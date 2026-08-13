package tools_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
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

// newGitReadTool wires the real tool over a real directory: the confinement is
// the run's own Scope, because the whole subject of this tool is what the
// allow-list and the floor do to a *history* read, and a stand-in scope would
// assert the message formatting and nothing else.
//
// The directory is not a git repository unless the test makes it one — every
// refusal below is decided before git would be reached, which is exactly what
// they assert.
func newGitReadTool(t *testing.T, root string, allow ...string) tools.Tool {
	t.Helper()

	scope, err := confine.OpenScope(root, allow)
	if err != nil {
		t.Fatalf("OpenScope(%q, %v): %v", root, allow, err)
	}
	t.Cleanup(func() {
		if err := scope.Close(); err != nil {
			t.Errorf("closing the scope: %v", err)
		}
	})
	return tools.GitRead(scope, root)
}

// gitReadTree writes the fixture files and returns the directory they live in.
// Names are wire paths, so a fixture reads the same on both matrix OSes.
func gitReadTree(t *testing.T, files map[string]string) string {
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
	return root
}

// gitReadFiles is the shape every fixture in this file starts from: an allowed
// subtree, a subtree no --allow grants, and a credential at the root that the
// floor covers whatever --allow said.
func gitReadFiles() map[string]string {
	return map[string]string{
		"docs/notes.md":    "notes\n",
		"secret/creds.txt": "token\n",
		".env":             "KEY=1\n",
	}
}

// withoutGitOnPath empties PATH for one test, so that any invocation the tool
// still makes fails as "git is not on PATH" instead of answering. It is how
// "a refused call never reaches git" is asserted as a property rather than
// inferred from a message: a refusal that is decided before the command is
// built reads identically with and without git installed.
func withoutGitOnPath(t *testing.T) {
	t.Helper()

	t.Setenv("PATH", "")
	t.Setenv("Path", "")
}

// mentions fails the test unless the result is an error carrying every
// fragment — the rule-naming half of a refusal, which is what a reviewer acts
// on.
func mentions(t *testing.T, result tools.Result, fragments ...string) {
	t.Helper()

	if !result.IsError {
		t.Fatalf("the call was answered rather than refused:\n%s", result.Text)
	}
	for _, fragment := range fragments {
		if !strings.Contains(result.Text, fragment) {
			t.Errorf("the refusal never mentions %q:\n%s", fragment, result.Text)
		}
	}
	if strings.Contains(result.Text, "not on PATH") {
		t.Errorf("the refusal reached git before deciding:\n%s", result.Text)
	}
}

func TestGitRead_RefusesASubcommandOutsideTheAllowlistWithoutReachingGit(t *testing.T) {
	withoutGitOnPath(t)
	tool := newGitReadTool(t, gitReadTree(t, gitReadFiles()), ".")

	for _, command := range []string{"commit", "push", "blame", "ls-tree", ""} {
		result := tool.Handler(t.Context(), map[string]any{"command": command})

		mentions(t, result, "log", "diff", "show", "status")
	}
}

func TestGitRead_RefusesArgsGivenAsAStringBecauseThereIsNoShell(t *testing.T) {
	tool := newGitReadTool(t, gitReadTree(t, gitReadFiles()), ".")

	result := tool.Handler(t.Context(), map[string]any{"command": "log", "args": "--oneline -- secret"})

	mentions(t, result, "args", "array")
}

func TestGitRead_RefusesAnArgsEntryThatIsNotAString(t *testing.T) {
	tool := newGitReadTool(t, gitReadTree(t, gitReadFiles()), ".")

	result := tool.Handler(t.Context(), map[string]any{"command": "log", "args": []any{"--oneline", 7.0}})

	mentions(t, result, "args", "string")
}

// TestGitRead_RefusesArgumentsThatWouldWidenTheConfinement covers the two
// shapes that reach past the pathspecs this tool appends: a second `--`, after
// which every following argument is a pathspec of the model's choosing, and
// `--output`, which is git writing a file on a binary that writes none.
func TestGitRead_RefusesArgumentsThatWouldWidenTheConfinement(t *testing.T) {
	withoutGitOnPath(t)
	tool := newGitReadTool(t, gitReadTree(t, gitReadFiles()), "docs")

	for _, arguments := range [][]any{
		{"--", "secret"},
		{"--oneline", "--", "secret/creds.txt"},
		{"--output=/tmp/leak.txt"},
		{"--output", "/tmp/leak.txt"},
	} {
		result := tool.Handler(t.Context(), map[string]any{"command": "log", "args": arguments})

		mentions(t, result, "pathspec")
	}
}

