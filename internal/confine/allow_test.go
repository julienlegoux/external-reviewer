package confine_test

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/julienlegoux/external-reviewer/internal/confine"
)

// allowFixture is the tree every allow-list test resolves against.
var allowFixture = map[string]string{
	"README.md":              "# fixture\n",
	"docs/notes.md":          "notes\n",
	"docs/epics/EPIC_2.md":   "# epic\n",
	"internal/cli/run.go":    "package cli\n",
	"internal/confine/x.go":  "package confine\n",
	"internal/cli/usage.txt": "usage\n",
}

// TestOpenScope_RejectsAllowValuesThatAreNotSubtreesOfTheRoot pins the
// allow-list's shape: a grant names an existing directory inside the root and
// nothing else. A regular file, a path that does not exist and a path that
// leaves the root are all refused, and the refusal names the offending value
// so a caller can tell which of several --allow arguments was wrong.
//
// Single-file grants are deliberately unsupported: every downstream mechanism
// — the resolver's subtree test, git_read's pathspecs, search's path
// parameter — reads the allow-list as subtrees.
func TestOpenScope_RejectsAllowValuesThatAreNotSubtreesOfTheRoot(t *testing.T) {
	tests := []struct {
		name  string
		allow []string
	}{
		{name: "no allow values at all", allow: nil},
		{name: "a regular file", allow: []string{"README.md"}},
		{name: "a regular file below a directory", allow: []string{"docs/notes.md"}},
		{name: "a path that does not exist", allow: []string{"nowhere"}},
		{name: "a path that does not exist below a directory", allow: []string{"docs/nowhere"}},
		{name: "traversal out of the root", allow: []string{"../outside"}},
		{name: "traversal back out through a real directory", allow: []string{"docs/../../outside"}},
		{name: "a posix absolute path", allow: []string{"/etc"}},
		{name: "one good value and one bad one", allow: []string{"docs", "nowhere"}},
	}

	repo := newRepo(t, allowFixture)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			scope, err := confine.OpenScope(repo, tc.allow)
			if err == nil {
				_ = scope.Close()
				t.Fatalf("OpenScope(%v) succeeded, want a refusal", tc.allow)
			}
			if scope != nil {
				t.Errorf("OpenScope(%v) returned a scope alongside its error, want nil", tc.allow)
			}
			if len(tc.allow) == 0 {
				if !strings.Contains(err.Error(), "--allow") {
					t.Errorf("error = %q, want it to name --allow as missing", err)
				}
				return
			}
			offending := tc.allow[len(tc.allow)-1]
			if !strings.Contains(err.Error(), offending) {
				t.Errorf("error = %q, want it to name the offending value %q", err, offending)
			}
		})
	}
}

// TestOpenScope_AcceptsExistingDirectories is the other half: every spelling
// of a real subtree is accepted, including "." for the whole repository —
// which grants everything but has to be said.
func TestOpenScope_AcceptsExistingDirectories(t *testing.T) {
	repo := newRepo(t, allowFixture)
	for _, allow := range [][]string{
		{"."},
		{"docs"},
		{"docs/epics"},
		{"docs", "internal"},
		{"docs/"},
		{"./docs"},
	} {
		t.Run(strings.Join(allow, " "), func(t *testing.T) {
			scope, err := confine.OpenScope(repo, allow)
			if err != nil {
				t.Fatalf("OpenScope(%v): %v", allow, err)
			}
			if err := scope.Close(); err != nil {
				t.Errorf("closing the scope: %v", err)
			}
		})
	}
}

// TestScope_Allowed_ReturnsCleanedWirePaths pins what the four tool issues
// read off the scope when they need the subtrees themselves — git_read's
// pathspecs, list's and search's enumeration roots. The values are wire
// paths, cleaned, deduplicated, and collapsed to "." when the whole
// repository was granted, so no tool has to re-derive any of that.
func TestScope_Allowed_ReturnsCleanedWirePaths(t *testing.T) {
	tests := []struct {
		name  string
		allow []string
		want  []string
	}{
		{name: "a single subtree", allow: []string{"docs"}, want: []string{"docs"}},
		{name: "trailing slashes and dot prefixes are cleaned", allow: []string{"docs/", "./internal"}, want: []string{"docs", "internal"}},
		{name: "duplicates collapse", allow: []string{"docs", "docs", "./docs/"}, want: []string{"docs"}},
		{name: "the whole repository", allow: []string{"."}, want: []string{"."}},
		{name: "the whole repository absorbs the rest", allow: []string{"docs", ".", "internal"}, want: []string{"."}},
		{name: "a nested subtree keeps its wire separator", allow: []string{"docs/epics"}, want: []string{"docs/epics"}},
	}

	repo := newRepo(t, allowFixture)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			scope := openScope(t, repo, tc.allow...)
			got := scope.Allowed()
			if !slices.Equal(got, tc.want) {
				t.Fatalf("Allowed() = %v, want %v", got, tc.want)
			}
			for _, value := range got {
				if strings.Contains(value, `\`) {
					t.Errorf("Allowed() = %v contains a backslash: allowed subtrees are wire paths", got)
				}
			}
		})
	}
}

// TestOpenScope_AcceptsNativeSeparatorsInAllowValues covers the one place an
// OS path legitimately reaches --allow: a human on Windows typing
// `--allow docs\epics` at their own shell. filepath.ToSlash converts it there
// and leaves it alone on Linux, where a backslash is an ordinary character in
// a filename rather than a separator — the conversion CONVENTIONS § Paths and
// platforms puts at the boundary, and nowhere else. The expectation is
// selected at runtime rather than by a build tag, so the one test runs on both
// matrix OSes.
func TestOpenScope_AcceptsNativeSeparatorsInAllowValues(t *testing.T) {
	repo := newRepo(t, allowFixture)
	native := filepath.FromSlash("docs/epics")

	scope, err := confine.OpenScope(repo, []string{native})
	if native == "docs/epics" {
		// Linux: the value was already a wire path and must resolve.
		if err != nil {
			t.Fatalf("OpenScope(%q): %v", native, err)
		}
		defer func() { _ = scope.Close() }()
		if got := scope.Allowed(); !slices.Equal(got, []string{"docs/epics"}) {
			t.Errorf("Allowed() = %v, want [docs/epics]", got)
		}
		return
	}
	// Windows: `docs\epics` is the same subtree spelled natively.
	if err != nil {
		t.Fatalf("OpenScope(%q): %v", native, err)
	}
	defer func() { _ = scope.Close() }()
	if got := scope.Allowed(); !slices.Equal(got, []string{"docs/epics"}) {
		t.Errorf("Allowed() = %v, want [docs/epics] — the value reaches the allow-list in wire form", got)
	}
}
