package confine_test

import (
	"errors"
	"testing"

	"github.com/julienlegoux/external-reviewer/internal/confine"
)

// TestScope_Resolve_RefusesTheSensitiveFileFloor is the floor's whole point:
// it applies *inside* an allowed subtree, so `--allow .` — the grant that has
// to be said and grants everything — still cannot read a credential. Every
// case here is refused as a denied filename and not as an allow-list miss;
// telling those two apart is what lets a reviewer know whether widening
// --allow would have helped (CONVENTIONS § Error handling, and specs 10's
// "a refusal message names which rule refused").
func TestScope_Resolve_RefusesTheSensitiveFileFloor(t *testing.T) {
	denied := []string{
		".env",
		".env.local",
		"server.pem",
		"id_rsa",
		"deploy.key",
		".npmrc",
		".netrc",
		"credentials.json",
		".git/config",
		"docs/.env",
		"docs/id_rsa.pub",
		"docs/client.p12",
		"docs/client.pfx",
		"docs/credentials/aws.txt",
		".git/refs/heads/main",
	}

	repo := newRepo(t, map[string]string{"docs/notes.md": "notes\n"})
	for _, name := range denied {
		writeFile(t, repo, name, "a secret\n")
	}
	scope := openScope(t, repo, ".")

	for _, name := range denied {
		t.Run(name, func(t *testing.T) {
			got, err := scope.Resolve(name)
			if err == nil {
				t.Fatalf("Resolve(%q) = %q, want a refusal", name, got)
			}
			assertRule(t, err, confine.ErrDeniedFilename, name)
			if _, err := scope.ReadFile(name); !errors.Is(err, confine.ErrDeniedFilename) {
				t.Errorf("ReadFile(%q) = %v, want the same refusal Resolve gives", name, err)
			}
		})
	}
}

// TestScope_Resolve_FloorIsCaseInsensitive keeps the floor from depending on
// which platform ran the binary: Windows would refuse .ENV whatever this code
// does, because its filesystem folds case, so folding here is what makes the
// two OSes answer the same way. Over-refusing a file called `NOTES.KEY` on
// Linux is the safe direction of that trade; a credential readable on one
// platform and not the other is not.
func TestScope_Resolve_FloorIsCaseInsensitive(t *testing.T) {
	repo := newRepo(t, map[string]string{"docs/notes.md": "notes\n"})
	scope := openScope(t, repo, ".")

	for _, name := range []string{".ENV", "SERVER.PEM", "ID_RSA", "Credentials.JSON", ".GIT/config"} {
		t.Run(name, func(t *testing.T) {
			if _, err := scope.Resolve(name); !errors.Is(err, confine.ErrDeniedFilename) {
				t.Errorf("Resolve(%q) = %v, want ErrDeniedFilename", name, err)
			}
		})
	}
}

// TestScope_Resolve_FloorLeavesOrdinaryFilenamesAlone is the other side of a
// deny-list: a floor that swallows ordinary source files would blind the
// reviewer without saying so. These are the near misses.
func TestScope_Resolve_FloorLeavesOrdinaryFilenamesAlone(t *testing.T) {
	allowed := []string{
		"docs/notes.md",
		"docs/environment.md",
		"docs/keychain.go",
		"docs/pem.md",
		"docs/gitignore.md",
		"internal/keys/store.go",
		"docs/env.example",
	}

	repo := newRepo(t, map[string]string{})
	for _, name := range allowed {
		writeFile(t, repo, name, "ordinary\n")
	}
	scope := openScope(t, repo, ".")

	for _, name := range allowed {
		t.Run(name, func(t *testing.T) {
			got, err := scope.Resolve(name)
			if err != nil {
				t.Fatalf("Resolve(%q): %v", name, err)
			}
			if got != name {
				t.Errorf("Resolve(%q) = %q, want it unchanged", name, got)
			}
		})
	}
}
