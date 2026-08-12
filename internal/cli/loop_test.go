package cli_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/julienlegoux/kern-link/ai"
	"github.com/julienlegoux/kern-link/ai/providers/faux"

	"github.com/julienlegoux/external-reviewer/internal/cli"
	"github.com/julienlegoux/external-reviewer/internal/fauxtest"
	"github.com/julienlegoux/external-reviewer/internal/reviewer"
)

// toolLineRE parses the "tool" lines a dispatch writes, so the assertions
// below read the tool name and the rendered arguments rather than matching a
// whole transcript verbatim.
var toolLineRE = regexp.MustCompile(`(?m)^tool {4}(\S+)(.*)$`)

// toolLines returns every "tool" line in a transcript as name + rendered
// arguments.
func toolLines(text string) [][2]string {
	matches := toolLineRE.FindAllStringSubmatch(text, -1)
	lines := make([][2]string, 0, len(matches))
	for _, m := range matches {
		lines = append(lines, [2]string{m[1], strings.TrimSpace(m[2])})
	}
	return lines
}

// listCall scripts one assistant turn that asks for `list` and nothing else,
// which is the only tool this epic has wired so far.
func listCall(pattern string) faux.ResponseStep {
	return faux.Step(faux.AssistantMessage(
		[]ai.AssistantContentPart{faux.ToolCall("list", map[string]any{"pattern": pattern}, nil)},
		&faux.AssistantMessageOptions{StopReason: ai.StopReasonToolUse},
	))
}

// repoWith builds a repository directory holding the named files, each with
// its own path as content so a listing is distinguishable from a read.
func repoWith(t *testing.T, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range names {
		full := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			t.Fatalf("creating %s: %v", name, err)
		}
		if err := os.WriteFile(full, []byte(name+"\n"), 0o600); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}
	return dir
}

// TestRun_ThreeTurnConversation_DispatchesToolsAndEndsOnFinalText is the
// issue's headline criterion: a scripted tool call → tool result → tool call
// → tool result → final text conversation runs to completion, the reviewer's
// own markdown reaches stdout verbatim, and the done line counts all three
// turns rather than the one Epic 1 took.
func TestRun_ThreeTurnConversation_DispatchesToolsAndEndsOnFinalText(t *testing.T) {
	models := scripted(t, 0,
		listCall("**/*.go"),
		listCall("**/*.md"),
		faux.Step(faux.TextMessage(markdownAnswer, nil)),
	)

	code, stdout, stderr := runReviewOver(t, models, repoWith(t, "main.go", "docs/plan.md"))

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	if stdout != markdownAnswer {
		t.Errorf("stdout = %q, want the final assistant text byte for byte: %q", stdout, markdownAnswer)
	}
	if fields := fauxtest.ParseDoneLine(t, stderr); fields.Turns != "3" {
		t.Errorf("done turns = %q, want 3 (stderr: %q)", fields.Turns, stderr)
	}
	if got := len(toolLineRE.FindAllString(stderr, -1)); got != 2 {
		t.Errorf("tool lines = %d, want 2 — one per dispatch (stderr: %q)", got, stderr)
	}
	if got := strings.Count(stderr, "\nturn "); got+strings.Count(stderr, "turn 1") == 0 {
		t.Errorf("turn lines missing from stderr: %q", stderr)
	}
}