// TestGitRead_RefusesAnObjectOnTheSensitiveFileFloor is the hole this whole
// issue exists to close: `git show HEAD:.env` reads a credential out of history
// through the one tool that never touches *os.Root. The path is validated
// before the command is built, so nothing reaches git at all.
func TestGitRead_RefusesAnObjectOnTheSensitiveFileFloor(t *testing.T) {
	withoutGitOnPath(t)
	tool := newGitReadTool(t, gitReadTree(t, gitReadFiles()), ".")

	for _, spec := range []string{"HEAD:.env", "HEAD:./.env", ":.env", ":0:.env", "HEAD:.git/config"} {
		result := tool.Handler(t.Context(), map[string]any{"command": "show", "args": []any{spec}})

		mentions(t, result, "floor")
	}
}

// TestGitRead_RefusesAnObjectOutsideTheAllowedSubtrees is the second rule, and
// it must read differently from the first: "widen --allow" and "that file is a
// credential" are different things for a reviewer to learn (specs 10).
func TestGitRead_RefusesAnObjectOutsideTheAllowedSubtrees(t *testing.T) {
	withoutGitOnPath(t)
	tool := newGitReadTool(t, gitReadTree(t, gitReadFiles()), "docs")

	result := tool.Handler(t.Context(), map[string]any{"command": "show", "args": []any{"HEAD:secret/creds.txt"}})

	mentions(t, result, "--allow", "secret/creds.txt")
	if strings.Contains(result.Text, "floor") {
		t.Errorf("the allow-list refusal is spelled as the floor refusal:\n%s", result.Text)
	}

	// `HEAD:` is the root tree — a listing of every top-level name, which is
	// exactly what a run granted one subtree must not have.
	root := tool.Handler(t.Context(), map[string]any{"command": "show", "args": []any{"HEAD:"}})
	mentions(t, root, "--allow")
}

// TestGitRead_RefusesABlobPairOnTheFloor closes the same hole in its second
// doorway: `git diff <blob> <blob>` prints both blobs' contents, so the
// <rev>:<path> form has to be validated in every subcommand's arguments, not
// only show's. `--allow .` is the fixture on purpose — the floor is what is
// left when the allow-list grants everything.
func TestGitRead_RefusesABlobPairOnTheFloor(t *testing.T) {
	withoutGitOnPath(t)
	tool := newGitReadTool(t, gitReadTree(t, gitReadFiles()), ".")

	result := tool.Handler(t.Context(), map[string]any{
		"command": "diff",
		"args":    []any{"HEAD:.env", "HEAD:docs/notes.md"},
	})

	mentions(t, result, "floor")
}

// TestGitRead_RefusesAnObjectSpelledAsSomethingOtherThanAWirePath keeps the
// third rule distinguishable too: a path that is not repo-relative and
// /-separated is refused as such rather than as an allow-list miss.
func TestGitRead_RefusesAnObjectSpelledAsSomethingOtherThanAWirePath(t *testing.T) {
	withoutGitOnPath(t)
	tool := newGitReadTool(t, gitReadTree(t, gitReadFiles()), ".")

	for _, spec := range []string{"HEAD:../outside.txt", "HEAD:/etc/passwd", `HEAD:C:\Windows\win.ini`} {
		result := tool.Handler(t.Context(), map[string]any{"command": "show", "args": []any{spec}})

		mentions(t, result, "repository")
	}
}

// TestGitRead_RefusesMixingAnObjectWithACommit is confinement's own bookkeeping:
// the granted subtrees are appended as pathspecs only when no argument names an
// object, since a pathspec makes `git diff <blob> <blob>` a usage error. A call
// that mixes the two would therefore be the one shape that runs a commit
// unscoped.
func TestGitRead_RefusesMixingAnObjectWithACommit(t *testing.T) {
	withoutGitOnPath(t)
	tool := newGitReadTool(t, gitReadTree(t, gitReadFiles()), "docs")

	result := tool.Handler(t.Context(), map[string]any{
		"command": "show",
		"args":    []any{"HEAD:docs/notes.md", "HEAD"},
	})

	mentions(t, result, "pathspec")
}

