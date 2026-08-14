package reviewer_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/julienlegoux/kern-link/ai"

	"github.com/julienlegoux/external-reviewer/internal/fauxtest"
	"github.com/julienlegoux/external-reviewer/internal/reviewer"
)

// turnAnswer is the report a scripted turn returns. It only has to be
// recognisable — the byte-for-byte stdout contract is asserted at the CLI
// boundary (internal/cli/roundtrip_test.go), not here.
const turnAnswer = "# Review\n\nnothing to report\n"

// turnTimeout is the stream timeout every test here configures. It is short
// enough that the silent-provider case finishes in well under a second, and
// every test's own deadline (turnDeadline) is more than an order of magnitude
// longer, so a slow CI runner cannot turn "the bound held" into a flake.
const turnTimeout = 100 * time.Millisecond

// turnDeadline bounds a scripted turn from the test's side. Any turn that
// takes longer than this has failed the bound the test is asserting, and
// failing loudly beats hanging the package until go test's own timeout.
const turnDeadline = 5 * time.Second

// scriptedTurn builds a Conversation whose one model streams exactly what
// script pushes, over the offline registry internal/fauxtest owns. The model
// carries a price sheet, so a message that leaves Usage.Cost unset still has
// something for the fallback computation to multiply.
func scriptedTurn(t *testing.T, script fauxtest.StreamScript) *reviewer.Conversation {
	t.Helper()

	models, _ := fauxtest.NewRegistry(t, fauxtest.RegistryOptions{
		ProviderID: fixtureProvider,
		ModelIDs:   []string{fixtureModel},
		Auth:       fauxtest.CredentialedAuth("OAuth"),
		Priced:     true,
		Stream:     script,
	})
	model := models.GetModel(fixtureProvider, fixtureModel)
	if model == nil {
		t.Fatalf("model %s/%s missing from the scripted registry", fixtureProvider, fixtureModel)
	}

	conversation := reviewer.NewConversation(models, model, callerSystemPrompt, "review this")
	conversation.StreamTimeout = turnTimeout
	return conversation
}

// terminalStream returns a script that pushes exactly one terminal event —
// message — and nothing else, synchronously, before the stream is even handed
// back to the caller. That is what makes "the result was already there when
// the cancellation landed" a fact of the test rather than a race it hopes to
// win.
func terminalStream(message *ai.AssistantMessage, after func()) fauxtest.StreamScript {
	return func(context.Context, *ai.Model, ai.Context, *ai.SimpleStreamOptions) *ai.Stream {
		stream := ai.NewStream()
		if message.StopReason == ai.StopReasonError || message.StopReason == ai.StopReasonAborted {
			stream.Push(ai.ErrorEvent{Reason: message.StopReason, Error: message})
		} else {
			stream.Push(ai.DoneEvent{Reason: message.StopReason, Message: message})
		}
		if after != nil {
			after()
		}
		return stream
	}
}

// pricedMessage is a complete assistant message carrying tokens, so both the
// adapter's own figure and the fallback computation have something to say
// about it. usageCost is the adapter's already-adjusted cost; the zero value
// stands for an adapter that filled none in.
func pricedMessage(usageCost ai.UsageCost) *ai.AssistantMessage {
	return &ai.AssistantMessage{
		Content:    []ai.AssistantContentPart{ai.TextContent{Text: turnAnswer}},
		StopReason: ai.StopReasonStop,
		Usage: ai.Usage{
			Input:       1000,
			Output:      500,
			TotalTokens: 1500,
			Cost:        usageCost,
		},
	}
}

// baseSheetCost is what ai.CalculateCost computes for pricedMessage against
// fauxtest's price sheet (input $100/1e6 tokens, output $200/1e6): the figure
// the second CalculateCost call used to overwrite the adapter's with.
const baseSheetCost = 100.0/1e6*1000 + 200.0/1e6*500

