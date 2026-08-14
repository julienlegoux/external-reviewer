package tools_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/julienlegoux/external-reviewer/internal/confine"
	"github.com/julienlegoux/external-reviewer/internal/repo"
	"github.com/julienlegoux/external-reviewer/internal/tools"
)

// newListTool wires the real list tool over a real repository: the tool's
// contract is "what the reviewer can see", and every rule that decides it —
// gitignore, the allow-list, the floor — lives in the enumeration underneath.
// A fake enumerator here would assert the formatting and nothing else.
func newListTool(t *testing.T, files map[string]string, allow ...string) tools.Tool {
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

	return tools.List(repo.NewEnumerator(scope, root, func(string) {}))
}

// call runs the tool's handler the way issue 03's dispatch will, with
// arguments already decoded from the model's JSON.
func call(t *testing.T, tool tools.Tool, arguments map[string]any) tools.Result {
	t.Helper()

	result := tool.Handler(t.Context(), arguments)
	if result.IsError {
		t.Logf("tool reported an error result: %s", result.Text)
	}
	return result
}

// lines splits a tool result into its lines, which is the whole result format:
// paths, one per line, with the truncation marker as the last of them.
func lines(result tools.Result) []string {
	trimmed := strings.TrimSuffix(result.Text, "\n")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}

func TestList_DefaultPatternReturnsOnlyPathsInsideTheAllowedSubtrees(t *testing.T) {
	tool := newListTool(t, map[string]string{
		"docs/notes.md":       "notes\n",
		"docs/deep/nested.md": "deeper\n",
		"internal/run.go":     "package main\n",
		"README.md":           "# fixture\n",
	}, "docs")

	result := call(t, tool, map[string]any{})

	want := []string{"docs/deep/nested.md", "docs/notes.md"}
	if got := lines(result); !slices.Equal(got, want) {
		t.Errorf("list returned\n\t%q\nwant\n\t%q", got, want)
	}
	if result.IsError {
		t.Error("a successful listing was reported as an error result")
	}
}

func TestList_NeverReturnsAPathOnTheSensitiveFileFloor(t *testing.T) {
	tool := newListTool(t, map[string]string{
		".env":              "SECRET=1\n",
		".env.local":        "SECRET=2\n",
		"server.pem":        "-----BEGIN\n",
		"deploy.key":        "-----BEGIN\n",
		"id_rsa":            "-----BEGIN\n",
		"bundle.p12":        "binary\n",
		"bundle.pfx":        "binary\n",
		".npmrc":            "//registry:_authToken=x\n",
		".netrc":            "machine example.invalid\n",
		"credentials.json":  "{}\n",
		".git/config":       "[remote]\n",
		"docs/readable.md":  "ordinary\n",
		"docs/mentions.env": "ordinary, despite the name\n",
	}, ".")

	result := call(t, tool, map[string]any{"pattern": "**/*"})

	want := []string{"docs/mentions.env", "docs/readable.md"}
	if got := lines(result); !slices.Equal(got, want) {
		t.Errorf("list returned\n\t%q\nwant\n\t%q", got, want)
	}
}

func TestList_MatchesGlobPatterns(t *testing.T) {
	files := map[string]string{
		"main.go":              "package main\n",
		"docs/notes.md":        "notes\n",
		"docs/deep/nested.md":  "deeper\n",
		"internal/cli/run.go":  "package cli\n",
		"internal/cli/run.txt": "not go\n",
	}

	tests := []struct {
		name    string
		pattern string
		want    []string
	}{
		{
			name:    "every path, by default",
			pattern: "**/*",
			want:    []string{"docs/deep/nested.md", "docs/notes.md", "internal/cli/run.go", "internal/cli/run.txt", "main.go"},
		},
		{
			name:    "a suffix at any depth",
			pattern: "**/*.go",
			want:    []string{"internal/cli/run.go", "main.go"},
		},
		{
			name:    "a single segment does not cross a directory",
			pattern: "*.go",
			want:    []string{"main.go"},
		},
		{
			name:    "one directory's immediate children",
			pattern: "docs/*",
			want:    []string{"docs/notes.md"},
		},
		{
			name:    "a whole subtree",
			pattern: "internal/**/*",
			want:    []string{"internal/cli/run.go", "internal/cli/run.txt"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tool := newListTool(t, files, ".")

			result := call(t, tool, map[string]any{"pattern": test.pattern})

			if got := lines(result); !slices.Equal(got, test.want) {
				t.Errorf("list %q returned\n\t%q\nwant\n\t%q", test.pattern, got, test.want)
			}
		})
	}
}

