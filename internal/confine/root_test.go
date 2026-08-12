package confine_test

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/julienlegoux/external-reviewer/internal/confine"
)

// rootFixture is the tree the resolution tests run against.
var rootFixture = map[string]string{
	"README.md":            "# fixture\n",
	"docs/notes.md":        "notes\n",
	"docs/epics/EPIC_2.md": "# epic\n",
	"internal/cli/run.go":  "package cli\n",
}

// TestScope_Resolve_RefusesPathsThatAreNotInsideTheRoot covers the first of
// the three rules. Traversal, a POSIX absolute path and a Windows absolute
// path are refused identically on both matrix OSes: a wire path is
// repo-relative and /-separated by definition (CONVENTIONS § Paths and
// platforms), so a value that is not one never becomes a filename on either
// platform. The kernel remains the authority on escapes it alone can see —
// symlinks and reserved device names below — and answers with the same rule.
func TestScope_Resolve_RefusesPathsThatAreNotInsideTheRoot(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "parent traversal", path: "../outside"},
		{name: "traversal back out through a real directory", path: "docs/../../outside"},
		{name: "the parent itself", path: ".."},
		{name: "a posix absolute path", path: "/etc/passwd"},
		{name: "a windows absolute path", path: `C:\Windows\win.ini`},
		{name: "a windows drive-relative path", path: "C:notes.md"},
		{name: "a native separator", path: `docs\notes.md`},
		{name: "the empty path", path: ""},
	}

	repo := newRepo(t, rootFixture)
	scope := openScope(t, repo, ".")
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := scope.Resolve(tc.path)
			if err == nil {
				t.Fatalf("Resolve(%q) = %q, want a refusal", tc.path, got)
			}
			assertRule(t, err, confine.ErrOutsideRoot, tc.path)
		})
	}
}

