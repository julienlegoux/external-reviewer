package cli_test

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/julienlegoux/kern-link/ai"
	"github.com/julienlegoux/kern-link/ai/providers/faux"

	"github.com/julienlegoux/external-reviewer/internal/cli"
	"github.com/julienlegoux/external-reviewer/internal/fauxtest"
	"github.com/julienlegoux/external-reviewer/internal/reviewer"
)

// markdownAnswer is what a scripted reviewer returns on the success path. It
// is deliberately awkward: fenced code, a trailing newline and a non-ASCII
// character, because stdout is a contract with the calling skill and the
// binary must hand the model's text over byte for byte rather than
// normalising it (SPECS § Interfaces).
const markdownAnswer = "# Review\n\n- `main.go:12` — the loop never terminates\n\n```go\nfor {}\n```\n"

// turnLineRE parses the "turn" line the round trip emits, so a test can
// assert on real token counts without matching an elapsed time that is real.
var turnLineRE = regexp.MustCompile(
	`(?m)^turn (\d+)\s+tools=(\d+)\s+in=(\d+) out=(\d+)\s+\$([0-9.]+)\s+(\S+)\s*$`)

// runReviewOver invokes a syntactically valid review of repoPath against
// models, returning the exit code and both streams.
func runReviewOver(t *testing.T, models ai.Models, repoPath string) (int, string, string) {
	t.Helper()

	var stdout, stderr bytes.Buffer
	code := cli.RunForTest([]string{"review", "--prompt", "review this", repoPath}, strings.NewReader(""), &stdout, &stderr, models)
	return code, stdout.String(), stderr.String()
}

// scripted builds the offline registry for a round trip: the hard-coded
// reviewer, a credential that resolves, and responses scripted onto
// kern-link's in-process faux provider. tokensPerSecond throttles the
// streaming so a run takes measurable wall clock; 0 streams as fast as the
// machine allows.
func scripted(t *testing.T, tokensPerSecond float64, responses ...faux.ResponseStep) ai.Models {
	t.Helper()
	models, handle := registryWith(t, fauxtest.CredentialedAuth("OAuth"), nil, tokensPerSecond, reviewer.DefaultModelID)
	handle.SetResponses(responses...)
	return models
}

// TestRun_SuccessfulTurn_WritesFinalTextVerbatim is the walking skeleton's
// reason to exist: one streamed round trip, and the reviewer's own markdown
// on stdout with no envelope of any kind around it.
func TestRun_SuccessfulTurn_WritesFinalTextVerbatim(t *testing.T) {
	models := scripted(t, 100, faux.Step(faux.TextMessage(markdownAnswer, nil)))

	code, stdout, stderr := runReviewOver(t, models, t.TempDir())

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	if stdout != markdownAnswer {
		t.Errorf("stdout = %q, want the final assistant text byte for byte: %q", stdout, markdownAnswer)
	}

	fields := parseDoneLine(t, stderr)
	// "stop" is kern-link's own StopReasonStop — faux.TextMessage's default
	// when no AssistantMessageOptions.StopReason is scripted. See
	// TestRun_SuccessfulTurn_DoneLineRendersModelsOwnStopReason below for the
	// explicit no-translation-table proof.
	if fields.stop != "stop" {
		t.Errorf("done stop = %q, want stop (the model's own reason)", fields.stop)
	}
	if fields.turns != "1" {
		t.Errorf("done turns = %q, want 1", fields.turns)
	}
	if fields.in == "0" || fields.out == "0" {
		t.Errorf("done in/out = %q/%q, want the turn's real token counts", fields.in, fields.out)
	}
	elapsed, err := time.ParseDuration(fields.elapsed)
	if err != nil {
		t.Fatalf("done elapsed = %q, unparseable: %v", fields.elapsed, err)
	}
	if elapsed <= 0 {
		t.Errorf("done elapsed = %q, want a non-zero wall clock", fields.elapsed)
	}

	turn := turnLineRE.FindStringSubmatch(stderr)
	if turn == nil {
		t.Fatalf("stderr = %q, want a turn line matching %s", stderr, turnLineRE)
	}
	if turn[1] != "1" {
		t.Errorf("turn line numbered %q, want turn 1", turn[1])
	}
	if fields.cost == "0.0000" {
		t.Errorf("done cost = %q, want the turn's calculated cost", fields.cost)
	}
	if strings.Contains(stderr, "# Review") {
		t.Errorf("stderr = %q, want the reviewer's prose kept off the transcript", stderr)
	}
}

