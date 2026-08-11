package cli_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"

	"github.com/julienlegoux/kern-link/ai/providers/faux"

	"github.com/julienlegoux/external-reviewer/internal/cli"
)

// failingStdout is the stdout a test controls: it hands back the first accept
// bytes and then fails with err, so both halves of a write failure — nothing
// written at all, and a prefix already gone out — are drivable black-box
// through Run without a real file descriptor.
type failingStdout struct {
	accept  int
	err     error
	written bytes.Buffer
	calls   int
}

func (w *failingStdout) Write(p []byte) (int, error) {
	w.calls++
	n := len(p)
	if n > w.accept {
		n = w.accept
	}
	w.written.Write(p[:n])
	w.accept -= n
	if n < len(p) {
		return n, w.err
	}
	return n, nil
}

// countingStdout is a plain io.Writer and deliberately not an
// io.StringWriter, so the success path is exercised through Write rather than
// through *os.File's and bytes.Buffer's WriteString shortcut.
type countingStdout struct {
	written bytes.Buffer
	calls   int
}

func (w *countingStdout) Write(p []byte) (int, error) {
	w.calls++
	return w.written.Write(p)
}

// errorLines returns every "error:" line on a transcript. The seam promises
// exactly one per exit-2 termination (issue 07), and a write failure is an
// ordinary exit-2 termination rather than an escape hatch around it.
func errorLines(stderr string) []string {
	var out []string
	for _, line := range strings.Split(stderr, "\n") {
		if strings.HasPrefix(line, "error:") {
			out = append(out, line)
		}
	}
	return out
}

// runReviewInto is runReviewOver with the stdout the caller supplies, so a
// test can fail the report's one write.
func runReviewInto(t *testing.T, stdout io.Writer, answer string) (int, string) {
	t.Helper()
	defer cli.SetModelsForTest(scripted(t, 0, faux.Step(faux.TextMessage(answer, nil))))()

	var stderr bytes.Buffer
	code := cli.Run([]string{"review", "--prompt", "review this", t.TempDir()},
		strings.NewReader(""), stdout, &stderr)
	return code, stderr.String()
}

// TestRun_StdoutRefusesEveryByte_ExitsTwo is the broken-pipe classification
// with the signal taken out of it: a stdout that returns syscall.EPIPE
// without accepting a byte. The run must come out of the ordinary exit-2
// seam — one error: line, a done line, stop=failed — and stdout stays empty,
// so the "empty on failure" half of the invariant still holds here.
func TestRun_StdoutRefusesEveryByte_ExitsTwo(t *testing.T) {
	stdout := &failingStdout{accept: 0, err: syscall.EPIPE}

	code, stderr := runReviewInto(t, stdout, markdownAnswer)

	if code != 2 {
		t.Errorf("exit code = %d, want 2 (stderr: %q)", code, stderr)
	}
	if stdout.written.Len() != 0 {
		t.Errorf("stdout = %q, want empty when no byte was accepted", stdout.written.String())
	}
	if lines := errorLines(stderr); len(lines) != 1 {
		t.Errorf("stderr carries %d error: lines, want exactly 1: %q", len(lines), stderr)
	}
	if fields := parseDoneLine(t, stderr); fields.stop != "failed" {
		t.Errorf("done stop = %q, want failed", fields.stop)
	}
	assertOnlyKnownPrefixedLines(t, stderr)
}

