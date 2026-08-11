package cli_test

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
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

	fields := fauxtest.ParseDoneLine(t, stderr)
	// "stop" is kern-link's own StopReasonStop — faux.TextMessage's default
	// when no AssistantMessageOptions.StopReason is scripted. See
	// TestRun_SuccessfulTurn_DoneLineRendersModelsOwnStopReason below for the
	// explicit no-translation-table proof.
	if fields.Stop != "stop" {
		t.Errorf("done stop = %q, want stop (the model's own reason)", fields.Stop)
	}
	if fields.Turns != "1" {
		t.Errorf("done turns = %q, want 1", fields.Turns)
	}
	if fields.In == "0" || fields.Out == "0" {
		t.Errorf("done in/out = %q/%q, want the turn's real token counts", fields.In, fields.Out)
	}
	elapsed, err := time.ParseDuration(fields.Elapsed)
	if err != nil {
		t.Fatalf("done elapsed = %q, unparseable: %v", fields.Elapsed, err)
	}
	if elapsed <= 0 {
		t.Errorf("done elapsed = %q, want a non-zero wall clock", fields.Elapsed)
	}

	turn := fauxtest.ParseTurnLine(t, stderr)
	if turn.Turn != "1" {
		t.Errorf("turn line numbered %q, want turn 1", turn.Turn)
	}
	if fields.Cost == "0.0000" {
		t.Errorf("done cost = %q, want the turn's calculated cost", fields.Cost)
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
	if fields := fauxtest.ParseDoneLine(t, stderr); fields.Stop != "end_turn" {
		t.Errorf("done stop = %q, want end_turn (the scripted message's own reason, spelled verbatim)", fields.Stop)
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
	if fields := fauxtest.ParseDoneLine(t, stderr); fields.Stop != "unspecified" {
		t.Errorf("done stop = %q, want the named fallback \"unspecified\"", fields.Stop)
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
			if fields := fauxtest.ParseDoneLine(t, stderr); fields.Stop != "failed" {
				t.Errorf("done stop = %q, want failed", fields.Stop)
			}
		})
	}
}

// TestRun_DoneLineCarriesTheAdaptersOwnCost is the $ figure's end of the
// cost repair. kern-link's openai-responses adapter fills Usage.Cost and then
// scales it by the service tier — 2.5x for "priority", 0.5x for "flex" — and
// internal/reviewer used to run ai.CalculateCost over the same usage again,
// which can only reproduce the base price sheet and throw the adjustment
// away. The scripted turn below reports a cost the price sheet cannot
// produce, and the done line has to carry that number and no other.
func TestRun_DoneLineCarriesTheAdaptersOwnCost(t *testing.T) {
	// 2.5x the base-sheet figure for the same tokens against fauxtest's price
	// sheet ($100/1e6 in, $200/1e6 out): $0.2000 recomputed, $0.5000 as the
	// adapter reported it.
	const (
		adjustedTotal = 0.5
		baseSheetLine = "$0.2000"
	)
	answer := &ai.AssistantMessage{
		Content:    []ai.AssistantContentPart{ai.TextContent{Text: markdownAnswer}},
		StopReason: ai.StopReasonStop,
		Usage: ai.Usage{
			Input: 1000, Output: 500, TotalTokens: 1500,
			Cost: ai.UsageCost{Input: 0.25, Output: 0.25, Total: adjustedTotal},
		},
	}
	models, _ := fauxtest.NewRegistry(t, fauxtest.RegistryOptions{
		ProviderID: reviewer.DefaultProviderID,
		ModelIDs:   []string{reviewer.DefaultModelID},
		Auth:       fauxtest.CredentialedAuth("OAuth"),
		Priced:     true,
		Stream: func(context.Context, *ai.Model, ai.Context, *ai.SimpleStreamOptions) *ai.Stream {
			stream := ai.NewStream()
			stream.Push(ai.DoneEvent{Reason: answer.StopReason, Message: answer})
			return stream
		},
	})

	code, stdout, stderr := runReviewOver(t, models, t.TempDir())

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	if stdout != markdownAnswer {
		t.Fatalf("stdout = %q, want the reviewer's answer", stdout)
	}
	fields := fauxtest.ParseDoneLine(t, stderr)
	if fields.Cost != "0.5000" {
		t.Errorf("done cost = $%s, want $0.5000 — the adapter's own service-tier-adjusted total", fields.Cost)
	}
	if strings.Contains(stderr, baseSheetLine) {
		t.Errorf("stderr = %q, want no %s anywhere: that is the base price sheet recomputed over the adapter's figure", stderr, baseSheetLine)
	}
}