// TestNext_ReportsTheAdaptersOwnCost pins the first repair. kern-link's
// openai-responses adapter fills Usage.Cost and then scales it by the service
// tier (2.5x for "priority", 0.5x for "flex"); calling ai.CalculateCost again
// over the same usage can only throw that adjustment away, because the price
// sheet knows nothing about tiers. The scripted cost below is deliberately
// 2.5x the base-sheet figure, so re-adding the second CalculateCost call turns
// this red rather than merely imprecise.
func TestNext_ReportsTheAdaptersOwnCost(t *testing.T) {
	adjusted := ai.UsageCost{Input: 0.25, Output: 0.25, Total: 0.5}
	conversation := scriptedTurn(t, terminalStream(pricedMessage(adjusted), nil))

	turn, err := conversation.Next(context.Background())
	if err != nil {
		t.Fatalf("Next returned %v, want a successful turn", err)
	}

	if turn.Usage.Cost.Total != adjusted.Total {
		t.Errorf("turn cost = %v, want the adapter's own service-tier-adjusted %v (base sheet would say %v)",
			turn.Usage.Cost.Total, adjusted.Total, baseSheetCost)
	}
	if turn.Usage.Cost.Total == baseSheetCost {
		t.Errorf("turn cost = %v, which is the base-sheet figure: the adapter's adjustment was recomputed away", turn.Usage.Cost.Total)
	}
}

// TestNext_ComputesCostTheAdapterLeftUnset is the other half of the same
// repair: dropping the second ai.CalculateCost call must not silently zero the
// $ figure for the adapters (and the faux provider) that report tokens without
// a cost. An unset Cost is the one case the price sheet is still the best
// available answer.
func TestNext_ComputesCostTheAdapterLeftUnset(t *testing.T) {
	conversation := scriptedTurn(t, terminalStream(pricedMessage(ai.UsageCost{}), nil))

	turn, err := conversation.Next(context.Background())
	if err != nil {
		t.Fatalf("Next returned %v, want a successful turn", err)
	}

	if turn.Usage.Cost.Total != baseSheetCost {
		t.Errorf("turn cost = %v, want the price sheet's %v computed as a fallback", turn.Usage.Cost.Total, baseSheetCost)
	}
}

// TestNext_SilentProviderEndsAtTheStreamTimeout pins the second repair. A
// backend that accepts the stream and then sends nothing used to block
// `for range stream.Events(ctx)` forever, with no output, no done line and no
// exit code: kern-link v0.1.1 arms no default deadline on either transport.
// The turn now ends at its own bound, and the failure it reports is a stream
// timeout rather than a cancellation — a self-imposed transport deadline is
// not a run a human interrupted, and internal/cli classifies the two
// differently.
func TestNext_SilentProviderEndsAtTheStreamTimeout(t *testing.T) {
	conversation := scriptedTurn(t, fauxtest.SilentStream)

	start := time.Now()
	turn, err := conversation.Next(context.Background())
	elapsed := time.Since(start)

	if err == nil {
		t.Fatalf("Next returned no error, want a stream timeout")
	}
	if elapsed > turnDeadline {
		t.Errorf("Next took %s, want it bounded well inside %s", elapsed, turnDeadline)
	}
	if !errors.Is(err, reviewer.ErrStreamTimeout) {
		t.Errorf("Next returned %v, want it to wrap ErrStreamTimeout", err)
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Next returned %v, want the timeout kept off the cancellation vocabulary — internal/cli reads that as an interruption", err)
	}
	if turn.Message != nil {
		t.Errorf("turn message = %+v, want none: the provider sent nothing", turn.Message)
	}
}