func TestList_AnnouncesTruncationWithBothRealNumbers(t *testing.T) {
	const total = 431

	files := make(map[string]string, total)
	for i := range total {
		files[fmt.Sprintf("docs/note-%03d.md", i)] = "note\n"
	}
	tool := newListTool(t, files, ".")

	result := call(t, tool, map[string]any{"pattern": "**/*.md"})

	got := lines(result)
	if len(got) != tools.MaxListPaths+1 {
		t.Fatalf("list returned %d lines, want %d paths plus one truncation line", len(got), tools.MaxListPaths)
	}
	marker := got[len(got)-1]
	want := "[truncated: showing " + strconv.Itoa(tools.MaxListPaths) + " of " + strconv.Itoa(total) + " paths]"
	if marker != want {
		t.Errorf("truncation line is %q, want %q", marker, want)
	}
	for _, line := range got[:len(got)-1] {
		if strings.HasPrefix(line, "[truncated") {
			t.Errorf("a truncation marker appeared among the paths: %q", line)
		}
	}
}

func TestList_SaysNothingAboutTruncationWhenNothingWasTruncated(t *testing.T) {
	files := make(map[string]string, tools.MaxListPaths)
	for i := range tools.MaxListPaths {
		files[fmt.Sprintf("docs/note-%03d.md", i)] = "note\n"
	}
	tool := newListTool(t, files, ".")

	result := call(t, tool, map[string]any{"pattern": "**/*.md"})

	got := lines(result)
	if len(got) != tools.MaxListPaths {
		t.Fatalf("list returned %d lines, want exactly %d paths and no truncation line", len(got), tools.MaxListPaths)
	}
	if strings.Contains(result.Text, "truncated") {
		t.Errorf("a listing at the cap but not over it announced truncation:\n%s", result.Text)
	}
}

func TestList_ReturnsWirePathsWhicheverOSRuns(t *testing.T) {
	tool := newListTool(t, map[string]string{"docs/deep/nested/note.md": "nested\n"}, ".")

	result := call(t, tool, map[string]any{})

	for _, line := range lines(result) {
		if strings.ContainsRune(line, '\\') || filepath.IsAbs(line) || strings.HasPrefix(line, "./") {
			t.Errorf("list returned %q, which is not a repo-relative wire path", line)
		}
	}
}

// TestList_ReportsABadPatternAsAToolErrorRatherThanAFailure is the loop's own
// rule: a tool that cannot answer says so and the reviewer tries again, which
// is why nothing here returns an error to the caller (CONVENTIONS § Error
// handling, specs 13).
func TestList_ReportsABadPatternAsAToolErrorRatherThanAFailure(t *testing.T) {
	tests := []struct {
		name      string
		arguments map[string]any
		mentions  string
	}{
		{name: "unclosed character class", arguments: map[string]any{"pattern": "docs/[a-"}, mentions: "pattern"},
		{name: "not a string", arguments: map[string]any{"pattern": 7}, mentions: "pattern"},
		{name: "an absolute path", arguments: map[string]any{"pattern": "/etc/*"}, mentions: "pattern"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tool := newListTool(t, map[string]string{"docs/notes.md": "notes\n"}, ".")

			result := tool.Handler(t.Context(), test.arguments)

			if !result.IsError {
				t.Fatalf("a bad pattern returned a successful result:\n%s", result.Text)
			}
			if !strings.Contains(result.Text, test.mentions) {
				t.Errorf("the error result %q does not mention %q", result.Text, test.mentions)
			}
		})
	}
}

