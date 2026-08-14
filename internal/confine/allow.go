package confine

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// wholeRepository is the one allow-list value that grants everything. It has
// to be said (specs 10 — Repository confinement): a run with no --allow at all
// is a usage error, never an implicit run over the whole tree.
const wholeRepository = "."

// cleanWirePath validates a wire path — repo-relative and /-separated on every
// platform (CONVENTIONS § Paths and platforms) — and returns its cleaned form.
//
// This is a check on the *vocabulary*, not a second implementation of
// confinement: it rejects the spellings that are not wire paths at all, so
// that what reaches os.Root is always a relative, slash-separated name and
// the two matrix OSes answer identically. `C:\Windows\win.ini` is an absolute
// path on Windows and an ordinary filename on Linux; refusing it here is what
// stops that difference from reaching the boundary. The escapes only the
// kernel can see — symlinks out of the tree, Windows reserved device names —
// stay entirely with *os.Root, which is why they are not enumerated here.
func cleanWirePath(wirePath string) (string, error) {
	switch {
	case wirePath == "":
		return "", errors.New("the path is empty")
	case strings.ContainsRune(wirePath, '\\'):
		return "", errors.New("the path uses a native separator rather than a wire path's /")
	case strings.HasPrefix(wirePath, "/"):
		return "", errors.New("the path is absolute")
	case len(wirePath) >= 2 && wirePath[1] == ':':
		// A drive-letter prefix: `C:\Windows` and the drive-relative `C:x`,
		// both of which name something outside the root on Windows and
		// nothing meaningful on Linux.
		return "", errors.New("the path names a drive")
	}

	cleaned := path.Clean(wirePath)
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", errors.New("the path traverses out of the repository")
	}
	return cleaned, nil
}

// resolveAllowList turns the raw --allow values into the cleaned wire paths
// the resolver tests membership against, checking each one names an existing
// directory inside root.
//
// filepath.ToSlash is applied first, and only here: --allow is the one place
// an OS path legitimately arrives, because a human types it at their own
// shell — `--allow docs\epics` on Windows means the same subtree as
// `--allow docs/epics`. On Linux the call is a no-op and a backslash stays the
// ordinary filename character it is there. Paths that arrive from the model
// get no such conversion (see cleanWirePath).
func resolveAllowList(root *os.Root, allow []string) ([]string, error) {
	if len(allow) == 0 {
		return nil, errors.New("at least one --allow <path> is required: nothing is readable until a subtree is granted")
	}

	resolved := make([]string, 0, len(allow))
	for _, raw := range allow {
		cleaned, err := cleanWirePath(filepath.ToSlash(raw))
		if err != nil {
			return nil, fmt.Errorf("--allow %q cannot be granted: %w", raw, err)
		}
		info, err := root.Stat(filepath.FromSlash(cleaned))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil, fmt.Errorf("--allow %q cannot be granted: no such directory in the repository", raw)
			}
			// os.Root declined for a reason only the kernel can see — an
			// escaping symlink, a reserved device name — which is the same
			// answer a read of that path would get.
			return nil, fmt.Errorf("--allow %q cannot be granted: %w", raw, ErrOutsideRoot)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("--allow %q cannot be granted: it names a file, and a grant is a subtree", raw)
		}
		resolved = append(resolved, cleaned)
	}
	return normaliseAllowList(resolved), nil
}

// normaliseAllowList removes duplicates and collapses the list to "." when the
// whole repository was granted, so every downstream consumer — the resolver's
// subtree test, git_read's pathspecs, search's path parameter — reads one
// canonical list rather than re-deriving it.
func normaliseAllowList(resolved []string) []string {
	unique := make([]string, 0, len(resolved))
	seen := make(map[string]bool, len(resolved))
	for _, value := range resolved {
		if value == wholeRepository {
			return []string{wholeRepository}
		}
		if seen[value] {
			continue
		}
		seen[value] = true
		unique = append(unique, value)
	}
	return unique
}

// allows reports whether a cleaned wire path lies inside the union of the
// granted subtrees. A subtree grants itself and everything beneath it, and
// nothing else: `docs` grants `docs` and `docs/notes.md`, and never
// `docsignored/notes.md`.
func (s *Scope) allows(cleaned string) bool {
	for _, subtree := range s.allowed {
		switch {
		case subtree == wholeRepository:
			return true
		case cleaned == subtree:
			return true
		case strings.HasPrefix(cleaned, subtree+"/"):
			return true
		}
	}
	return false
}
