package tools

import (
	"github.com/julienlegoux/external-reviewer/internal/confine"
)

// Deps is everything a read-only tool in this package can need from the run
// it belongs to, gathered once at the call site that starts a review.
//
// It exists so that adding a tool is adding a file, not changing a signature.
// A tool constructor takes whichever of these fields it needs; NewRunRegistry
// below is the single place they are supplied, and the loop
// (internal/reviewer) never sees this type at all.
type Deps struct {
	// Scope is the run's confinement: the repository's single *os.Root and
	// the allow-list, travelling as one value. Every path a tool touches
	// resolves through it, and there is no other way to reach a file.
	Scope *confine.Scope
	// RepoPath is the repository's own OS path, for the one tool that has to
	// point a subprocess at it (git_read, issue 06) rather than open a file.
	RepoPath string
	// Files is the enumeration `list` and `search` both answer "what is
	// there" from, so what the reviewer can glob and what it can grep never
	// disagree.
	Files Enumerator
	// Warn is the run's non-fatal diagnostic sink (diag.WriteWarn on stderr,
	// in a real run), for a tool that degrades rather than fails.
	Warn func(message string)
}

// NewRunRegistry assembles the tool set one review offers, in the order the
// model is told about them.
//
// **This is the single registration point.** A new tool is a new file in this
// package exporting a constructor, plus one line below. Nothing in
// internal/reviewer and nothing in internal/cli changes for it: the loop takes
// a ToolSet interface this *Registry satisfies, and the CLI builds Deps once.
func NewRunRegistry(deps Deps) *Registry {
	return NewRegistry(
		List(deps.Files),
		// read_file — issue 04 registers it here.
		Search(deps.Scope, deps.Files),
		// git_read — issue 06 registers it here.
	)
}
