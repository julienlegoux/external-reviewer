package cli_test

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/julienlegoux/kern-link/ai"
	"github.com/julienlegoux/kern-link/ai/providers/faux"

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
