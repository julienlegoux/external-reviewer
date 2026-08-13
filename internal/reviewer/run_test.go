package reviewer_test

import (
	"context"
	"testing"

	"github.com/julienlegoux/kern-link/ai"
	"github.com/julienlegoux/kern-link/ai/providers/faux"

	"github.com/julienlegoux/external-reviewer/internal/fauxtest"
	"github.com/julienlegoux/external-reviewer/internal/reviewer"
)

// reviewerModel builds the offline registry a Conversation test resolves
// against and streams from, returning the one model it serves so a test can
// call NewConversation directly without going through Resolver.
func reviewerModel(t *testing.T) (ai.Models, *faux.Handle, *ai.Model) {
	t.Helper()
	models, handle := fauxtest.NewRegistry(t, fauxtest.RegistryOptions{
		ProviderID: fixtureProvider,
		ModelIDs:   []string{fixtureModel},
		Auth:       fauxtest.CredentialedAuth("OAuth"),
	})
	model := models.GetModel(fixtureProvider, fixtureModel)
	if model == nil {
		t.Fatalf("GetModel(%s, %s) = nil, want the model the registry was built with", fixtureProvider, fixtureModel)
	}
	return models, handle, model
}

// TestConversation_Next_SendsTaskAsUserMessageVerbatim is the request-side
// half of the round trip report 2 found unobserved: NewConversation seeds the
// user message from the caller's task with ai.UserText, and this asserts
// against what the provider actually received — from inside the scripted
// step, following the pattern already used at
// internal/cli/roundtrip_test.go:118 — not against the value NewConversation
// was called with. Mutating ai.UserText(task) to ai.UserText("") at
// run.go:53 must turn this red.
func TestConversation_Next_SendsTaskAsUserMessageVerbatim(t *testing.T) {
	const task = "review this repository for security issues, focusing on auth"

	var gotMessages []ai.Message
	capture := faux.StepFunc(func(_ context.Context, chat ai.Context, _ *ai.StreamOptions, _ *faux.State, _ *ai.Model) (*ai.AssistantMessage, error) {
		gotMessages = chat.Messages
		return faux.TextMessage("acknowledged", nil), nil
	})

	models, handle, model := reviewerModel(t)
	handle.SetResponses(capture)

	conv := reviewer.NewConversation(models, model, task)
	if _, err := conv.Next(context.Background()); err != nil {
		t.Fatalf("Next() error = %v, want nil", err)
	}

	if len(gotMessages) != 1 {
		t.Fatalf("the provider received %d messages, want 1 (the seeded user message)", len(gotMessages))
	}
	userMessage, ok := gotMessages[0].(*ai.UserMessage)
	if !ok {
		t.Fatalf("the provider's first message = %T, want *ai.UserMessage", gotMessages[0])
	}
	if got := userMessage.Content.Text(); got != task {
		t.Errorf("the prompt reaching the model = %q, want the task verbatim: %q", got, task)
	}
}

// TestConversation_Next_AccumulatesMessagesInOrder pins the messages
// accumulator Epic 2's multi-turn loop lands on: after a completed turn, the
// next request carries the user message and the prior assistant message, in
// that order. Deleting c.messages = append(c.messages, message) at run.go:87
// must turn this red — the second turn's request would carry only the
// original user message.
func TestConversation_Next_AccumulatesMessagesInOrder(t *testing.T) {
	const task = "review this repository"
	const firstAnswer = "first turn's answer"

	var secondCallMessages []ai.Message
	models, handle, model := reviewerModel(t)
	handle.SetResponses(
		faux.Step(faux.TextMessage(firstAnswer, nil)),
		faux.StepFunc(func(_ context.Context, chat ai.Context, _ *ai.StreamOptions, _ *faux.State, _ *ai.Model) (*ai.AssistantMessage, error) {
			secondCallMessages = chat.Messages
			return faux.TextMessage("second turn's answer", nil), nil
		}),
	)

	conv := reviewer.NewConversation(models, model, task)
	if _, err := conv.Next(context.Background()); err != nil {
		t.Fatalf("first Next() error = %v, want nil", err)
	}
	if _, err := conv.Next(context.Background()); err != nil {
		t.Fatalf("second Next() error = %v, want nil", err)
	}

	if len(secondCallMessages) != 2 {
		t.Fatalf("the second turn's request carried %d messages, want 2 (user, assistant): %+v", len(secondCallMessages), secondCallMessages)
	}
	userMessage, ok := secondCallMessages[0].(*ai.UserMessage)
	if !ok || userMessage.Content.Text() != task {
		t.Errorf("the first message = %+v, want the original user task %q", secondCallMessages[0], task)
	}
	assistantMessage, ok := secondCallMessages[1].(*ai.AssistantMessage)
	if !ok || reviewer.FinalText(assistantMessage) != firstAnswer {
		t.Errorf("the second message = %+v, want the first turn's assistant reply %q", secondCallMessages[1], firstAnswer)
	}
}

// TestFinalText_ConcatenatesAndFiltersContent is the report side of the
// round trip: FinalText's doc comment (run.go:100-102) promises the text
// blocks concatenated in order with nothing inserted between them, and that
// thinking blocks and tool calls never reach the answer. Inserting a break
// after out.WriteString(text.Text) at run.go:119 must turn the first case
// red — it would return only the first text block instead of both
// concatenated.
func TestFinalText_ConcatenatesAndFiltersContent(t *testing.T) {
	tests := []struct {
		name    string
		content []ai.AssistantContentPart
		want    string
	}{
		{
			name:    "two text blocks concatenate in order with nothing inserted between",
			content: []ai.AssistantContentPart{faux.Text("first "), faux.Text("second")},
			want:    "first second",
		},
		{
			name: "a thinking block and a tool call are filtered from the answer",
			content: []ai.AssistantContentPart{
				faux.Thinking("mulling it over"),
				faux.Text("finding one.\n"),
				faux.ToolCall("read_file", map[string]any{"path": "a.go"}, nil),
				faux.Text("finding two.\n"),
			},
			want: "finding one.\nfinding two.\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			message := faux.AssistantMessage(tc.content, nil)
			if got := reviewer.FinalText(message); got != tc.want {
				t.Errorf("FinalText() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestFinalText_NilMessage_ReturnsEmptyString guards the doc comment's other
// promise: a turn that produced no assistant message at all (Message is nil)
// reports no text rather than panicking.
func TestFinalText_NilMessage_ReturnsEmptyString(t *testing.T) {
	if got := reviewer.FinalText(nil); got != "" {
		t.Errorf("FinalText(nil) = %q, want empty", got)
	}
}
