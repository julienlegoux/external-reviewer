package cli_test

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/julienlegoux/kern-link/ai"
	"github.com/julienlegoux/kern-link/ai/providers/faux"

	"github.com/julienlegoux/external-reviewer/internal/cli"
	"github.com/julienlegoux/external-reviewer/internal/fauxtest"
)

// capturedRequest is what a scripted faux step saw of the request that
// actually reached the model: the system prompt kern-link carries on
// ai.Context, and the first user message's plain text. Asserting on these
// rather than on stderr is what the acceptance criteria ask for — stderr can
// only ever prove that the binary *thought* it sent the right thing.
type capturedRequest struct {
	system string
	task   string
	calls  int
}

// capturingModels builds the offline registry every request-object test runs
// against and returns the record its one scripted step writes into. The
// answer is unremarkable: nothing here is about what comes back.
func capturingModels(t *testing.T) (ai.Models, *capturedRequest) {
	t.Helper()

	captured := &capturedRequest{}
	step := faux.StepFunc(func(_ context.Context, chat ai.Context, _ *ai.StreamOptions, _ *faux.State, _ *ai.Model) (*ai.AssistantMessage, error) {
		captured.calls++
		captured.system = chat.SystemPrompt
		if len(chat.Messages) == 0 {
			t.Errorf("the request carried no messages at all")
			return faux.TextMessage(scriptedAnswer, nil), nil
		}
		user, ok := chat.Messages[0].(*ai.UserMessage)
		if !ok || user.Content.Plain == nil {
			t.Errorf("chat.Messages[0] = %#v, want a *ai.UserMessage carrying plain text", chat.Messages[0])
			return faux.TextMessage(scriptedAnswer, nil), nil
		}
		captured.task = *user.Content.Plain
		return faux.TextMessage(scriptedAnswer, nil), nil
	})
	return scripted(t, 0, step), captured
}

// unreadStdin is stdin that must never be read. Every flag-shorthand test
// passes one: "giving either flag means stdin is not read at all" is not
// observable from the outcome — a run that read stdin and then ignored what
// it found would look identical — so the reader itself is the assertion.
type unreadStdin struct{ t *testing.T }

func (r unreadStdin) Read([]byte) (int, error) {
	r.t.Errorf("stdin was read: --system/--prompt must be mutually exclusive with the request object")
	return 0, io.EOF
}

// requestObject renders the JSON request object stdin carries, so the tests
// below state the two values rather than the encoding around them.
func requestObject(system, task string) string {
	var out bytes.Buffer
	out.WriteString(`{"system":`)
	writeJSONString(&out, system)
	out.WriteString(`,"task":`)
	writeJSONString(&out, task)
	out.WriteString(`}`)
	return out.String()
}

// writeJSONString escapes the two characters the fixtures below actually
// contain. encoding/json would do it too, but a test that builds its input
// with the same package the implementation decodes it with cannot catch an
// encoding disagreement.
func writeJSONString(out *bytes.Buffer, s string) {
	out.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			out.WriteString(`\"`)
		case '\\':
			out.WriteString(`\\`)
		case '\n':
			out.WriteString(`\n`)
		case '\t':
			out.WriteString(`\t`)
		default:
			out.WriteRune(r)
		}
	}
	out.WriteByte('"')
}

// TestRun_Review_RequestObjectOnStdin_ReachesModelVerbatim is the cross-repo
// contract in one assertion: `{"system":…,"task":…}` on stdin arrives at the
// model as the system message and the first user message, byte for byte. It
// is asserted against the recorded request rather than against stderr,
// because stderr never carries either value.
func TestRun_Review_RequestObjectOnStdin_ReachesModelVerbatim(t *testing.T) {
	const system = "You review repositories you did not write. Report leads, not findings."
	const task = "Review the diff on branch feature/x for concurrency mistakes."

	models, captured := capturingModels(t)

	var stdout, stderr bytes.Buffer
	code := cli.RunForTest([]string{"review", "--allow", ".", t.TempDir()},
		strings.NewReader(requestObject(system, task)), &stdout, &stderr, models)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr.String())
	}
	if captured.system != system {
		t.Errorf("system prompt reaching the model = %q, want %q", captured.system, system)
	}
	if captured.task != task {
		t.Errorf("task reaching the model = %q, want %q", captured.task, task)
	}
	if stdout.String() != scriptedAnswer {
		t.Errorf("stdout = %q, want the reviewer's answer %q", stdout.String(), scriptedAnswer)
	}
	assertOnlyKnownPrefixedLines(t, stderr.String())
}

