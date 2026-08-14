package diag_test

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/julienlegoux/external-reviewer/internal/diag"
	"github.com/julienlegoux/external-reviewer/internal/fauxtest"
)

func TestNewState_StartsClockRunning(t *testing.T) {
	before := time.Now()
	s := diag.NewState()
	after := time.Now()

	if s.Start.Before(before) || s.Start.After(after) {
		t.Errorf("Start = %v, want between %v and %v", s.Start, before, after)
	}
}

// TestWriteDone_RendersFields is table-driven over WriteDone's callers
// (CONVENTIONS § Testing): the ordinary accumulated-fields render, the
// sub-minute elapsed format, the zero-fields-not-absent case for a path that
// never reached a model, and the hour-unit rendering formatElapsed gains in
// this issue. Mutating formatElapsed's `d < time.Minute` boundary to
// `d <= time.Minute` was hand-verified to turn this suite red (see
// TestWriteTurn_RendersFields's exact-one-minute row below, which pins that
// boundary deterministically — WriteDone's own elapsed is real wall-clock
// time and always lands a few microseconds past whatever Start it is given,
// so it cannot distinguish "<" from "<=" on its own).
func TestWriteDone_RendersFields(t *testing.T) {
	tests := []struct {
		name        string
		configure   func(s *diag.State)
		wantTurns   string
		wantIn      string
		wantOut     string
		wantCost    string
		wantElapsed string // "" skips the elapsed assertion
		wantStop    string
	}{
		{
			name: "accumulated fields after 7 turns over 2m14s",
			configure: func(s *diag.State) {
				s.Start = time.Now().Add(-(2*time.Minute + 14*time.Second))
				s.Turns = 7
				s.InputTokens = 182430
				s.OutputTokens = 9106
				s.Cost = 0.0918
				s.StopReason = "end_turn"
			},
			wantTurns: "7", wantIn: "182430", wantOut: "9106", wantCost: "0.0918",
			wantElapsed: "2m14s", wantStop: "end_turn",
		},
		{
			name: "sub-minute elapsed renders one decimal place, no minutes unit",
			configure: func(s *diag.State) {
				s.Start = time.Now().Add(-42800 * time.Millisecond)
				s.StopReason = "end_turn"
			},
			wantTurns: "0", wantIn: "0", wantOut: "0",
			wantElapsed: "42.8s", wantStop: "end_turn",
		},
		{
			name: "token fields are zero, not absent, when no model was reached",
			configure: func(s *diag.State) {
				s.StopReason = "usage"
			},
			wantTurns: "0", wantIn: "0", wantOut: "0", wantStop: "usage",
		},
		{
			name: "a 95-minute run renders an hour unit",
			configure: func(s *diag.State) {
				s.Start = time.Now().Add(-(1*time.Hour + 35*time.Minute))
				s.StopReason = "end_turn"
			},
			wantTurns: "0", wantIn: "0", wantOut: "0",
			wantElapsed: "1h35m0s", wantStop: "end_turn",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			s := diag.NewState()
			tc.configure(s)

			diag.WriteDone(&buf, s)

			fields := fauxtest.ParseDoneLine(t, buf.String())
			if fields.Turns != tc.wantTurns {
				t.Errorf("turns = %q, want %q", fields.Turns, tc.wantTurns)
			}
			if fields.In != tc.wantIn {
				t.Errorf("in = %q, want %q", fields.In, tc.wantIn)
			}
			if fields.Out != tc.wantOut {
				t.Errorf("out = %q, want %q", fields.Out, tc.wantOut)
			}
			if tc.wantCost != "" && fields.Cost != tc.wantCost {
				t.Errorf("cost = %q, want %q", fields.Cost, tc.wantCost)
			}
			if tc.wantElapsed != "" && fields.Elapsed != tc.wantElapsed {
				t.Errorf("elapsed = %q, want %q", fields.Elapsed, tc.wantElapsed)
			}
			if fields.Stop != tc.wantStop {
				t.Errorf("stop = %q, want %q", fields.Stop, tc.wantStop)
			}
		})
	}
}

