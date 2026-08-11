package cli_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/julienlegoux/external-reviewer/internal/cli"
	"github.com/julienlegoux/external-reviewer/internal/fauxtest"
	"github.com/julienlegoux/external-reviewer/internal/reviewer"
)

// TestRun_Review drives the review/help subcommand grammar through Run,
// table-driven over every usage case the invocation grammar has to reject
// (or accept) before anything talks to a model. The rows that get past
// parsing run pre-flight for real, so the whole table resolves against an
// offline registry.
func TestRun_Review(t *testing.T) {
	tests := []struct {
		name             string
		argv             func(t *testing.T) []string
		stdin            string
		wantExit         int
		wantStdoutEmpty  bool
		wantStderrNonEmp bool
	}{
		{
			name: "prompt flag and repo path parses and reaches the run path",
			argv: func(t *testing.T) []string {
				return []string{"review", "--prompt", "review this", t.TempDir()}
			},
			wantExit:        0,
			wantStdoutEmpty: false,
		},
		{
			name: "prompt from stdin parses identically",
			argv: func(t *testing.T) []string {
				return []string{"review", t.TempDir()}
			},
			stdin:           "review this",
			wantExit:        0,
			wantStdoutEmpty: false,
		},
		{
			name: "missing repository path is a usage error",
			argv: func(t *testing.T) []string {
				return []string{"review", "--prompt", "x"}
			},
			wantExit:         2,
			wantStdoutEmpty:  true,
			wantStderrNonEmp: true,
		},
		{
			name: "extra positional arguments are a usage error",
			argv: func(t *testing.T) []string {
				return []string{"review", "--prompt", "x", t.TempDir(), "extra"}
			},
			wantExit:         2,
			wantStdoutEmpty:  true,
			wantStderrNonEmp: true,
		},
		{
			name: "nonexistent repository path is a usage error",
			argv: func(t *testing.T) []string {
				return []string{"review", "--prompt", "x", filepath.Join(t.TempDir(), "does-not-exist")}
			},
			wantExit:         2,
			wantStdoutEmpty:  true,
			wantStderrNonEmp: true,
		},
		{
			name: "a repository path to a regular file is a usage error",
			argv: func(t *testing.T) []string {
				dir := t.TempDir()
				file := filepath.Join(dir, "not-a-directory.txt")
				if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
					t.Fatalf("WriteFile fixture: %v", err)
				}
				return []string{"review", "--prompt", "x", file}
			},
			wantExit:         2,
			wantStdoutEmpty:  true,
			wantStderrNonEmp: true,
		},
		{
			name: "empty stdin with no prompt flag is a usage error",
			argv: func(t *testing.T) []string {
				return []string{"review", t.TempDir()}
			},
			stdin:            "",
			wantExit:         2,
			wantStdoutEmpty:  true,
			wantStderrNonEmp: true,
		},
		{
			name: "an unknown flag is a usage error",
			argv: func(t *testing.T) []string {
				return []string{"review", "--nope", t.TempDir()}
			},
			wantExit:         2,
			wantStdoutEmpty:  true,
			wantStderrNonEmp: true,
		},
		{
			name: "an unknown subcommand is a usage error",
			argv: func(t *testing.T) []string {
				return []string{"frobnicate"}
			},
			wantExit:         2,
			wantStdoutEmpty:  true,
			wantStderrNonEmp: true,
		},
		{
			name: "help prints usage and exits 0",
			argv: func(t *testing.T) []string {
				return []string{"help"}
			},
			wantExit:        0,
			wantStdoutEmpty: false,
		},
		{
			name: "--help prints usage and exits 0",
			argv: func(t *testing.T) []string {
				return []string{"--help"}
			},
			wantExit:        0,
			wantStdoutEmpty: false,
		},
		{
			name: "-h prints usage and exits 0",
			argv: func(t *testing.T) []string {
				return []string{"-h"}
			},
			wantExit:        0,
			wantStdoutEmpty: false,
		},
		{
			// review --help must exit 0 like top-level help, not the exit-2
			// usage error flag.ErrHelp used to be classified as.
			name: "review --help prints usage and exits 0",
			argv: func(t *testing.T) []string {
				return []string{"review", "--help"}
			},
			wantExit:        0,
			wantStdoutEmpty: false,
		},
		{
			name: "review -h prints usage and exits 0",
			argv: func(t *testing.T) []string {
				return []string{"review", "-h"}
			},
			wantExit:        0,
			wantStdoutEmpty: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// A registry per row rather than one for the table: each row that
			// gets past parsing consumes a scripted response, and a shared
			// faux provider would leave the second one talking to an empty
			// queue.
			argv := tc.argv(t)

			var stdout, stderr bytes.Buffer
			code := cli.RunForTest(argv, strings.NewReader(tc.stdin), &stdout, &stderr,
				registry(t, fauxtest.CredentialedAuth("OAuth"), nil, reviewer.DefaultModelID))

			if code != tc.wantExit {
				t.Errorf("exit code = %d, want %d (stderr: %q)", code, tc.wantExit, stderr.String())
			}
			if tc.wantStdoutEmpty && stdout.Len() != 0 {
				t.Errorf("stdout = %q, want empty", stdout.String())
			}
			if !tc.wantStdoutEmpty && stdout.Len() == 0 {
				t.Error("stdout is empty, want non-empty")
			}
			if tc.wantStderrNonEmp && stderr.Len() == 0 {
				t.Error("stderr is empty, want a reason naming what was wrong")
			}
			assertOnlyKnownPrefixedLines(t, stderr.String())
			if tc.wantExit == 2 {
				if n := strings.Count(stderr.String(), "error:  "); n != 1 {
					t.Errorf("stderr = %q, want exactly one error: line, found %d", stderr.String(), n)
				}
			}
		})
	}
}

