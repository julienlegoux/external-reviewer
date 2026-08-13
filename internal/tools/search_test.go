package tools_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/julienlegoux/kern-link/ai"
	"github.com/julienlegoux/kern-link/ai/providers/faux"

	"github.com/julienlegoux/external-reviewer/internal/confine"
	"github.com/julienlegoux/external-reviewer/internal/diag"
	"github.com/julienlegoux/external-reviewer/internal/fauxtest"
	"github.com/julienlegoux/external-reviewer/internal/repo"
	"github.com/julienlegoux/external-reviewer/internal/reviewer"
	"github.com/julienlegoux/external-reviewer/internal/tools"
)

// newSearchTool wires the real search tool over a real repository, the same
// way newListTool does: every rule that decides what may be matched —
// gitignore, the allow-list, the sensitive-file floor — lives in the
// enumeration and the scope underneath, and a fake would assert the formatting
// and nothing else.
func newSearchTool(t *testing.T, files map[string]string, allow ...string) tools.Tool {
	t.Helper()

	return tools.Search(newSearchScope(t, files, allow...))
}

// newSearchScope builds the fixture repository and returns the two things
// search is made of: the run's confinement and the enumeration it draws
// candidates from.
func newSearchScope(t *testing.T, files map[string]string, allow ...string) (*confine.Scope, *repo.Enumerator) {
	t.Helper()

	root := t.TempDir()
	writeFixture(t, root, files)

	scope, err := confine.OpenScope(root, allow)
	if err != nil {
		t.Fatalf("OpenScope(%q, %v): %v", root, allow, err)
	}
	t.Cleanup(func() {
		if err := scope.Close(); err != nil {
			t.Errorf("closing the scope: %v", err)
		}
	})

	return scope, repo.NewEnumerator(scope, root, func(string) {})
}

func writeFixture(t *testing.T, root string, files map[string]string) {
	t.Helper()

	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatalf("creating fixture directory for %s: %v", name, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("writing fixture %s: %v", name, err)
		}
	}
}

// TestSearch_ReturnsPathLineTextPerMatchInEnumerationOrder is the result shape
// specs 09 fixes: one match per line, repo-relative and /-separated, in the
// order enumeration produced the files rather than in any ranking of its own.
func TestSearch_ReturnsPathLineTextPerMatchInEnumerationOrder(t *testing.T) {
	tool := newSearchTool(t, map[string]string{
		"docs/a.md":   "alpha\nneedle here\nomega\n",
		"docs/b.md":   "nothing\nneedle again\n",
		"src/main.go": "// needle\n",
	}, ".")

	result := call(t, tool, map[string]any{"pattern": "needle", "context": 0})

	want := []string{
		"docs/a.md:2:needle here",
		"docs/b.md:2:needle again",
		"src/main.go:1:// needle",
	}
	if got := lines(result); !slices.Equal(got, want) {
		t.Errorf("search returned\n\t%q\nwant\n\t%q", got, want)
	}
	if result.IsError {
		t.Error("a successful search was reported as an error result")
	}
}

// TestSearch_ContextLinesSurroundTheMatchAndReadDifferently covers the
// parameter this tool exists for: a match with its surroundings, and a
// surrounding line a model can tell apart from a matching one at a glance —
// grep's own convention, path-line-text against path:line:text.
func TestSearch_ContextLinesSurroundTheMatchAndReadDifferently(t *testing.T) {
	files := map[string]string{
		"docs/a.md": "one\ntwo\nthree\nneedle\nfive\nsix\nseven\n",
	}

	tests := []struct {
		name    string
		context any
		want    []string
	}{
		{
			name:    "two lines either side",
			context: 2,
			want: []string{
				"docs/a.md-2-two",
				"docs/a.md-3-three",
				"docs/a.md:4:needle",
				"docs/a.md-5-five",
				"docs/a.md-6-six",
			},
		},
		{
			name:    "none at all",
			context: 0,
			want:    []string{"docs/a.md:4:needle"},
		},
		{
			name:    "the default, when the model says nothing",
			context: nil,
			want: []string{
				"docs/a.md-2-two",
				"docs/a.md-3-three",
				"docs/a.md:4:needle",
				"docs/a.md-5-five",
				"docs/a.md-6-six",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tool := newSearchTool(t, files, ".")

			arguments := map[string]any{"pattern": "needle"}
			if test.context != nil {
				arguments["context"] = test.context
			}
			result := call(t, tool, arguments)

			if got := lines(result); !slices.Equal(got, test.want) {
				t.Errorf("search returned\n\t%q\nwant\n\t%q", got, test.want)
			}
		})
	}
}

