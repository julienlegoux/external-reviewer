package cli_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/julienlegoux/external-reviewer/internal/cli"
	"github.com/julienlegoux/external-reviewer/internal/fauxtest"
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
				return []string{"review", "--allow", ".", "--system", fixtureSystem, "--prompt", "review this", t.TempDir()}
			},
			wantExit:        0,
			wantStdoutEmpty: false,
		},
		{
			name: "the request object on stdin parses identically",
			argv: func(t *testing.T) []string {
				return []string{"review", "--allow", ".", t.TempDir()}
			},
			stdin:           requestObject(fixtureSystem, "review this"),
			wantExit:        0,
			wantStdoutEmpty: false,
		},
		{
			name: "missing repository path is a usage error",
			argv: func(t *testing.T) []string {
				return []string{"review", "--allow", ".", "--system", fixtureSystem, "--prompt", "x"}
			},
			wantExit:         2,
			wantStdoutEmpty:  true,
			wantStderrNonEmp: true,
		},
		{
			name: "extra positional arguments are a usage error",
			argv: func(t *testing.T) []string {
				return []string{"review", "--allow", ".", "--system", fixtureSystem, "--prompt", "x", t.TempDir(), "extra"}
			},
			wantExit:         2,
			wantStdoutEmpty:  true,
			wantStderrNonEmp: true,
		},
		{
			name: "nonexistent repository path is a usage error",
			argv: func(t *testing.T) []string {
				return []string{"review", "--allow", ".", "--system", fixtureSystem, "--prompt", "x", filepath.Join(t.TempDir(), "does-not-exist")}
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
				return []string{"review", "--allow", ".", "--system", fixtureSystem, "--prompt", "x", file}
			},
			wantExit:         2,
			wantStdoutEmpty:  true,
			wantStderrNonEmp: true,
		},
		{
			name: "empty stdin with no prompt flag is a usage error",
			argv: func(t *testing.T) []string {
				return []string{"review", "--allow", ".", t.TempDir()}
			},
			stdin:            "",
			wantExit:         2,
			wantStdoutEmpty:  true,
			wantStderrNonEmp: true,
		},
		{
			// echo with nothing to say pipes a single newline, not an
			// empty stream. Since issue 08 both are the same refusal for
			// the same reason -- neither is a JSON request object -- but
			// the row stays, because the two are distinct states of a real
			// pipe and a decoder could plausibly accept one of them.
			name: "whitespace-only stdin (echo with no arguments) is a usage error",
			argv: func(t *testing.T) []string {
				return []string{"review", "--allow", ".", t.TempDir()}
			},
			stdin:            "\n",
			wantExit:         2,
			wantStdoutEmpty:  true,
			wantStderrNonEmp: true,
		},
		{
			name: "whitespace-only --prompt is a usage error",
			argv: func(t *testing.T) []string {
				return []string{"review", "--allow", ".", "--system", fixtureSystem, "--prompt", " ", t.TempDir()}
			},
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
				registry(t, fauxtest.CredentialedAuth("OAuth"), nil, fixtureModel))

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

// TestRun_Review_WithoutAllow_IsUsageError is the epic's confinement made
// non-optional at the grammar: a review that granted nothing reads nothing,
// and saying so is a usage error rather than an implicit run over the whole
// tree. `--allow .` grants everything and has to be said.
func TestRun_Review_WithoutAllow_IsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cli.RunForTest([]string{"review", "--system", fixtureSystem, "--prompt", "x", t.TempDir()}, strings.NewReader(""), &stdout, &stderr,
		registry(t, fauxtest.CredentialedAuth("OAuth"), nil, fixtureModel))

	if code != 2 {
		t.Fatalf("exit code = %d, want 2 (stderr: %q)", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "--allow") {
		t.Errorf("stderr = %q, want an error: line naming --allow as missing", stderr.String())
	}
	if n := strings.Count(stderr.String(), "error:  "); n != 1 {
		t.Errorf("stderr = %q, want exactly one error: line", stderr.String())
	}
	assertOnlyKnownPrefixedLines(t, stderr.String())
	if fields := fauxtest.ParseDoneLine(t, stderr.String()); fields.Stop != "usage" {
		t.Errorf("done stop = %q, want usage", fields.Stop)
	}
}

