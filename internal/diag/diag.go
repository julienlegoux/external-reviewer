// Package diag implements the run's stderr diagnostics: prefixed,
// human-readable lines with no colour and no spinner, and the run state the
// "done" line renders from (SPECS § Interfaces).
package diag

import (
	"fmt"
	"io"
	"time"
)

// State accumulates one run's diagnostics — turn count, accumulated input
// and output tokens, accumulated cost, start time and stop reason — in one
// struct so the "done" line can be rendered from it on every termination
// path. A zero State (via NewState) already carries everything a path that
// never reached a model needs: the token and turn fields stay zero rather
// than absent.
type State struct {
	Turns        int
	InputTokens  int64
	OutputTokens int64
	Cost         float64
	Start        time.Time
	StopReason   string
}

// NewState starts a run's diagnostic state with the clock running.
func NewState() *State {
	return &State{Start: time.Now()}
}

// WriteTurn writes a "turn" line: the running snapshot after one model round
// trip — turn number, tool-call count, accumulated tokens and cost so far,
// and how long the turn took.
func WriteTurn(w io.Writer, n, tools int, inTok, outTok int64, cost float64, elapsed time.Duration) {
	_, _ = fmt.Fprintf(w, "turn %d  tools=%d  in=%d out=%d  $%.4f  %s\n",
		n, tools, inTok, outTok, cost, formatElapsed(elapsed))
}

// WriteTool writes a "tool" line naming the call the reviewer just made.
func WriteTool(w io.Writer, summary string) {
	_, _ = fmt.Fprintf(w, "tool    %s\n", summary)
}

// WriteModel writes the "model" line pre-flight emits once a reviewer has
// been resolved:
//
//	model   openai-codex/gpt-5.5  auth=OAuth
//
// authSource is AuthResult.Source — a label like "OAuth" or
// "ANTHROPIC_API_KEY". It is the only credential-adjacent value that ever
// reaches stderr; this binary never sees a credential itself, and nothing
// here formats one.
func WriteModel(w io.Writer, provider, id, authSource string) {
	_, _ = fmt.Fprintf(w, "model   %s/%s  auth=%s\n", provider, id, authSource)
}

// WriteWarn writes a "warn" line for a non-fatal diagnostic — a kern-link
// AssistantMessageDiagnostic (already redacted) or an unrecognised
// configuration key.
func WriteWarn(w io.Writer, message string) {
	_, _ = fmt.Fprintf(w, "warn    %s\n", message)
}

// WriteDone writes the "done" line SPECS fixes:
//
//	done    turns=7  in=182430 out=9106  $0.0918  2m14s  stop=end_turn
//
// It is meant to be called on every termination path — success, no reviewer
// available, failure and interruption — so a transcript reader never has to
// sum the "turn" lines for totals; the elapsed time is measured from s.Start
// to the moment this is called.
func WriteDone(w io.Writer, s *State) {
	_, _ = fmt.Fprintf(w, "done    turns=%d  in=%d out=%d  $%.4f  %s  stop=%s\n",
		s.Turns, s.InputTokens, s.OutputTokens, s.Cost, formatElapsed(time.Since(s.Start)), s.StopReason)
}

// formatElapsed renders a duration the way SPECS' examples show it: seconds
// with one decimal place under a minute ("42.8s"), minutes and whole seconds
// at or above a minute ("2m14s").
func formatElapsed(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	m := d / time.Minute
	s := (d % time.Minute) / time.Second
	return fmt.Sprintf("%dm%ds", m, s)
}
