package repo_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/julienlegoux/external-reviewer/internal/repo"
)

// The reviewed repository's own configuration is untrusted input. Every git
// this binary runs runs with that repository's local configuration in effect,
// and several read-only subcommands run a configured helper program as a
// normal part of their work — so a repository can execute code on the
// reviewer's machine without any argument this binary builds ever being wrong
// (issue #58).
//
// Neutralising GIT_CONFIG_GLOBAL and GIT_CONFIG_SYSTEM does not touch this:
// those cover the machine's configuration, and the third scope — the
// repository's own .git/config — is the one the attacker controls.

// hostileVectors is one entry per way a repository can name a program git will
// run during a read. Each configures the fixture and is expected to fire
// without the hardening and never with it; the two tests below assert those
// two halves against the same table, because a fixture that stopped biting
// would otherwise turn this whole file green while proving nothing.
var hostileVectors = []struct {
	name      string
	configure func(t *testing.T, root, payload string)
}{
	{
		// Fires on diff, log -p and show, via .gitattributes assigning the
		// driver. The driver name is the repository's to choose, so no
		// pre-written -c override can name it.
		name: "diff.<driver>.textconv",
		configure: func(t *testing.T, root, payload string) {
			runGit(t, root, "config", "diff.hostile.textconv", payload)
		},
	},
	{
		// An external diff driver selected by attribute. Distinct from
		// diff.external, and likewise attacker-named.
		name: "diff.<driver>.command",
		configure: func(t *testing.T, root, payload string) {
			runGit(t, root, "config", "diff.hostile.command", payload)
		},
	},
	{
		name: "diff.external",
		configure: func(t *testing.T, root, payload string) {
			runGit(t, root, "config", "diff.external", payload)
		},
	},
	{
		// The enumeration vector: fsmonitor is consulted by status and by the
		// two ls-files calls internal/repo makes to answer "which files does
		// this repository contain".
		name: "core.fsmonitor",
		configure: func(t *testing.T, root, payload string) {
			runGit(t, root, "config", "core.fsmonitor", payload)
		},
	},
	{
		// A clean filter converts a modified working-tree file before it is
		// compared, so a worktree diff runs it. No diff flag disables filters,
		// which is why the fixture below leaves a.txt dirty.
		name: "filter.<driver>.clean",
		configure: func(t *testing.T, root, payload string) {
			runGit(t, root, "config", "filter.hostile.clean", payload)
		},
	},
	{
		// Not configuration at all: the hook is a file inside the repository
		// directory, so no -c override reaches it, and blanking core.hooksPath
		// only falls back to this same .git/hooks. `git status` refreshes and
		// writes the index, which runs it.
		name: "the post-index-change hook",
		configure: func(t *testing.T, root, payload string) {
			t.Helper()

			hook := filepath.Join(root, ".git", "hooks", "post-index-change")
			if err := os.WriteFile(hook, []byte("#!/bin/sh\n"+payload+"\n"), 0o700); err != nil { //nolint:gosec // a fixture's hook has to be executable to be a hook
				t.Fatalf("writing the fixture hook: %v", err)
			}
		},
	},
}

// hostileReads are the reads this binary performs against a repository it was
// pointed at: git_read's four allowed subcommands, in the shapes that reach a
// diff, plus the enumeration call. `diff` with no revision is listed
// separately from the two-commit form because only it compares the working
// tree, which is where a clean filter runs.
var hostileReads = [][]string{
	{"diff", "HEAD~1", "HEAD"},
	{"diff"},
	{"log", "-p"},
	{"show", "HEAD"},
	{"status"},
	{"ls-files", "-z"},
}

// Every read below gets a repository of its own rather than sharing one per
// vector. Reads are not independent when they share a repository: `git diff`
// refreshes and rewrites the index, so a later `git status` finds nothing to
// write and the post-index-change hook that would have fired does not. Batched,
// that made both tests flaky in the same direction — the dangerous thing
// silently stopped happening — which is the one failure mode a security test
// must not have.

func TestAHostileRepositoryRunsNoProgramItsOwnConfigurationNames(t *testing.T) {
	requireGit(t)

	forEachHostileRead(t, func(t *testing.T, root, marker string, args []string) {
		// A read that fails is not this test's concern: the assertion is that
		// nothing ran, not that everything succeeded.
		_, _ = repo.Git(t.Context(), root, args...)

		if markerWritten(t, marker) {
			t.Errorf("the repository ran a program of its own choosing during git %v", args)
		}
	})
}

