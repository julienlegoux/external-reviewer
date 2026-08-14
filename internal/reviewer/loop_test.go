package reviewer_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/julienlegoux/kern-link/ai"
	"github.com/julienlegoux/kern-link/ai/providers/faux"

	"github.com/julienlegoux/external-reviewer/internal/confine"
	"github.com/julienlegoux/external-reviewer/internal/diag"
	"github.com/julienlegoux/external-reviewer/internal/fauxtest"
	"github.com/julienlegoux/external-reviewer/internal/reviewer"
	"github.com/julienlegoux/external-reviewer/internal/tools"
)

// loopAnswer is the report a scripted loop ends on.
const loopAnswer = "# Review\n\nno leads\n"

// queuedLoop builds a Loop over the faux provider's own response queue, which
// is what makes a multi-turn conversation scriptable turn by turn. It returns
// the loop and the buffer its diagnostics land in.
func queuedLoop(t *testing.T, toolSet reviewer.ToolSet, responses ...faux.ResponseStep) (*reviewer.Loop, *bytes.Buffer, *diag.State) {
	t.Helper()

	models, handle := fauxtest.NewRegistry(t, fauxtest.RegistryOptions{
		ProviderID: fixtureProvider,
		ModelIDs:   []string{fixtureModel},
		Auth:       fauxtest.CredentialedAuth("OAuth"),
		Priced:     true,
	})
	handle.SetResponses(responses...)

	model := models.GetModel(fixtureProvider, fixtureModel)
	if model == nil {
		t.Fatalf("model %s/%s missing from the registry", fixtureProvider, fixtureModel)
	}

	stderr := &bytes.Buffer{}
	state := diag.NewState()
	loop := &reviewer.Loop{
		Conversation: reviewer.NewConversation(models, model, callerSystemPrompt, "review this"),
		Tools:        toolSet,
		State:        state,
		Stderr:       stderr,
	}
	return loop, stderr, state
}

// toolUse scripts one assistant turn that asks for name and stops for tools.
func toolUse(name string, arguments map[string]any, text string) faux.ResponseStep {
	content := []ai.AssistantContentPart{}
	if text != "" {
		content = append(content, faux.Text(text))
	}
	content = append(content, faux.ToolCall(name, arguments, nil))
	return faux.Step(faux.AssistantMessage(content, &faux.AssistantMessageOptions{
		StopReason: ai.StopReasonToolUse,
	}))
}

// echoTool is a tool that answers with what it was asked, so a loop test can
// assert what came back without depending on any real tool's formatting.
func echoTool() tools.Tool {
	return tools.Tool{
		Declaration: ai.Tool{
			Name:        "echo",
			Description: "Echo the text argument back.",
			Parameters:  ai.JSONSchema(`{"type":"object","properties":{"text":{"type":"string"}}}`),
		},
		Handler: func(_ context.Context, arguments map[string]any) tools.Result {
			text, _ := arguments["text"].(string)
			return tools.Result{Text: "echo: " + text}
		},
	}
}

// TestLoop_Run_StopsAtTheTurnBoundAndReturnsWhatItHas is the bounds seam with
// a number in it, which only a test ever puts there: the third turn is never
// taken, and the run ends with the text it had rather than with a failure.
func TestLoop_Run_StopsAtTheTurnBoundAndReturnsWhatItHas(t *testing.T) {
	loop, stderr, state := queuedLoop(t, tools.NewRegistry(echoTool()),
		toolUse("echo", map[string]any{"text": "one"}, "first thoughts\n"),
		toolUse("echo", map[string]any{"text": "two"}, "second thoughts\n"),
		faux.Step(faux.TextMessage(loopAnswer, nil)),
	)
	loop.Bounds = reviewer.Bounds{MaxTurns: 2}

	report, err := loop.Run(context.Background())

	if err != nil {
		t.Fatalf("Run() error = %v, want nil — an exceeded bound is not a failure", err)
	}
	if report != "second thoughts\n" {
		t.Errorf("Run() = %q, want the last assistant text the run actually had", report)
	}
	if state.Turns != 2 {
		t.Errorf("state.Turns = %d, want 2 — the third turn is never taken", state.Turns)
	}
	if state.StopReason != "bounds" {
		t.Errorf("state.StopReason = %q, want bounds", state.StopReason)
	}
	if !strings.Contains(stderr.String(), "turn") {
		t.Errorf("stderr = %q, want the turn lines on it", stderr.String())
	}
}

