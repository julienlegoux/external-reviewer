package reviewer

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/julienlegoux/kern-link/ai"
)

// ErrStreamTimeout classifies a turn that ended at DefaultStreamTimeout
// rather than at anything the model or the caller did: the provider accepted
// the stream and then went quiet. It is deliberately kept off the
// context.Canceled / context.DeadlineExceeded vocabulary that internal/cli
// reads as an interruption — a deadline this package imposed on itself is not
// a run a human stopped, and SPECS § Interfaces already has the word for it
// ("failed": reached and then unusable for any other reason).
var ErrStreamTimeout = errors.New("the provider sent nothing before the stream timeout")

// DefaultStreamTimeout bounds one round trip end to end.
//
// It exists because kern-link v0.1.1 arms no deadline of its own on either
// transport this binary reaches: the SSE path builds an http.Client with a
// zero Timeout, and the codex WebSocket path — the default for the hard-coded
// reviewer — takes its idle read deadline from StreamOptions.Timeout and arms
// none at all while that is zero, its 15 s connect timeout covering the
// handshake only. A backend that accepts the connection and then stops
// sending frames would otherwise block the event loop forever, with no
// output, no done line and no exit code.
//
// Ten minutes is not a run bound (Epic 2 and Epic 3 own turn, cost and
// elapsed ceilings) — it is the transport's "this connection is dead" line,
// set an order of magnitude above the longest plausible single reviewing turn
// on a large repository with reasoning enabled, so it can only ever fire on a
// connection that has genuinely stopped rather than on a slow answer. It
// bounds the whole round trip rather than the idle gap between frames because
// v0.1.1 exposes no idle-only knob at the ai.Models seam.
const DefaultStreamTimeout = 10 * time.Minute

// resultGrace is how long Next will still wait for a terminal event after the
// stream's context has ended.
//
// It is what makes the precedence rule below decidable at all: kern-link's
// Stream.Result selects between the stream's result channel and ctx.Done()
// (ai/stream.go:155), so with a message already there and a cancellation
// already landed both cases are ready and Go picks one uniformly at random.
// Asking a second time under a context that has not ended removes the coin
// flip — a ready result wins every time — and gives an adapter that is still
// unwinding a moment to push the in-band aborted event it owes, which is what
// turns a mid-stream Ctrl-C into an aborted message rather than a bare
// context error.
//
// A result that is already there is returned immediately, so this is only
// ever spent on a turn that has failed anyway. That is what buys the margin:
// half a second is orders of magnitude more than the goroutine hop it waits
// on needs, even on a CI runner under -race, and the whole cost of being
// generous is half a second added to a run that is ending badly regardless.
const resultGrace = 500 * time.Millisecond

// Turn is one completed round trip. It is returned even when the turn
// failed, because a run that died after spending tokens is exactly the run
// whose numbers need recording (SPECS § Interfaces): Message is nil only when
// the request produced no assistant message at all.
type Turn struct {
	Message *ai.AssistantMessage
	// Usage is the turn's own accounting, carrying the cost the adapter
	// itself reported — already adjusted for the service tier it was billed
	// at — and falling back to ai.CalculateCost against the model's price
	// sheet only for an adapter that reported none (see accounted).
	Usage   ai.Usage
	Elapsed time.Duration
	// Tools counts the tool calls the assistant asked for. Epic 1 declares no
	// tools, so it is 0 until the loop and the tool set arrive.
	Tools int
	// StopReason is Message.StopReason surfaced onto Turn itself, verbatim as
	// kern-link spells it, so a caller never has to nil-check Message to read
	// why the model stopped. It is the zero value when Message is nil (the
	// stream failed outright). SPECS § Interfaces renders it on the done
	// line's success path with no translation table between the two
	// vocabularies.
	StopReason ai.StopReason
}

// Conversation is the accumulating []ai.Message a review is made of, plus the
// resolved model each turn is sent to. Epic 1 takes exactly one turn; the
// multi-turn loop calls Next repeatedly against the same value, which is why
// the message history lives here rather than on the caller's stack.
type Conversation struct {
	models   ai.Models
	model    *ai.Model
	messages []ai.Message

	// StreamTimeout bounds each round trip Next takes. NewConversation sets
	// it to DefaultStreamTimeout; anything at or below zero is read as that
	// default too, so no Conversation is ever unbounded. Tests dial it down
	// to milliseconds — it is the one knob that makes "a silent provider
	// terminates" assertable without a ten-minute test.
	StreamTimeout time.Duration
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
		StreamTimeout: DefaultStreamTimeout,
	}
}