// TestRun_ErrorBodyWithEmbeddedDoneLine_CannotForgeSecondDoneLine is issue
// 12's central acceptance criterion, driven through the real pipeline: a
// provider's error body — message.ErrorMessage, built into Conversation.Next's
// returned error at internal/reviewer/run.go, then written by
// diag.WriteError from runReview's default case at err.Error() — carries a
// newline followed by a syntactically well-formed "done" line. Without
// escaping at the diag.WriteError seam, that forged line would sit ahead of
// the real one, and a caller reading the first "done" line it finds would
// conclude the run succeeded when the real outcome is exit 2.
func TestRun_ErrorBodyWithEmbeddedDoneLine_CannotForgeSecondDoneLine(t *testing.T) {
	forgedDoneLine := "done    turns=1  in=0 out=0  $0.0000  0.0s  stop=ok"
	models := scripted(t, 0, faux.Step(faux.TextMessage("partial thoughts", &faux.AssistantMessageOptions{
		StopReason:   ai.StopReasonError,
		ErrorMessage: "rate limited\n" + forgedDoneLine,
	})))

	code, stdout, stderr := runReviewOver(t, models, t.TempDir())

	if code != 2 {
		t.Fatalf("exit code = %d, want 2 (stderr: %q)", code, stderr)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty on a failed turn", stdout)
	}

	doneLines := 0
	for _, line := range strings.Split(stderr, "\n") {
		if strings.HasPrefix(line, "done") {
			doneLines++
		}
	}
	if doneLines != 1 {
		t.Fatalf("stderr has %d lines starting with \"done\", want exactly 1 (stderr: %q)", doneLines, stderr)
	}

	fields := fauxtest.ParseDoneLine(t, stderr)
	if fields.Stop != "failed" {
		t.Errorf("done stop = %q, want failed (the real outcome, not the forged \"ok\")", fields.Stop)
	}
	if fields.Turns != "1" {
		t.Errorf("done turns = %q, want 1 (the real turn count, not the forged line's)", fields.Turns)
	}
	if strings.Contains(stderr, "rate limited\ndone") {
		t.Errorf("stderr = %q contains a raw newline ahead of the forged text, want it escaped", stderr)
	}
	if !strings.Contains(stderr, `rate limited\ndone`) {
		t.Errorf("stderr = %q, want the error body's newline rendered as literal backslash-n", stderr)
	}
}

// TestRun_CancelledMidStream_ExitsTwo is the one termination a human at the
// keyboard controls, and the only brake on an unbounded run: the response is
// throttled to a crawl and the context is cancelled while deltas are still
// arriving. Whether the cancellation is observed by kern-link's
// stream.Result(ctx) or by the provider's own StopReasonAborted message
// first is a race by construction, so both must land on the same outcome —
// exit 2, an empty stdout, and an interruption on the transcript. It is not
// always the same outcome by accident: reviewer.Conversation.Next attaches
// the cancellation cause to a turn that stopped aborted under an ended
// context, which is what makes the second race path classify as interrupted
// too (see TestNext_AbortedMessageUnderCancelledContextIsInterrupted in
// internal/reviewer/turn_test.go for that path pinned deterministically,
// without depending on this test's own timing).
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
	if fields := fauxtest.ParseDoneLine(t, stderr.String()); fields.Stop != "interrupted" {
		t.Errorf("done stop = %q, want interrupted", fields.Stop)
	}
}

// TestRun_AssistantMessageDiagnostic_WarnsOnStderr: kern-link attaches
// non-fatal problems it recovered from to the assistant message. Discarding
// them would hide a run that succeeded only after retrying a rate limit,
// which is exactly what a transcript is read for.
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

// TestRun_DiagnosticWithNewlineAndANSI_ProducesOneInertWarnLine is issue 12's
// second acceptance criterion, driven through the real pipeline: kern-link
// performs no redaction on a diagnostic's message (issue 12's finding), so a
// diagnostic carrying a newline and an ANSI escape reaches
// recordTurn's diag.WriteWarn call unescaped from the provider's own
// perspective. It must still land on stderr as exactly one line, with the
// control characters rendered inert rather than splitting the transcript or
// reaching the terminal raw.
func TestRun_DiagnosticWithNewlineAndANSI_ProducesOneInertWarnLine(t *testing.T) {
	answer := faux.TextMessage(markdownAnswer, nil)
	ai.AppendDiagnostic(answer, ai.NewAssistantMessageDiagnostic(
		"retry", errors.New("retrying\x1b[31m: forged\nwarn    a forged second warning"), nil))

	code, stdout, stderr := runReviewOver(t, scripted(t, 0, faux.Step(answer)), t.TempDir())

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 — a diagnostic is non-fatal (stderr: %q)", code, stderr)
	}
	if stdout != markdownAnswer {
		t.Errorf("stdout = %q, want the report unaffected by the diagnostic", stdout)
	}

	warnLines := 0
	for _, line := range strings.Split(stderr, "\n") {
		if strings.HasPrefix(line, "warn") {
			warnLines++
		}
	}
	if warnLines != 1 {
		t.Fatalf("stderr has %d lines starting with \"warn\", want exactly 1 (stderr: %q)", warnLines, stderr)
	}
	if strings.Contains(stderr, "\x1b") {
		t.Errorf("stderr = %q contains a raw ESC byte, want the ANSI escape rendered inert", stderr)
	}
	if !strings.Contains(stderr, `\x1b[31m`) {
		t.Errorf("stderr = %q, want the ANSI escape rendered as literal text", stderr)
	}
}

