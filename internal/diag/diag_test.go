package diag_test

import (
	"bytes"
	"regexp"
	"testing"
	"time"

	"github.com/julienlegoux/external-reviewer/internal/diag"
)

// doneLineRE parses the "done" line's key=value fields rather than matching
// the line verbatim — elapsed time is real and cannot be asserted exactly.
var doneLineRE = regexp.MustCompile(
	`^done\s+turns=(\d+)\s+in=(\d+) out=(\d+)\s+\$([0-9.]+)\s+(\S+)\s+stop=(\S+)\s*$`)

func parseDone(t *testing.T, line string) (turns, in, out, cost, elapsed, stop string) {
	t.Helper()
	m := doneLineRE.FindStringSubmatch(line)
	if m == nil {
		t.Fatalf("done line %q does not match expected shape", line)
	}
	return m[1], m[2], m[3], m[4], m[5], m[6]
}

func TestNewState_StartsClockRunning(t *testing.T) {
	before := time.Now()
	s := diag.NewState()
	after := time.Now()

	if s.Start.Before(before) || s.Start.After(after) {
		t.Errorf("Start = %v, want between %v and %v", s.Start, before, after)
	}
}

func TestWriteDone_RendersAccumulatedFields(t *testing.T) {
	var buf bytes.Buffer
	s := diag.NewState()
	s.Start = time.Now().Add(-(2*time.Minute + 14*time.Second))
	s.Turns = 7
	s.InputTokens = 182430
	s.OutputTokens = 9106
	s.Cost = 0.0918
	s.StopReason = "end_turn"

	diag.WriteDone(&buf, s)

	turns, in, out, cost, elapsed, stop := parseDone(t, buf.String())
	if turns != "7" {
		t.Errorf("turns = %q, want 7", turns)
	}
	if in != "182430" {
		t.Errorf("in = %q, want 182430", in)
	}
	if out != "9106" {
		t.Errorf("out = %q, want 9106", out)
	}
	if cost != "0.0918" {
		t.Errorf("cost = %q, want 0.0918", cost)
	}
	if stop != "end_turn" {
		t.Errorf("stop = %q, want end_turn", stop)
	}
	if elapsed != "2m14s" {
		t.Errorf("elapsed = %q, want 2m14s", elapsed)
	}
}

func TestWriteDone_SubMinuteElapsedHasOneDecimal(t *testing.T) {
	var buf bytes.Buffer
	s := diag.NewState()
	s.Start = time.Now().Add(-42800 * time.Millisecond)
	s.StopReason = "end_turn"

	diag.WriteDone(&buf, s)

	_, _, _, _, elapsed, _ := parseDone(t, buf.String())
	if elapsed != "42.8s" {
		t.Errorf("elapsed = %q, want 42.8s", elapsed)
	}
}

func TestWriteDone_TokenFieldsAreZeroNotAbsentWhenNoModelWasReached(t *testing.T) {
	var buf bytes.Buffer
	s := diag.NewState()
	s.StopReason = "usage"

	diag.WriteDone(&buf, s)

	turns, in, out, _, _, stop := parseDone(t, buf.String())
	if turns != "0" {
		t.Errorf("turns = %q, want 0", turns)
	}
	if in != "0" || out != "0" {
		t.Errorf("in/out = %q/%q, want 0/0 (present, not absent)", in, out)
	}
	if stop != "usage" {
		t.Errorf("stop = %q, want usage", stop)
	}
}

func TestWriteTurn_RendersFields(t *testing.T) {
	var buf bytes.Buffer

	diag.WriteTurn(&buf, 3, 2, 48210, 1104, 0.0231, 42800*time.Millisecond)

	got := buf.String()
	want := "turn 3  tools=2  in=48210 out=1104  $0.0231  42.8s\n"
	if got != want {
		t.Errorf("WriteTurn wrote %q, want %q", got, want)
	}
}

func TestWriteTool_RendersSummary(t *testing.T) {
	var buf bytes.Buffer

	diag.WriteTool(&buf, "read_file docs/epics/epic-2/EPIC_2.md:1-2000")

	got := buf.String()
	want := "tool    read_file docs/epics/epic-2/EPIC_2.md:1-2000\n"
	if got != want {
		t.Errorf("WriteTool wrote %q, want %q", got, want)
	}
}

func TestWriteWarn_RendersMessage(t *testing.T) {
	var buf bytes.Buffer

	diag.WriteWarn(&buf, `config: unrecognised key "models" in [tiers.standard]`)

	got := buf.String()
	want := "warn    config: unrecognised key \"models\" in [tiers.standard]\n"
	if got != want {
		t.Errorf("WriteWarn wrote %q, want %q", got, want)
	}
}
