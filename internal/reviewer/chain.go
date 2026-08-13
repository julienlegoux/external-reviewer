package reviewer

import (
	"context"

	"github.com/julienlegoux/kern-link/ai"

	"github.com/julienlegoux/external-reviewer/internal/family"
)

// Chain is the whole of reviewer selection: a tier name goes in, and either a
// reachable, credentialed, allowed model comes out or the run ends before a
// single request is sent. It is [Assigner] — the precedence chain specs 06
// fixes — in front of [Resolver] — the pre-flight SPECS orders — and it is
// what the CLI calls (issues 05, 06 and 07); nothing above it needs to know
// that selection has two halves.
//
// The two failure classes come out of it exactly as [Resolver] produces them,
// unflattened: ErrNoReviewer (exit 1) as a *NoReviewerError naming the rule
// that refused, and an error wrapping *ai.ModelsError for a credential that
// exists and is broken (exit 2). A value that is not provider/id is a third
// thing — a *MalformedAssignmentError, a mistake to report rather than a tier
// to skip.
type Chain struct {
	// Models is the registry resolution runs against.
	Models ai.Models
	// Assigner decides which provider and model a tier means.
	Assigner Assigner
	// Exclusion is the family rule; nil is the product default, Anthropic
	// excluded.
	Exclusion *family.Exclusion
	// Warn receives non-fatal diagnostics — currently only a failed catalog
	// refresh. Optional; a nil Warn discards them.
	Warn func(message string)
}

// ResolveTier runs both halves for tier. An explicit assignment bypasses
// tiers entirely, so a Chain whose Assigner carries one resolves the same way
// for any tier, including the empty string.
//
// A successful Resolution carries the assignment that produced it, because
// "why did this resolve to that?" is not answerable after the fact — the
// three precedence layers all yield a bare provider and id.
func (c Chain) ResolveTier(ctx context.Context, tier string) (Resolution, error) {
	assignment, err := c.Assigner.Assign(tier)
	if err != nil {
		return Resolution{}, err
	}

	resolution, err := Resolver{
		Models:     c.Models,
		ProviderID: assignment.Provider,
		ModelID:    assignment.Model,
		Exclusion:  c.Exclusion,
		Warn:       c.Warn,
	}.Resolve(ctx)
	if err != nil {
		return Resolution{}, err
	}

	resolution.Assignment = assignment
	return resolution, nil
}