// errStdin is an io.Reader that always fails, used to exercise the "stdin
// read failure" exit-2 path — the one exit-2 reason the existing table
// cannot reach with a plain strings.Reader.
type errStdin struct{}

func (errStdin) Read([]byte) (int, error) { return 0, errors.New("device error") }

// TestRun_Review_StdinReadFailure_ExitsTwo covers the one exit-2 usage
// reason not reachable through TestRun_Review's table: a stdin read that
// itself errors, rather than merely returning nothing.
func TestRun_Review_StdinReadFailure_ExitsTwo(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{"review", t.TempDir()}, errStdin{}, &stdout, &stderr)

	if code != 2 {
		t.Errorf("exit code = %d, want 2 (stderr: %q)", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "error:  reading task prompt from stdin") {
		t.Errorf("stderr = %q, want an error: line naming the stdin failure", stderr.String())
	}
	assertOnlyKnownPrefixedLines(t, stderr.String())
	if fields := parseDoneLine(t, stderr.String()); fields.stop != "usage" {
		t.Errorf("done stop = %q, want usage", fields.stop)
	}
}

// TestRun_NoReviewerReached_WritesNoErrorLine is the exit-1 path's silence,
// asserted rather than assumed: ErrNoReviewer is SPECS' silent-fallback
// case, the one termination that gets no error: line at all. Adding
// fmt.Fprintln(stderr, "error:", err) to runReview's errors.Is(err,
// ErrNoReviewer) branch is the change that must turn this red — the missing
// half of the pair TestRun_BrokenCredential_ExitsTwo already writes for the
// exit-2 sibling.
func TestRun_NoReviewerReached_WritesNoErrorLine(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cli.RunForTest([]string{"review", "--prompt", "x", t.TempDir()}, strings.NewReader(""), &stdout, &stderr,
		registry(t, fauxtest.UnconfiguredAuth(), nil, reviewer.DefaultModelID))

	if code != 1 {
		t.Fatalf("exit code = %d, want 1 (stderr: %q)", code, stderr.String())
	}
	if strings.Contains(stderr.String(), "error:") {
		t.Errorf("stderr = %q, want no error: line on the silent-fallback exit-1 path", stderr.String())
	}
	if fields := parseDoneLine(t, stderr.String()); fields.stop != "no_reviewer" {
		t.Errorf("done stop = %q, want no_reviewer", fields.stop)
	}
}

func TestRun_Review_NoTestCausesProcessExit(t *testing.T) {
	// A regression guard: FlagSet must be ContinueOnError, never ExitOnError,
	// or a bad flag would call os.Exit from inside this test process. The
	// table above already exercises "--nope"; reaching this line at all is
	// the assertion — os.Exit would have terminated the test binary instead.
	var stdout, stderr bytes.Buffer
	_ = cli.Run([]string{"review", "--nope", t.TempDir()}, strings.NewReader(""), &stdout, &stderr)
}
