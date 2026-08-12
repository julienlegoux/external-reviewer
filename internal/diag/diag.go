// Package diag implements the run's stderr diagnostics: prefixed,
// human-readable lines with no colour and no spinner, and the run state the
// "done" line renders from (SPECS § Interfaces).
package diag

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode"
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
//
// summary is built from the model's own tool-call arguments, so it is
// provider-controlled text and gets the same escapeControlChars treatment
// WriteWarn and WriteError get, for the same reason: without it a newline in
// an argument could forge a second, well-formed line — a fake "done … stop=ok"
// ahead of the real one — or an ANSI/OSC sequence could reach the terminal
// raw. This function had no caller when that defence was added to the other
// two (Epic 0 issue 12); the loop is its first (Epic 2 issue 03).
func WriteTool(w io.Writer, summary string) {
	_, _ = fmt.Fprintf(w, "tool    %s\n", escapeControlChars(summary))
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
// AssistantMessageDiagnostic or an unrecognised configuration key. message
// is escaped by escapeControlChars first: no version of kern-link redacts a
// diagnostic's message (issue 12), so this is provider-controlled text, and
// without escaping, a newline in it could forge a second line — a fake
// "done … stop=ok" ahead of the real one — or an ANSI/OSC sequence could
// reach the terminal raw. Escaping control characters is not redaction: a
// credential value that reached this function would still print, verbatim
// with its control characters spelled out; nothing here removes or scans
// for one (SPECS § Security — this binary formats no credential in the
// first place).
func WriteWarn(w io.Writer, message string) {
	_, _ = fmt.Fprintf(w, "warn    %s\n", escapeControlChars(message))
}

// WriteError writes the "error:" line naming what was wrong on an exit-2
// path — the one prefixed rendering every hand-rolled fmt.Fprint* used to
// duplicate across internal/cli, padded to the same 8-column width every
// other diag.Write* uses ("error:" is 6 characters, so 2 trailing spaces
// rather than WriteWarn's 4). For a hand-written call site message is
// already lowercase and unpunctuated, stating what was attempted
// (CONVENTIONS § Error handling); for a reached-and-failed turn it also
// carries err.Error(), which can hold a provider's own error body verbatim.
// Either way message is escaped by escapeControlChars first — the same
// defence WriteWarn applies, for the same reason: a newline in a provider's
// error text must not forge a second, well-formed line ahead of the real
// done line. Nothing here formats or classifies the underlying error, only
// renders text a caller already decided to print. It is never called on the
// exit-1 silent-fallback path — SPECS calls that the case the caller needs
// no line about.
func WriteError(w io.Writer, message string) {
	_, _ = fmt.Fprintf(w, "error:  %s\n", escapeControlChars(message))
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
// from a minute up to an hour ("2m14s"), and hours, minutes and whole
// seconds at or above an hour ("1h35m0s") — a run reviewing a large
// repository can run long enough that "95m0s" would otherwise be the only
// rendering with no unit boundary above minutes.
func formatElapsed(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%.1fs", d.Seconds())
	case d < time.Hour:
		m := d / time.Minute
		s := (d % time.Minute) / time.Second
		return fmt.Sprintf("%dm%ds", m, s)
	default:
		h := d / time.Hour
		m := (d % time.Hour) / time.Minute
		s := (d % time.Minute) / time.Second
		return fmt.Sprintf("%dh%dm%ds", h, m, s)
	}
}

// escapeControlChars renders s safe for a single line of the line-oriented
// stderr protocol SPECS fixes: every rune unicode.IsControl reports true for
// — C0 controls including "\n", "\r" and ESC (0x1B, the byte every ANSI/OSC
// sequence opens with), plus the C1 range — is replaced by its Go escape
// ("\n", "\x1b", …) via strconv.QuoteRune with the surrounding quotes
// stripped. Everything else, any printable script included, passes through
// unchanged. This is deliberately narrow: it stops a control character from
// splitting the line or reaching the terminal raw, and it is not a general
// string sanitiser or a secret scanner — a credential that reached this
// function would still print, just with any control characters in it spelled
// out rather than a redaction removing them (issue 12's Out of scope).
func escapeControlChars(s string) string {
	if !strings.ContainsFunc(s, unicode.IsControl) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if !unicode.IsControl(r) {
			b.WriteRune(r)
			continue
		}
		q := strconv.QuoteRune(r)
		b.WriteString(q[1 : len(q)-1]) // strip the surrounding single quotes
	}
	return b.String()
}