// TestNext_AbortingProviderStillReportsTheStreamTimeout covers the same bound
// against a well-behaved adapter rather than a catatonic one: a real
// transport watches the context it was handed and answers a deadline with an
// in-band aborted message, so the turn ends through the stop-reason branch
// with no context error in sight. It must still be reported as this package's
// stream timeout — the same word for the same event, whichever half of the
// stack noticed it — rather than as a bare upstream abort.
func TestNext_AbortingProviderStillReportsTheStreamTimeout(t *testing.T) {
	script := func(ctx context.Context, _ *ai.Model, _ ai.Context, _ *ai.SimpleStreamOptions) *ai.Stream {
		stream := ai.NewStream()
		go func() {
			<-ctx.Done()
			stream.Push(ai.ErrorEvent{Reason: ai.StopReasonAborted, Error: &ai.AssistantMessage{
				Content:      []ai.AssistantContentPart{},
				StopReason:   ai.StopReasonAborted,
				ErrorMessage: "Request was aborted",
			}})
		}()
		return stream
	}
	conversation := scriptedTurn(t, script)

	start := time.Now()
	_, err := conversation.Next(context.Background())
	elapsed := time.Since(start)

	if elapsed > turnDeadline {
		t.Errorf("Next took %s, want it bounded well inside %s", elapsed, turnDeadline)
	}
	if !errors.Is(err, reviewer.ErrStreamTimeout) {
		t.Errorf("Next returned %v, want it to wrap ErrStreamTimeout", err)
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Next returned %v, want the timeout kept off the cancellation vocabulary", err)
	}
}

// TestNext_CancelledSilentProviderIsCancellation is the precedence order
// between the two deadlines: when the caller's own context ends first, the
// turn reports that cancellation rather than this package's stream bound,
// even though both are context errors by the time Result gives up. The
// stream timeout is deliberately set out of reach here, so the assertion
// cannot pass by the two deadlines racing on a loaded runner.
func TestNext_CancelledSilentProviderIsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conversation := scriptedTurn(t, func(context.Context, *ai.Model, ai.Context, *ai.SimpleStreamOptions) *ai.Stream {
		cancel()
		return ai.NewStream()
	})
	conversation.StreamTimeout = turnDeadline

	_, err := conversation.Next(ctx)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Next returned %v, want it to wrap context.Canceled", err)
	}
	if errors.Is(err, reviewer.ErrStreamTimeout) {
		t.Errorf("Next returned %v, want the caller's cancellation to win over the stream timeout", err)
	}
}