// TestRun_Review_MalformedRequestObject_IsUsageError is the fail-closed half,
// and the reason the object is decoded with DisallowUnknownFields at all: a
// request the binary cannot read in full is exit 2 with stop=usage, and no
// request is ever sent.
//
// stop=usage rather than stop=failed is a deliberate placement in the
// exit-code contract issue 05 landed. The request object is written by the
// same caller, in the same breath, as argv — a mistyped key is a mistyped
// invocation, exactly like an unknown tier — so it takes the word that sends
// a script's author back to its own template. stop=failed is reserved for a
// machine that is misconfigured while the invocation is well formed.
func TestRun_Review_MalformedRequestObject_IsUsageError(t *testing.T) {
	tests := []struct {
		name  string
		stdin string
		// wantMessage is a fragment the single error: line has to carry, so
		// the caller can tell which of these seven states it is in.
		wantMessage string
	}{
		{
			name:        "a mistyped key",
			stdin:       `{"systm":"S","task":"T"}`,
			wantMessage: "systm",
		},
		{
			name:        "an empty system prompt",
			stdin:       requestObject("", "T"),
			wantMessage: "system",
		},
		{
			name:        "a whitespace-only system prompt",
			stdin:       requestObject("  \n\t ", "T"),
			wantMessage: "system",
		},
		{
			name:        "an empty task",
			stdin:       requestObject("S", ""),
			wantMessage: "task",
		},
		{
			name:        "a whitespace-only task",
			stdin:       requestObject("S", " \n "),
			wantMessage: "task",
		},
		{
			name:        "the task field absent altogether",
			stdin:       `{"system":"S"}`,
			wantMessage: "task",
		},
		{
			name:        "the system field absent altogether",
			stdin:       `{"task":"T"}`,
			wantMessage: "system",
		},
		{
			name:        "raw task text, the pre-request-object shape",
			stdin:       "review this repository please",
			wantMessage: "JSON request object",
		},
		{
			name:        "a JSON value that is not an object",
			stdin:       `["review this"]`,
			wantMessage: "JSON request object",
		},
		{
			name:        "a second object after the first",
			stdin:       requestObject("S", "T") + requestObject("S2", "T2"),
			wantMessage: "JSON request object",
		},
		{
			name:        "a system prompt of the wrong type",
			stdin:       `{"system":42,"task":"T"}`,
			wantMessage: "JSON request object",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			models, captured := capturingModels(t)

			var stdout, stderr bytes.Buffer
			code := cli.RunForTest([]string{"review", "--allow", ".", t.TempDir()},
				strings.NewReader(tc.stdin), &stdout, &stderr, models)

			if code != 2 {
				t.Fatalf("exit code = %d, want 2 (stderr: %q)", code, stderr.String())
			}
			if captured.calls != 0 {
				t.Errorf("the model was called %d times, want 0: a malformed request must never be sent", captured.calls)
			}
			if stdout.Len() != 0 {
				t.Errorf("stdout = %q, want empty", stdout.String())
			}
			lines := errorLines(stderr.String())
			if len(lines) != 1 {
				t.Fatalf("stderr carries %d error: lines, want exactly 1: %q", len(lines), stderr.String())
			}
			if !strings.Contains(lines[0], tc.wantMessage) {
				t.Errorf("error line = %q, want it to carry %q", lines[0], tc.wantMessage)
			}
			if fields := fauxtest.ParseDoneLine(t, stderr.String()); fields.Stop != "usage" {
				t.Errorf("done stop = %q, want usage", fields.Stop)
			}
			assertOnlyKnownPrefixedLines(t, stderr.String())
		})
	}
}

// TestRun_Review_SystemAndPromptFlags_MatchTheStdinForm is journey 4's
// by-hand shorthand: the two flags carry exactly what the request object
// would have, and stdin is not read at all — asserted by a reader that fails
// the test the moment anything calls Read.
func TestRun_Review_SystemAndPromptFlags_MatchTheStdinForm(t *testing.T) {
	const system = "You review repositories you did not write."
	const task = "Review internal/cli for exit-code mistakes."

	models, captured := capturingModels(t)

	var stdout, stderr bytes.Buffer
	code := cli.RunForTest(
		[]string{"review", "--allow", ".", "--system", system, "--prompt", task, t.TempDir()},
		unreadStdin{t}, &stdout, &stderr, models)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr.String())
	}
	if captured.system != system {
		t.Errorf("system prompt reaching the model = %q, want %q", captured.system, system)
	}
	if captured.task != task {
		t.Errorf("task reaching the model = %q, want %q", captured.task, task)
	}
	assertOnlyKnownPrefixedLines(t, stderr.String())
}