// TestSearch_ContextAtTheEdgesOfAFileIsWhatExists — a match on the first or
// last line returns the context there is, with nothing padded and nothing
// refused.
func TestSearch_ContextAtTheEdgesOfAFileIsWhatExists(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "the first line",
			content: "needle\ntwo\nthree\nfour\n",
			want:    []string{"docs/a.md:1:needle", "docs/a.md-2-two", "docs/a.md-3-three"},
		},
		{
			name:    "the last line",
			content: "one\ntwo\nthree\nneedle\n",
			want:    []string{"docs/a.md-2-two", "docs/a.md-3-three", "docs/a.md:4:needle"},
		},
		{
			name:    "the only line, with no trailing newline",
			content: "needle",
			want:    []string{"docs/a.md:1:needle"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tool := newSearchTool(t, map[string]string{"docs/a.md": test.content}, ".")

			result := call(t, tool, map[string]any{"pattern": "needle"})

			if got := lines(result); !slices.Equal(got, test.want) {
				t.Errorf("search returned\n\t%q\nwant\n\t%q", got, test.want)
			}
		})
	}
}

// TestSearch_OverlappingContextWindowsAreMergedNotRepeated is the property
// that keeps the result's size sane: two matches three lines apart with two
// lines of context each share four lines, and a result that printed them twice
// would spend the reviewer's context window on the same text.
func TestSearch_OverlappingContextWindowsAreMergedNotRepeated(t *testing.T) {
	tool := newSearchTool(t, map[string]string{
		"docs/a.md": "one\ntwo\nneedle\nfour\nneedle\nsix\nseven\n",
	}, ".")

	result := call(t, tool, map[string]any{"pattern": "needle"})

	want := []string{
		"docs/a.md-1-one",
		"docs/a.md-2-two",
		"docs/a.md:3:needle",
		"docs/a.md-4-four",
		"docs/a.md:5:needle",
		"docs/a.md-6-six",
		"docs/a.md-7-seven",
	}
	if got := lines(result); !slices.Equal(got, want) {
		t.Errorf("search returned\n\t%q\nwant\n\t%q", got, want)
	}
}

// TestSearch_SeparatesDiscontiguousGroups — with context in play, two windows
// that do not touch are separated the way grep separates them, so a model
// never reads two distant regions as one passage.
func TestSearch_SeparatesDiscontiguousGroups(t *testing.T) {
	tool := newSearchTool(t, map[string]string{
		"docs/a.md": "needle\ntwo\nthree\nfour\nfive\nsix\nneedle\n",
	}, ".")

	result := call(t, tool, map[string]any{"pattern": "needle", "context": 1})

	want := []string{
		"docs/a.md:1:needle",
		"docs/a.md-2-two",
		"--",
		"docs/a.md-6-six",
		"docs/a.md:7:needle",
	}
	if got := lines(result); !slices.Equal(got, want) {
		t.Errorf("search returned\n\t%q\nwant\n\t%q", got, want)
	}
}

// TestSearch_SaysSoWhenNothingMatches — an empty result is a legitimate
// answer to a legitimate question, not a refusal.
func TestSearch_SaysSoWhenNothingMatches(t *testing.T) {
	tool := newSearchTool(t, map[string]string{"docs/a.md": "alpha\n"}, ".")

	result := call(t, tool, map[string]any{"pattern": "needle"})

	if result.IsError {
		t.Error("an empty result is not a tool error: the reviewer asked a legitimate question")
	}
	if !strings.Contains(result.Text, "needle") {
		t.Errorf("an empty result does not name the pattern that matched nothing: %q", result.Text)
	}
}

