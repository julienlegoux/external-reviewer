package fauxtest

import (
	"regexp"
	"testing"
)

// doneLineRE parses one "done" line's key=value fields out of a run's
// stderr (SPECS § Interfaces) rather than matching it verbatim — elapsed
// time is real and cannot be asserted exactly. It is anchored per line
// (multiline mode) so it finds the done line whether text is exactly one
// line (internal/diag's own package tests, which hand WriteDone a fresh
// buffer) or a full multi-line transcript (internal/cli's black-box tests,
// which run a whole review). Both packages parsed this with their own copy
// of the same pattern before this one replaced them (issue 12).
var doneLineRE = regexp.MustCompile(
	`(?m)^done\s+turns=(\d+)\s+in=(\d+) out=(\d+)\s+\$([0-9.]+)\s+(\S+)\s+stop=(\S+)\s*$`)

// DoneFields holds one done line's key=value fields for assertions.
type DoneFields struct {
	Turns, In, Out, Cost, Elapsed, Stop string
}

// ParseDoneLine finds and parses the done line in text, failing t if none is
// present — every termination path must emit exactly one.
func ParseDoneLine(t testing.TB, text string) DoneFields {
	t.Helper()
	m := doneLineRE.FindStringSubmatch(text)
	if m == nil {
		t.Fatalf("text = %q, want a done line matching %s", text, doneLineRE)
	}
	return DoneFields{Turns: m[1], In: m[2], Out: m[3], Cost: m[4], Elapsed: m[5], Stop: m[6]}
}

// turnLineRE parses one "turn" line's key=value fields the same way
// doneLineRE does for the done line.
var turnLineRE = regexp.MustCompile(
	`(?m)^turn (\d+)\s+tools=(\d+)\s+in=(\d+) out=(\d+)\s+\$([0-9.]+)\s+(\S+)\s*$`)

// TurnFields holds one turn line's key=value fields for assertions.
type TurnFields struct {
	Turn, Tools, In, Out, Cost, Elapsed string
}

// ParseTurnLine finds and parses the turn line in text, failing t if none is
// present.
func ParseTurnLine(t testing.TB, text string) TurnFields {
	t.Helper()
	m := turnLineRE.FindStringSubmatch(text)
	if m == nil {
		t.Fatalf("text = %q, want a turn line matching %s", text, turnLineRE)
	}
	return TurnFields{Turn: m[1], Tools: m[2], In: m[3], Out: m[4], Cost: m[5], Elapsed: m[6]}
}