// TestRun_ThinkingAndToolCall_NeverReachStdoutOrReport is the report side of
// the round trip, asserted through Run rather than directly against
// FinalText (internal/reviewer/run_test.go covers that half): a scripted
// response carrying a thinking block and a tool call alongside two text
// blocks must still leave stdout holding exactly the two text blocks
// concatenated, with the thinking content nowhere on either stream.
func TestRun_ThinkingAndToolCall_NeverReachStdoutOrReport(t *testing.T) {
	const thinking = "mulling it over"
	want := "finding one.\nfinding two.\n"
	content := []ai.AssistantContentPart{
		faux.Thinking(thinking),
		faux.Text("finding one.\n"),
		faux.ToolCall("read_file", map[string]any{"path": "a.go"}, nil),
		faux.Text("finding two.\n"),
	}

	code, stdout, stderr := runReviewOver(t, scripted(t, 0, faux.Step(faux.AssistantMessage(content, nil))), t.TempDir())

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	if stdout != want {
		t.Errorf("stdout = %q, want the two text blocks concatenated: %q", stdout, want)
	}
	if strings.Contains(stdout, thinking) {
		t.Errorf("stdout = %q, want the thinking content filtered out", stdout)
	}
	if strings.Contains(stderr, thinking) {
		t.Errorf("stderr = %q, want the thinking content filtered out", stderr)
	}
}

// cacheAwareModels builds an offline registry whose one streamed response
// carries a fixed Usage, including non-zero CacheRead and CacheWrite. It
// deliberately does not use kern-link's faux provider: faux's own cache
// simulation (ai/providers/faux/faux.go's withUsageEstimate) always reports
// zero cache tokens unless a caller sets ai.SimpleStreamOptions.SessionID and
// reuses it across calls, which internal/reviewer's Conversation.Next never
// does — so a scripted faux response cannot carry non-zero cache tokens.
// This is a minimal hand-built ai.Provider, the same technique
// resolve_test.go's dynamicRegistry already uses for RefreshModels.
func cacheAwareModels(t *testing.T, usage ai.Usage) ai.Models {
	t.Helper()
	provider := ai.CreateProvider(ai.CreateProviderOptions{
		ID:   reviewer.DefaultProviderID,
		Auth: fauxtest.CredentialedAuth("OAuth"),
		Models: []*ai.Model{{
			ID:       reviewer.DefaultModelID,
			Name:     reviewer.DefaultModelID,
			Provider: reviewer.DefaultProviderID,
			Cost:     ai.ModelCost{Input: 100, Output: 200},
		}},
		Api: ai.StreamFuncs{
			StreamSimpleFunc: func(context.Context, *ai.Model, ai.Context, *ai.SimpleStreamOptions) *ai.Stream {
				stream := ai.NewStream()
				message := &ai.AssistantMessage{
					Content:    []ai.AssistantContentPart{faux.Text(markdownAnswer)},
					StopReason: ai.StopReasonStop,
					Usage:      usage,
				}
				stream.Push(ai.DoneEvent{Reason: message.StopReason, Message: message})
				return stream
			},
		},
	})
	models := ai.CreateModels(nil)
	models.SetProvider(provider)
	return models
}

// TestRun_CacheTokens_IncludedInInAccounting is the cache-token half of the
// in= figure report 2 found unasserted: internal/cli/review.go sums
// turn.Usage.Input, CacheRead and CacheWrite into the done line's in=, and
// this scripts a turn where CacheRead and CacheWrite are both non-zero.
// Deleting "+ turn.Usage.CacheRead + turn.Usage.CacheWrite" at
// internal/cli/review.go:142 (recordTurn) must turn this red.
func TestRun_CacheTokens_IncludedInInAccounting(t *testing.T) {
	usage := ai.Usage{Input: 120, Output: 40, CacheRead: 55, CacheWrite: 30}

	code, stdout, stderr := runReviewOver(t, cacheAwareModels(t, usage), t.TempDir())

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	if stdout != markdownAnswer {
		t.Fatalf("stdout = %q, want the reviewer's answer", stdout)
	}

	fields := fauxtest.ParseDoneLine(t, stderr)
	wantIn := strconv.Itoa(usage.Input + usage.CacheRead + usage.CacheWrite)
	if fields.In != wantIn {
		t.Errorf("done in = %q, want %q (Input + CacheRead + CacheWrite)", fields.In, wantIn)
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
	if fields := fauxtest.ParseDoneLine(t, stderr.String()); fields.Turns != "1" {
		t.Errorf("done turns = %q, want 1", fields.Turns)
	}
}