// TestLoop_Run_BoundBeatsAPendingToolUse is the first half of the termination
// order SPECS fixes: the bound is checked at the top of the turn, so a model
// that has just asked for another tool does not get one.
func TestLoop_Run_BoundBeatsAPendingToolUse(t *testing.T) {
	loop, _, state := queuedLoop(t, tools.NewRegistry(echoTool()),
		toolUse("echo", map[string]any{"text": "one"}, "partial\n"),
		faux.Step(faux.TextMessage(loopAnswer, nil)),
	)
	loop.Bounds = reviewer.Bounds{MaxTurns: 1}

	report, err := loop.Run(context.Background())

	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if report != "partial\n" {
		t.Errorf("Run() = %q, want the text from the bounded turn, not the second turn's answer", report)
	}
	if state.Turns != 1 {
		t.Errorf("state.Turns = %d, want 1 — the pending tool-use turn loses to the bound", state.Turns)
	}
	if state.StopReason != "bounds" {
		t.Errorf("state.StopReason = %q, want bounds", state.StopReason)
	}
}

// TestLoop_Run_CancellationBeatsAPendingToolUse is the second half of the
// order: with no bound in play, a context that ended between turns stops the
// loop before the next request rather than after it.
func TestLoop_Run_CancellationBeatsAPendingToolUse(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	loop, _, state := queuedLoop(t, tools.NewRegistry(cancellingTool(cancel)),
		toolUse("stop", nil, "partial\n"),
		faux.Step(faux.TextMessage(loopAnswer, nil)),
	)

	_, err := loop.Run(ctx)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v, want a context.Canceled — a cancelled run is not a completed one", err)
	}
	if state.Turns != 1 {
		t.Errorf("state.Turns = %d, want 1 — the second turn is never requested", state.Turns)
	}
}

// TestLoop_Run_BoundBeatsCancellation pins the order between the two
// terminations themselves: with both a bound exceeded and the context already
// cancelled, the bound wins, so a run that reached its ceiling still returns
// its report rather than being reported as interrupted.
func TestLoop_Run_BoundBeatsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	loop, _, state := queuedLoop(t, tools.NewRegistry(cancellingTool(cancel)),
		toolUse("stop", nil, "partial\n"),
		faux.Step(faux.TextMessage(loopAnswer, nil)),
	)
	loop.Bounds = reviewer.Bounds{MaxTurns: 1}

	report, err := loop.Run(ctx)

	if err != nil {
		t.Fatalf("Run() error = %v, want nil — the bound is checked before cancellation", err)
	}
	if report != "partial\n" {
		t.Errorf("Run() = %q, want the bounded run's own text", report)
	}
	if state.StopReason != "bounds" {
		t.Errorf("state.StopReason = %q, want bounds — the bound wins over the cancellation", state.StopReason)
	}
}

// cancellingTool cancels the run's context from inside a dispatch, which is
// the only way to make "the context ended between turns" a fact of the test
// rather than a race against the scheduler.
func cancellingTool(cancel context.CancelFunc) tools.Tool {
	return tools.Tool{
		Declaration: ai.Tool{
			Name:        "stop",
			Description: "Cancel the run.",
			Parameters:  ai.JSONSchema(`{"type":"object","properties":{}}`),
		},
		Handler: func(_ context.Context, _ map[string]any) tools.Result {
			cancel()
			return tools.Result{Text: "cancelled"}
		},
	}
}

// TestLoop_Run_UnsetBoundsNeverStopARun is the shipped configuration: this
// epic ships the seam with no numbers in it, so a run ends where the model
// ends it (SCOPE defers the values to Epic 3's measurements).
func TestLoop_Run_UnsetBoundsNeverStopARun(t *testing.T) {
	loop, _, state := queuedLoop(t, tools.NewRegistry(echoTool()),
		toolUse("echo", map[string]any{"text": "one"}, ""),
		toolUse("echo", map[string]any{"text": "two"}, ""),
		toolUse("echo", map[string]any{"text": "three"}, ""),
		faux.Step(faux.TextMessage(loopAnswer, nil)),
	)

	report, err := loop.Run(context.Background())

	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if report != loopAnswer {
		t.Errorf("Run() = %q, want %q", report, loopAnswer)
	}
	if state.Turns != 4 {
		t.Errorf("state.Turns = %d, want 4 — an unset bound stops nothing", state.Turns)
	}
	if state.StopReason == "bounds" {
		t.Errorf("state.StopReason = bounds, want the model's own reason on an unbounded run")
	}
}

// TestLoop_Run_ElapsedBoundStopsTheRun covers the second quantity the seam
// watches. The run's start is moved back an hour rather than the bound being
// dialled down to nanoseconds: two time.Now calls in immediate succession can
// return the same instant on Windows, which would make a sub-millisecond
// bound a coin flip rather than an assertion.
func TestLoop_Run_ElapsedBoundStopsTheRun(t *testing.T) {
	loop, stderr, state := queuedLoop(t, tools.NewRegistry(echoTool()),
		toolUse("echo", map[string]any{"text": "one"}, "partial\n"),
		faux.Step(faux.TextMessage(loopAnswer, nil)),
	)
	state.Start = time.Now().Add(-time.Hour)
	loop.Bounds = reviewer.Bounds{MaxElapsed: time.Minute}

	report, err := loop.Run(context.Background())

	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if state.Turns != 0 {
		t.Errorf("state.Turns = %d, want 0 — the bound is checked before the first request too", state.Turns)
	}
	if report != "" {
		t.Errorf("Run() = %q, want empty — no turn was ever taken", report)
	}
	if state.StopReason != "bounds" {
		t.Errorf("state.StopReason = %q, want bounds", state.StopReason)
	}
	if !strings.Contains(stderr.String(), "elapsed bound") {
		t.Errorf("stderr = %q, want it to name the elapsed bound, not just say bounds", stderr.String())
	}
}

