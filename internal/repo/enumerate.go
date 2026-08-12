// Package repo owns the one question `list` and `search` both have to answer
// the same way: which files does this repository contain?
//
// The answer is git's — `git ls-files`, so that .gitignore means exactly what
// git means by it and a walk never wanders into node_modules/ and fills the
// reviewer's context with vendored code — falling back to a walk of the
// granted subtrees where there is no git to ask. Either way the file set is
// only what the run's confinement allows: every path either mechanism produces
// is re-resolved through the Scope, so a path git names but the boundary
// refuses is skipped rather than trusted.
package repo

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
	"path"
	"slices"
	"sync"

	"github.com/julienlegoux/external-reviewer/internal/confine"
)

// Enumerator answers "which files are in this repository" for one run. It
// holds the run's confinement and the repository path git is pointed at, plus
// the sink a degraded enumeration announces itself through.
//
// It caches nothing: one review per process with sequential tool execution
// (SPECS § Distribution & operations), so each call asks git again and sees a
// repository that may have changed under it. The one thing it remembers is
// whether it has already said that enumeration is degraded, because a warn
// line per tool call would bury the transcript in the same sentence.
type Enumerator struct {
	scope    *confine.Scope
	repoPath string
	warn     func(message string)
	degraded sync.Once
}

// NewEnumerator wires an enumerator to the run's scope, the repository path,
// and the warn sink (Epic 1's diag.WriteWarn, in a real run).
func NewEnumerator(scope *confine.Scope, repoPath string, warn func(message string)) *Enumerator {
	return &Enumerator{scope: scope, repoPath: repoPath, warn: warn}
}

// Files returns the repository's files as sorted, de-duplicated wire paths,
// every one of them inside the granted subtrees and clear of the floor.
//
// git is asked first and its failure is not fatal: a machine without git, or a
// directory that is not a repository, degrades to a filesystem walk rather
// than ending the review. The degradation is announced, because a review that
// read a vendored dependency tree should be explicable rather than inferred
// from a thin report.
func (e *Enumerator) Files(ctx context.Context) ([]string, error) {
	candidates, err := e.gitFiles(ctx)
	if err != nil {
		e.announceDegraded(err)
		candidates, err = e.walkFiles()
		if err != nil {
			return nil, err
		}
	}
	return e.confined(candidates), nil
}

// gitFiles asks git for the two halves of "not ignored": the tracked files,
// and the untracked ones .gitignore does not cover. Their union is what a
// human sees in the working tree minus what the repository has decided is
// noise.
func (e *Enumerator) gitFiles(ctx context.Context) ([]string, error) {
	tracked, err := e.gitPaths(ctx, "ls-files", "-z")
	if err != nil {
		return nil, err
	}
	untracked, err := e.gitPaths(ctx, "ls-files", "-z", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	return append(tracked, untracked...), nil
}

func (e *Enumerator) gitPaths(ctx context.Context, args ...string) ([]string, error) {
	output, err := runGit(ctx, e.repoPath, args...)
	if err != nil {
		return nil, err
	}
	return splitNUL(output), nil
}

// announceDegraded states, once per run, that the file set is a plain walk
// rather than git's answer, and which of the two things went wrong: git is not
// installed at all, or it declined to enumerate this directory — most often
// because the directory is not a repository. git's own sentence is not quoted,
// since it is capitalised, full-stopped and spelled per platform
// (CONVENTIONS § Error handling); what the line carries is the consequence.
func (e *Enumerator) announceDegraded(cause error) {
	e.degraded.Do(func() {
		if e.warn == nil {
			return
		}
		reason := "git could not enumerate the repository"
		if errors.Is(cause, exec.ErrNotFound) {
			reason = "git is not on PATH"
		}
		e.warn("enumeration is degraded: " + reason +
			", falling back to a filesystem walk that cannot apply .gitignore")
	})
}

// walkFiles is the fallback: a walk of each granted subtree, through the
// Scope. It starts at the allow-list rather than at the repository root
// because those are the only directories this run may read at all — walking
// the root and filtering afterwards would list the names of directories the
// invocation never granted.
func (e *Enumerator) walkFiles() ([]string, error) {
	var files []string
	for _, subtree := range e.scope.Allowed() {
		entries, err := e.scope.ReadDir(subtree)
		if err != nil {
			return nil, fmt.Errorf("listing the granted subtree %q: %w", subtree, err)
		}
		files = e.walk(subtree, entries, files)
	}
	return files, nil
}

// walk descends one directory whose entries have already been listed,
// collecting regular files.
//
// Symlinks are neither followed nor collected: a DirEntry reports the link
// itself, so a link is neither a directory to descend nor a regular file to
// collect, and the walk cannot loop on one or leave the tree through one.
// (git's own enumeration lists a tracked symlink, and confined() drops it for
// the same reason — Scope.Stat follows it and the root refuses what leaves.)
//
// A subdirectory this run cannot list is skipped rather than fatal: the file
// set is what is readable, and one directory the operating system refuses
// should not end a review. The three rules refuse here too, which is what
// prunes `.git/` and a credentials directory without a second copy of the
// floor living in this package.
func (e *Enumerator) walk(dir string, entries []fs.DirEntry, files []string) []string {
	for _, entry := range entries {
		child := path.Join(dir, entry.Name())
		if _, err := e.scope.Resolve(child); err != nil {
			continue
		}
		switch {
		case entry.IsDir():
			children, err := e.scope.ReadDir(child)
			if err != nil {
				continue
			}
			files = e.walk(child, children, files)
		case entry.Type().IsRegular():
			files = append(files, child)
		}
	}
	return files
}

// confined is the funnel both enumeration paths end in, and the place this
// package's central claim is true: nothing is returned that the run's
// confinement has not just answered for. Resolve applies the three rules to
// the spelling, and Scope.Stat makes the root itself confirm that the name
// still denotes a regular file inside the tree — which is how a path git
// tracks but the boundary refuses (a symlink out of the tree, a file deleted
// since the index was written, a submodule) is skipped rather than trusted.
func (e *Enumerator) confined(candidates []string) []string {
	files := make([]string, 0, len(candidates))
	seen := make(map[string]bool, len(candidates))
	for _, candidate := range candidates {
		cleaned, err := e.scope.Resolve(candidate)
		if err != nil || seen[cleaned] {
			continue
		}
		info, err := e.scope.Stat(cleaned)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		seen[cleaned] = true
		files = append(files, cleaned)
	}
	slices.Sort(files)
	return files
}
