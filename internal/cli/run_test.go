package cli_test

import (
	"bytes"
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
}