// TestSearch_NarrowsTheCandidateFilesByGlobAndPath — neither parameter can
// widen the file set; both only narrow what enumeration already confined.
func TestSearch_NarrowsTheCandidateFilesByGlobAndPath(t *testing.T) {
	files := map[string]string{
		"README.md":       "needle at the top\n",
		"docs/notes.md":   "needle in the docs\n",
		"internal/run.go": "// needle in the code\n",
		"internal/run.md": "needle beside the code\n",
	}

	tests := []struct {
		name      string
		arguments map[string]any
		want      []string
	}{
		{
			name:      "a glob keeps only the matching files",
			arguments: map[string]any{"glob": "**/*.go"},
			want:      []string{"internal/run.go:1:// needle in the code"},
		},
		{
			name:      "a path keeps only one subtree",
			arguments: map[string]any{"path": "internal"},
			want: []string{
				"internal/run.go:1:// needle in the code",
				"internal/run.md:1:needle beside the code",
			},
		},
		{
			name:      "the two narrow together",
			arguments: map[string]any{"path": "internal", "glob": "**/*.md"},
			want:      []string{"internal/run.md:1:needle beside the code"},
		},
		{
			name:      "the default path is the whole granted surface",
			arguments: map[string]any{},
			want: []string{
				"README.md:1:needle at the top",
				"docs/notes.md:1:needle in the docs",
				"internal/run.go:1:// needle in the code",
				"internal/run.md:1:needle beside the code",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tool := newSearchTool(t, files, ".")

			arguments := map[string]any{"pattern": "needle", "context": 0}
			for name, value := range test.arguments {
				arguments[name] = value
			}
			result := call(t, tool, arguments)

			if got := lines(result); !slices.Equal(got, test.want) {
				t.Errorf("search returned\n\t%q\nwant\n\t%q", got, test.want)
			}
		})
	}
}

// TestSearch_RefusesAPathOutsideTheAllowList is the confinement half of the
// path parameter: a subtree the invocation never granted is refused by the
// rule that refused it, because "widen --allow" and "that file is a
// credential" are different things for a reviewer to learn (specs 10).
func TestSearch_RefusesAPathOutsideTheAllowList(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		mentions string
	}{
		{
			name:     "a sibling subtree the invocation never granted",
			path:     "internal",
			mentions: "outside the allowed subtrees",
		},
		{
			name:     "a traversal out of the repository",
			path:     "../elsewhere",
			mentions: "not inside the repository root",
		},
		{
			name:     "a native path spelling",
			path:     `docs\notes.md`,
			mentions: "not inside the repository root",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tool := newSearchTool(t, map[string]string{
				"docs/notes.md":   "needle in the docs\n",
				"internal/run.go": "// needle in the code\n",
			}, "docs")

			result := tool.Handler(t.Context(), map[string]any{"pattern": "needle", "path": test.path})

			if !result.IsError {
				t.Fatalf("a path outside the allow-list returned a successful result:\n%s", result.Text)
			}
			if !strings.Contains(result.Text, test.mentions) {
				t.Errorf("the refusal %q does not name the rule that refused (%q)", result.Text, test.mentions)
			}
		})
	}
}

// TestSearch_NeverReportsAFileOutsideTheAllowList — the allow-list is not
// only checked when the model names a path. A file inside the repository but
// outside the granted subtrees is never a candidate at all.
func TestSearch_NeverReportsAFileOutsideTheAllowList(t *testing.T) {
	tool := newSearchTool(t, map[string]string{
		"docs/notes.md":   "needle in the docs\n",
		"internal/run.go": "// needle in the code\n",
		"secrets/key.txt": "needle in the secrets\n",
	}, "docs")

	result := call(t, tool, map[string]any{"pattern": "needle", "context": 0})

	want := []string{"docs/notes.md:1:needle in the docs"}
	if got := lines(result); !slices.Equal(got, want) {
		t.Errorf("search returned\n\t%q\nwant\n\t%q", got, want)
	}
}