// TestRun_SuccessfulTurn_DoneLineRendersModelsOwnStopReason is the
// no-translation-table proof SPECS § Interfaces requires: the scripted
// message's StopReason is a value no CLI word matches, and it must still
// reach the done line byte for byte — not "ok", not any hand-written
// vocabulary.
func TestRun_SuccessfulTurn_DoneLineRendersModelsOwnStopReason(t *testing.T) {
	models := scripted(t, 0, faux.Step(faux.TextMessage(markdownAnswer, &faux.AssistantMessageOptions{
		StopReason: "end_turn",
	})))

	code, stdout, stderr := runReviewOver(t, models, t.TempDir())

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	if stdout != markdownAnswer {
		t.Fatalf("stdout = %q, want the reviewer's answer", stdout)
	}
	if fields := parseDoneLine(t, stderr); fields.stop != "end_turn" {
		t.Errorf("done stop = %q, want end_turn (the scripted message's own reason, spelled verbatim)", fields.stop)
	}
}

// TestRun_SuccessfulTurn_EmptyStopReasonRendersNamedFallback pins the named
// fallback SPECS § Interfaces requires so stop= can never render with
// nothing after it: a scripted response built directly (bypassing
// faux.TextMessage's non-empty default) carries no StopReason at all.
func TestRun_SuccessfulTurn_EmptyStopReasonRendersNamedFallback(t *testing.T) {
	noStopReason := faux.StepFunc(func(context.Context, ai.Context, *ai.StreamOptions, *faux.State, *ai.Model) (*ai.AssistantMessage, error) {
		return &ai.AssistantMessage{Content: []ai.AssistantContentPart{faux.Text(markdownAnswer)}}, nil
	})
	models := scripted(t, 0, noStopReason)

	code, stdout, stderr := runReviewOver(t, models, t.TempDir())

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	if stdout != markdownAnswer {
		t.Fatalf("stdout = %q, want the reviewer's answer", stdout)
	}
	if fields := parseDoneLine(t, stderr); fields.stop != "unspecified" {
		t.Errorf("done stop = %q, want the named fallback \"unspecified\"", fields.stop)
	}
}

// TestRun_FailedTurn_ExitsTwo covers every way one round trip can end without
// a usable report. All four are "reached and then failed", so all four are
// exit 2 with an empty stdout and a reason on stderr — including the two that
// stream text first, which is why stdout is written once at the end rather
// than as deltas arrive.
func TestRun_FailedTurn_ExitsTwo(t *testing.T) {
	tests := []struct {
		name      string
		responses []faux.ResponseStep
		wantOnErr string
	}{
		{
			name: "the request itself fails",
			responses: []faux.ResponseStep{faux.StepFunc(
				func(context.Context, ai.Context, *ai.StreamOptions, *faux.State, *ai.Model) (*ai.AssistantMessage, error) {
					return nil, errors.New("provider returned 503")
				})},
			wantOnErr: "503",
		},
		{
			name: "the turn stops with reason error",
			responses: []faux.ResponseStep{faux.Step(faux.TextMessage("partial thoughts", &faux.AssistantMessageOptions{
				StopReason:   ai.StopReasonError,
				ErrorMessage: "context window exceeded",
			}))},
			wantOnErr: "context window exceeded",
		},
		{
			name: "the turn stops with reason aborted",
			responses: []faux.ResponseStep{faux.Step(faux.TextMessage("partial thoughts", &faux.AssistantMessageOptions{
				StopReason:   ai.StopReasonAborted,
				ErrorMessage: "request was aborted upstream",
			}))},
			wantOnErr: "aborted",
		},
		{
			name:      "the final assistant message carries no text",
			responses: []faux.ResponseStep{faux.Step(faux.TextMessage("", nil))},
			wantOnErr: "no text",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, stdout, stderr := runReviewOver(t, scripted(t, 0, tc.responses...), t.TempDir())

			if code != 2 {
				t.Errorf("exit code = %d, want 2 (stderr: %q)", code, stderr)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty on a failed turn", stdout)
			}
			if !strings.Contains(stderr, tc.wantOnErr) {
				t.Errorf("stderr = %q, want it to name the failure (%q)", stderr, tc.wantOnErr)
			}
			if fields := parseDoneLine(t, stderr); fields.stop != "failed" {
				t.Errorf("done stop = %q, want failed", fields.stop)
			}
		})
	}
}

