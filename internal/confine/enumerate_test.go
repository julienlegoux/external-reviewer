package confine_test

import (
	"errors"
	"io/fs"
	"slices"
	"testing"

	"github.com/julienlegoux/external-reviewer/internal/confine"
)

// TestScope_ReadDir_ListsThroughTheRoot covers the primitive enumeration
// walks on. It is the Scope's answer to "what is in this directory", and it
// exists instead of an exported fs.FS over the root: a raw fs.FS reads on the
// strength of the root alone, with neither the allow-list nor the floor in the
// way (see the drift record for issue 02).
func TestScope_ReadDir_ListsThroughTheRoot(t *testing.T) {
	scope := openScope(t, newRepo(t, rootFixture), ".")

	entries, err := scope.ReadDir("docs")
	if err != nil {
		t.Fatalf("ReadDir(%q): %v", "docs", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	slices.Sort(names)
	if want := []string{"epics", "notes.md"}; !slices.Equal(names, want) {
		t.Errorf("ReadDir(%q) listed %q, want %q", "docs", names, want)
	}
}

// TestScope_ReadDir_RefusesDirectoriesTheThreeRulesRefuse is why enumeration
// goes through the Scope rather than around it: the gate answers for a
// directory exactly as it answers for a file, so a walk cannot descend into a
// subtree the invocation never granted, nor into one the floor covers.
func TestScope_ReadDir_RefusesDirectoriesTheThreeRulesRefuse(t *testing.T) {
	root := newRepo(t, map[string]string{
		"docs/notes.md":       "notes\n",
		"internal/cli/run.go": "package cli\n",
		"credentials/aws.md":  "secret\n",
	})

	tests := []struct {
		name    string
		allow   []string
		dir     string
		refusal error
	}{
		{name: "outside the root", allow: []string{"."}, dir: "../outside", refusal: confine.ErrOutsideRoot},
		{name: "outside the allow-list", allow: []string{"docs"}, dir: "internal", refusal: confine.ErrOutsideAllowList},
		{name: "on the floor", allow: []string{"."}, dir: "credentials", refusal: confine.ErrDeniedFilename},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scope := openScope(t, root, test.allow...)

			if _, err := scope.ReadDir(test.dir); !errors.Is(err, test.refusal) {
				t.Errorf("ReadDir(%q) returned %v, want %v", test.dir, err, test.refusal)
			}
		})
	}
}

// TestScope_Stat_ReportsThroughTheRoot covers the other half enumeration
// needs: a path git named is only trusted once the root itself says it is a
// file inside the tree.
func TestScope_Stat_ReportsThroughTheRoot(t *testing.T) {
	scope := openScope(t, newRepo(t, rootFixture), ".")

	info, err := scope.Stat("docs/notes.md")
	if err != nil {
		t.Fatalf("Stat(%q): %v", "docs/notes.md", err)
	}
	if !info.Mode().IsRegular() {
		t.Errorf("Stat(%q) reported mode %v, want a regular file", "docs/notes.md", info.Mode())
	}

	if _, err := scope.Stat("docs/absent.md"); !errors.Is(err, confine.ErrNotFound) {
		t.Errorf("Stat of a missing file returned %v, want %v", err, confine.ErrNotFound)
	}
	if _, err := scope.Stat(".env"); !errors.Is(err, confine.ErrDeniedFilename) {
		t.Errorf("Stat of a floor filename returned %v, want %v", err, confine.ErrDeniedFilename)
	}
}

// TestScope_ReadDir_MissingDirectoryIsNotARefusal keeps the vocabulary
// straight: a granted directory that is simply absent is ErrNotFound, not one
// of the three refusals, because "widen --allow" is the wrong advice for it.
func TestScope_ReadDir_MissingDirectoryIsNotARefusal(t *testing.T) {
	scope := openScope(t, newRepo(t, rootFixture), ".")

	_, err := scope.ReadDir("docs/absent")
	if !errors.Is(err, confine.ErrNotFound) {
		t.Fatalf("ReadDir of a missing directory returned %v, want %v", err, confine.ErrNotFound)
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("ReadDir of a missing directory did not wrap fs.ErrNotExist: %v", err)
	}
}
