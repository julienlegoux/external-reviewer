package tools_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/julienlegoux/kern-link/ai"
	"github.com/julienlegoux/kern-link/ai/providers/faux"

	"github.com/julienlegoux/external-reviewer/internal/confine"
	"github.com/julienlegoux/external-reviewer/internal/diag"
	"github.com/julienlegoux/external-reviewer/internal/fauxtest"
	"github.com/julienlegoux/external-reviewer/internal/reviewer"
	"github.com/julienlegoux/external-reviewer/internal/tools"
)

// newReadFileTool wires the real read_file tool over a real confine.Scope:
// every rule that can refuse a path — the root, the allow-list, the floor —
// lives there, so a fake scope here would assert nothing but the formatting.
func newReadFileTool(t *testing.T, files map[string]string, allow ...string) tools.Tool {
	t.Helper()
	return tools.ReadFile(newReadFileScope(t, files, allow...))
}

func newReadFileScope(t *testing.T, files map[string]string, allow ...string) *confine.Scope {
	t.Helper()

	root := t.TempDir()
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatalf("creating fixture directory for %s: %v", name, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("writing fixture %s: %v", name, err)
		}
	}

	scope, err := confine.OpenScope(root, allow)
	if err != nil {
		t.Fatalf("OpenScope(%q, %v): %v", root, allow, err)
	}
	t.Cleanup(func() {
		if err := scope.Close(); err != nil {
			t.Errorf("closing the scope: %v", err)
		}
	})
	return scope
}

// numberedLines builds a fixture file's content of n lines, each reading
// "line <k>", so a test can assert on both the text and the line number a
// window returned.
func numberedLines(n int) string {
	var out strings.Builder
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&out, "line %d\n", i)
	}
	return out.String()
}

func readCall(t *testing.T, tool tools.Tool, arguments map[string]any) tools.Result {
	t.Helper()
	return tool.Handler(t.Context(), arguments)
}

func TestReadFile_BatchOfThreeValidPathsReturnsThreeHeadedSectionsInOrder(t *testing.T) {
	tool := newReadFileTool(t, map[string]string{
		"a.md": "alpha\n",
		"b.md": "bravo\n",
		"c.md": "charlie\n",
	}, ".")

	result := readCall(t, tool, map[string]any{"paths": []any{"c.md", "a.md", "b.md"}})

	if result.IsError {
		t.Fatalf("a batch of valid paths was reported as an error result: %s", result.Text)
	}
	order := []string{"c.md", "a.md", "b.md"}
	lastIndex := -1
	for _, path := range order {
		index := strings.Index(result.Text, path)
		if index == -1 {
			t.Fatalf("result does not mention %q:\n%s", path, result.Text)
		}
		if index < lastIndex {
			t.Errorf("path %q appeared out of the requested order:\n%s", path, result.Text)
		}
		lastIndex = index
	}
	for _, want := range []string{"1\talpha", "1\tbravo", "1\tcharlie"} {
		if !strings.Contains(result.Text, want) {
			t.Errorf("result does not contain the numbered line %q:\n%s", want, result.Text)
		}
	}
}

func TestReadFile_OneMissingPathIsInlineWhileTheOthersReturnInFull(t *testing.T) {
	tool := newReadFileTool(t, map[string]string{
		"exists-1.md": "one\n",
		"exists-2.md": "two\n",
	}, ".")

	result := readCall(t, tool, map[string]any{"paths": []any{"exists-1.md", "missing.md", "exists-2.md"}})

	if result.IsError {
		t.Errorf("a batch with one missing path was reported as an error result — the call itself must not be an error: %s", result.Text)
	}
	if !strings.Contains(result.Text, "1\tone") {
		t.Errorf("result does not contain exists-1.md's content:\n%s", result.Text)
	}
	if !strings.Contains(result.Text, "1\ttwo") {
		t.Errorf("result does not contain exists-2.md's content:\n%s", result.Text)
	}
	if !strings.Contains(result.Text, "missing.md") || !strings.Contains(result.Text, confine.ErrNotFound.Error()) {
		t.Errorf("result does not name the missing path and the rule that refused it:\n%s", result.Text)
	}
}

