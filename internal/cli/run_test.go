package cli_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/julienlegoux/external-reviewer/internal/cli"
)

func TestRun_EmptyArgv_UsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := cli.Run(nil, strings.NewReader(""), &stdout, &stderr)

	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
	if stderr.Len() == 0 {
		t.Error("stderr is empty, want a non-empty usage message")
	}
	fields := parseDoneLine(t, stderr.String())
	if fields.stop != "usage" {
		t.Errorf("done stop = %q, want usage", fields.stop)
	}
}

// TestRun_Help_DoneLineStopIsHelp pins the CLI word for the one termination
// the model never even gets a chance to run for: help/--help/-h all exit 0
// with a done line whose stop= reads "help", not the model's vocabulary
// (there was no turn to have one).
func TestRun_Help_DoneLineStopIsHelp(t *testing.T) {
	tests := []struct {
		name string
		argv []string
	}{
		{"help", []string{"help"}},
		{"--help", []string{"--help"}},
		{"-h", []string{"-h"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := cli.Run(tc.argv, strings.NewReader(""), &stdout, &stderr)

			if code != 0 {
				t.Errorf("exit code = %d, want 0 (stderr: %q)", code, stderr.String())
			}
			if fields := parseDoneLine(t, stderr.String()); fields.stop != "help" {
				t.Errorf("done stop = %q, want help", fields.stop)
			}
		})
	}
}

// TestRunContext_CancelledContext_ExitsTwo drives the interruption path
// directly against the inner run seam (RunWithModelsForTest) with a
// pre-cancelled context, since delivering a real SIGINT is not portable to
// the Windows CI runner — Run's signal.NotifyContext wiring itself is
// exercised only by constructing Run in the other tests in this package.
func TestRunContext_CancelledContext_ExitsTwo(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var stdout, stderr bytes.Buffer
	code := cli.RunWithModelsForTest(ctx, []string{"review", "--prompt", "x", t.TempDir()}, strings.NewReader(""), &stdout, &stderr, nil)

	if code != 2 {
		t.Errorf("exit code = %d, want 2 (stderr: %q)", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(strings.ToLower(stderr.String()), "interrupt") {
		t.Errorf("stderr = %q, want an interruption reason", stderr.String())
	}
	fields := parseDoneLine(t, stderr.String())
	if fields.stop != "interrupted" {
		t.Errorf("done stop = %q, want interrupted", fields.stop)
	}
	if fields.in != "0" || fields.out != "0" {
		t.Errorf("done in/out = %q/%q, want 0/0 (a path that never reached a model)", fields.in, fields.out)
	}
}
