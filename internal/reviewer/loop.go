package reviewer

import (
	"context"
	"fmt"
	"io"
	"maps"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/julienlegoux/kern-link/ai"

	"github.com/julienlegoux/external-reviewer/internal/diag"
)

// ToolSet is everything the loop needs from a tool registry, and the whole of
// the seam between this package and internal/tools.
//
// It is stated as an interface here, in terms kern-link already defines,
// rather than as a dependency on internal/tools, for one structural reason:
// adding a tool must never mean editing the loop. A new tool is a new file in
// internal/tools plus one line in that package's own registry builder —
// nothing in internal/reviewer changes, and nothing in internal/reviewer can
// break a tool. Dispatch returns (text, isError) rather than a tools.Result so
// the direction of that dependency stays one-way.
//
// Dispatch answers every call, including one naming a tool that does not
// exist: an invented name is the model's mistake to learn from, so it is a
// result with isError true, never an error that ends a run (CONVENTIONS §
// Error handling).
type ToolSet interface {
	// Declarations is what the model is told it may call, in declaration
	// order. It travels on every request, not only the first.
	Declarations() []ai.Tool
	// Dispatch runs one call and returns the text the model sees, plus
	// whether that text is a refusal.
	Dispatch(ctx context.Context, call ai.ToolCall) (text string, isError bool)
}

// Loop drives a Conversation until something ends it, dispatching the tool
// calls the reviewer makes along the way. It is what turns the single round
// trip Epic 1 shipped into a reviewer: the model declares what it wants to
// read, the loop reads it, and the answer comes back as another turn.
//
// The run's accumulated totals live here, in State, rather than one call frame
// above: the bound check at the top of every turn reads turns, cost and
// elapsed wall clock, and a total threaded back up to internal/cli and down
// again on every turn would be the same number in two places. internal/cli
// keeps creating the State before the loop starts and rendering the done line
// from it once the loop returns.
type Loop struct {
	// Conversation is the accumulating message history and the model each
	// turn is sent to.
	Conversation *Conversation
	// Tools is the run's tool set. A nil ToolSet declares nothing, which is
	// Epic 1's behaviour — a model that calls a tool anyway is told so.
	Tools ToolSet
	// Bounds is the run ceiling, consulted at the top of every turn. The zero
	// value bounds nothing, which is what this epic ships (see Bounds).
	Bounds Bounds
	// State is the run state the done line renders from. The loop updates it
	// in place after every turn.
	State *diag.State
	// Stderr is where the turn, tool and warn lines go.
	Stderr io.Writer
}

// Run drives the conversation to its end and returns the report — the last
// assistant text the run produced.
//
// Termination is checked in exactly this order, at the top of every turn, and
// the order is the contract rather than an accident of how the code fell out
// (specs 07):
//
//  1. **A bound is exceeded** → stop and return what the run has. This is not
//     a failure: the accumulated report goes to stdout, the exit code is 0,
//     and the done line says stop=bounds. It is checked first so that a run
//     which reached its ceiling reports as bounded even if a cancellation
//     landed in the same instant — a ceiling is a decision the invocation
//     made, and it is more informative than the interruption that followed it.
//  2. **The context ended** → stop and return the cancellation. A human at the
//     keyboard is the only thing standing between an unbounded run and real
//     money while the bounds above ship unset (scope risk 5), so this must
//     work; internal/cli renders it stop=interrupted at exit 2.
//  3. **StopReason != StopReasonToolUse** → the model is done. The normal
//     exit.
//
// A failing *turn* — the stream failing, a StopReason of error or aborted, the
// stream timeout — is the fourth way out and the only one that returns an
// error. A failing *tool* never appears here at all: it is a value the next
// turn carries (see dispatch).
//
// The report is the last *non-empty* assistant text rather than the last
// message's text flat, so a bound that bites after a turn which asked for a
// tool without saying anything still returns the prose the run had produced
// before it.
func (l *Loop) Run(ctx context.Context) (string, error) {
	if l.Tools != nil {
		l.Conversation.declare(l.Tools.Declarations())
	}

	report := ""
	for {
		if bound := l.Bounds.check(l.State); bound != "" {
			diag.WriteWarn(l.Stderr, "the run stopped at its "+bound+" bound")
			l.State.StopReason = BoundsStopReason
			return report, nil
		}
		if err := ctx.Err(); err != nil {
			return "", fmt.Errorf("driving the reviewer's loop: %w", err)
		}

		turn, err := l.Conversation.Next(ctx)
		l.record(turn)
		if err != nil {
			return "", err
		}
		if text := FinalText(turn.Message); text != "" {
			report = text
		}
		if turn.StopReason != ai.StopReasonToolUse {
			return report, nil
		}
		l.dispatch(ctx, turn.Message)
	}
}