func TestReadFile_MixedRefusalsNameDifferentRules(t *testing.T) {
	tool := newReadFileTool(t, map[string]string{
		"src/main.go":     "package main\n",
		"secrets/key.pem": "-----BEGIN\n",
		"src/.env":        "SECRET=1\n",
	}, "src")

	result := readCall(t, tool, map[string]any{"paths": []any{"src/main.go", "secrets/key.pem", "src/.env"}})

	if result.IsError {
		t.Errorf("a batch mixing valid and refused paths was reported as an error result: %s", result.Text)
	}
	if !strings.Contains(result.Text, "1\tpackage main") {
		t.Errorf("result does not contain the valid file's content:\n%s", result.Text)
	}
	if !strings.Contains(result.Text, confine.ErrOutsideAllowList.Error()) {
		t.Errorf("result does not name the allow-list refusal for secrets/key.pem:\n%s", result.Text)
	}
	if !strings.Contains(result.Text, confine.ErrDeniedFilename.Error()) {
		t.Errorf("result does not name the floor refusal for src/.env:\n%s", result.Text)
	}
}

func TestReadFile_OffsetAndLimitReturnExactlyTheRequestedWindow(t *testing.T) {
	tool := newReadFileTool(t, map[string]string{"notes.md": numberedLines(50)}, ".")

	tests := []struct {
		name       string
		arguments  map[string]any
		wantHeader string
		wantFirst  string
		wantLast   string
		wantAbsent []string
	}{
		{
			name:       "default window starts at line 1",
			arguments:  map[string]any{"paths": []any{"notes.md"}},
			wantHeader: "notes.md (lines 1–50)",
			wantFirst:  "1\tline 1",
			wantLast:   "50\tline 50",
		},
		{
			name:       "offset moves the window",
			arguments:  map[string]any{"paths": []any{"notes.md"}, "offset": 10, "limit": 5},
			wantHeader: "notes.md (lines 10–14)",
			wantFirst:  "10\tline 10",
			wantLast:   "14\tline 14",
			wantAbsent: []string{"9\tline 9", "15\tline 15"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := readCall(t, tool, test.arguments)

			if result.IsError {
				t.Fatalf("Handler() reported an error result: %s", result.Text)
			}
			if !strings.Contains(result.Text, test.wantHeader) {
				t.Errorf("result does not contain the header %q:\n%s", test.wantHeader, result.Text)
			}
			if !strings.Contains(result.Text, test.wantFirst) {
				t.Errorf("result does not contain %q:\n%s", test.wantFirst, result.Text)
			}
			if !strings.Contains(result.Text, test.wantLast) {
				t.Errorf("result does not contain %q:\n%s", test.wantLast, result.Text)
			}
			for _, absent := range test.wantAbsent {
				if strings.Contains(result.Text, absent) {
					t.Errorf("result contains %q, which is outside the requested window:\n%s", absent, result.Text)
				}
			}
		})
	}
}

func TestReadFile_LongerFileEndsWithTruncationCarryingBothRealNumbers(t *testing.T) {
	tool := newReadFileTool(t, map[string]string{"big.md": numberedLines(30)}, ".")

	result := readCall(t, tool, map[string]any{"paths": []any{"big.md"}, "limit": 10})

	want := "[truncated: lines 1–10 of 30]"
	if !strings.Contains(result.Text, want) {
		t.Errorf("result does not contain the truncation marker %q:\n%s", want, result.Text)
	}
}

func TestReadFile_ShorterFileCarriesNoTruncationLine(t *testing.T) {
	tool := newReadFileTool(t, map[string]string{"small.md": numberedLines(3)}, ".")

	result := readCall(t, tool, map[string]any{"paths": []any{"small.md"}, "limit": 2000})

	if strings.Contains(result.Text, "truncated") {
		t.Errorf("a file shorter than the cap announced truncation:\n%s", result.Text)
	}
}

