package reviewer

import (
	"time"

	"github.com/julienlegoux/external-reviewer/internal/diag"
)

// BoundsStopReason is the done line's word for a run that ended at its own
// ceiling rather than at anything the model, the caller or the provider did.
// It is the value SPECS § Interfaces reserved for this epic, added to the
// closed set rather than repurposing one of the five that were already there:
// a bounded run neither failed nor was interrupted, and it carries a report.
const BoundsStopReason = "bounds"

// Bounds is the run ceiling the loop consults at the top of every turn.
//
// It ships with **no values**: every field's zero means "unset", so the shape
// this epic delivers is a seam rather than a policy. SCOPE defers the real
// numbers until a review has actually been run and measured (issue 07), on the
// reasoning that a cap guessed before any measurement silently truncates
// legitimate reviews — and having the seam here means Epic 3 sets numbers
// instead of restructuring the loop (specs 07).
//
// Which quantities are worth bounding is already decided and is not the same
// as which are representable: on a subscription provider the cost figure is
// notional, so MaxTurns and MaxElapsed are the meaningful ones, and MaxCost is
// carried because a tier that names an API-key provider makes it real again
// and the check is one comparison either way (specs 07 § Verdict).
type Bounds struct {
	// MaxTurns caps how many round trips a run may take. 0 is unset.
	MaxTurns int
	// MaxCost caps the accumulated dollar figure. 0 is unset.
	MaxCost float64
	// MaxElapsed caps the run's wall clock, measured from diag.State.Start.
	// 0 is unset.
	MaxElapsed time.Duration
}

// check names the bound that has been reached — "turn", "cost" or "elapsed" —
// or returns "" while every bound is unset or unmet.
//
// It is deliberately not an error: an exceeded bound stops the loop and
// returns what the run has, which is a different outcome from a failure and
// carries a different exit code. The name it returns is for the stderr line
// that says which ceiling stopped the run; the done line's stop= word is
// BoundsStopReason for all three, because a caller parsing the transcript
// needs one value to match on.
//
// The comparison is >= rather than >, so MaxTurns: 1 means "one turn, then
// stop" rather than "two". state.Turns is the count of turns already taken
// when this is consulted at the top of the next one.
func (b Bounds) check(state *diag.State) string {
	switch {
	case b.MaxTurns > 0 && state.Turns >= b.MaxTurns:
		return "turn"
	case b.MaxCost > 0 && state.Cost >= b.MaxCost:
		return "cost"
	case b.MaxElapsed > 0 && time.Since(state.Start) >= b.MaxElapsed:
		return "elapsed"
	default:
		return ""
	}
}