// TestGitRead_RefusesAPathsEntryOutsideTheAllowedSubtreesWithoutReachingGit is
// the first rule over the new parameter, and it is asserted with PATH emptied
// because the point is not the wording alone: a pathspec the allow-list never
// granted must be decided before a command exists to run.
func TestGitRead_RefusesAPathsEntryOutsideTheAllowedSubtreesWithoutReachingGit(t *testing.T) {
	withoutGitOnPath(t)
	tool := newGitReadTool(t, gitReadTree(t, gitReadFiles()), "docs")

	result := tool.Handler(t.Context(), map[string]any{"command": "diff", "paths": []any{"secret"}})

	mentions(t, result, "--allow", "secret")
	if strings.Contains(result.Text, "floor") {
		t.Errorf("the allow-list refusal is spelled as the floor refusal:\n%s", result.Text)
	}
}

// TestGitRead_RefusesAPathsEntryOnTheSensitiveFileFloor is the second rule, and
// `--allow .` is the fixture on purpose: the floor is what is left when the
// allow-list grants everything, and a pathspec is how a diff would otherwise
// print a credential's contents.
func TestGitRead_RefusesAPathsEntryOnTheSensitiveFileFloor(t *testing.T) {
	withoutGitOnPath(t)
	tool := newGitReadTool(t, gitReadTree(t, gitReadFiles()), ".")

	for _, entry := range []string{".env", ".git/config"} {
		result := tool.Handler(t.Context(), map[string]any{"command": "diff", "paths": []any{entry}})

		mentions(t, result, "floor")
	}
}

// TestGitRead_RefusesAPathsEntrySpelledAsSomethingOtherThanAWirePath is the
// third rule. All three spellings are refused on both matrix OSes, which is the
// whole reason the vocabulary is checked above *os.Root: `C:\Windows\win.ini`
// is an absolute path on Windows and an ordinary filename on Linux.
func TestGitRead_RefusesAPathsEntrySpelledAsSomethingOtherThanAWirePath(t *testing.T) {
	withoutGitOnPath(t)
	tool := newGitReadTool(t, gitReadTree(t, gitReadFiles()), ".")

	for _, entry := range []string{"../outside", "/etc/passwd", `C:\Windows\win.ini`} {
		result := tool.Handler(t.Context(), map[string]any{"command": "diff", "paths": []any{entry}})

		mentions(t, result, "repository")
	}
}

// TestGitRead_RefusesPathsAlongsideAnObject is the same bookkeeping that
// refuses mixing an object with a commit: git takes pathspecs or an object and
// never both, so a call that named both would have to drop one silently.
func TestGitRead_RefusesPathsAlongsideAnObject(t *testing.T) {
	withoutGitOnPath(t)
	tool := newGitReadTool(t, gitReadTree(t, gitReadFiles()), ".")

	result := tool.Handler(t.Context(), map[string]any{
		"command": "show",
		"args":    []any{"HEAD:docs/notes.md"},
		"paths":   []any{"docs"},
	})

	mentions(t, result, "pathspec", "object")
}

