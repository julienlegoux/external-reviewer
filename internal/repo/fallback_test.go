package repo_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/julienlegoux/external-reviewer/internal/diag"
	"github.com/julienlegoux/external-reviewer/internal/repo"
)

// nonRepository is a tree that is not a git repository, `.git` directory and
// all: a stray `.git/config` is exactly the file a walk must not hand to a
// reviewer, since it carries remote URLs and credential-helper configuration.
var nonRepository = map[string]string{
	"docs/notes.md":       "notes\n",
	"docs/deep/nested.md": "deeper\n",
	"README.md":           "# fixture\n",
	".git/config":         "[remote \"origin\"]\n\turl = https://token@example.invalid/x.git\n",
	".env":                "SECRET=1\n",
}

func TestFiles_OutsideAGitRepositoryWalksTheFilesystemInstead(t *testing.T) {
	root := newTree(t, nonRepository)

	enumerator, warnings := newEnumerator(t, root, ".")
	files, err := enumerator.Files(t.Context())
	if err != nil {
		t.Fatalf("Files: %v", err)
	}

	assertFiles(t, files, []string{"README.md", "docs/deep/nested.md", "docs/notes.md"})
	if len(*warnings) != 1 {
		t.Fatalf("a degraded enumeration emitted %d warnings, want exactly 1: %q", len(*warnings), *warnings)
	}
	if !strings.Contains((*warnings)[0], "degraded") {
		t.Errorf("the degradation warning does not say the enumeration is degraded: %q", (*warnings)[0])
	}
}

// TestFiles_DegradationIsAnnouncedOncePerRun keeps the transcript readable:
// enumeration is not cached, so every tool call runs it again, and a warn line
// per call would bury the run in one repeated sentence.
func TestFiles_DegradationIsAnnouncedOncePerRun(t *testing.T) {
	root := newTree(t, nonRepository)

	enumerator, warnings := newEnumerator(t, root, ".")
	for range 3 {
		if _, err := enumerator.Files(t.Context()); err != nil {
			t.Fatalf("Files: %v", err)
		}
	}

	if len(*warnings) != 1 {
		t.Errorf("three degraded enumerations emitted %d warnings, want exactly 1: %q", len(*warnings), *warnings)
	}
}

// TestFiles_DegradationReachesStderrAsAWarnLine wires the sink to the real
// diagnostic writer, because the criterion is about what a reader of the run
// sees, not about a callback having fired.
func TestFiles_DegradationReachesStderrAsAWarnLine(t *testing.T) {
	root := newTree(t, nonRepository)

	var stderr bytes.Buffer
	enumerator := repo.NewEnumerator(openScope(t, root, "."), root, func(message string) {
		diag.WriteWarn(&stderr, message)
	})
	if _, err := enumerator.Files(t.Context()); err != nil {
		t.Fatalf("Files: %v", err)
	}

	lines := strings.Split(strings.TrimSuffix(stderr.String(), "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("stderr carried %d lines, want exactly 1:\n%s", len(lines), stderr.String())
	}
	if !strings.HasPrefix(lines[0], "warn ") || !strings.Contains(lines[0], "degraded") {
		t.Errorf("stderr line %q is not a warn line saying enumeration is degraded", lines[0])
	}
}

// TestFiles_TheFallbackWalkStaysInsideTheAllowedSubtrees is the fallback's
// half of the boundary: the walk starts at the granted subtrees rather than at
// the repository root, so a directory the invocation never granted is never
// listed, let alone read.
func TestFiles_TheFallbackWalkStaysInsideTheAllowedSubtrees(t *testing.T) {
	root := newTree(t, map[string]string{
		"docs/notes.md":       "granted\n",
		"docs/deep/nested.md": "granted, deeper\n",
		"internal/run.go":     "package main\n",
		"secrets/.env":        "SECRET=1\n",
	})

	enumerator, _ := newEnumerator(t, root, "docs")
	files, err := enumerator.Files(t.Context())
	if err != nil {
		t.Fatalf("Files: %v", err)
	}

	assertFiles(t, files, []string{"docs/deep/nested.md", "docs/notes.md"})
}
