package repo_test

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// assertFiles compares an enumeration against the wire paths a test expects,
// in full: enumeration's contract is the *set* of files, so an extra path is
// as wrong as a missing one.
func assertFiles(t *testing.T, got, want []string) {
	t.Helper()

	if !slices.Equal(got, want) {
		t.Errorf("enumeration returned\n\t%q\nwant\n\t%q", got, want)
	}
}

func TestFiles_ReturnsTrackedAndUntrackedFilesButNothingGitignored(t *testing.T) {
	requireGit(t)

	root := newTree(t, map[string]string{
		".gitignore":                "node_modules/\n",
		"docs/tracked.md":           "tracked\n",
		"docs/untracked.md":         "untracked, and not ignored\n",
		"node_modules/pkg/index.js": "vendored\n",
	})
	initGitRepo(t, root, ".gitignore", "docs/tracked.md")

	enumerator, warnings := newEnumerator(t, root, ".")
	files, err := enumerator.Files(t.Context())
	if err != nil {
		t.Fatalf("Files: %v", err)
	}

	assertFiles(t, files, []string{".gitignore", "docs/tracked.md", "docs/untracked.md"})
	if len(*warnings) != 0 {
		t.Errorf("a git repository enumerated with no degradation, yet warned: %q", *warnings)
	}
}

func TestFiles_ReturnsAFilenameGitWouldQuoteIntact(t *testing.T) {
	requireGit(t)

	const quoted = "docs/café.md" // git quotes the non-ASCII byte unless -z is used

	root := newTree(t, map[string]string{quoted: "unusual name\n"})
	initGitRepo(t, root, quoted)
	// core.quotepath is git's default, but the default is not the contract
	// under test: set it explicitly so this asserts -z rather than a machine's
	// configuration.
	runGit(t, root, "config", "core.quotepath", "true")

	enumerator, _ := newEnumerator(t, root, ".")
	files, err := enumerator.Files(t.Context())
	if err != nil {
		t.Fatalf("Files: %v", err)
	}

	assertFiles(t, files, []string{quoted})
}

func TestFiles_IgnoresAHostileMachineLevelGitConfiguration(t *testing.T) {
	requireGit(t)

	root := newTree(t, map[string]string{
		"docs/tracked.md":   "tracked\n",
		"docs/untracked.md": "untracked, and not ignored\n",
	})
	initGitRepo(t, root, "docs/tracked.md")

	// A global configuration that excludes the untracked file from
	// `ls-files --others --exclude-standard`: if the machine's git
	// configuration reached the subprocess, the file would vanish from the
	// enumeration.
	hostile := newTree(t, map[string]string{"ignore": "untracked.md\n"})
	writeFile(t, hostile, "gitconfig", "[core]\n\texcludesFile = "+
		filepath.ToSlash(filepath.Join(hostile, "ignore"))+"\n")
	configPath := filepath.Join(hostile, "gitconfig")
	t.Setenv("GIT_CONFIG_GLOBAL", configPath)
	t.Setenv("GIT_CONFIG_SYSTEM", configPath)

	enumerator, _ := newEnumerator(t, root, ".")
	files, err := enumerator.Files(t.Context())
	if err != nil {
		t.Fatalf("Files: %v", err)
	}

	assertFiles(t, files, []string{"docs/tracked.md", "docs/untracked.md"})
}

func TestFiles_OmitsPathsOutsideTheAllowedSubtrees(t *testing.T) {
	requireGit(t)

	root := newTree(t, map[string]string{
		"docs/tracked.md":   "granted\n",
		"internal/main.go":  "not granted\n",
		"README.md":         "not granted either\n",
		"docs/deep/note.md": "granted, deeper\n",
	})
	initGitRepo(t, root, "docs/tracked.md", "internal/main.go", "README.md", "docs/deep/note.md")

	enumerator, _ := newEnumerator(t, root, "docs")
	files, err := enumerator.Files(t.Context())
	if err != nil {
		t.Fatalf("Files: %v", err)
	}

	assertFiles(t, files, []string{"docs/deep/note.md", "docs/tracked.md"})
}

func TestFiles_OmitsPathsOnTheSensitiveFileFloor(t *testing.T) {
	requireGit(t)

	root := newTree(t, map[string]string{
		".env":               "SECRET=1\n",
		"deploy.key":         "-----BEGIN\n",
		"docs/server.pem":    "-----BEGIN\n",
		"credentials.json":   "{}\n",
		"docs/readable.md":   "ordinary\n",
		"config/.npmrc":      "//registry:_authToken=x\n",
		"config/settings.md": "ordinary\n",
	})
	initGitRepo(t, root, ".env", "deploy.key", "docs/server.pem", "credentials.json",
		"docs/readable.md", "config/.npmrc", "config/settings.md")

	enumerator, _ := newEnumerator(t, root, ".")
	files, err := enumerator.Files(t.Context())
	if err != nil {
		t.Fatalf("Files: %v", err)
	}

	assertFiles(t, files, []string{"config/settings.md", "docs/readable.md"})
	for _, file := range files {
		if strings.Contains(file, ".git/") {
			t.Errorf("enumeration returned a path under .git/: %q", file)
		}
	}
}

func TestFiles_ReturnsWirePathsWhicheverOSRuns(t *testing.T) {
	requireGit(t)

	root := newTree(t, map[string]string{"docs/deep/nested/note.md": "nested\n"})
	initGitRepo(t, root, "docs/deep/nested/note.md")

	enumerator, _ := newEnumerator(t, root, ".")
	files, err := enumerator.Files(t.Context())
	if err != nil {
		t.Fatalf("Files: %v", err)
	}

	assertFiles(t, files, []string{"docs/deep/nested/note.md"})
	for _, file := range files {
		if strings.ContainsRune(file, '\\') || filepath.IsAbs(file) {
			t.Errorf("enumeration returned %q, which is not a repo-relative wire path", file)
		}
	}
}