func TestGitRead_DeclarationIsAWireContract(t *testing.T) {
	tool := newGitReadTool(t, gitReadTree(t, gitReadFiles()), ".")
	declaration := tool.Declaration

	if declaration.Name != "git_read" {
		t.Errorf("tool name is %q, want %q", declaration.Name, "git_read")
	}
	for _, mention := range []string{"log", "diff", "show", "status", "array", "paths parameter"} {
		if !strings.Contains(declaration.Description, mention) {
			t.Errorf("the tool description never mentions %q:\n%s", mention, declaration.Description)
		}
	}

	var schema struct {
		Type       string `json:"type"`
		Required   []string
		Properties struct {
			Command struct {
				Type        string   `json:"type"`
				Enum        []string `json:"enum"`
				Description string   `json:"description"`
			} `json:"command"`
			Args struct {
				Type        string `json:"type"`
				Description string `json:"description"`
				Items       struct {
					Type string `json:"type"`
				} `json:"items"`
			} `json:"args"`
			Paths struct {
				Type        string `json:"type"`
				Description string `json:"description"`
				Items       struct {
					Type string `json:"type"`
				} `json:"items"`
			} `json:"paths"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(declaration.Parameters, &schema); err != nil {
		t.Fatalf("the parameter schema is not valid JSON: %v", err)
	}
	if schema.Type != "object" {
		t.Errorf("the parameter schema is of type %q, want %q", schema.Type, "object")
	}
	if want := []string{"log", "diff", "show", "status"}; !slices.Equal(schema.Properties.Command.Enum, want) {
		t.Errorf("the command parameter allows %q, want %q", schema.Properties.Command.Enum, want)
	}
	if schema.Properties.Args.Type != "array" {
		t.Errorf("the args parameter is of type %q, want %q — a string implies a shell", schema.Properties.Args.Type, "array")
	}
	if schema.Properties.Args.Items.Type != "string" {
		t.Errorf("the args items are of type %q, want %q", schema.Properties.Args.Items.Type, "string")
	}
	if schema.Properties.Paths.Type != "array" {
		t.Errorf("the paths parameter is of type %q, want %q", schema.Properties.Paths.Type, "array")
	}
	if schema.Properties.Paths.Items.Type != "string" {
		t.Errorf("the paths items are of type %q, want %q", schema.Properties.Paths.Items.Type, "string")
	}
	if !slices.Contains(schema.Required, "command") {
		t.Errorf("the schema requires %q, want command among them", schema.Required)
	}
	if slices.Contains(schema.Required, "paths") {
		t.Errorf("the schema requires %q, want paths optional — its absence is today's whole-grant behaviour", schema.Required)
	}
	if schema.Properties.Command.Description == "" || schema.Properties.Args.Description == "" ||
		schema.Properties.Paths.Description == "" {
		t.Error("a parameter carries no description")
	}
}

// gitReadRepo builds the repository every behaviour test below reads: an
// allowed subtree and an ungranted one, each with a commit of its own and each
// left dirty in the working tree, so scoping is asserted against history *and*
// against uncommitted state. It returns the repository's OS path.
//
// The three commits are ordered so that HEAD touches only the ungranted
// subtree: a pathspec that were not applied would show it first.
func gitReadRepo(t *testing.T, extra map[string]string) string {
	t.Helper()
	requireGit(t)

	files := gitReadFiles()
	for name, content := range extra {
		files[name] = content
	}
	root := gitReadTree(t, files)

	fixtureGit(t, root, "init")
	fixtureGit(t, root, "add", "-A")
	fixtureCommit(t, root, "initial commit")
	appendFixture(t, root, "docs/notes.md", "more\n")
	fixtureGit(t, root, "add", "-A")
	fixtureCommit(t, root, "docs only change")
	appendFixture(t, root, "secret/creds.txt", "rotated\n")
	fixtureGit(t, root, "add", "-A")
	fixtureCommit(t, root, "secret only change")

	appendFixture(t, root, "docs/notes.md", "uncommitted docs edit\n")
	appendFixture(t, root, "secret/creds.txt", "uncommitted secret edit\n")
	return root
}

// fixtureGit runs one git command in a fixture repository and fails the test
// with its output when git refuses. Tests are exempt from the exec guard the
// implementation carries; a fixture that shells out to git is the only way to
// assert against a real repository's history.
func fixtureGit(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...) //nolint:forbidigo,gosec // a fixture builds and reads its repository with real git; argv only, no shell
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, stderr.String())
	}
	return stdout.String()
}

// fixtureCommit commits the index with an identity and signing supplied per
// command, so the fixture needs nothing from the machine's git configuration —
// which the implementation neutralises anyway.
func fixtureCommit(t *testing.T, dir, message string) {
	t.Helper()

	fixtureGit(t, dir, "-c", "user.email=fixture@example.test", "-c", "user.name=Fixture",
		"-c", "commit.gpgsign=false", "commit", "-q", "-m", message)
}

// appendFixture adds a line to a fixture file, which is what makes one commit
// touch one subtree and leaves the working tree dirty for `status` and `diff`.
func appendFixture(t *testing.T, root, wireName, content string) {
	t.Helper()

	name := filepath.Join(root, filepath.FromSlash(wireName))
	existing, err := os.ReadFile(name) //nolint:gosec // a fixture reads back the file it wrote under t.TempDir()
	if err != nil {
		t.Fatalf("reading fixture %s: %v", wireName, err)
	}
	if err := os.WriteFile(name, append(existing, content...), 0o600); err != nil { //nolint:gosec // a fixture writes under t.TempDir(), to a name joined from a literal wire path
		t.Fatalf("appending to fixture %s: %v", wireName, err)
	}
}

// TestGitRead_ReturnsGitStdoutVerbatim asserts the result *is* git's output —
// not a reformatting of it — for all four subcommands, by running the same
// scoped command beside it. Anything this tool added or dropped would show up
// as a difference.
func TestGitRead_ReturnsGitStdoutVerbatim(t *testing.T) {
	root := gitReadRepo(t, nil)
	tool := newGitReadTool(t, root, ".")

	for _, testCase := range []struct {
		command string
		args    []any
		want    []string
	}{
		{command: "status", args: []any{"--porcelain"}, want: []string{"status", "--porcelain"}},
		{command: "log", args: []any{"--oneline"}, want: []string{"log", "--oneline"}},
		{command: "diff", args: []any{"--name-only"}, want: []string{"diff", "--name-only"}},
		{command: "show", args: []any{"--oneline", "HEAD~1"}, want: []string{"show", "--oneline", "HEAD~1"}},
	} {
		t.Run(testCase.command, func(t *testing.T) {
			result := call(t, tool, map[string]any{"command": testCase.command, "args": testCase.args})

			want := fixtureGit(t, root, append(testCase.want, "--", ".")...)
			if result.IsError {
				t.Fatalf("git %s was refused: %s", testCase.command, result.Text)
			}
			if want == "" {
				t.Fatalf("the fixture produced no output for %s, so the comparison proves nothing", testCase.command)
			}
			if result.Text != want {
				t.Errorf("git_read returned\n%q\nwant git's own stdout\n%q", result.Text, want)
			}
		})
	}
}

// TestGitRead_ShowReturnsTheFileContentsAtThatRevision is the tool doing its
// job: a granted path at an older revision, which is material no other tool in
// this run can reach.
func TestGitRead_ShowReturnsTheFileContentsAtThatRevision(t *testing.T) {
	root := gitReadRepo(t, nil)
	tool := newGitReadTool(t, root, "docs")

	result := call(t, tool, map[string]any{"command": "show", "args": []any{"HEAD~2:docs/notes.md"}})

	if result.IsError {
		t.Fatalf("show of a granted path was refused: %s", result.Text)
	}
	if result.Text != "notes\n" {
		t.Errorf("show returned %q, want the file as it was at that revision", result.Text)
	}
}

// TestGitRead_ScopesLogAndDiffToTheGrantedSubtrees is the pathspec half of the
// confinement: history is scoped the way the working tree is, so a commit that
// only touched an ungranted subtree is not reported at all.
func TestGitRead_ScopesLogAndDiffToTheGrantedSubtrees(t *testing.T) {
	root := gitReadRepo(t, nil)
	tool := newGitReadTool(t, root, "docs")

	log := call(t, tool, map[string]any{"command": "log", "args": []any{"--oneline"}})
	if !strings.Contains(log.Text, "docs only change") {
		t.Errorf("log dropped the granted subtree's own commit:\n%s", log.Text)
	}
	if strings.Contains(log.Text, "secret only change") {
		t.Errorf("log reported a commit from a subtree no --allow granted:\n%s", log.Text)
	}

	diff := call(t, tool, map[string]any{"command": "diff", "args": []any{"--name-only"}})
	if !strings.Contains(diff.Text, "docs/notes.md") {
		t.Errorf("diff dropped the granted subtree's own change:\n%s", diff.Text)
	}
	if strings.Contains(diff.Text, "secret/creds.txt") {
		t.Errorf("diff reported a change from a subtree no --allow granted:\n%s", diff.Text)
	}
}

// TestGitRead_ScopesStatusToTheGrantedSubtrees is asserted separately from log
// and diff because an unscoped `status` is the one form that leaks path names
// without reading a file: it reports the working-tree state of the whole
// repository, including the paths the floor exists to hide.
func TestGitRead_ScopesStatusToTheGrantedSubtrees(t *testing.T) {
	root := gitReadRepo(t, nil)
	tool := newGitReadTool(t, root, "docs")

	result := call(t, tool, map[string]any{"command": "status", "args": []any{"--porcelain"}})

	if !strings.Contains(result.Text, "docs/notes.md") {
		t.Errorf("status dropped the granted subtree's uncommitted change:\n%s", result.Text)
	}
	if strings.Contains(result.Text, "secret/creds.txt") {
		t.Errorf("status named a path from a subtree no --allow granted:\n%s", result.Text)
	}
}

// TestGitRead_PathsScopesTheDiffToOneGrantedSubtree is the gap this parameter
// closes. Before it, the only diff a reviewer could obtain was the whole
// granted surface — which is exactly the read that hits the line cap — because
// `diff <range> -- <path>` is refused and `diff <range> <path>` makes the path
// a revision.
func TestGitRead_PathsScopesTheDiffToOneGrantedSubtree(t *testing.T) {
	root := gitReadRepo(t, nil)
	tool := newGitReadTool(t, root, ".")

	result := call(t, tool, map[string]any{
		"command": "diff",
		"args":    []any{"--unified=0", "HEAD~2"},
		"paths":   []any{"docs"},
	})

	if result.IsError {
		t.Fatalf("a diff scoped inside the grant was refused: %s", result.Text)
	}
	if !strings.Contains(result.Text, "docs/notes.md") {
		t.Errorf("the scoped diff dropped the subtree it was scoped to:\n%s", result.Text)
	}
	if strings.Contains(result.Text, "secret/creds.txt") {
		t.Errorf("the scoped diff reported a subtree paths never named:\n%s", result.Text)
	}
}

// TestGitRead_PathsNarrowTheGrantAndNeverWidenIt is the same refusal as
// TestGitRead_RefusesAPathsEntryOutsideTheAllowedSubtreesWithoutReachingGit,
// asserted where the mutation is visible: git is on PATH here on purpose, so
// deleting the resolution over `paths` does not merely change the wording — the
// call succeeds and this test fails carrying the ungranted subtree's own diff.
func TestGitRead_PathsNarrowTheGrantAndNeverWidenIt(t *testing.T) {
	root := gitReadRepo(t, nil)
	tool := newGitReadTool(t, root, "docs")

	result := tool.Handler(t.Context(), map[string]any{
		"command": "diff",
		"args":    []any{"HEAD~2"},
		"paths":   []any{"secret"},
	})

	mentions(t, result, "--allow", "secret")
	if strings.Contains(result.Text, "rotated") {
		t.Errorf("the ungranted subtree's own diff came back:\n%s", result.Text)
	}
}

// TestGitRead_AnnouncesTruncationWithBothRealNumbers uses one big blob rather
// than a long history, because the cap is about the answer's size and a fixture
// with 2500 commits would only be slower.
func TestGitRead_AnnouncesTruncationWithBothRealNumbers(t *testing.T) {
	const total = tools.MaxGitReadLines + 500

	root := gitReadRepo(t, map[string]string{
		"docs/big.txt": strings.Repeat("a line of a very long file\n", total),
	})
	tool := newGitReadTool(t, root, "docs")

	result := call(t, tool, map[string]any{"command": "show", "args": []any{"HEAD:docs/big.txt"}})

	got := lines(result)
	if len(got) != tools.MaxGitReadLines+1 {
		t.Fatalf("show returned %d lines, want %d plus the truncation line", len(got), tools.MaxGitReadLines)
	}
	want := fmt.Sprintf("[truncated: showing %d of %d lines]", tools.MaxGitReadLines, total)
	if last := got[len(got)-1]; last != want {
		t.Errorf("the last line is %q, want %q", last, want)
	}
}

func TestGitRead_SaysNothingAboutTruncationWhenNothingWasTruncated(t *testing.T) {
	root := gitReadRepo(t, nil)
	tool := newGitReadTool(t, root, "docs")

	result := call(t, tool, map[string]any{"command": "log", "args": []any{"--oneline"}})

	if strings.Contains(result.Text, "truncated") {
		t.Errorf("a short answer announced truncation:\n%s", result.Text)
	}
}

// TestGitRead_ANonZeroGitExitIsAToolErrorCarryingGitsStderr: the revision is
// the model's mistake to learn from, and git's own sentence is the only thing
// that says which mistake it was.
func TestGitRead_ANonZeroGitExitIsAToolErrorCarryingGitsStderr(t *testing.T) {
	root := gitReadRepo(t, nil)
	tool := newGitReadTool(t, root, "docs")

	result := tool.Handler(t.Context(), map[string]any{
		"command": "show",
		"args":    []any{"deadbeefdeadbeef:docs/notes.md"},
	})

	if !result.IsError {
		t.Fatalf("a failing git was reported as an answer:\n%s", result.Text)
	}
	if !strings.Contains(result.Text, "deadbeefdeadbeef") {
		t.Errorf("the tool error carries none of git's stderr:\n%s", result.Text)
	}
}

// TestGitRead_AMissingGitIsAToolErrorNotARunFailure: the other three tools
// still work on the working tree, and a review that read files without reading
// history is a degraded review rather than no review (specs 11).
func TestGitRead_AMissingGitIsAToolErrorNotARunFailure(t *testing.T) {
	tool := newGitReadTool(t, gitReadTree(t, gitReadFiles()), "docs")
	withoutGitOnPath(t)

	result := tool.Handler(t.Context(), map[string]any{"command": "log", "args": []any{"--oneline"}})

	if !result.IsError {
		t.Fatalf("a missing git was reported as an answer:\n%s", result.Text)
	}
	if !strings.Contains(result.Text, "PATH") {
		t.Errorf("the tool error never says git could not be found:\n%s", result.Text)
	}
}

// TestGitRead_AFailingGitLeavesTheRunAlive is the same failure asserted where
// it matters — through issue 03's loop. Nothing a tool returns can end a run:
// the refusal comes back as a tool result the model reads, and the review ends
// with the report it went on to write.
func TestGitRead_AFailingGitLeavesTheRunAlive(t *testing.T) {
	const answer = "# Review\n\nthe revision was not there\n"

	root := gitReadRepo(t, nil)
	tool := newGitReadTool(t, root, "docs")

	models, handle := fauxtest.NewRegistry(t, fauxtest.RegistryOptions{
		ProviderID: fixtureProvider,
		ModelIDs:   []string{fixtureModel},
		Auth:       fauxtest.CredentialedAuth("OAuth"),
	})
	handle.SetResponses(
		faux.Step(faux.AssistantMessage(
			[]ai.AssistantContentPart{faux.ToolCall("git_read", map[string]any{
				"command": "show",
				"args":    []any{"deadbeefdeadbeef:docs/notes.md"},
			}, nil)},
			&faux.AssistantMessageOptions{StopReason: ai.StopReasonToolUse},
		)),
		faux.Step(faux.TextMessage(answer, nil)),
	)
	model := models.GetModel(fixtureProvider, fixtureModel)
	if model == nil {
		t.Fatalf("model %s/%s missing from the registry", fixtureProvider, fixtureModel)
	}

	stderr := &bytes.Buffer{}
	state := diag.NewState()
	loop := &reviewer.Loop{
		Conversation: reviewer.NewConversation(models, model, "you review repositories you did not write", "review this"),
		Tools:        tools.NewRegistry(tool),
		State:        state,
		Stderr:       stderr,
	}

	report, err := loop.Run(t.Context())

	if err != nil {
		t.Fatalf("Run() error = %v, want nil — a failing tool never ends a run", err)
	}
	if report != answer {
		t.Errorf("Run() = %q, want the report the run went on to write", report)
	}
	if state.Turns != 2 {
		t.Errorf("state.Turns = %d, want 2 — the run continued past the failing tool", state.Turns)
	}
	if !strings.Contains(stderr.String(), "git_read") {
		t.Errorf("stderr carries no tool line for the call:\n%s", stderr.String())
	}
}
