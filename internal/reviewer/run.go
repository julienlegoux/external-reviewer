package reviewer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/julienlegoux/kern-link/ai"
)

// Turn is one completed round trip. It is returned even when the turn
// failed, because a run that died after spending tokens is exactly the run
// whose numbers need recording (SPECS § Interfaces): Message is nil only when
// the request produced no assistant message at all.
type Turn struct {
	Message *ai.AssistantMessage
	// Usage is the turn's own accounting with Cost filled in by
	// ai.CalculateCost against the model's price sheet.
	Usage   ai.Usage
	Elapsed time.Duration
	// Tools counts the tool calls the assistant asked for. Epic 1 declares no
	// tools, so it is 0 until the loop and the tool set arrive.
	Tools int
}

// Conversation is the accumulating []ai.Message a review is made of, plus the
// resolved model each turn is sent to. Epic 1 takes exactly one turn; the
// multi-turn loop calls Next repeatedly against the same value, which is why
// the message history lives here rather than on the caller's stack.
type Conversation struct {
	models   ai.Models
	model    *ai.Model
	messages []ai.Message
}

// NewConversation seeds a conversation with the caller's task and nothing
// else. No system prompt (the caller owns it, and Epic 1 sends none), no file
// listing, no repository summary: the reviewer chooses what to read, and
// choosing is the reason the loop exists.
func NewConversation(models ai.Models, model *ai.Model, task string) *Conversation {
	return &Conversation{
		models: models,
		model:  model,
		messages: []ai.Message{&ai.UserMessage{
			Content:   ai.UserText(task),
			Timestamp: time.Now().UnixMilli(),
		}},
	}
}

// Next runs one streamed round trip, appends the assistant message to the
// conversation, and returns what the turn cost.
//
// The error it returns is a *failing turn*, which ends a run (exit 2): the
// stream failing outright, or a StopReason of "error" or "aborted". A
// cancelled context is reported as an error wrapping ctx.Err() whichever way
// the cancellation surfaced — kern-link's Result observing it, or the
// provider aborting the request first, a race by construction that callers
// must not have to distinguish.
func (c *Conversation) Next(ctx context.Context) (Turn, error) {
	start := time.Now()

	chat := ai.Context{Messages: c.messages}
	stream := c.models.StreamSimple(ctx, c.model, chat, &ai.SimpleStreamOptions{})
	// Draining the events rather than only awaiting Result is the whole
	// reason SPECS chose StreamSimple over CompleteSimple: this is where a
	// turn becomes observable while it is still running, and where Epic 2
	// reads tool calls off the wire to write its "tool" lines. Epic 1 has
	// nothing to say about a text delta — assistant prose is deliberately
	// kept off stderr — so the body is empty and the loop's job is to keep
	// pace with the stream until it terminates or ctx is cancelled.
	for range stream.Events(ctx) {
	}

	message, err := stream.Result(ctx)
	if err != nil {
		return Turn{Elapsed: time.Since(start)}, fmt.Errorf("streaming the reviewer's turn: %w", err)
	}

	c.messages = append(c.messages, message)
	usage := message.Usage
	ai.CalculateCost(c.model, &usage)
	turn := Turn{Message: message, Usage: usage, Elapsed: time.Since(start), Tools: toolCalls(message)}

	// A cancelled context outranks whatever the message says. kern-link's
	// Result observes the cancellation first almost every time, so this is
	// belt to that braces: when the provider's own aborted message wins the
	// race instead, an interrupted run must still be reported as interrupted
	// rather than as a turn that failed on its own.
	if err := ctx.Err(); err != nil {
		return turn, fmt.Errorf("the reviewer's turn was interrupted: %w", err)
	}
	if message.StopReason == ai.StopReasonError || message.StopReason == ai.StopReasonAborted {
		return turn, fmt.Errorf("the reviewer's turn stopped with reason %q: %s", message.StopReason, message.ErrorMessage)
	}
	return turn, nil
}

// FinalText is the report: the assistant message's text blocks concatenated
// in order, with nothing inserted between them. Thinking blocks and tool
// calls are not part of the answer and never reach stdout.
func FinalText(message *ai.AssistantMessage) string {
	if message == nil {
		return ""
	}
	var out strings.Builder
	for _, part := range message.Content {
		if text, ok := part.(ai.TextContent); ok {
			out.WriteString(text.Text)
		}
	}
	return out.String()
}

func toolCalls(message *ai.AssistantMessage) int {
	n := 0
	for _, part := range message.Content {
		if _, ok := part.(ai.ToolCall); ok {
			n++
		}
	}
	return n
}