// record folds a completed round trip into the run state the done line
// renders from and writes the turn's own line, plus a warn line for every
// non-fatal diagnostic kern-link attached to the message.
//
// It runs on the failure paths too: a turn that stopped with an error still
// consumed tokens, and its diagnostics are usually why it stopped.
// State.StopReason is set here to the turn's own reason on every call —
// internal/cli's runReview unconditionally overwrites it with a CLI word on
// any termination the model's own turn does not explain — so the value set
// here is only ever what the done line renders on the success path.
func (l *Loop) record(turn Turn) {
	if turn.Message == nil {
		return
	}

	l.State.Turns++
	// Cached prompt tokens are prompt tokens: providers report them beside
	// Input rather than inside it, so summing the three is what makes the in=
	// figure comparable between a cold run and a warm one.
	l.State.InputTokens += int64(turn.Usage.Input + turn.Usage.CacheRead + turn.Usage.CacheWrite)
	l.State.OutputTokens += int64(turn.Usage.Output)
	l.State.Cost += turn.Usage.Cost.Total
	l.State.StopReason = modelStopReason(turn.StopReason)

	for _, diagnostic := range turn.Message.Diagnostics {
		diag.WriteWarn(l.Stderr, diagnosticMessage(diagnostic))
	}
	diag.WriteTurn(l.Stderr, l.State.Turns, turn.Tools, l.State.InputTokens, l.State.OutputTokens, l.State.Cost, turn.Elapsed)
}

// dispatch answers every tool call in one assistant message, in the order the
// model made them, and appends each answer to the conversation so the next
// turn carries it.
//
// Execution is sequential: one review per process, one tool at a time
// (SPECS § Distribution & operations). Nothing here can return an error —
// that is the single place CONVENTIONS' "never swallow an error" rule
// inverts. A reviewer that globs a path which does not exist should try
// another path, not kill the review, so every refusal comes back as a tool
// result the model reads and can act on.
func (l *Loop) dispatch(ctx context.Context, message *ai.AssistantMessage) {
	for _, part := range message.Content {
		call, ok := part.(ai.ToolCall)
		if !ok {
			continue
		}
		diag.WriteTool(l.Stderr, toolSummary(call))
		text, isError := l.answer(ctx, call)
		l.Conversation.appendToolResult(call, text, isError)
	}
}

// answer runs one call through the tool set, or explains that this run has no
// tool set at all — which is not a state any real invocation reaches, but is
// a value rather than a nil dereference if one ever does.
func (l *Loop) answer(ctx context.Context, call ai.ToolCall) (string, bool) {
	if l.Tools == nil {
		return fmt.Sprintf("this run declared no tools, so %q cannot be called", call.Name), true
	}
	return l.Tools.Dispatch(ctx, call)
}

// toolSummary renders one call for its "tool" line: the tool's name, then its
// arguments as name=value in a stable (sorted) order, so two runs of the same
// conversation produce the same transcript.
//
// The loop renders generically rather than per tool — it cannot know that
// read_file's interesting fields are a path and a line range — which is why
// this is name=value rather than the hand-composed summary SPECS § Interfaces
// illustrates.
func toolSummary(call ai.ToolCall) string {
	var out strings.Builder
	out.WriteString(call.Name)
	for _, name := range slices.Sorted(maps.Keys(call.Arguments)) {
		fmt.Fprintf(&out, " %s=%s", name, wireValue(call.Arguments[name]))
	}
	return out.String()
}

// wireValue renders one argument for a tool line. A string is converted to
// wire form and quoted: every string a tool takes in this product is a path, a
// glob or a git revision, all of which are /-separated by contract, so
// filepath.ToSlash is the boundary conversion CONVENTIONS § Paths and
// platforms requires and it also stops strconv.Quote from doubling a
// backslash the way flag's own %q rendering once did (Epic 0 issue 08).
// Anything else is a number or a boolean the model sent, printed as it is.
func wireValue(value any) string {
	if text, ok := value.(string); ok {
		return strconv.Quote(filepath.ToSlash(text))
	}
	return fmt.Sprintf("%v", value)
}

// stopReasonUnspecified is the done line's named fallback for a successful
// turn whose assistant message carried no StopReason at all, so stop= can
// never render with nothing after it (SPECS § Interfaces).
const stopReasonUnspecified = "unspecified"

// modelStopReason renders a turn's own StopReason for the done line's success
// path, spelled exactly as kern-link spells it — no translation table between
// the model's vocabulary and the CLI's.
func modelStopReason(reason ai.StopReason) string {
	if reason == "" {
		return stopReasonUnspecified
	}
	return string(reason)
}

// diagnosticMessage renders one AssistantMessageDiagnostic as a warn line's
// text. Nothing here re-formats or re-derives anything from the underlying
// error, so no credential is reconstructed from it; what keeps the message
// from breaking the transcript is diag.WriteWarn's own control-character
// escaping, which is not a redaction (SPECS § Security — this binary formats
// no credential in the first place).
func diagnosticMessage(diagnostic ai.AssistantMessageDiagnostic) string {
	kind := diagnostic.Type
	if kind == "" {
		kind = "diagnostic"
	}
	if diagnostic.Error == nil || diagnostic.Error.Message == "" {
		return kind
	}
	return kind + ": " + diagnostic.Error.Message
}