// TestWriteTurn_RendersFields is table-driven over WriteTurn's rendering,
// including the sub-minute/at-minute boundary formatElapsed's elapsed
// argument crosses. Unlike WriteDone's tests above, elapsed here is an
// explicit time.Duration argument rather than real wall-clock time, so the
// "exactly one minute" row is a deterministic mutation guard: with the
// production `d < time.Minute`, d == time.Minute takes the minutes branch
// ("1m0s"); mutated to `d <= time.Minute`, the same input takes the seconds
// branch ("60.0s") instead, and this row goes red.
func TestWriteTurn_RendersFields(t *testing.T) {
	tests := []struct {
		name    string
		n       int
		tools   int
		inTok   int64
		outTok  int64
		cost    float64
		elapsed time.Duration
		want    string
	}{
		{
			name: "sub-minute elapsed renders seconds with one decimal",
			n:    3, tools: 2, inTok: 48210, outTok: 1104, cost: 0.0231, elapsed: 42800 * time.Millisecond,
			want: "turn 3  tools=2  in=48210 out=1104  $0.0231  42.8s\n",
		},
		{
			name: "elapsed of exactly one minute takes the minutes branch, not seconds",
			n:    1, tools: 0, inTok: 100, outTok: 50, cost: 0.0001, elapsed: time.Minute,
			want: "turn 1  tools=0  in=100 out=50  $0.0001  1m0s\n",
		},
		{
			name: "elapsed of 95 minutes renders an hour unit",
			n:    1, tools: 0, inTok: 100, outTok: 50, cost: 0.0001, elapsed: 95 * time.Minute,
			want: "turn 1  tools=0  in=100 out=50  $0.0001  1h35m0s\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			diag.WriteTurn(&buf, tc.n, tc.tools, tc.inTok, tc.outTok, tc.cost, tc.elapsed)

			if got := buf.String(); got != tc.want {
				t.Errorf("WriteTurn wrote %q, want %q", got, tc.want)
			}
		})
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

func TestWriteModel_RendersResolvedModelAndAuthSource(t *testing.T) {
	var buf bytes.Buffer

	diag.WriteModel(&buf, "openai-codex", "gpt-5.5", "OAuth")

	got := buf.String()
	want := "model   openai-codex/gpt-5.5  auth=OAuth\n"
	if got != want {
		t.Errorf("WriteModel wrote %q, want %q", got, want)
	}
}

// TestWriteWarnAndWriteError_EscapeControlCharacters is table-driven over
// both writers at once, since they share the same escaping seam: an
// ordinary message passes through unchanged, and a message carrying a
// newline, an ANSI escape, or both together is rendered as a single line
// with the control characters spelled out inertly rather than reaching the
// terminal raw. The "forges a done line" and "forges a second warn/error
// line" rows are this issue's central acceptance criterion, expressed at
// the unit level the injection is actually stopped at; roundtrip_test.go
// carries the same property through the real reviewer/cli/diag pipeline.
func TestWriteWarnAndWriteError_EscapeControlCharacters(t *testing.T) {
	tests := []struct {
		name    string
		write   func(w io.Writer, message string)
		message string
		want    string
	}{
		{
			name:    "WriteWarn renders an ordinary message unchanged",
			write:   diag.WriteWarn,
			message: `config: unrecognised key "models" in [tiers.standard]`,
			want:    "warn    config: unrecognised key \"models\" in [tiers.standard]\n",
		},
		{
			name:    "WriteError renders an ordinary message unchanged, padded to 8 columns",
			write:   diag.WriteError,
			message: "review requires a repository path",
			want:    "error:  review requires a repository path\n",
		},
		{
			name:    "WriteWarn escapes a newline so a diagnostic cannot forge a second warn line",
			write:   diag.WriteWarn,
			message: "rate limited\nwarn    a forged second warning",
			want:    `warn    rate limited\nwarn    a forged second warning` + "\n",
		},
		{
			name:    "WriteError escapes a newline so an error body cannot forge a done line ahead of the real one",
			write:   diag.WriteError,
			message: "rate limited\ndone    turns=1  in=0 out=0  $0.0000  0.0s  stop=ok",
			want:    `error:  rate limited\ndone    turns=1  in=0 out=0  $0.0000  0.0s  stop=ok` + "\n",
		},
		{
			name:    "WriteWarn escapes a newline and an ANSI escape in the same message",
			write:   diag.WriteWarn,
			message: "warning\x1b[31m: bad\nrate limited",
			want:    `warn    warning\x1b[31m: bad\nrate limited` + "\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			tc.write(&buf, tc.message)

			got := buf.String()
			if got != tc.want {
				t.Errorf("wrote %q, want %q", got, tc.want)
			}
			if n := strings.Count(got, "\n"); n != 1 {
				t.Errorf("wrote %d newlines in %q, want exactly 1 (the line's own terminator)", n, got)
			}
		})
	}
}