// TestRun_StdoutFailsMidReport_NamesTheReportIncomplete is the judgment this
// issue asked to be made explicit. The bytes already on stdout cannot be
// unwritten, so the run cannot restore an empty stdout; what it can do is
// terminate through the same seam every other failure does and say on stderr
// that what the caller captured is a fragment. A caller that reads the
// transcript can then tell a truncated report from a complete one.
func TestRun_StdoutFailsMidReport_NamesTheReportIncomplete(t *testing.T) {
	const accepted = 12
	stdout := &failingStdout{accept: accepted, err: syscall.ENOSPC}

	code, stderr := runReviewInto(t, stdout, markdownAnswer)

	if code != 2 {
		t.Errorf("exit code = %d, want 2 (stderr: %q)", code, stderr)
	}
	if got := stdout.written.String(); got != markdownAnswer[:accepted] {
		t.Errorf("stdout = %q, want the %d bytes the writer accepted", got, accepted)
	}
	lines := errorLines(stderr)
	if len(lines) != 1 {
		t.Fatalf("stderr carries %d error: lines, want exactly 1: %q", len(lines), stderr)
	}
	if !strings.Contains(lines[0], "incomplete") {
		t.Errorf("error line = %q, want it to name the report as incomplete", lines[0])
	}
	counts := fmt.Sprintf("%d of %d", accepted, len(markdownAnswer))
	if !strings.Contains(lines[0], counts) {
		t.Errorf("error line = %q, want it to carry %q — the bytes written and the report's own length",
			lines[0], counts)
	}
	if fields := parseDoneLine(t, stderr); fields.stop != "failed" {
		t.Errorf("done stop = %q, want failed", fields.stop)
	}
	assertOnlyKnownPrefixedLines(t, stderr)
}

// TestRun_StdoutShortWriteWithoutError_IsStillIncomplete covers the writer
// that breaks io.Writer's own contract by reporting a short write with a nil
// error. os.File cannot do this — internal/poll loops — but the seam must not
// depend on that: an incomplete report reported as a success is the one
// outcome the calling skill cannot recover from.
func TestRun_StdoutShortWriteWithoutError_IsStillIncomplete(t *testing.T) {
	stdout := &failingStdout{accept: 5, err: nil}

	code, stderr := runReviewInto(t, stdout, markdownAnswer)

	if code != 2 {
		t.Errorf("exit code = %d, want 2 (stderr: %q)", code, stderr)
	}
	lines := errorLines(stderr)
	if len(lines) != 1 {
		t.Fatalf("stderr carries %d error: lines, want exactly 1: %q", len(lines), stderr)
	}
	if !strings.Contains(lines[0], "incomplete") {
		t.Errorf("error line = %q, want it to name the report as incomplete", lines[0])
	}
}

// TestRun_SuccessfulReport_ReachesAPlainWriterIntact is the other side of the
// classification: nothing above may cost the success path. A complete write
// through a bare io.Writer still exits 0 with the report byte for byte and no
// error: line at all.
func TestRun_SuccessfulReport_ReachesAPlainWriterIntact(t *testing.T) {
	stdout := &countingStdout{}

	code, stderr := runReviewInto(t, stdout, markdownAnswer)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	if got := stdout.written.String(); got != markdownAnswer {
		t.Errorf("stdout = %q, want the report byte for byte: %q", got, markdownAnswer)
	}
	if stdout.calls != 1 {
		t.Errorf("stdout was written %d times, want exactly 1 — the report goes out once, at the end", stdout.calls)
	}
	if lines := errorLines(stderr); len(lines) != 0 {
		t.Errorf("stderr carries %d error: lines on the success path, want 0: %q", len(lines), stderr)
	}
}

// brokenPipeChildEnv opts one process into being the child of
// TestRun_ClosedStdoutPipe_NeverDiesBySignal rather than an ordinary test. It
// is checked at runtime rather than behind a build tag, so the whole
// arrangement compiles and runs on both matrix OSes.
const brokenPipeChildEnv = "EXTERNAL_REVIEWER_BROKEN_PIPE_CHILD"

// brokenPipeReport is deliberately larger than any pipe buffer either OS
// gives an anonymous pipe, so the write cannot be absorbed and silently
// succeed on a platform whose pipe semantics differ from Linux's.
var brokenPipeReport = strings.Repeat("# Review\n\n- a finding that goes on and on\n\n", 4000)