// TestSearch_NeverSearchesAFileOnTheSensitiveFileFloor — the floor holds
// under `--allow .`, which is the whole point of it being a floor rather than
// a default.
func TestSearch_NeverSearchesAFileOnTheSensitiveFileFloor(t *testing.T) {
	tool := newSearchTool(t, map[string]string{ //nolint:gosec // G101 reads the fixture's filenames as credentials, which is the point: these are the floor's own names, and every value is the search needle
		".env":             "SETTING=needle\n",
		"id_rsa":           "needle\n",
		"credentials.json": "{\"value\": \"needle\"}\n",
		".git/config":      "needle\n",
		"docs/notes.md":    "needle in the docs\n",
	}, ".")

	result := call(t, tool, map[string]any{"pattern": "needle", "context": 0})

	want := []string{"docs/notes.md:1:needle in the docs"}
	if got := lines(result); !slices.Equal(got, want) {
		t.Errorf("search returned\n\t%q\nwant\n\t%q", got, want)
	}
}

// TestSearch_HonoursGitignore is the promise `list` makes, made by the other
// half of the pair: what the reviewer can glob and what it can grep are the
// same file set, so a vendored dependency tree never reaches its context.
func TestSearch_HonoursGitignore(t *testing.T) {
	requireGit(t)

	root := t.TempDir()
	writeFixture(t, root, map[string]string{
		".gitignore":                "node_modules/\n",
		"docs/notes.md":             "needle in the docs\n",
		"node_modules/pkg/index.js": "// needle in a vendored package\n",
	})
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
	tool := tools.Search(scope, repo.NewEnumerator(scope, root, func(string) {}))

	result := call(t, tool, map[string]any{"pattern": "needle", "context": 0})

	want := []string{"docs/notes.md:1:needle in the docs"}
	if got := lines(result); !slices.Equal(got, want) {
		t.Errorf("search returned\n\t%q\nwant\n\t%q", got, want)
	}
}

// TestSearch_ReportsABadPatternAsAToolErrorRatherThanAFailure — RE2 has no
// backreferences and no lookaround, and a pattern using them failing to
// compile is the intended behaviour rather than a gap: the model is told what
// was wrong and rewrites it.
func TestSearch_ReportsABadPatternAsAToolErrorRatherThanAFailure(t *testing.T) {
	tests := []struct {
		name      string
		arguments map[string]any
		mentions  string
	}{
		{name: "an unclosed group", arguments: map[string]any{"pattern": "func ("}, mentions: "missing closing )"},
		{name: "a backreference RE2 does not implement", arguments: map[string]any{"pattern": `(\w+) \1`}, mentions: "invalid escape sequence"},
		{name: "a lookahead RE2 does not implement", arguments: map[string]any{"pattern": "foo(?=bar)"}, mentions: "invalid or unsupported Perl syntax"},
		{name: "no pattern at all", arguments: map[string]any{}, mentions: "pattern"},
		{name: "a pattern that is not a string", arguments: map[string]any{"pattern": 7}, mentions: "pattern"},
		{name: "a glob that cannot match a wire path", arguments: map[string]any{"pattern": "needle", "glob": "/etc/*"}, mentions: "glob"},
		{name: "a context that is not a number", arguments: map[string]any{"pattern": "needle", "context": "two"}, mentions: "context"},
		{name: "a max_results that is not a number", arguments: map[string]any{"pattern": "needle", "max_results": "many"}, mentions: "max_results"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tool := newSearchTool(t, map[string]string{"docs/a.md": "needle\n"}, ".")

			result := tool.Handler(t.Context(), test.arguments)

			if !result.IsError {
				t.Fatalf("a bad argument returned a successful result:\n%s", result.Text)
			}
			if !strings.Contains(result.Text, test.mentions) {
				t.Errorf("the error result %q does not mention %q", result.Text, test.mentions)
			}
		})
	}
}