// TestNext_AlreadyCancelledContext_SendsNoRequest pins the one place a
// cancellation does short-circuit: before a stream exists there is no
// complete message to prefer over it, so the turn ends without asking the
// provider anything at all. The done line's token counts stay at 0 on a path
// that never reached a model (SPECS § Interfaces), which is only true if no
// request was made.
func TestNext_AlreadyCancelledContext_SendsNoRequest(t *testing.T) {
	called := make(chan struct{}, 1)
	conversation := scriptedTurn(t, func(context.Context, *ai.Model, ai.Context, *ai.SimpleStreamOptions) *ai.Stream {
		called <- struct{}{}
		return ai.NewStream()
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	turn, err := conversation.Next(ctx)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Next returned %v, want it to wrap context.Canceled", err)
	}
	if turn.Message != nil {
		t.Errorf("turn message = %+v, want none", turn.Message)
	}
	select {
	case <-called:
		t.Errorf("the provider was called under an already-cancelled context")
	default:
	}
}

// TestNext_CompleteMessageBeatsALateCancellation pins the third repair, and
// the one that costs real money: a 90-second turn finishes, the report is on
// the wire and already billed, and the user — who stopped watching — presses
// Ctrl-C a millisecond later. kern-link's Stream.Result selects between its
// result channel and ctx.Done() (ai/stream.go:155), so with the result already
// there and the context already ended both cases are ready and Go picks one
// uniformly at random. The script below makes that exact state a certainty:
// the terminal event is pushed before the stream is handed back, and the
// cancellation lands immediately after.
//
// Two interleavings are in play, because kern-link's LazyStream copies the
// adapter's events onto the stream Next holds through a goroutine of its own:
// the terminal event may or may not have been copied across by the time
// Result runs. The loop covers both, and both must hand back the report.
func TestNext_CompleteMessageBeatsALateCancellation(t *testing.T) {
	const repeats = 20

	for i := range repeats {
		ctx, cancel := context.WithCancel(context.Background())
		conversation := scriptedTurn(t, terminalStream(pricedMessage(ai.UsageCost{}), cancel))

		turn, err := conversation.Next(ctx)
		cancel()

		if err != nil {
			t.Fatalf("repeat %d: Next returned %v, want the completed report rather than a late cancellation", i, err)
		}
		if got := reviewer.FinalText(turn.Message); got != turnAnswer {
			t.Fatalf("repeat %d: final text = %q, want the completed report %q", i, got, turnAnswer)
		}
	}
}

// TestNext_AbortedMessageUnderCancelledContextIsInterrupted is the guard that
// was deleted as unreachable and then cost a windows-latest `go test -race`
// round on issue 05's PR. When a cancellation lands mid-stream the provider's
// own goroutine can notice it first and finish with a StopReasonAborted
// message; Result then returns that message with a *nil* error, so the turn's
// constructed "stopped with reason" error wraps nothing errors.Is can match
// against context.Canceled, and the run is reported as failed rather than
// interrupted. Pushing the aborted message and only then cancelling reaches
// exactly that state, in that order, without racing anything.
//
// Deleting the ctx.Err() branch from Next's stop-reason check turns this red.
func TestNext_AbortedMessageUnderCancelledContextIsInterrupted(t *testing.T) {
	aborted := &ai.AssistantMessage{
		Content:      []ai.AssistantContentPart{},
		StopReason:   ai.StopReasonAborted,
		ErrorMessage: "Request was aborted",
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	conversation := scriptedTurn(t, terminalStream(aborted, cancel))

	turn, err := conversation.Next(ctx)

	if err == nil {
		t.Fatalf("Next returned no error, want an aborted turn")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Next returned %v, want it to carry the cancellation cause so internal/cli reads it as an interruption", err)
	}
	if turn.StopReason != ai.StopReasonAborted {
		t.Errorf("turn stop reason = %q, want %q surfaced verbatim", turn.StopReason, ai.StopReasonAborted)
	}
}

// TestNext_AbortedMessageUnderLiveContextIsAFailure is the other side of the
// same branch: an abort the provider decided on its own, with nobody
// interrupting anything, stays a plain failing turn. Without this, wrapping
// every aborted message in a cancellation cause would pass the test above
// while quietly relabelling every upstream abort as a user interruption.
func TestNext_AbortedMessageUnderLiveContextIsAFailure(t *testing.T) {
	aborted := &ai.AssistantMessage{
		Content:      []ai.AssistantContentPart{},
		StopReason:   ai.StopReasonAborted,
		ErrorMessage: "request was aborted upstream",
	}
	conversation := scriptedTurn(t, terminalStream(aborted, nil))

	_, err := conversation.Next(context.Background())

	if err == nil {
		t.Fatalf("Next returned no error, want a failing turn")
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Next returned %v, want an abort under a live context to stay a failure", err)
	}
}

// TestNext_PassesTheStreamTimeoutToTheAdapter checks the belt beside the
// braces: the local deadline bounds the turn whatever the adapter does, but
// the adapter is also told the timeout, so kern-link's own transports arm
// their read deadlines (the codex WebSocket path sets idleTimeout from it and
// arms nothing at all when it is zero) instead of relying on this package to
// unblock them.
func TestNext_PassesTheStreamTimeoutToTheAdapter(t *testing.T) {
	seen := make(chan time.Duration, 1)
	script := func(_ context.Context, _ *ai.Model, _ ai.Context, opts *ai.SimpleStreamOptions) *ai.Stream {
		timeout := time.Duration(0)
		if opts != nil {
			timeout = opts.Timeout
		}
		seen <- timeout
		return terminalStream(pricedMessage(ai.UsageCost{}), nil)(context.Background(), nil, ai.Context{}, nil)
	}

	conversation := scriptedTurn(t, script)
	if _, err := conversation.Next(context.Background()); err != nil {
		t.Fatalf("Next returned %v, want a successful turn", err)
	}

	select {
	case got := <-seen:
		if got != turnTimeout {
			t.Errorf("StreamOptions.Timeout = %s, want the conversation's %s", got, turnTimeout)
		}
	case <-time.After(turnDeadline):
		t.Fatalf("the scripted adapter was never called")
	}
}