// TestBrokenStdoutPipeChild is the child half of the closed-pipe test, not a
// test of its own: without the environment variable it skips. Run as a child
// it is a faithful external-reviewer process — real Run, real os.Stdout on
// fd 1, an offline registry — that exits with the code Run returns, so the
// parent can read the exit status a shell would have read.
func TestBrokenStdoutPipeChild(t *testing.T) {
	if os.Getenv(brokenPipeChildEnv) == "" {
		t.Skipf("%s is not set: this is the child half of the closed-pipe test", brokenPipeChildEnv)
	}

	// Block until the parent has closed the read end of our stdout. Reaching
	// EOF on stdin is that handshake; the CLI never reads stdin itself here
	// because --prompt is supplied.
	_, _ = io.Copy(io.Discard, os.Stdin)

	defer cli.SetModelsForTest(scripted(t, 0, faux.Step(faux.TextMessage(brokenPipeReport, nil))))()
	code := cli.Run([]string{"review", "--prompt", "review this", t.TempDir()},
		strings.NewReader(""), os.Stdout, os.Stderr)
	os.Exit(code)
}

// TestRun_ClosedStdoutPipe_NeverDiesBySignal is the acceptance criterion
// `external-reviewer review <repo> --prompt "…" | head -20` states, run as a
// process rather than by hand: a real child with a real pipe on fd 1 whose
// reader is gone before it writes — exactly what a `head` that has already
// exited leaves behind.
//
// On Linux that is SIGPIPE, and the Go runtime's default rule kills a process
// that takes one on fd 1 or 2, which is why the shell sees 141 and the done
// line never gets written. On Windows there is no SIGPIPE at all and the
// write simply fails. The assertion is therefore selected at runtime rather
// than compiled per OS: whatever the platform does to the write, the process
// must terminate through the seam with a real exit code and a done line, and
// must never be killed by a signal.
func TestRun_ClosedStdoutPipe_NeverDiesBySignal(t *testing.T) {
	pipeRead, pipeWrite, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating the stdout pipe: %v", err)
	}
	stdinRead, stdinWrite, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating the handshake pipe: %v", err)
	}

	// os.Args[0] is this very test binary and the argument is a constant, so
	// there is no input to taint; re-executing the test binary is the only way
	// to get a real pipe onto a real fd 1. -test.paniconexit0 is deliberately
	// not passed on, so the child's os.Exit carries Run's code out unchanged.
	//nolint:gosec // G204/G702: the command is this process's own binary and a literal flag.
	cmd := exec.Command(os.Args[0], "-test.run=^TestBrokenStdoutPipeChild$")
	cmd.Env = append(os.Environ(), brokenPipeChildEnv+"=1")
	cmd.Stdin = stdinRead
	cmd.Stdout = pipeWrite
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("starting the child process: %v", err)
	}

	// Drop every handle the parent holds, then the reader itself: from here
	// the child's fd 1 is a pipe nobody is reading.
	_ = pipeWrite.Close()
	_ = stdinRead.Close()
	if err := pipeRead.Close(); err != nil {
		t.Fatalf("closing the read end of the child's stdout: %v", err)
	}
	_ = stdinWrite.Close()

	waitErr := cmd.Wait()
	state := cmd.ProcessState
	if state == nil {
		t.Fatalf("waiting for the child: %v", waitErr)
	}

	code := state.ExitCode()
	if code < 0 {
		t.Fatalf("the child was killed by a signal (%v) instead of terminating through the seam — "+
			"a shell would report 141 here (stderr: %q)", state, stderr.String())
	}
	if code != 0 && code != 1 && code != 2 {
		t.Errorf("child exit code = %d, want 0, 1 or 2 (stderr: %q)", code, stderr.String())
	}
	if fields := parseDoneLine(t, stderr.String()); fields.stop == "" {
		t.Errorf("done line carries no stop reason: %q", stderr.String())
	}
}