// TestScope_Resolve_RefusesPathsOutsideTheAllowList covers the second rule,
// and the reason the allow-list exists at all: a path that is perfectly
// inside the root is still unreadable unless a --allow value granted its
// subtree.
func TestScope_Resolve_RefusesPathsOutsideTheAllowList(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "a file at the repository root", path: "README.md"},
		{name: "a file in an ungranted subtree", path: "internal/cli/run.go"},
		{name: "the ungranted subtree itself", path: "internal"},
		{name: "a sibling whose name shares the granted prefix", path: "docsignored/notes.md"},
	}

	repo := newRepo(t, rootFixture)
	writeFile(t, repo, "docsignored/notes.md", "not the granted subtree\n")
	scope := openScope(t, repo, "docs")
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := scope.Resolve(tc.path)
			if err == nil {
				t.Fatalf("Resolve(%q) = %q, want a refusal", tc.path, got)
			}
			assertRule(t, err, confine.ErrOutsideAllowList, tc.path)
			if strings.Contains(err.Error(), `\`) {
				t.Errorf("error = %q contains a backslash: refusals name the path in wire form", err)
			}
		})
	}
}

// TestScope_Resolve_AllowsPathsInsideAllowedSubtrees is the positive case, and
// the assertion that a resolved path comes back /-separated and repo-relative
// on both OSes rather than in whatever form the platform stores it.
func TestScope_Resolve_AllowsPathsInsideAllowedSubtrees(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "a file in the granted subtree", path: "docs/notes.md", want: "docs/notes.md"},
		{name: "a file deeper in the granted subtree", path: "docs/epics/EPIC_2.md", want: "docs/epics/EPIC_2.md"},
		{name: "the granted subtree itself", path: "docs", want: "docs"},
		{name: "a redundant dot segment is cleaned away", path: "docs/./epics/EPIC_2.md", want: "docs/epics/EPIC_2.md"},
		{name: "an interior traversal that stays inside", path: "docs/epics/../notes.md", want: "docs/notes.md"},
		{name: "a file in the second granted subtree", path: "internal/cli/run.go", want: "internal/cli/run.go"},
	}

	repo := newRepo(t, rootFixture)
	scope := openScope(t, repo, "docs", "internal")
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := scope.Resolve(tc.path)
			if err != nil {
				t.Fatalf("Resolve(%q): %v", tc.path, err)
			}
			if got != tc.want {
				t.Errorf("Resolve(%q) = %q, want %q", tc.path, got, tc.want)
			}
			if strings.Contains(got, `\`) {
				t.Errorf("Resolve(%q) = %q contains a backslash: a resolved path is a wire path", tc.path, got)
			}
		})
	}
}

// TestScope_ReadFile_ReadsThroughTheRoot proves the resolver is not only a
// predicate: a path inside an allowed subtree that matches nothing on the
// floor resolves and its contents come back, read through the one *os.Root.
func TestScope_ReadFile_ReadsThroughTheRoot(t *testing.T) {
	repo := newRepo(t, rootFixture)
	scope := openScope(t, repo, "docs")

	data, err := scope.ReadFile("docs/epics/EPIC_2.md")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "# epic\n" {
		t.Errorf("ReadFile = %q, want the fixture's contents", data)
	}

	file, err := scope.Open("docs/notes.md")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = file.Close() }()
	buf := make([]byte, 6)
	if _, err := file.Read(buf); err != nil {
		t.Fatalf("reading the opened file: %v", err)
	}
	if string(buf) != "notes\n" {
		t.Errorf("read %q, want the fixture's contents", buf)
	}
}

// TestScope_ReadFile_RefusesBeforeTouchingTheFilesystem pins the order the
// three rules run in, which is what makes the refusal vocabulary meaningful:
// an ungranted path is refused as ungranted whether or not a file is there.
func TestScope_ReadFile_RefusesBeforeTouchingTheFilesystem(t *testing.T) {
	repo := newRepo(t, rootFixture)
	scope := openScope(t, repo, "docs")

	if _, err := scope.ReadFile("internal/cli/run.go"); !errors.Is(err, confine.ErrOutsideAllowList) {
		t.Errorf("ReadFile of an existing but ungranted file = %v, want ErrOutsideAllowList", err)
	}
	if _, err := scope.ReadFile("internal/cli/nowhere.go"); !errors.Is(err, confine.ErrOutsideAllowList) {
		t.Errorf("ReadFile of a missing ungranted file = %v, want ErrOutsideAllowList, not a not-found", err)
	}
}

// TestScope_ReadFile_MissingFileIsNotARefusal keeps the three refusal forms
// meaning what they say: a granted path that simply is not there is an
// ordinary absence, reported as one, and never dressed up as a confinement
// refusal.
func TestScope_ReadFile_MissingFileIsNotARefusal(t *testing.T) {
	repo := newRepo(t, rootFixture)
	scope := openScope(t, repo, "docs")

	_, err := scope.ReadFile("docs/nowhere.md")
	if err == nil {
		t.Fatal("ReadFile of a missing granted file succeeded, want an error")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("error = %v, want it to satisfy fs.ErrNotExist", err)
	}
	for _, rule := range []error{confine.ErrOutsideRoot, confine.ErrOutsideAllowList, confine.ErrDeniedFilename} {
		if errors.Is(err, rule) {
			t.Errorf("error = %v also reports %v: a missing file is not a refusal", err, rule)
		}
	}
	if !strings.Contains(err.Error(), "docs/nowhere.md") {
		t.Errorf("error = %q, want it to name the path that was attempted", err)
	}
}

// TestScope_Open_RefusesSymlinksThatLeaveTheRoot is the escape only the
// kernel can see, and the reason os.OpenRoot carries this requirement rather
// than a filepath.Clean plus a prefix comparison. Creating a symlink is
// attempted for real and the case is skipped with a reason when the platform
// refuses — never guarded by a build tag, so the same test runs on both
// matrix OSes and the CI runner that can create one exercises it.
func TestScope_Open_RefusesSymlinksThatLeaveTheRoot(t *testing.T) {
	tests := []struct {
		name string
		link string
		// target is the symlink's target, native and relative to the
		// repository root's parent when relative.
		target func(outside string) string
		read   string
	}{
		{
			name:   "an absolute symlink to a file outside the root",
			link:   "docs/absolute",
			target: func(outside string) string { return filepath.Join(outside, "secret.txt") },
			read:   "docs/absolute",
		},
		{
			name:   "a relative symlink that traverses out of the root",
			link:   "docs/relative",
			target: func(string) string { return filepath.Join("..", "..", "..") },
			read:   "docs/relative",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			outside := t.TempDir()
			writeFile(t, outside, "secret.txt", "a credential\n")
			repo := newRepo(t, rootFixture)

			link := filepath.Join(repo, filepath.FromSlash(tc.link))
			if err := os.Symlink(tc.target(outside), link); err != nil {
				t.Skipf("this platform refuses to create a symlink (%v); the refusal is exercised wherever one can be created", err)
			}

			scope := openScope(t, repo, ".")
			file, err := scope.Open(tc.read)
			if err == nil {
				_ = file.Close()
				t.Fatalf("Open(%q) succeeded through a symlink that leaves the root", tc.read)
			}
			assertRule(t, err, confine.ErrOutsideRoot, tc.read)
		})
	}
}

// TestScope_Open_WindowsReservedDeviceNames is the platform split asserted by
// one test that runs on both matrix OSes and picks its expectation from
// runtime.GOOS: os.Root refuses NUL and COM1 on Windows, and on Linux they are
// ordinary filenames that must stay readable. A hand-rolled reserved-name
// blocklist would make the Linux file permanently unreadable and reintroduce
// exactly the path validation os.Root was chosen to avoid
// (CONVENTIONS § Paths and platforms).
func TestScope_Open_WindowsReservedDeviceNames(t *testing.T) {
	for _, name := range []string{"NUL", "COM1"} {
		t.Run(name, func(t *testing.T) {
			repo := newRepo(t, rootFixture)
			if runtime.GOOS != "windows" {
				writeFile(t, repo, "docs/"+name, "an ordinary file\n")
			}
			scope := openScope(t, repo, ".")

			data, err := scope.ReadFile("docs/" + name)
			if runtime.GOOS == "windows" {
				if err == nil {
					t.Fatalf("ReadFile(%q) = %q, want a refusal on Windows", "docs/"+name, data)
				}
				assertRule(t, err, confine.ErrOutsideRoot, "docs/"+name)
				return
			}
			if err != nil {
				t.Fatalf("ReadFile(%q): %v — %s is an ordinary filename on %s", "docs/"+name, err, name, runtime.GOOS)
			}
			if string(data) != "an ordinary file\n" {
				t.Errorf("ReadFile(%q) = %q, want the fixture's contents", "docs/"+name, data)
			}
		})
	}
}

// TestScope_Close_ReleasesTheRoot pins the other half of "opened once at
// startup": the handle is closed on every termination path, and a scope that
// has been closed reads nothing more.
func TestScope_Close_ReleasesTheRoot(t *testing.T) {
	repo := newRepo(t, rootFixture)
	scope, err := confine.OpenScope(repo, []string{"docs"})
	if err != nil {
		t.Fatalf("OpenScope: %v", err)
	}
	if err := scope.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := scope.ReadFile("docs/notes.md"); err == nil {
		t.Error("ReadFile succeeded after Close, want an error")
	}
}

// TestOpenScope_RejectsARepositoryPathThatIsNotADirectory guards the seam the
// CLI's own os.Stat cannot: that earlier check follows symlinks and is a
// classic TOCTOU gap, so the root is opened fresh from repoPath here and
// establishes its own facts rather than inheriting that check's.
func TestOpenScope_RejectsARepositoryPathThatIsNotADirectory(t *testing.T) {
	repo := newRepo(t, rootFixture)

	scope, err := confine.OpenScope(filepath.Join(repo, "README.md"), []string{"."})
	if err == nil {
		_ = scope.Close()
		t.Fatal("OpenScope over a regular file succeeded, want an error")
	}
	if scope != nil {
		t.Error("OpenScope returned a scope alongside its error, want nil")
	}
}

// assertRule checks that err reports the given rule, both to errors.Is — the
// classification CONVENTIONS § Error handling requires — and in its rendered
// message, which is what the model and a human reading stderr actually see.
// It also checks the other two rules are *not* reported, because a refusal
// that claims two rules tells the reader nothing.
func assertRule(t *testing.T, err error, want error, attempted string) {
	t.Helper()

	if !errors.Is(err, want) {
		t.Fatalf("error = %v, want it to report %v", err, want)
	}
	for _, other := range []error{confine.ErrOutsideRoot, confine.ErrOutsideAllowList, confine.ErrDeniedFilename} {
		if other != want && errors.Is(err, other) {
			t.Errorf("error = %v reports %v as well as %v: a refusal names exactly one rule", err, other, want)
		}
	}
	if !strings.Contains(err.Error(), want.Error()) {
		t.Errorf("error = %q, want its message to carry the rule %q", err, want)
	}
	// The quoted form, because that is how the path reaches the message: %q is
	// what keeps a path carrying a newline or a control character from forging
	// a line on stderr, and it is what a caller comparing against the value it
	// sent will see.
	if attempted != "" && !strings.Contains(err.Error(), fmt.Sprintf("%q", attempted)) {
		t.Errorf("error = %q, want it to state what was attempted (%q)", err, attempted)
	}
}