// TestRun_CancelledMidStream_ExitsTwo is the one termination a human at the
// keyboard controls, and the only brake on an unbounded run: the response is
// throttled to a crawl and the context is cancelled while deltas are still
// arriving. Whether the cancellation is observed by kern-link's
// stream.Result(ctx) or by the provider's own StopReasonAborted message
// first is a race by construction, so both must land on the same outcome —
// exit 2, an empty stdout, and an interruption on the transcript. It is not
// always the same outcome by accident: asInterrupted (review.go) is what
// makes the second race path also classify as interrupted, added after a
// windows-latest `go test -race` run caught the outcome diverging (see
// TestAsInterrupted_ReclassifiesFailingTurnWhenContextEnded in run_test.go
// for that race pinned deterministically, without depending on this test's
// own timing).
func TestRun_CancelledMidStream_ExitsTwo(t *testing.T) {
	models := scripted(t, 1, faux.Step(faux.TextMessage(strings.Repeat("a long answer that is still streaming. ", 20), nil)))

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(25*time.Millisecond, cancel)
	defer cancel()

	var stdout, stderr bytes.Buffer
	code := cli.RunWithModelsForTest(ctx, []string{"review", "--prompt", "review this", t.TempDir()}, strings.NewReader(""), &stdout, &stderr, models)

	if code != 2 {
		t.Errorf("exit code = %d, want 2 (stderr: %q)", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty on an interrupted run", stdout.String())
	}
	if !strings.Contains(strings.ToLower(stderr.String()), "interrupt") {
		t.Errorf("stderr = %q, want an interruption reason", stderr.String())
	}
	if fields := parseDoneLine(t, stderr.String()); fields.stop != "interrupted" {
		t.Errorf("done stop = %q, want interrupted", fields.stop)
	}
}

// TestRun_AssistantMessageDiagnostic_WarnsOnStderr: kern-link attaches
// non-fatal problems it recovered from to the assistant message, already
// redacted. Discarding them would hide a run that succeeded only after
// retrying a rate limit, which is exactly what a transcript is read for.
func TestRun_AssistantMessageDiagnostic_WarnsOnStderr(t *testing.T) {
	answer := faux.TextMessage(markdownAnswer, nil)
	ai.AppendDiagnostic(answer, ai.NewAssistantMessageDiagnostic(
		"retry", errors.New("rate limited, retried after 2s"), nil))

	code, stdout, stderr := runReviewOver(t, scripted(t, 0, faux.Step(answer)), t.TempDir())

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 — a diagnostic is non-fatal (stderr: %q)", code, stderr)
	}
	if stdout != markdownAnswer {
		t.Errorf("stdout = %q, want the report unaffected by the diagnostic", stdout)
	}
	warned := false
	for _, line := range strings.Split(stderr, "\n") {
		if strings.HasPrefix(line, "warn ") && strings.Contains(line, "rate limited, retried after 2s") {
			warned = true
		}
	}
	if !warned {
		t.Errorf("stderr = %q, want a warn line carrying the diagnostic", stderr)
	}
}