// TestRun_Review_OneShorthandFlagWithoutTheOther_IsUsageError keeps the
// shorthand from becoming a back door onto the thing specs 08 forbids. Half
// the pair is not a shorter invocation: --prompt alone would be a review with
// no system prompt, and the binary has none to substitute, while --system
// alone would be a system prompt with nothing to do.
//
// It is refused from argv alone, before stdin is touched, because the two
// flags mean "do not read stdin" whether or not they are complete — falling
// back to stdin on a half-given pair would silently answer a different
// invocation from the one that was typed.
func TestRun_Review_OneShorthandFlagWithoutTheOther_IsUsageError(t *testing.T) {
	tests := []struct {
		name  string
		flags []string
	}{
		{name: "--prompt without --system", flags: []string{"--prompt", "review this"}},
		{name: "--system without --prompt", flags: []string{"--system", "you are a reviewer"}},
		{name: "--prompt empty, without --system", flags: []string{"--prompt", ""}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			models, captured := capturingModels(t)

			argv := append([]string{"review", "--allow", "."}, tc.flags...)
			argv = append(argv, t.TempDir())

			var stdout, stderr bytes.Buffer
			code := cli.RunForTest(argv, unreadStdin{t}, &stdout, &stderr, models)

			if code != 2 {
				t.Fatalf("exit code = %d, want 2 (stderr: %q)", code, stderr.String())
			}
			if captured.calls != 0 {
				t.Errorf("the model was called %d times, want 0", captured.calls)
			}
			if stdout.Len() != 0 {
				t.Errorf("stdout = %q, want empty", stdout.String())
			}
			lines := errorLines(stderr.String())
			if len(lines) != 1 {
				t.Fatalf("stderr carries %d error: lines, want exactly 1: %q", len(lines), stderr.String())
			}
			if !strings.Contains(lines[0], "--system") || !strings.Contains(lines[0], "--prompt") {
				t.Errorf("error line = %q, want it to name both --system and --prompt", lines[0])
			}
			if fields := fauxtest.ParseDoneLine(t, stderr.String()); fields.Stop != "usage" {
				t.Errorf("done stop = %q, want usage", fields.Stop)
			}
			assertOnlyKnownPrefixedLines(t, stderr.String())
		})
	}
}

// TestRun_Review_EmptyShorthandValue_IsUsageError is the flag channel's own
// half of the "no default system prompt is ever substituted" rule: a pair
// that is complete but empty is refused exactly as the request object's empty
// fields are, and for the same reason.
func TestRun_Review_EmptyShorthandValue_IsUsageError(t *testing.T) {
	tests := []struct {
		name   string
		system string
		task   string
	}{
		{name: "an empty system prompt", system: "", task: "review this"},
		{name: "a whitespace-only system prompt", system: " \n ", task: "review this"},
		{name: "an empty task", system: "you are a reviewer", task: ""},
		{name: "a whitespace-only task", system: "you are a reviewer", task: " "},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			models, captured := capturingModels(t)

			var stdout, stderr bytes.Buffer
			code := cli.RunForTest(
				[]string{"review", "--allow", ".", "--system", tc.system, "--prompt", tc.task, t.TempDir()},
				unreadStdin{t}, &stdout, &stderr, models)

			if code != 2 {
				t.Fatalf("exit code = %d, want 2 (stderr: %q)", code, stderr.String())
			}
			if captured.calls != 0 {
				t.Errorf("the model was called %d times, want 0", captured.calls)
			}
			if stdout.Len() != 0 {
				t.Errorf("stdout = %q, want empty", stdout.String())
			}
			if n := len(errorLines(stderr.String())); n != 1 {
				t.Fatalf("stderr carries %d error: lines, want exactly 1: %q", n, stderr.String())
			}
			if fields := fauxtest.ParseDoneLine(t, stderr.String()); fields.Stop != "usage" {
				t.Errorf("done stop = %q, want usage", fields.Stop)
			}
			assertOnlyKnownPrefixedLines(t, stderr.String())
		})
	}
}

// systemPromptPattern is the acceptance criteria's grep, compiled: any
// spelling of the role sentence a system prompt would open with.
var systemPromptPattern = regexp.MustCompile(`(?i)you are an? (external )?reviewer`)

// TestBinary_CarriesNoSystemPrompt is specs 08's verdict as a test rather
// than as a one-off grep, so it keeps holding: the caller owns the system
// prompt, the binary supplies none, and there is no default to fall back to.
// A prompt compiled into the binary would put the fastest-moving text in the
// project behind the slowest release path — and would be invisible to every
// other test in this suite.
//
// Test files are exempt (a fixture has to say something), and so are
// dot-directories, which on the development machine hold sibling git
// worktrees of this same repository.
func TestBinary_CarriesNoSystemPrompt(t *testing.T) {
	root := filepath.Join("..", "..")

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != root && strings.HasPrefix(entry.Name(), ".") {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		source, err := os.ReadFile(path) //nolint:gosec // G304/G122: a test reads this module's own source tree, whose paths come from walking it
		if err != nil {
			return err
		}
		if systemPromptPattern.Match(source) {
			t.Errorf("%s carries a system prompt; the caller owns it (specs 08)", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking the module: %v", err)
	}
}