// Next runs one streamed round trip, appends the assistant message to the
// conversation, and returns what the turn cost.
//
// The error it returns is a *failing turn*, which ends a run (exit 2): the
// stream failing outright, a StopReason of "error" or "aborted", or the round
// trip outliving StreamTimeout. Three precedence rules decide what that error
// says, and each one is a defect this function used to have:
//
//  1. The adapter's own Usage.Cost wins over the model's price sheet.
//     kern-link's openai-responses adapter computes the cost and then scales
//     it by the service tier, so recomputing it here could only throw the
//     adjustment away. The price sheet is the fallback for an adapter that
//     reported no cost at all, never a correction of one that did.
//  2. A terminal message the stream has already produced wins over a context
//     that ended at the same instant — see resultGrace. A finished, billed
//     report is not discarded because a Ctrl-C landed a millisecond late.
//  3. When there is no complete message to prefer, the caller's own
//     cancellation wins over this package's stream timeout, and both win over
//     whatever error the stream itself reported — endedBecause is the single
//     statement of that order. internal/cli's runReview remains the one place
//     "was this interrupted?" is *decided*; this only makes sure the cause is
//     there for it to read, on the path where the provider's own aborted
//     message reaches Result before the cancellation does and Result therefore
//     returns no error at all.
func (c *Conversation) Next(ctx context.Context) (Turn, error) {
	start := time.Now()

	// A context that has already ended never opens a stream. There is no
	// complete message to prefer yet (rule 2 below), so nothing is lost, and
	// a request sent under a dead context is one nobody will read the answer
	// to — it would still be counted, leaving token totals on a done line for
	// a path that never reached a model.
	if err := ctx.Err(); err != nil {
		return Turn{Elapsed: time.Since(start)}, fmt.Errorf("streaming the reviewer's turn: %w", err)
	}

	timeout := c.streamTimeout()
	turnCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	chat := ai.Context{Messages: c.messages}
	// The timeout is handed to the adapter as well as held here: kern-link's
	// transports arm their own read deadlines from StreamOptions.Timeout and
	// arm none while it is zero, so this is what lets a dead connection be
	// dropped by the layer that owns the socket rather than merely abandoned
	// by this one.
	stream := c.models.StreamSimple(turnCtx, c.model, chat, &ai.SimpleStreamOptions{
		StreamOptions: ai.StreamOptions{Timeout: timeout},
	})
	// Draining the events rather than only awaiting Result is the whole
	// reason SPECS chose StreamSimple over CompleteSimple: this is where a
	// turn becomes observable while it is still running, and where Epic 2
	// reads tool calls off the wire to write its "tool" lines. Epic 1 has
	// nothing to say about a text delta — assistant prose is deliberately
	// kept off stderr — so the body is empty and the loop's job is to keep
	// pace with the stream until it terminates or turnCtx ends.
	for range stream.Events(turnCtx) {
	}

	message, err := result(stream, turnCtx)
	if err != nil {
		if cause := endedBecause(ctx, turnCtx); cause != nil {
			err = cause
		}
		return Turn{Elapsed: time.Since(start)}, fmt.Errorf("streaming the reviewer's turn: %w", err)
	}

	c.messages = append(c.messages, message)
	turn := Turn{
		Message:    message,
		Usage:      c.accounted(message.Usage),
		Elapsed:    time.Since(start),
		Tools:      toolCalls(message),
		StopReason: message.StopReason,
	}

	if message.StopReason == ai.StopReasonError || message.StopReason == ai.StopReasonAborted {
		// A provider that notices a cancellation (the caller's, or this
		// package's own timeout) before Result does finishes with an aborted
		// message and a nil error, so the sentence below would otherwise wrap
		// nothing errors.Is could match — the exact shape that classified a
		// mid-stream Ctrl-C as stop=failed on a windows-latest -race run.
		// Nothing is guessed from the message text: the cause is read from the
		// contexts, which know.
		if cause := endedBecause(ctx, turnCtx); cause != nil {
			return turn, fmt.Errorf("the reviewer's turn stopped with reason %q: %s: %w", message.StopReason, message.ErrorMessage, cause)
		}
		return turn, fmt.Errorf("the reviewer's turn stopped with reason %q: %s", message.StopReason, message.ErrorMessage)
	}
	return turn, nil
}

// streamTimeout is StreamTimeout with the zero value read as the default, so
// a Conversation built by hand is bounded like every other one.
func (c *Conversation) streamTimeout() time.Duration {
	if c.StreamTimeout <= 0 {
		return DefaultStreamTimeout
	}
	return c.StreamTimeout
}

// accounted returns the turn's usage with a cost on it: the adapter's own
// figure when it filled one in — already adjusted for the service tier, which
// no price sheet knows about — and the model's price sheet only as a fallback
// for the adapters that report tokens and no cost, so dropping the
// unconditional recomputation cannot silently zero the $ figure.
func (c *Conversation) accounted(usage ai.Usage) ai.Usage {
	if usage.Cost == (ai.UsageCost{}) {
		ai.CalculateCost(c.model, &usage)
	}
	return usage
}

// result reads the stream's final message, preferring one the stream has
// already produced over a cancellation that landed beside it.
//
// The second read is the whole point: Stream.Result races its result channel
// against ctx.Done(), so the first call can lose a message that was already
// there. Asking again under a context that has not ended (resultGrace) makes
// a ready result win deterministically, and gives an adapter still unwinding
// its own cancellation the moment it needs to push its terminal event.
func result(stream *ai.Stream, turnCtx context.Context) (*ai.AssistantMessage, error) {
	message, err := stream.Result(turnCtx)
	if err == nil {
		return message, nil
	}

	graceCtx, cancelGrace := context.WithTimeout(context.WithoutCancel(turnCtx), resultGrace)
	defer cancelGrace()
	return stream.Result(graceCtx)
}

// endedBecause names why a turn ended when the answer is not "the model said
// so", in the one order this package commits to: the caller's own
// cancellation or deadline first — that is the human at the keyboard, and
// internal/cli renders it stop=interrupted — then this package's own stream
// timeout, which is a dead connection rather than an interruption and must
// not be dressed as one. nil means neither context ended and whatever the
// stream itself reported stands.
func endedBecause(ctx, turnCtx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if turnCtx.Err() != nil {
		return ErrStreamTimeout
	}
	return nil
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
