package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/julienlegoux/external-reviewer/internal/cli"
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
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// A registry per row rather than one for the table: each row that
			// gets past parsing consumes a scripted response, and a shared
			// faux provider would leave the second one talking to an empty
			// queue.
			defer cli.SetModelsForTest(registry(t, credentialedAuth("OAuth"), nil, reviewer.DefaultModelID))()

			argv := tc.argv(t)

			var stdout, stderr bytes.Buffer
			code := cli.Run(argv, strings.NewReader(tc.stdin), &stdout, &stderr)

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
