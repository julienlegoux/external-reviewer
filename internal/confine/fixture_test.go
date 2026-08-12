package confine_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/julienlegoux/external-reviewer/internal/confine"
)

// newRepo builds a throwaway repository whose contents are given as wire
// paths, so a test's fixture reads the same on both matrix OSes. The write
// half of the filesystem API is legitimate here and nowhere else: fixtures
// are built with t.TempDir() (CONVENTIONS § Code style & formatting).
func newRepo(t *testing.T, files map[string]string) string {
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

	path := filepath.Join(root, filepath.FromSlash(wireName))
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("creating fixture directory for %s: %v", wireName, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writing fixture %s: %v", wireName, err)
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