// TestSearch_AnInvalidPatternLeavesTheRunAlive is the two-level failure rule
// end to end, through issue 03's loop: a tool error is a turn the model reads
// and corrects from, never a run that ends. The corrected pattern is
// dispatched, and the run finishes with the report.
func TestSearch_AnInvalidPatternLeavesTheRunAlive(t *testing.T) {
	const answer = "# Review\n\nno leads\n"

	tool := newSearchTool(t, map[string]string{"docs/notes.md": "needle in the docs\n"}, ".")
	loop, state, dispatched := searchLoop(t, tool,
		toolCallStep("search", map[string]any{"pattern": "func ("}),
		toolCallStep("search", map[string]any{"pattern": "needle", "context": 0}),
		faux.Step(faux.TextMessage(answer, nil)),
	)

	report, err := loop.Run(context.Background())

	if err != nil {
		t.Fatalf("Run() error = %v, want nil — a bad regex is a tool error, not a run failure", err)
	}
	if report != answer {
		t.Errorf("Run() = %q, want %q", report, answer)
	}
	if state.Turns != 3 {
		t.Errorf("state.Turns = %d, want 3 — the run continued past the invalid pattern", state.Turns)
	}
	results := dispatched.results
	if len(results) != 2 {
		t.Fatalf("the conversation carries %d tool results, want 2", len(results))
	}
	if !results[0].IsError || !strings.Contains(results[0].Text, "not a valid regular expression") {
		t.Errorf("the first tool result is %+v, want an error naming the pattern problem", results[0])
	}
	if results[1].IsError || !strings.Contains(results[1].Text, "docs/notes.md:1:needle in the docs") {
		t.Errorf("the corrected search returned %+v, want the real matches", results[1])
	}
}

// TestSearch_AnnouncesTruncationWithBothRealNumbers — a result that silently
// showed half the matches would read exactly as confident as one that showed
// them all (specs 09), so both numbers are real: the matches beyond the cap
// are counted even though nothing is kept for them.
func TestSearch_AnnouncesTruncationWithBothRealNumbers(t *testing.T) {
	const total = 431

	var content strings.Builder
	for i := range total {
		fmt.Fprintf(&content, "needle %d\n", i)
	}

	tests := []struct {
		name       string
		maxResults any
		wantShown  int
	}{
		{name: "the default cap", maxResults: nil, wantShown: tools.DefaultSearchMatches},
		{name: "a cap the model chose", maxResults: 5, wantShown: 5},
		{name: "a cap above the maximum is clamped", maxResults: 5000, wantShown: tools.MaxSearchMatches},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tool := newSearchTool(t, map[string]string{"docs/a.md": content.String()}, ".")

			arguments := map[string]any{"pattern": "^needle", "context": 0}
			if test.maxResults != nil {
				arguments["max_results"] = test.maxResults
			}
			result := call(t, tool, arguments)

			got := lines(result)
			if len(got) != test.wantShown+1 {
				t.Fatalf("search returned %d lines, want %d matches plus one truncation line", len(got), test.wantShown)
			}
			want := "[truncated: showing " + strconv.Itoa(test.wantShown) + " of " + strconv.Itoa(total) + " matches]"
			if marker := got[len(got)-1]; marker != want {
				t.Errorf("the truncation line is %q, want %q", marker, want)
			}
		})
	}
}

// TestSearch_SaysNothingAboutTruncationWhenNothingWasTruncated — the marker
// is a statement about this result, not decoration on every result.
func TestSearch_SaysNothingAboutTruncationWhenNothingWasTruncated(t *testing.T) {
	var content strings.Builder
	for i := range tools.MaxSearchMatches {
		fmt.Fprintf(&content, "needle %d\n", i)
	}
	tool := newSearchTool(t, map[string]string{"docs/a.md": content.String()}, ".")

	result := call(t, tool, map[string]any{"pattern": "^needle", "context": 0, "max_results": tools.MaxSearchMatches})

	if len(lines(result)) != tools.MaxSearchMatches {
		t.Fatalf("search returned %d lines, want exactly %d matches", len(lines(result)), tools.MaxSearchMatches)
	}
	if strings.Contains(result.Text, "truncated") {
		t.Error("a result at the cap but not over it announced truncation")
	}
}