func TestReadFile_LimitAboveTheCapIsClampedAndTruncationReflectsIt(t *testing.T) {
	tool := newReadFileTool(t, map[string]string{"huge.md": numberedLines(5001)}, ".")

	result := readCall(t, tool, map[string]any{"paths": []any{"huge.md"}, "limit": 999999})

	want := "[truncated: lines 1–5000 of 5001]"
	if !strings.Contains(result.Text, want) {
		t.Errorf("result does not contain the clamped truncation marker %q:\n%s", want, result.Text)
	}
	if strings.Contains(result.Text, "5001\tline 5001") {
		t.Errorf("result returned a line beyond the clamped 5000-line cap:\n%s", result.Text)
	}
}

func TestReadFile_DirectoryAndNonUTF8EachReturnAnInlineMessage(t *testing.T) {
	tool := newReadFileTool(t, map[string]string{
		"a-directory/file.md": "content\n",
		"binary.dat":          "before\xff\xfebroken\n",
		"text.md":             "ordinary\n",
	}, ".")

	result := readCall(t, tool, map[string]any{"paths": []any{"a-directory", "binary.dat", "text.md"}})

	if result.IsError {
		t.Errorf("a batch with a directory and a non-UTF-8 file was reported as an error result: %s", result.Text)
	}
	if !strings.Contains(result.Text, "1\tordinary") {
		t.Errorf("result does not contain the ordinary file's content:\n%s", result.Text)
	}
	if !strings.Contains(result.Text, "a-directory") {
		t.Errorf("result does not mention the directory path:\n%s", result.Text)
	}
	if strings.Contains(result.Text, "\xff\xfe") {
		t.Errorf("result contains raw non-UTF-8 bytes rather than an explanatory message:\n%s", result.Text)
	}
	if !strings.Contains(result.Text, "binary.dat") {
		t.Errorf("result does not mention the non-UTF-8 path:\n%s", result.Text)
	}
}

func TestReadFile_PathsInHeadersAndRefusalsAreRepoRelativeWirePaths(t *testing.T) {
	tool := newReadFileTool(t, map[string]string{
		"deep/nested/file.md": "content\n",
		"deep/other/blocked":  "content\n",
	}, "deep/nested")

	result := readCall(t, tool, map[string]any{"paths": []any{"deep/nested/file.md", "deep/other/blocked"}})

	for _, line := range strings.Split(result.Text, "\n") {
		if strings.HasPrefix(line, "===") && (strings.ContainsRune(line, '\\') || filepath.IsAbs(line)) {
			t.Errorf("header line is not a repo-relative wire path: %q", line)
		}
	}
}

func TestReadFile_ArgumentErrors(t *testing.T) {
	tool := newReadFileTool(t, map[string]string{"a.md": "one\n"}, ".")

	tests := []struct {
		name      string
		arguments map[string]any
		mentions  string
	}{
		{name: "no paths given", arguments: map[string]any{}, mentions: "paths"},
		{name: "empty paths array", arguments: map[string]any{"paths": []any{}}, mentions: "paths"},
		{name: "paths is not an array", arguments: map[string]any{"paths": "a.md"}, mentions: "paths"},
		{name: "a path element is not a string", arguments: map[string]any{"paths": []any{7}}, mentions: "paths"},
		{name: "offset is not a number", arguments: map[string]any{"paths": []any{"a.md"}, "offset": "one"}, mentions: "offset"},
		{name: "offset below 1", arguments: map[string]any{"paths": []any{"a.md"}, "offset": 0}, mentions: "offset"},
		{name: "limit is not a number", arguments: map[string]any{"paths": []any{"a.md"}, "limit": "many"}, mentions: "limit"},
		{name: "limit below 1", arguments: map[string]any{"paths": []any{"a.md"}, "limit": 0}, mentions: "limit"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := tool.Handler(t.Context(), test.arguments)

			if !result.IsError {
				t.Fatalf("a bad call returned a successful result:\n%s", result.Text)
			}
			if !strings.Contains(result.Text, test.mentions) {
				t.Errorf("the error result %q does not mention %q", result.Text, test.mentions)
			}
		})
	}
}

