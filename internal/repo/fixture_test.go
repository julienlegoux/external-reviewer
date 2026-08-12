package repo_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/julienlegoux/external-reviewer/internal/confine"
	"github.com/julienlegoux/external-reviewer/internal/repo"
)

// newTree builds a throwaway repository whose contents are given as wire
// paths, so a test's fixture reads the same on both matrix OSes. The write
// half of the filesystem API is legitimate here and nowhere else: fixtures
// are built with t.TempDir() (CONVENTIONS § Code style & formatting).
func newTree(t *testing.T, files map[string]string) string {
	t.Helper()

	root := t.TempDir()
	for name, content := range files {
		writeFile(t, root, name, content)
	}
	return root
}

// writeFile creates one fixture file under root, named by its wire path,
// making the parent directories it needs.
func writeFile(t *testing.T, root, wireName, content string) {
	t.Helper()

	name := filepath.Join(root, filepath.FromSlash(wireName))
	if err := os.MkdirAll(filepath.Dir(name), 0o750); err != nil {
		t.Fatalf("creating fixture directory for %s: %v", wireName, err)
	}
	if err := os.WriteFile(name, []byte(content), 0o600); err != nil {
		t.Fatalf("writing fixture %s: %v", wireName, err)
	}
}

// requireGit skips the calling test at runtime when git is absent from PATH,
// naming it — never a build tag (CONVENTIONS § Testing, and this issue's
// acceptance criteria). Enumeration's whole gitignore-exact half is git's
// answer, so a machine without git can only test the fallback.
func requireGit(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git is not on PATH: %v", err)
	}
}

// initGitRepo turns dir into a git repository and stages the named wire
// paths, so that `git ls-files` reports exactly them as tracked. Staging is
// enough: the index is what ls-files reads, so no commit — and therefore no
// user identity — is needed.
func initGitRepo(t *testing.T, dir string, track ...string) {
	t.Helper()

	runGit(t, dir, "init")
	if len(track) > 0 {
		runGit(t, dir, append([]string{"add", "--"}, track...)...)
	}
}

// runGit runs one git command in dir and fails the test with its output when
// git refuses. Tests are exempt from the exec guard the implementation
// carries; a fixture that shells out to git is the only way to assert against
// real gitignore semantics.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...) //nolint:forbidigo // test fixture
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, output)
	}
}

// openScope opens a scope over root that the test does not expect to fail,
// closed when the test ends.
func openScope(t *testing.T, root string, allow ...string) *confine.Scope {
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
	return scope
}

// newEnumerator wires an enumerator over root with a warn sink the test can
// read back, since "enumeration was degraded" is a diagnostic the run has to
// state rather than a value it returns.
func newEnumerator(t *testing.T, root string, allow ...string) (*repo.Enumerator, *[]string) {
	t.Helper()

	warnings := new([]string)
	enumerator := repo.NewEnumerator(openScope(t, root, allow...), root, func(message string) {
		*warnings = append(*warnings, message)
	})
	return enumerator, warnings
}