// TestLoop_Run_AllowListRefusalReachesTheModelAsAToolError is the confinement
// half of the two-level failure rule, asserted through the real
// confine.Scope: a tool that resolves a path outside the granted subtrees
// hands the reviewer the rule that refused, and the run carries on to the next
// turn.
func TestLoop_Run_AllowListRefusalReachesTheModelAsAToolError(t *testing.T) {
	scope := grantedScope(t)
	defer func() { _ = scope.Close() }()

	var secondRequest ai.Context
	loop, stderr, _ := queuedLoop(t, tools.NewRegistry(resolveTool(scope)),
		toolUse("resolve", map[string]any{"path": "secrets/key.pem"}, ""),
		faux.StepFunc(func(_ context.Context, chat ai.Context, _ *ai.StreamOptions, _ *faux.State, _ *ai.Model) (*ai.AssistantMessage, error) {
			secondRequest = chat
			return faux.TextMessage(loopAnswer, nil), nil
		}),
	)

	report, err := loop.Run(context.Background())

	if err != nil {
		t.Fatalf("Run() error = %v, want nil — a refused path ends a turn, not a run", err)
	}
	if report != loopAnswer {
		t.Errorf("Run() = %q, want the model's next turn to have been dispatched", report)
	}
	result := lastToolResult(t, secondRequest)
	if !result.IsError {
		t.Errorf("tool result IsError = false, want true for a path outside the allow-list")
	}
	if text := resultText(result); !strings.Contains(text, confine.ErrOutsideAllowList.Error()) {
		t.Errorf("tool result = %q, want it to name the rule that refused (%q)", text, confine.ErrOutsideAllowList)
	}
	if !strings.Contains(stderr.String(), "secrets/key.pem") {
		t.Errorf("stderr = %q, want the tool line naming the wire path asked for", stderr.String())
	}
}

// grantedScope opens a scope over a repository that grants src/ only, so
// anything under secrets/ is inside the root and outside the allow-list —
// the one refusal a reviewer is meant to learn from and retry around.
func grantedScope(t *testing.T) *confine.Scope {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"src/main.go", "secrets/key.pem"} {
		full := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			t.Fatalf("creating %s: %v", name, err)
		}
		if err := os.WriteFile(full, []byte("x\n"), 0o600); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}
	scope, err := confine.OpenScope(dir, []string{"src"})
	if err != nil {
		t.Fatalf("OpenScope: %v", err)
	}
	return scope
}

// resolveTool is the smallest possible path-taking tool: it asks the run's own
// Scope whether a path may be read and returns the answer. Issues 04–06 add
// the real ones; this exists so the loop's contract with a confinement
// refusal is asserted against confine itself rather than a hand-written
// message.
func resolveTool(scope *confine.Scope) tools.Tool {
	return tools.Tool{
		Declaration: ai.Tool{
			Name:        "resolve",
			Description: "Resolve a repo-relative path through the run's confinement.",
			Parameters:  ai.JSONSchema(`{"type":"object","properties":{"path":{"type":"string"}}}`),
		},
		Handler: func(_ context.Context, arguments map[string]any) tools.Result {
			path, _ := arguments["path"].(string)
			cleaned, err := scope.Resolve(path)
			if err != nil {
				return tools.Result{Text: err.Error(), IsError: true}
			}
			return tools.Result{Text: cleaned}
		},
	}
}

// lastToolResult returns the last tool result a request carried, which is how
// the loop's accumulation is observed from outside the package: not through
// an accessor added for the test, but through what the model was actually
// sent next.
func lastToolResult(t *testing.T, request ai.Context) *ai.ToolResultMessage {
	t.Helper()
	var last *ai.ToolResultMessage
	for _, message := range request.Messages {
		if result, ok := message.(*ai.ToolResultMessage); ok {
			last = result
		}
	}
	if last == nil {
		t.Fatalf("the request carries no ToolResultMessage: %#v", request.Messages)
	}
	return last
}

func resultText(result *ai.ToolResultMessage) string {
	var out strings.Builder
	for _, part := range result.Content {
		if text, ok := part.(ai.TextContent); ok {
			out.WriteString(text.Text)
		}
	}
	return out.String()
}