// TestReadFile_DeclarationIsAWireContract checks the half of the tool the
// model reads before it ever calls it: snake_case schema fields (CONVENTIONS
// § Naming), and a description that stands alone and states explicitly that a
// batch failure is partial rather than total (specs 09).
func TestReadFile_DeclarationIsAWireContract(t *testing.T) {
	tool := newReadFileTool(t, map[string]string{"a.md": "one\n"}, ".")
	declaration := tool.Declaration

	if declaration.Name != "read_file" {
		t.Errorf("tool name is %q, want %q", declaration.Name, "read_file")
	}
	for _, mention := range []string{"paths", "offset", "limit", "partial", "truncat"} {
		if !strings.Contains(declaration.Description, mention) {
			t.Errorf("the tool description never mentions %q:\n%s", mention, declaration.Description)
		}
	}

	var schema struct {
		Type       string   `json:"type"`
		Required   []string `json:"required"`
		Properties struct {
			Paths struct {
				Type  string `json:"type"`
				Items struct {
					Type string `json:"type"`
				} `json:"items"`
			} `json:"paths"`
			Offset struct {
				Type string `json:"type"`
			} `json:"offset"`
			Limit struct {
				Type string `json:"type"`
			} `json:"limit"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(declaration.Parameters, &schema); err != nil {
		t.Fatalf("the parameter schema is not valid JSON: %v", err)
	}
	if schema.Type != "object" {
		t.Errorf("the parameter schema is of type %q, want %q", schema.Type, "object")
	}
	if schema.Properties.Paths.Type != "array" || schema.Properties.Paths.Items.Type != "string" {
		t.Errorf("the paths parameter is not an array of strings: %+v", schema.Properties.Paths)
	}
	if schema.Properties.Offset.Type != "integer" {
		t.Errorf("the offset parameter is of type %q, want %q", schema.Properties.Offset.Type, "integer")
	}
	if schema.Properties.Limit.Type != "integer" {
		t.Errorf("the limit parameter is of type %q, want %q", schema.Properties.Limit.Type, "integer")
	}
	if !slicesContain(schema.Required, "paths") {
		t.Errorf("the schema does not require %q, required = %v", "paths", schema.Required)
	}
}

func slicesContain(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// TestReadFile_EndToEndThroughTheLoop drives the real read_file tool through
// issue 03's Loop and the faux provider: a scripted call over a batch with
// one bad path must leave the run alive, with the model's next turn
// dispatched — the loop-level half of "the call itself is not an error"
// (issue 03's own AllowListRefusal test is the template this follows).
func TestReadFile_EndToEndThroughTheLoop(t *testing.T) {
	scope := newReadFileScope(t, map[string]string{"exists.md": "hello\n"}, ".")
	registry := tools.NewRegistry(tools.ReadFile(scope))

	models, handle := fauxtest.NewRegistry(t, fauxtest.RegistryOptions{
		ProviderID: fixtureProvider,
		ModelIDs:   []string{fixtureModel},
		Auth:       fauxtest.CredentialedAuth("OAuth"),
	})
	const finalAnswer = "# Review\n\nno leads\n"
	handle.SetResponses(
		faux.Step(faux.AssistantMessage(
			[]ai.AssistantContentPart{faux.ToolCall("read_file", map[string]any{"paths": []any{"exists.md", "missing.md"}}, nil)},
			&faux.AssistantMessageOptions{StopReason: ai.StopReasonToolUse},
		)),
		faux.Step(faux.TextMessage(finalAnswer, nil)),
	)

	model := models.GetModel(fixtureProvider, fixtureModel)
	if model == nil {
		t.Fatalf("model %s/%s missing from the registry", fixtureProvider, fixtureModel)
	}

	loop := &reviewer.Loop{
		Conversation: reviewer.NewConversation(models, model, "you review repositories you did not write", "review this"),
		Tools:        registry,
		State:        diag.NewState(),
		Stderr:       &strings.Builder{},
	}

	report, err := loop.Run(context.Background())

	if err != nil {
		t.Fatalf("Run() error = %v, want nil — a bad path in a read_file batch is a value, not a failure", err)
	}
	if report != finalAnswer {
		t.Errorf("Run() = %q, want the model's next turn to have been dispatched (%q)", report, finalAnswer)
	}
}