// TestSearch_BoundsTheSizeOfOneResult is what makes the worst case a number
// rather than a hope: the context window is clamped like the match count, so
// no single call can return more than MaxSearchMatches * (2*MaxSearchContext+1)
// lines however large the arguments are.
func TestSearch_BoundsTheSizeOfOneResult(t *testing.T) {
	var content strings.Builder
	for i := range 200 {
		fmt.Fprintf(&content, "line %d\n", i)
	}
	content.WriteString("needle\n")
	for i := range 200 {
		fmt.Fprintf(&content, "line %d\n", i)
	}
	tool := newSearchTool(t, map[string]string{"docs/a.md": content.String()}, ".")

	result := call(t, tool, map[string]any{"pattern": "needle", "context": 1000})

	want := 2*tools.MaxSearchContext + 1
	if got := len(lines(result)); got != want {
		t.Errorf("a context of 1000 returned %d lines, want %d — the window is clamped to %d either side",
			got, want, tools.MaxSearchContext)
	}
}

// TestSearch_ClipsAVeryLongLine — a minified bundle is a text file by every
// other test, and one of its lines would otherwise be the whole result.
func TestSearch_ClipsAVeryLongLine(t *testing.T) {
	tool := newSearchTool(t, map[string]string{
		"docs/a.md": "needle" + strings.Repeat("x", 4000) + "\n",
	}, ".")

	result := call(t, tool, map[string]any{"pattern": "needle", "context": 0})

	got := lines(result)
	if len(got) != 1 {
		t.Fatalf("search returned %d lines, want 1", len(got))
	}
	if len(got[0]) > 600 {
		t.Errorf("a 4000-character line was rendered in full (%d characters)", len(got[0]))
	}
	if !strings.HasSuffix(got[0], "…") {
		t.Errorf("a clipped line does not say it was clipped: %q", got[0])
	}
}

// TestSearch_SkipsFilesThatAreNotMaterialAReviewerReads — a NUL byte in the
// first 8 KB marks a compiled artefact, whose matches would be noise its
// context lines multiply (specs 11).
func TestSearch_SkipsFilesThatAreNotMaterialAReviewerReads(t *testing.T) {
	tool := newSearchTool(t, map[string]string{
		"docs/notes.md": "needle in the docs\n",
		"build/app.bin": "needle\x00 in a binary\n",
	}, ".")

	result := call(t, tool, map[string]any{"pattern": "needle", "context": 0})

	want := []string{"docs/notes.md:1:needle in the docs"}
	if got := lines(result); !slices.Equal(got, want) {
		t.Errorf("search returned\n\t%q\nwant\n\t%q", got, want)
	}
}

// TestSearch_APathologicalPatternCompletesPromptly demonstrates RE2 rather
// than a backtracking engine: (a+)+$ against a long non-matching line is the
// classic exponential blow-up, and a model supplies this pattern by accident
// long before it supplies one on purpose.
func TestSearch_APathologicalPatternCompletesPromptly(t *testing.T) {
	tool := newSearchTool(t, map[string]string{
		"docs/a.md": strings.Repeat("a", 4000) + "b\n",
	}, ".")

	done := make(chan tools.Result, 1)
	start := time.Now()
	go func() { done <- tool.Handler(context.Background(), map[string]any{"pattern": "(a+)+$"}) }()

	select {
	case result := <-done:
		if result.IsError {
			t.Fatalf("the pattern was refused rather than run: %s", result.Text)
		}
		t.Logf("(a+)+$ over 4000 characters completed in %s", time.Since(start))
	case <-time.After(10 * time.Second):
		t.Fatal("(a+)+$ over a long non-matching line did not complete in 10s — this is not RE2")
	}
}