// TestRun_Review_RepeatedAllow_ParsesEveryValue: --allow is repeatable, and a
// run that grants two subtrees reaches exactly the point a run with one does —
// the same success path Epic 1 issue 02's parsing tests assert, now with the
// grant this epic requires.
func TestRun_Review_RepeatedAllow_ParsesEveryValue(t *testing.T) {
	repo := t.TempDir()
	writeFixture(t, repo, filepath.Join("docs", "notes.md"), "notes\n")
	writeFixture(t, repo, filepath.Join("internal", "cli", "run.go"), "package cli\n")

	var stdout, stderr bytes.Buffer
	code := cli.RunForTest(
		[]string{"review", "--allow", "docs", "--allow", "internal", "--system", fixtureSystem, "--prompt", "review this", repo},
		strings.NewReader(""), &stdout, &stderr,
		registry(t, fauxtest.CredentialedAuth("OAuth"), nil, fixtureModel))

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr.String())
	}
	if stdout.Len() == 0 {
		t.Error("stdout is empty, want the reviewer's answer: both --allow values must parse")
	}
	assertOnlyKnownPrefixedLines(t, stderr.String())
}

// TestRun_Review_AllowValueThatIsNotASubtree_IsUsageError: a grant names an
// existing directory. A regular file and a path that is not there are both
// refused at exit 2, naming the offending value, and — asserted by the absence
// of the model line pre-flight writes as soon as a reviewer resolves — before
// anything talks to a model.
func TestRun_Review_AllowValueThatIsNotASubtree_IsUsageError(t *testing.T) {
	tests := []struct {
		name  string
		allow string
	}{
		{name: "a regular file", allow: "README.md"},
		{name: "a path that does not exist", allow: "nowhere"},
		{name: "traversal out of the repository", allow: "../outside"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := t.TempDir()
			writeFixture(t, repo, "README.md", "# fixture\n")

			var stdout, stderr bytes.Buffer
			code := cli.RunForTest([]string{"review", "--allow", tc.allow, "--system", fixtureSystem, "--prompt", "x", repo},
				strings.NewReader(""), &stdout, &stderr,
				registry(t, fauxtest.CredentialedAuth("OAuth"), nil, fixtureModel))

			if code != 2 {
				t.Fatalf("exit code = %d, want 2 (stderr: %q)", code, stderr.String())
			}
			if stdout.Len() != 0 {
				t.Errorf("stdout = %q, want empty", stdout.String())
			}
			if !strings.Contains(stderr.String(), fmt.Sprintf("%q", tc.allow)) {
				t.Errorf("stderr = %q, want it to name the offending value %q", stderr.String(), tc.allow)
			}
			if strings.Contains(stderr.String(), "model   ") {
				t.Errorf("stderr = %q carries a model line: the grant must be rejected before any model request", stderr.String())
			}
			if n := strings.Count(stderr.String(), "error:  "); n != 1 {
				t.Errorf("stderr = %q, want exactly one error: line", stderr.String())
			}
			assertOnlyKnownPrefixedLines(t, stderr.String())
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
	code := cli.Run([]string{"review", "--allow", ".", t.TempDir()}, errStdin{}, &stdout, &stderr)

	if code != 2 {
		t.Errorf("exit code = %d, want 2 (stderr: %q)", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "error:  reading the request object from stdin") {
		t.Errorf("stderr = %q, want an error: line naming the stdin failure", stderr.String())
	}
	assertOnlyKnownPrefixedLines(t, stderr.String())
	if fields := fauxtest.ParseDoneLine(t, stderr.String()); fields.Stop != "usage" {
		t.Errorf("done stop = %q, want usage", fields.Stop)
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
	code := cli.RunForTest([]string{"review", "--allow", ".", "--system", fixtureSystem, "--prompt", "x", t.TempDir()}, strings.NewReader(""), &stdout, &stderr,
		registry(t, fauxtest.UnconfiguredAuth(), nil, fixtureModel))

	if code != 1 {
		t.Fatalf("exit code = %d, want 1 (stderr: %q)", code, stderr.String())
	}
	if strings.Contains(stderr.String(), "error:") {
		t.Errorf("stderr = %q, want no error: line on the silent-fallback exit-1 path", stderr.String())
	}
	if fields := fauxtest.ParseDoneLine(t, stderr.String()); fields.Stop != "no_reviewer" {
		t.Errorf("done stop = %q, want no_reviewer", fields.Stop)
	}
}

// TestRun_Review_PathDiagnostics_AreWireForm proves the OS-path/wire-path
// boundary CONVENTIONS § Paths and platforms names is actually implemented:
// the repository path reaches stderr /-separated, with no OS separator and
// no OS-authored sentence, and the wording carries no trailing full stop.
// The expectation is built from filepath.ToSlash of the same native path Run
// is given, so this passes unchanged on ubuntu-latest and windows-latest —
// no runtime.GOOS branch and no //go:build divergence.
func TestRun_Review_PathDiagnostics_AreWireForm(t *testing.T) {
	tests := []struct {
		name       string
		buildPath  func(t *testing.T) string
		wantSuffix string
	}{
		{
			name: "nonexistent repository path",
			buildPath: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "does-not-exist")
			},
			wantSuffix: "could not be accessed",
		},
		{
			name: "repository path is a regular file",
			buildPath: func(t *testing.T) string {
				dir := t.TempDir()
				file := filepath.Join(dir, "not-a-directory.txt")
				if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
					t.Fatalf("WriteFile fixture: %v", err)
				}
				return file
			},
			wantSuffix: "is not a directory",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			nativePath := tc.buildPath(t)
			wirePath := filepath.ToSlash(filepath.Clean(nativePath))

			var stdout, stderr bytes.Buffer
			code := cli.Run([]string{"review", "--allow", ".", "--system", fixtureSystem, "--prompt", "x", nativePath}, strings.NewReader(""), &stdout, &stderr)

			if code != 2 {
				t.Fatalf("exit code = %d, want 2 (stderr: %q)", code, stderr.String())
			}
			if stdout.Len() != 0 {
				t.Errorf("stdout = %q, want empty", stdout.String())
			}

			want := fmt.Sprintf("error:  repository path %q %s\n", wirePath, tc.wantSuffix)
			if !strings.Contains(stderr.String(), want) {
				t.Errorf("stderr = %q, want a line %q", stderr.String(), want)
			}
			if strings.Contains(stderr.String(), `\`) {
				t.Errorf("stderr = %q contains a backslash: the path must reach stderr in wire form", stderr.String())
			}
			assertOnlyKnownPrefixedLines(t, stderr.String())
		})
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

// TestRun_Review_PromptFlagEmptyString_IsUsageError pins the flag-presence
// check's fs.Visit form: `--prompt ""` must be read as "the flag was given,
// with an empty value" — not "the flag was not given, fall back to stdin".
// Stdin here carries a real, non-empty prompt so the two readings diverge:
// the correct fs.Visit-based check takes the (empty) flag value and rejects
// it, exit 2, without ever touching stdin. The mutation this pins —
// collapsing the check to `promptSet := *prompt != ""` — would instead see
// *prompt == "" and read the mismatched value as "not set", fall back to the
// stdin prompt below, and succeed at exit 0 with the reviewer's answer on
// stdout. Applying that exact mutation by hand and re-running this test
// turns it red, as recorded in this issue's PR body.
func TestRun_Review_PromptFlagEmptyString_IsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cli.RunForTest([]string{"review", "--allow", ".", "--system", fixtureSystem, "--prompt", "", t.TempDir()}, strings.NewReader("review this"), &stdout, &stderr,
		registry(t, fauxtest.CredentialedAuth("OAuth"), nil, fixtureModel))

	if code != 2 {
		t.Fatalf("exit code = %d, want 2 (stderr: %q)", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty — the empty --prompt value must not fall back to reading stdin", stdout.String())
	}
	if n := strings.Count(stderr.String(), "error:  "); n != 1 {
		t.Errorf("stderr = %q, want exactly one error: line", stderr.String())
	}
	assertOnlyKnownPrefixedLines(t, stderr.String())
}