// TestRun_ToolResult_CarriesTheRealListOutput proves the loop and the tool are
// exercised together rather than through a stub: what the second request
// carries back is the enumeration of a real repository directory, produced by
// the real `list` tool over the run's real confinement.
func TestRun_ToolResult_CarriesTheRealListOutput(t *testing.T) {
	var secondRequest ai.Context
	models, handle := registryWith(t, fauxtest.CredentialedAuth("OAuth"), nil, 0, reviewer.DefaultModelID)
	handle.SetResponses(
		listCall("**/*"),
		faux.StepFunc(func(_ context.Context, chat ai.Context, _ *ai.StreamOptions, _ *faux.State, _ *ai.Model) (*ai.AssistantMessage, error) {
			secondRequest = chat
			return faux.TextMessage(markdownAnswer, nil), nil
		}),
	)

	code, _, stderr := runReviewOver(t, models, repoWith(t, "main.go", "docs/plan.md"))

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}

	var result *ai.ToolResultMessage
	for _, message := range secondRequest.Messages {
		if tr, ok := message.(*ai.ToolResultMessage); ok {
			result = tr
		}
	}
	if result == nil {
		t.Fatalf("the second request carried no ToolResultMessage: %#v", secondRequest.Messages)
	}
	if result.IsError {
		t.Errorf("tool result IsError = true, want a successful listing: %q", toolResultText(result))
	}
	text := toolResultText(result)
	for _, want := range []string{"main.go", "docs/plan.md"} {
		if !strings.Contains(text, want) {
			t.Errorf("tool result = %q, want it to contain %q — the real list output", text, want)
		}
	}
	if strings.Contains(text, `\`) {
		t.Errorf("tool result = %q, want /-separated wire paths", text)
	}
}

// TestRun_UnknownToolName_IsAToolErrorAndTheRunContinues pins the first half
// of the two-level failure rule: a tool the registry never declared is the
// model's mistake to learn from, not a run-ending failure.
func TestRun_UnknownToolName_IsAToolErrorAndTheRunContinues(t *testing.T) {
	var secondRequest ai.Context
	models, handle := registryWith(t, fauxtest.CredentialedAuth("OAuth"), nil, 0, reviewer.DefaultModelID)
	handle.SetResponses(
		faux.Step(faux.AssistantMessage(
			[]ai.AssistantContentPart{faux.ToolCall("write_file", map[string]any{"path": "main.go"}, nil)},
			&faux.AssistantMessageOptions{StopReason: ai.StopReasonToolUse},
		)),
		faux.StepFunc(func(_ context.Context, chat ai.Context, _ *ai.StreamOptions, _ *faux.State, _ *ai.Model) (*ai.AssistantMessage, error) {
			secondRequest = chat
			return faux.TextMessage(markdownAnswer, nil), nil
		}),
	)

	code, stdout, stderr := runReviewOver(t, models, repoWith(t, "main.go"))

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 — an unknown tool ends a turn, not a run (stderr: %q)", code, stderr)
	}
	if stdout != markdownAnswer {
		t.Errorf("stdout = %q, want the reviewer's answer after it corrected itself", stdout)
	}

	var result *ai.ToolResultMessage
	for _, message := range secondRequest.Messages {
		if tr, ok := message.(*ai.ToolResultMessage); ok {
			result = tr
		}
	}
	if result == nil {
		t.Fatalf("the second request carried no ToolResultMessage: %#v", secondRequest.Messages)
	}
	if !result.IsError {
		t.Errorf("tool result IsError = false, want true for an undeclared tool name")
	}
	if text := toolResultText(result); !strings.Contains(text, "write_file") || !strings.Contains(text, "list") {
		t.Errorf("tool result = %q, want it to name both the invented tool and the ones that exist", text)
	}
}

// TestRun_RefusedToolArguments_AreAToolErrorAndTheRunContinues drives the
// same rule through the real `list` tool: an argument the declared schema
// rules out comes back as a refusal naming what was wrong, the run continues,
// and the next scripted turn still exits 0.
func TestRun_RefusedToolArguments_AreAToolErrorAndTheRunContinues(t *testing.T) {
	var secondRequest ai.Context
	models, handle := registryWith(t, fauxtest.CredentialedAuth("OAuth"), nil, 0, reviewer.DefaultModelID)
	handle.SetResponses(
		listCall("/etc/*"),
		faux.StepFunc(func(_ context.Context, chat ai.Context, _ *ai.StreamOptions, _ *faux.State, _ *ai.Model) (*ai.AssistantMessage, error) {
			secondRequest = chat
			return faux.TextMessage(markdownAnswer, nil), nil
		}),
	)

	code, stdout, stderr := runReviewOver(t, models, repoWith(t, "main.go"))

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	if stdout != markdownAnswer {
		t.Errorf("stdout = %q, want the reviewer's answer", stdout)
	}

	var result *ai.ToolResultMessage
	for _, message := range secondRequest.Messages {
		if tr, ok := message.(*ai.ToolResultMessage); ok {
			result = tr
		}
	}
	if result == nil {
		t.Fatalf("the second request carried no ToolResultMessage: %#v", secondRequest.Messages)
	}
	if !result.IsError {
		t.Errorf("tool result IsError = false, want true for a refused pattern")
	}
	if text := toolResultText(result); !strings.Contains(text, "repo-relative") {
		t.Errorf("tool result = %q, want it to name the rule that refused", text)
	}
}

// TestRun_ToolDeclarationsReachTheModelOnEveryRequest pins the other half of
// the dispatch contract: the model can only ask for what it was told about,
// so the registry's declarations travel on every turn, not just the first.
func TestRun_ToolDeclarationsReachTheModelOnEveryRequest(t *testing.T) {
	var declared [][]ai.Tool
	models, handle := registryWith(t, fauxtest.CredentialedAuth("OAuth"), nil, 0, reviewer.DefaultModelID)
	handle.SetResponses(
		faux.StepFunc(func(_ context.Context, chat ai.Context, _ *ai.StreamOptions, _ *faux.State, _ *ai.Model) (*ai.AssistantMessage, error) {
			declared = append(declared, chat.Tools)
			return faux.AssistantMessage(
				[]ai.AssistantContentPart{faux.ToolCall("list", nil, nil)},
				&faux.AssistantMessageOptions{StopReason: ai.StopReasonToolUse},
			), nil
		}),
		faux.StepFunc(func(_ context.Context, chat ai.Context, _ *ai.StreamOptions, _ *faux.State, _ *ai.Model) (*ai.AssistantMessage, error) {
			declared = append(declared, chat.Tools)
			return faux.TextMessage(markdownAnswer, nil), nil
		}),
	)

	if code, _, stderr := runReviewOver(t, models, repoWith(t, "main.go")); code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}

	if len(declared) != 2 {
		t.Fatalf("requests = %d, want 2", len(declared))
	}
	for i, tools := range declared {
		if len(tools) == 0 {
			t.Fatalf("request %d declared no tools", i+1)
		}
		if tools[0].Name != "list" {
			t.Errorf("request %d declared %q first, want list", i+1, tools[0].Name)
		}
		if tools[0].Description == "" || len(tools[0].Parameters) == 0 {
			t.Errorf("request %d declared list with no description or schema", i+1)
		}
	}
}

// TestRun_FailingTurnMidLoop_ExitsTwo pins the second half of the two-level
// failure rule: a turn that fails after tools have already been dispatched
// ends the run at exit 2, with nothing on stdout and a reason on stderr.
func TestRun_FailingTurnMidLoop_ExitsTwo(t *testing.T) {
	models := scripted(t, 0,
		listCall("**/*"),
		faux.Step(faux.AssistantMessage(nil, &faux.AssistantMessageOptions{
			StopReason:   ai.StopReasonError,
			ErrorMessage: "the provider gave up mid-review",
		})),
	)

	code, stdout, stderr := runReviewOver(t, models, repoWith(t, "main.go"))

	if code != 2 {
		t.Fatalf("exit code = %d, want 2 (stderr: %q)", code, stderr)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty on a failing turn", stdout)
	}
	if !strings.Contains(stderr, "the provider gave up mid-review") {
		t.Errorf("stderr = %q, want the reason on it", stderr)
	}
	fields := fauxtest.ParseDoneLine(t, stderr)
	if fields.Stop != "failed" {
		t.Errorf("done stop = %q, want failed", fields.Stop)
	}
	if fields.Turns != "2" {
		t.Errorf("done turns = %q, want 2 — the failing turn is still a turn that cost money", fields.Turns)
	}
}

// TestRun_ToolLine_NamesTheToolAndItsArgumentsInWireForm pins the
// instrumentation SPECS § Interfaces fixes: one line per dispatch, naming the
// tool and what it was asked for, /-separated whatever the host OS spells
// paths with.
func TestRun_ToolLine_NamesTheToolAndItsArgumentsInWireForm(t *testing.T) {
	models := scripted(t, 0,
		listCall("docs/**/*.md"),
		faux.Step(faux.TextMessage(markdownAnswer, nil)),
	)

	code, _, stderr := runReviewOver(t, models, repoWith(t, "docs/plan.md"))

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	lines := toolLines(stderr)
	if len(lines) != 1 {
		t.Fatalf("tool lines = %d, want 1 (stderr: %q)", len(lines), stderr)
	}
	if lines[0][0] != "list" {
		t.Errorf("tool line named %q, want list", lines[0][0])
	}
	if !strings.Contains(lines[0][1], "docs/**/*.md") {
		t.Errorf("tool line arguments = %q, want the pattern the model asked for", lines[0][1])
	}
	if strings.Contains(lines[0][1], `\`) {
		t.Errorf("tool line arguments = %q, want /-separated wire paths", lines[0][1])
	}
}

// TestRun_ExceededBound_ReturnsWhatTheRunHasAtExitZero drives the bounds seam
// end to end, with the number a test puts there and no invocation does yet:
// the run stops at its ceiling, the text it had reaches stdout, the exit code
// is 0 because nothing failed, and the done line names the new stop reason.
func TestRun_ExceededBound_ReturnsWhatTheRunHasAtExitZero(t *testing.T) {
	const partial = "# Review\n\nstill reading\n"
	models := scripted(t, 0,
		faux.Step(faux.AssistantMessage(
			[]ai.AssistantContentPart{faux.Text(partial), faux.ToolCall("list", nil, nil)},
			&faux.AssistantMessageOptions{StopReason: ai.StopReasonToolUse},
		)),
		faux.Step(faux.TextMessage(markdownAnswer, nil)),
	)

	var stdout, stderr bytes.Buffer
	code := cli.RunWithBoundsForTest(
		context.Background(),
		[]string{"review", "--allow", ".", "--prompt", "review this", repoWith(t, "main.go")},
		strings.NewReader(""), &stdout, &stderr, models,
		reviewer.Bounds{MaxTurns: 1},
	)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 — an exceeded bound is not a failure (stderr: %q)", code, stderr.String())
	}
	if stdout.String() != partial {
		t.Errorf("stdout = %q, want the last assistant text the run had: %q", stdout.String(), partial)
	}
	fields := fauxtest.ParseDoneLine(t, stderr.String())
	if fields.Stop != "bounds" {
		t.Errorf("done stop = %q, want bounds", fields.Stop)
	}
	if fields.Turns != "1" {
		t.Errorf("done turns = %q, want 1 — the second turn is never taken", fields.Turns)
	}
	if !strings.Contains(stderr.String(), "turn bound") {
		t.Errorf("stderr = %q, want it to say which bound stopped the run", stderr.String())
	}
}

// TestRun_UnsetBounds_AreWhatAnOrdinaryInvocationGets pins what this epic
// actually ships: no invocation can set a bound, so nothing bounds a run. The
// same conversation that stops at one turn above runs to its end here.
func TestRun_UnsetBounds_AreWhatAnOrdinaryInvocationGets(t *testing.T) {
	models := scripted(t, 0,
		listCall("**/*"),
		listCall("**/*.go"),
		faux.Step(faux.TextMessage(markdownAnswer, nil)),
	)

	code, stdout, stderr := runReviewOver(t, models, repoWith(t, "main.go"))

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	if stdout != markdownAnswer {
		t.Errorf("stdout = %q, want the run to have reached its own end", stdout)
	}
	if fields := fauxtest.ParseDoneLine(t, stderr); fields.Stop == "bounds" {
		t.Errorf("done stop = bounds, want the model's own reason — this epic ships no values")
	}
}

// TestRun_CancelledBetweenTurns_ExitsTwoWithADoneLine is the loop's own
// cancellation path — the one Epic 1 could not have: a run interrupted after a
// turn completed and its tools were dispatched, rather than mid-stream. It
// must still terminate through the same seam every other path does, so the
// transcript ends with exactly one done line saying which of failure and
// interruption happened.
//
// The stream is scripted rather than queued on faux's response queue, and the
// cancellation lands *after* the terminal event has already been pushed. That
// is what makes this an assertion instead of a race: Conversation.Next
// deliberately prefers a message that is already there over a cancellation
// beside it, so turn 1 completes and the loop's own check is what stops the
// run before turn 2.
func TestRun_CancelledBetweenTurns_ExitsTwoWithADoneLine(t *testing.T) {
	const partial = "# Review\n\nstill reading\n"
	ctx, cancel := context.WithCancel(context.Background())
	var calls atomic.Int64

	models, _ := fauxtest.NewRegistry(t, fauxtest.RegistryOptions{
		ProviderID: reviewer.DefaultProviderID,
		ModelIDs:   []string{reviewer.DefaultModelID},
		Auth:       fauxtest.CredentialedAuth("OAuth"),
		Priced:     true,
		Stream: func(context.Context, *ai.Model, ai.Context, *ai.SimpleStreamOptions) *ai.Stream {
			stream := ai.NewStream()
			// A second call means the loop kept going through a cancelled
			// context: answer with a final message so the test fails on the
			// exit code rather than hanging.
			if calls.Add(1) > 1 {
				stream.Push(ai.DoneEvent{Reason: ai.StopReasonStop, Message: &ai.AssistantMessage{
					Content:    []ai.AssistantContentPart{ai.TextContent{Text: markdownAnswer}},
					StopReason: ai.StopReasonStop,
				}})
				return stream
			}
			stream.Push(ai.DoneEvent{Reason: ai.StopReasonToolUse, Message: &ai.AssistantMessage{
				Content: []ai.AssistantContentPart{
					ai.TextContent{Text: partial},
					ai.ToolCall{ID: "call-1", Name: "list"},
				},
				StopReason: ai.StopReasonToolUse,
			}})
			cancel()
			return stream
		},
	})

	var stdout, stderr bytes.Buffer
	code := cli.RunWithModelsForTest(
		ctx,
		[]string{"review", "--allow", ".", "--prompt", "review this", repoWith(t, "main.go")},
		strings.NewReader(""), &stdout, &stderr, models,
	)

	if code != 2 {
		t.Fatalf("exit code = %d, want 2 (stderr: %q)", code, stderr.String())
	}
	if stdout.String() != "" {
		t.Errorf("stdout = %q, want empty on an interrupted run", stdout.String())
	}
	fields := fauxtest.ParseDoneLine(t, stderr.String())
	if fields.Stop != "interrupted" {
		t.Errorf("done stop = %q, want interrupted", fields.Stop)
	}
	if fields.Turns != "1" {
		t.Errorf("done turns = %q, want 1 — the completed turn still counts", fields.Turns)
	}
	if got := strings.Count(stderr.String(), "\ndone    "); got != 1 {
		t.Errorf("done lines = %d, want exactly 1 (stderr: %q)", got, stderr.String())
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("requests = %d, want 1 — the second turn is never sent under a cancelled context", got)
	}
}

// toolResultText concatenates a tool result's text blocks, which is how the
// loop hands a tool's answer to the model.
func toolResultText(result *ai.ToolResultMessage) string {
	var out strings.Builder
	for _, part := range result.Content {
		if text, ok := part.(ai.TextContent); ok {
			out.WriteString(text.Text)
		}
	}
	return out.String()
}