// TestSearch_DeclarationIsAWireContract checks the half of the tool the model
// reads before it ever calls it: snake_case schema fields, and a description
// that stands alone because the caller owns the system prompt (specs 09).
func TestSearch_DeclarationIsAWireContract(t *testing.T) {
	tool := newSearchTool(t, map[string]string{"docs/a.md": "needle\n"}, ".")
	declaration := tool.Declaration

	if declaration.Name != "search" {
		t.Errorf("tool name is %q, want %q", declaration.Name, "search")
	}
	for _, mention := range []string{"path:line:text", "path-line-text", "truncat", "context", "max_results", "/"} {
		if !strings.Contains(declaration.Description, mention) {
			t.Errorf("the tool description never mentions %q:\n%s", mention, declaration.Description)
		}
	}

	var schema struct {
		Type       string   `json:"type"`
		Required   []string `json:"required"`
		Properties map[string]struct {
			Type        string `json:"type"`
			Description string `json:"description"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(declaration.Parameters, &schema); err != nil {
		t.Fatalf("the parameter schema is not valid JSON: %v", err)
	}
	if schema.Type != "object" {
		t.Errorf("the parameter schema is of type %q, want %q", schema.Type, "object")
	}
	if !slices.Contains(schema.Required, "pattern") {
		t.Errorf("the schema requires %v, want pattern among them", schema.Required)
	}
	for name, want := range map[string]string{
		"pattern":     "string",
		"path":        "string",
		"glob":        "string",
		"max_results": "integer",
		"context":     "integer",
	} {
		property, declared := schema.Properties[name]
		if !declared {
			t.Errorf("the schema declares no %q parameter", name)
			continue
		}
		if property.Type != want {
			t.Errorf("the %s parameter is of type %q, want %q", name, property.Type, want)
		}
		if property.Description == "" {
			t.Errorf("the %s parameter carries no description", name)
		}
	}
}

// searchLoop wires the tool into issue 03's real loop over the faux
// provider's response queue, which is how "a tool error leaves the run alive"
// is asserted end to end rather than at the handler.
func searchLoop(t *testing.T, tool tools.Tool, responses ...faux.ResponseStep) (*reviewer.Loop, *diag.State, *recordingToolSet) {
	t.Helper()

	models, handle := fauxtest.NewRegistry(t, fauxtest.RegistryOptions{
		ProviderID: fixtureProvider,
		ModelIDs:   []string{fixtureModel},
		Auth:       fauxtest.CredentialedAuth("OAuth"),
	})
	handle.SetResponses(responses...)

	model := models.GetModel(fixtureProvider, fixtureModel)
	if model == nil {
		t.Fatalf("model %s/%s missing from the registry", fixtureProvider, fixtureModel)
	}

	state := diag.NewState()
	dispatched := &recordingToolSet{inner: tools.NewRegistry(tool)}
	loop := &reviewer.Loop{
		Conversation: reviewer.NewConversation(models, model, "you review repositories you did not write", "review this"),
		Tools:        dispatched,
		State:        state,
		Stderr:       io.Discard,
	}
	return loop, state, dispatched
}

// recordingToolSet is the real registry with a notebook: the loop's ToolSet
// seam is what a test can see the answers through, since the conversation's
// own messages are the loop's business rather than this package's.
type recordingToolSet struct {
	inner   reviewer.ToolSet
	results []tools.Result
}

func (r *recordingToolSet) Declarations() []ai.Tool { return r.inner.Declarations() }

func (r *recordingToolSet) Dispatch(ctx context.Context, call ai.ToolCall) (string, bool) {
	text, isError := r.inner.Dispatch(ctx, call)
	r.results = append(r.results, tools.Result{Text: text, IsError: isError})
	return text, isError
}

// toolCallStep scripts one assistant turn that calls a tool and stops for its
// result.
func toolCallStep(name string, arguments map[string]any) faux.ResponseStep {
	return faux.Step(faux.AssistantMessage(
		[]ai.AssistantContentPart{faux.ToolCall(name, arguments, nil)},
		&faux.AssistantMessageOptions{StopReason: ai.StopReasonToolUse},
	))
}