func TestList_SaysSoWhenNothingMatches(t *testing.T) {
	tool := newListTool(t, map[string]string{"docs/notes.md": "notes\n"}, ".")

	result := call(t, tool, map[string]any{"pattern": "**/*.go"})

	if result.IsError {
		t.Error("an empty listing is not a tool error: the reviewer asked a legitimate question")
	}
	if !strings.Contains(result.Text, "**/*.go") {
		t.Errorf("an empty listing does not name the pattern that matched nothing: %q", result.Text)
	}
}

// TestList_DeclarationIsAWireContract checks the half of the tool the model
// reads before it ever calls it. snake_case names and schema fields, because
// they are a contract with the model rather than Go API (CONVENTIONS §
// Naming), and a description that stands alone, because the caller owns the
// system prompt and this may be the model's only instruction (specs 09).
func TestList_DeclarationIsAWireContract(t *testing.T) {
	tool := newListTool(t, map[string]string{"docs/notes.md": "notes\n"}, ".")
	declaration := tool.Declaration

	if declaration.Name != "list" {
		t.Errorf("tool name is %q, want %q", declaration.Name, "list")
	}
	for _, mention := range []string{"pattern", "truncat", "/"} {
		if !strings.Contains(declaration.Description, mention) {
			t.Errorf("the tool description never mentions %q:\n%s", mention, declaration.Description)
		}
	}

	var schema struct {
		Type       string `json:"type"`
		Properties struct {
			Pattern struct {
				Type        string `json:"type"`
				Description string `json:"description"`
			} `json:"pattern"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(declaration.Parameters, &schema); err != nil {
		t.Fatalf("the parameter schema is not valid JSON: %v", err)
	}
	if schema.Type != "object" {
		t.Errorf("the parameter schema is of type %q, want %q", schema.Type, "object")
	}
	if schema.Properties.Pattern.Type != "string" {
		t.Errorf("the pattern parameter is of type %q, want %q", schema.Properties.Pattern.Type, "string")
	}
	if schema.Properties.Pattern.Description == "" {
		t.Error("the pattern parameter carries no description")
	}
}

// requireGit skips at runtime when git is absent, naming it — never a build
// tag (CONVENTIONS § Testing).
func requireGit(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git is not on PATH: %v", err)
	}
}

// TestList_HonoursGitignore is the tool-level half of enumeration's central
// promise: what the reviewer can glob is what git says is in the repository,
// so a vendored dependency tree never reaches its context.
func TestList_HonoursGitignore(t *testing.T) {
	requireGit(t)

	root := t.TempDir()
	for name, content := range map[string]string{
		".gitignore":                "node_modules/\n",
		"docs/notes.md":             "notes\n",
		"node_modules/pkg/index.js": "vendored\n",
	} {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatalf("creating fixture directory for %s: %v", name, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("writing fixture %s: %v", name, err)
		}
	}
	for _, args := range [][]string{{"init"}, {"add", "--", ".gitignore", "docs/notes.md"}} {
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...) //nolint:forbidigo,gosec // a fixture builds its repository with real git; argv only, no shell
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
	}

	scope, err := confine.OpenScope(root, []string{"."})
	if err != nil {
		t.Fatalf("OpenScope: %v", err)
	}
	t.Cleanup(func() { _ = scope.Close() })
	tool := tools.List(repo.NewEnumerator(scope, root, func(string) {}))

	result := call(t, tool, map[string]any{})

	want := []string{".gitignore", "docs/notes.md"}
	if got := lines(result); !slices.Equal(got, want) {
		t.Errorf("list returned\n\t%q\nwant\n\t%q", got, want)
	}
}