// TestAHostileRepositoryRunsNoProgramItsOwnConfigurationNamesDuringEnumeration
// covers the other caller. Enumeration runs ls-files against the same hostile
// repository, so a hardening that only covered git_read would leave the file
// listing — which every review begins with — open.
func TestAHostileRepositoryRunsNoProgramItsOwnConfigurationNamesDuringEnumeration(t *testing.T) {
	requireGit(t)

	for _, vector := range hostileVectors {
		t.Run(vector.name, func(t *testing.T) {
			root, marker := hostileRepository(t)
			vector.configure(t, root, markerPayload(marker))

			enumerator, _ := newEnumerator(t, root, ".")
			if _, err := enumerator.Files(t.Context()); err != nil {
				t.Fatalf("enumerating the fixture: %v", err)
			}

			if markerWritten(t, marker) {
				t.Errorf("the repository's %s ran a program of its choosing during enumeration", vector.name)
			}
		})
	}
}

// TestTheHostileFixtureRunsThatProgramWithoutTheHardening is the negative
// control, and it is not optional. Each payload fires only under conditions
// the fixture has to get right — a commit touching the file the attribute
// selects, a modified working tree for the clean filter, an unrefreshed index
// for the hook — and a fixture that quietly stopped meeting them would leave
// the tests above passing against a repository that was never dangerous.
//
// It asserts per vector rather than per read, because no single vector fires
// on every read: textconv does not reach status, fsmonitor does not reach a
// two-commit diff. What must hold is that each vector still fires somewhere.
func TestTheHostileFixtureRunsThatProgramWithoutTheHardening(t *testing.T) {
	requireGit(t)

	for _, vector := range hostileVectors {
		t.Run(vector.name, func(t *testing.T) {
			fired := false
			for _, args := range hostileReads {
				root, marker := hostileRepository(t)
				vector.configure(t, root, markerPayload(marker))

				// Deliberately not repo.Git: this is the bare invocation, the
				// one this binary made before the hardening.
				cmd := exec.Command("git", append([]string{"-C", root}, args...)...) //nolint:forbidigo,gosec // the control has to be the unhardened invocation; argv only, no shell
				_ = cmd.Run()

				if markerWritten(t, marker) {
					fired = true
				}
			}

			if !fired {
				t.Errorf("the fixture is not dangerous: %s never ran on any read, so the tests above prove nothing", vector.name)
			}
		})
	}
}

// forEachHostileRead runs assert against a repository built fresh for every
// (vector, read) pair, so that neither the vector's payload nor the read's own
// side effects can carry into the next case.
func forEachHostileRead(t *testing.T, assert func(t *testing.T, root, marker string, args []string)) {
	t.Helper()

	for _, vector := range hostileVectors {
		t.Run(vector.name, func(t *testing.T) {
			for _, args := range hostileReads {
				t.Run(strings.Join(args, " "), func(t *testing.T) {
					root, marker := hostileRepository(t)
					vector.configure(t, root, markerPayload(marker))

					assert(t, root, marker, args)
				})
			}
		})
	}
}

// hostileRepository builds a repository carrying everything a payload needs to
// fire, and returns it with the path its payload would write.
//
// Three properties matter and none is incidental: .gitattributes selects both
// a diff driver and a filter driver for a.txt, HEAD touches a.txt so a
// two-commit diff reaches the attribute, and the working tree is left modified
// so a clean filter has something to convert. The marker lives outside the
// repository, since a file inside it would change what enumeration reports.
func hostileRepository(t *testing.T) (root, marker string) {
	t.Helper()

	root = newTree(t, map[string]string{
		".gitattributes": "*.txt diff=hostile filter=hostile\n",
		"a.txt":          "one\n",
	})
	runGit(t, root, "init")
	commitEverything(t, root, "first")
	writeFile(t, root, "a.txt", "two\n")
	commitEverything(t, root, "second")
	writeFile(t, root, "a.txt", "modified, uncommitted\n")

	return root, filepath.Join(t.TempDir(), "marker")
}

// commitEverything commits the fixture's whole working tree under an identity
// and signing policy of its own, so the test does not depend on — or fail
// because of — whatever the developer's machine configures globally.
func commitEverything(t *testing.T, root, message string) {
	t.Helper()

	runGit(t, root, "add", "-A")
	runGit(t, root,
		"-c", "user.name=fixture",
		"-c", "user.email=fixture@example.invalid",
		"-c", "commit.gpgsign=false",
		"commit", "--quiet", "--message", message)
}

// markerPayload is the stand-in for an attacker's program: shell redirection
// and nothing else, so it needs no command on PATH and runs identically on
// both matrix OSes — git runs a configured helper through its own shell on
// Windows too. The path is spelled with forward slashes because it is being
// read by that shell rather than by the OS.
func markerPayload(marker string) string {
	return fmt.Sprintf("echo pwned > %q", filepath.ToSlash(marker))
}

func markerWritten(t *testing.T, marker string) bool {
	t.Helper()

	_, err := os.Stat(marker)
	return err == nil
}
