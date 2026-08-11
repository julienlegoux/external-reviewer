package cli_test

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/julienlegoux/kern-link/ai"
	"github.com/julienlegoux/kern-link/ai/providers/faux"

	"github.com/julienlegoux/external-reviewer/internal/cli"
	"github.com/julienlegoux/external-reviewer/internal/fauxtest"
	"github.com/julienlegoux/external-reviewer/internal/reviewer"
)

// blockingReader models stdin with nothing typed into it yet — a terminal a
// human hasn't pressed a key at, the scenario this issue's acceptance
// criteria describes. Read never returns; the goroutine reading it is
// abandoned once ctx wins the race in readPromptFromStdin (see that
// function's own doc comment for the mechanism and what happens to the
// goroutine left behind).
type blockingReader struct{ block chan struct{} }

func newBlockingReader() blockingReader { return blockingReader{block: make(chan struct{})} }

func (r blockingReader) Read([]byte) (int, error) {
	<-r.block
	return 0, io.EOF
}

// TestRun_Review_StdinCancelledMidRead_ExitsInterrupted is the deterministic
// stand-in for a real Ctrl-C at a terminal prompt: delivering a genuine
// SIGINT is not portable to the windows-latest CI runner (the same
// constraint issue 03 of Epic 1 documented beside the signal.NotifyContext
// wiring), so this drives the inner run directly with a context that ends
// while the stdin read is still blocked, through RunWithModelsForTest — the
// same seam TestRun_CancelledMidStream_ExitsTwo (roundtrip_test.go) uses for
// the model-call half of cancellation. The hand-run terminal transcript for
// the real SIGINT is recorded separately in the PR body, as the acceptance
// criteria require.
func TestRun_Review_StdinCancelledMidRead_ExitsInterrupted(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(25*time.Millisecond, cancel)
	defer cancel()

	var stdout, stderr bytes.Buffer
	code := cli.RunWithModelsForTest(ctx, []string{"review", t.TempDir()}, newBlockingReader(), &stdout, &stderr,
		registry(t, fauxtest.CredentialedAuth("OAuth"), nil, reviewer.DefaultModelID))

	if code != 2 {
		t.Errorf("exit code = %d, want 2 (stderr: %q)", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
	if fields := fauxtest.ParseDoneLine(t, stderr.String()); fields.Stop != "interrupted" {
		t.Errorf("done stop = %q, want interrupted", fields.Stop)
	}
	if n := strings.Count(stderr.String(), "error:  "); n != 1 {
		t.Errorf("stderr = %q, want exactly one error: line", stderr.String())
	}
	assertOnlyKnownPrefixedLines(t, stderr.String())
}

// repeatingReader serves an endless stream of the same byte without ever
// materializing more than one Read call's worth of it — how
// TestRun_Review_StdinOverTheBound_NamedError drives a stdin body larger
// than the byte bound without allocating a bound-sized fixture in the test
// source, the acceptance criterion's own wording.
type repeatingReader byte

func (b repeatingReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = byte(b)
	}
	return len(p), nil
}

// TestRun_Review_StdinOverTheBound_NamedError proves the stdin read is
// bounded rather than buffering an unbounded pipe whole: a body larger than
// the limit is rejected at exit 2 with one error: line naming the limit,
// long before anything resembling 8GB.bin (the acceptance criteria's own
// example) could be fully read into memory.
func TestRun_Review_StdinOverTheBound_NamedError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cli.RunForTest([]string{"review", t.TempDir()}, repeatingReader('a'), &stdout, &stderr,
		registry(t, fauxtest.CredentialedAuth("OAuth"), nil, reviewer.DefaultModelID))

	if code != 2 {
		t.Fatalf("exit code = %d, want 2 (stderr: %q)", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
	lines := errorLines(stderr.String())
	if len(lines) != 1 {
		t.Fatalf("stderr carries %d error: lines, want exactly 1: %q", len(lines), stderr.String())
	}
	if !strings.Contains(lines[0], "1048576") {
		t.Errorf("error line = %q, want it to name the byte limit (1048576)", lines[0])
	}
	assertOnlyKnownPrefixedLines(t, stderr.String())
}

// TestRun_Review_PromptWhitespaceReachesModelVerbatim is the trim's own
// boundary: strings.TrimSpace decides emptiness, and nothing else — the
// prompt the model actually receives is the caller's own text, leading and
// trailing whitespace included. The assertion runs from inside a scripted
// faux step, capturing the exact ai.UserMessage the round trip sent, on
// issue 06's pattern of asserting the request rather than only the
// response.
func TestRun_Review_PromptWhitespaceReachesModelVerbatim(t *testing.T) {
	const raw = "  please review this carefully  \n"
	var captured string
	capture := faux.StepFunc(func(_ context.Context, chat ai.Context, _ *ai.StreamOptions, _ *faux.State, _ *ai.Model) (*ai.AssistantMessage, error) {
		user, ok := chat.Messages[0].(*ai.UserMessage)
		if !ok || user.Content.Plain == nil {
			t.Fatalf("chat.Messages[0] = %#v, want a *ai.UserMessage carrying plain text", chat.Messages[0])
		}
		captured = *user.Content.Plain
		return faux.TextMessage(scriptedAnswer, nil), nil
	})
	models := scripted(t, 0, capture)

	var stdout, stderr bytes.Buffer
	code := cli.RunForTest([]string{"review", t.TempDir()}, strings.NewReader(raw), &stdout, &stderr, models)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr.String())
	}
	if captured != raw {
		t.Errorf("prompt reaching the model = %q, want the caller's own text untouched: %q", captured, raw)
	}
}