// TestRun_SuccessfulReview_TouchesNoFile is the product's central guarantee
// asserted as behaviour rather than as a lint rule: the calling skill owns
// the report file, so a full successful review over a real tree must leave
// that tree byte-identical, down to the modification times. The forbidigo
// guard in .golangci.yml says the write API is not in the source; this says
// nothing the binary reaches writes either.
func TestRun_SuccessfulReview_TouchesNoFile(t *testing.T) {
	repo := t.TempDir()
	writeFixture(t, repo, "README.md", "# fixture\n")
	writeFixture(t, repo, filepath.Join("cmd", "app", "main.go"), "package main\n\nfunc main() {}\n")
	writeFixture(t, repo, filepath.Join("docs", "notes.txt"), "notes\n")

	before := snapshotTree(t, repo)

	code, stdout, stderr := runReviewOver(t, scripted(t, 0, faux.Step(faux.TextMessage(markdownAnswer, nil))), repo)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	if stdout != markdownAnswer {
		t.Fatalf("stdout = %q, want the review to have actually run", stdout)
	}

	after := snapshotTree(t, repo)
	for name, was := range before {
		is, present := after[name]
		if !present {
			t.Errorf("%s was deleted by the run", name)
			continue
		}
		if is != was {
			t.Errorf("%s changed: %+v -> %+v", name, was, is)
		}
	}
	for name := range after {
		if _, present := before[name]; !present {
			t.Errorf("%s was created by the run", name)
		}
	}
}

// entry is one node of a directory tree, in the terms that change when
// something writes to it.
//
// Directories carry only their existence. On Windows two consecutive walks of
// an untouched tree report different directory modification times — the walk
// itself is enough to refresh them — so asserting on those would fail on the
// CI matrix for a reason that has nothing to do with this binary. Nothing is
// lost: a directory's timestamp only ever moves because an entry inside it
// was created, renamed or deleted, and the entry set below already catches
// every one of those.
type entry struct {
	dir      bool
	size     int64
	mode     fs.FileMode
	modified time.Time
}

// snapshotTree records every node under root, keyed by its path relative to
// root, so two snapshots taken around a run can be compared entry by entry.
func snapshotTree(t *testing.T, root string) map[string]entry {
	t.Helper()

	out := map[string]entry{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if d.IsDir() {
			out[filepath.ToSlash(name)] = entry{dir: true}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		out[filepath.ToSlash(name)] = entry{
			size:     info.Size(),
			mode:     info.Mode(),
			modified: info.ModTime(),
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}
	return out
}

// writeFixture creates one file of the throwaway repository a review runs
// over, making the parent directories it needs.
func writeFixture(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("creating fixture directory for %s: %v", name, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writing fixture %s: %v", name, err)
	}
}

// liveModelEnv opts a run of this package into the one test that spends
// money. It is checked at runtime rather than behind a build tag, so
// `go test ./...` stays green offline on a clean checkout and CI — which
// never sets it — always skips.
const liveModelEnv = "EXTERNAL_REVIEWER_LIVE_MODEL"

// TestRun_LiveModel_ReturnsMarkdown is the automated echo of the epic's
// hand-run risk retirement: the same Run, against the machine's real
// credential store and the real hard-coded reviewer. It asserts only what is
// deterministic about a foreign model — exit 0 and a non-empty stdout —
// because a report's text is not, and a golden file of one would fail for the
// wrong reasons.
func TestRun_LiveModel_ReturnsMarkdown(t *testing.T) {
	if os.Getenv(liveModelEnv) == "" {
		t.Skipf("%s is not set: skipping the live model round trip", liveModelEnv)
	}

	var stdout, stderr bytes.Buffer
	code := cli.Run(
		[]string{"review", "--prompt", "Reply with a one-line markdown heading and nothing else.", t.TempDir()},
		strings.NewReader(""), &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr.String())
	}
	if stdout.Len() == 0 {
		t.Errorf("stdout is empty, want the reviewer's markdown")
	}
	if fields := parseDoneLine(t, stderr.String()); fields.turns != "1" {
		t.Errorf("done turns = %q, want 1", fields.turns)
	}
}
