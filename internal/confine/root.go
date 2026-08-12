// Package confine owns the one boundary every filesystem read in this binary
// crosses: a single *os.Root over the repository under review, travelling with
// the allow-list of subtrees the invocation granted.
//
// Nothing outside this package opens a file, and no tool holds the *os.Root
// itself. A tool asks the Scope, and the Scope answers in one of five ways:
// the path resolves, or it is refused by one of the three rules
// (ErrOutsideRoot, ErrOutsideAllowList, ErrDeniedFilename), or it is simply
// not there (ErrNotFound), or it is there and unreadable (ErrUnreadable).
// Which rule refused is part of the answer, because "widen --allow" and "that
// file is a credential" are different things for a reviewer to learn.
//
// Only the read half of the filesystem API appears here, which is what makes
// "the binary writes nothing inside the repository under review" a property of
// the code that exists rather than a promise (CONVENTIONS § Code style &
// formatting).
package confine

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
)

// The three rules a read can be refused by, plus the two ways a granted path
// can still fail. A caller classifies with errors.Is and never by matching
// message text (CONVENTIONS § Error handling).
var (
	// ErrOutsideRoot is the boundary os.Root enforces: traversal, an absolute
	// path, a symlink whose target leaves the tree, a Windows reserved device
	// name.
	ErrOutsideRoot = errors.New("path is not inside the repository root")
	// ErrOutsideAllowList is the boundary the invocation drew: the path is
	// inside the repository but no --allow value granted its subtree.
	ErrOutsideAllowList = errors.New("path is outside the allowed subtrees")
	// ErrDeniedFilename is the floor inside the allowed subtrees: a filename
	// that names a credential, whatever --allow said.
	ErrDeniedFilename = errors.New("filename is denied by the sensitive-file floor")
	// ErrNotFound is not a refusal. The path was granted and there is nothing
	// there.
	ErrNotFound = errors.New("no such file in the repository")
	// ErrUnreadable is not a refusal either. The path was granted and exists,
	// and the operating system would not open it.
	ErrUnreadable = errors.New("the file could not be opened")
)

// PathError is what every failed resolution returns: the wire path the caller
// asked for, and the reason, in a form both errors.Is and a human reading
// stderr can use.
//
// Err carries the operating system's own error for errors.Is (fs.ErrNotExist
// and friends) and is deliberately never rendered: the OS's sentence is
// capitalised, full-stopped and spelled differently per platform, and none of
// that belongs on a transcript or in a tool result (CONVENTIONS § Error
// handling, and Epic 1's diagnostics).
type PathError struct {
	// Path is the wire path as the caller spelled it, not its cleaned form:
	// a refusal states what was attempted.
	Path string
	// Reason is one of the six sentinels above.
	Reason error
	// Err is the underlying operating system error, when there was one.
	Err error
}

func (e *PathError) Error() string {
	return fmt.Sprintf("reading %q: %s", e.Path, e.Reason)
}

func (e *PathError) Unwrap() []error {
	if e.Err == nil {
		return []error{e.Reason}
	}
	return []error{e.Reason, e.Err}
}

// Scope is the confinement one review run reads through: the repository's
// single *os.Root and the allow-list, travelling together as one value,
// because a root without the allow-list is not the boundary this product
// sells and an allow-list without the root is a string comparison.
type Scope struct {
	root    *os.Root
	allowed []string
}

// OpenScope opens repoPath as the run's single root and resolves the raw
// --allow values against it. It is called once, at startup, and the Scope is
// closed on every termination path.
//
// The root is opened fresh from repoPath and establishes its own facts. It
// does not build on the os.Stat the CLI already ran over the same argument:
// that Stat follows symlinks, so a symlink to a directory passes IsDir, and
// the gap between it and any later read is a classic TOCTOU window. Every
// read below runs the escape, allow-list and floor resolution on its own
// merits, through this handle.
func OpenScope(repoPath string, allow []string) (*Scope, error) {
	root, err := os.OpenRoot(repoPath)
	if err != nil {
		return nil, fmt.Errorf("opening the repository root: %w", err)
	}

	allowed, err := resolveAllowList(root, allow)
	if err != nil {
		// A scope that cannot be granted leaves no handle behind: the caller
		// gets nil and has nothing to close.
		_ = root.Close()
		return nil, err
	}
	return &Scope{root: root, allowed: allowed}, nil
}

// Close releases the root handle.
func (s *Scope) Close() error { return s.root.Close() }

// Allowed returns the granted subtrees as cleaned wire paths — the value
// git_read turns into pathspecs and the enumeration tools walk. "." means the
// whole repository, and is the only element when it appears. The slice is a
// copy: the allow-list is fixed for the life of the run.
func (s *Scope) Allowed() []string { return slices.Clone(s.allowed) }

// Resolve is the gate. It answers whether a wire path may be read at all, in
// the order the three rules are layered — inside the root, then inside the
// allow-list, then not on the floor — and returns the cleaned wire path when
// it may.
//
// It performs no I/O, so it also answers for paths that do not exist in the
// working tree: git_read validates `show <rev>:<path>` through here before
// building a command, and a path that only exists in history must be
// answerable. The escapes only the kernel can see are therefore *not* decided
// here; they are decided by os.Root when Open actually opens something, and
// come back as the same ErrOutsideRoot.
func (s *Scope) Resolve(wirePath string) (string, error) {
	cleaned, err := cleanWirePath(wirePath)
	if err != nil {
		return "", &PathError{Path: wirePath, Reason: ErrOutsideRoot, Err: err}
	}
	if !s.allows(cleaned) {
		return "", &PathError{Path: wirePath, Reason: ErrOutsideAllowList}
	}
	if deniedByFloor(cleaned) {
		return "", &PathError{Path: wirePath, Reason: ErrDeniedFilename}
	}
	return cleaned, nil
}

// Open resolves a wire path and opens it for reading through the root. The
// returned *os.File is the caller's to close.
func (s *Scope) Open(wirePath string) (*os.File, error) {
	cleaned, err := s.Resolve(wirePath)
	if err != nil {
		return nil, err
	}
	file, err := s.root.Open(filepath.FromSlash(cleaned))
	if err != nil {
		return nil, openError(wirePath, err)
	}
	return file, nil
}

// ReadFile resolves a wire path and reads the whole file through the root.
func (s *Scope) ReadFile(wirePath string) ([]byte, error) {
	cleaned, err := s.Resolve(wirePath)
	if err != nil {
		return nil, err
	}
	data, err := s.root.ReadFile(filepath.FromSlash(cleaned))
	if err != nil {
		return nil, openError(wirePath, err)
	}
	return data, nil
}

// openError classifies what os.Root refused. Only two of its outcomes are
// separable by sentinel — a plain absence and a permission failure — and
// everything else it declines is the one fact it exists to state: this name
// does not denote a file inside the root. A symlink whose target leaves the
// tree, an absolute symlink, a Windows reserved device name and a directory
// swapped for a link between Resolve and the open all arrive here, all as
// "path escapes from parent", with no exported error to tell them apart.
// Reporting them as ErrOutsideRoot is what they are; enumerating them here
// instead would be the hand-rolled path validation os.Root was chosen to
// avoid (CONVENTIONS § Paths and platforms).
func openError(wirePath string, err error) error {
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return &PathError{Path: wirePath, Reason: ErrNotFound, Err: err}
	case errors.Is(err, fs.ErrPermission):
		return &PathError{Path: wirePath, Reason: ErrUnreadable, Err: err}
	default:
		return &PathError{Path: wirePath, Reason: ErrOutsideRoot, Err: err}
	}
}
