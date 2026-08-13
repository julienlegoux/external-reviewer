package reviewer

import (
	"fmt"
	"slices"
	"strings"

	"github.com/julienlegoux/external-reviewer/internal/config"
)

// TierEnvPrefix is the stem of the per-tier override variables:
// EXTERNAL_REVIEWER_TIER_LIGHT, …_STANDARD, …_HEAVY, each holding one
// provider/id (specs 06). One of them overrides that tier's table and leaves
// the other tiers alone, which is what makes "try this model once" a shell
// prefix rather than an edit to a file that lives on the machine.
const TierEnvPrefix = "EXTERNAL_REVIEWER_TIER_"

// Source is the layer of the precedence chain an assignment came from. It
// exists because "why did this tier resolve to that model?" is the question
// setup actually asks, and the answer is not derivable after the fact — the
// three layers all yield the same provider and id.
type Source string

const (
	// SourceFlag is an explicit provider/id supplied by the caller, which
	// bypasses tiers entirely. The CLI carries it in --model (issue 05).
	SourceFlag Source = "flag"
	// SourceEnvironment is a EXTERNAL_REVIEWER_TIER_<TIER> variable.
	SourceEnvironment Source = "environment"
	// SourceConfig is the tier's table in the machine-local config file.
	SourceConfig Source = "config"
)

// Assignment is one tier's provider and model together with where they came
// from: kern-link's own ProviderId and Model.ID verbatim, with no aliasing or
// translation layer (specs 05).
type Assignment struct {
	Provider string
	Model    string
	// Source is the layer that supplied this assignment.
	Source Source
	// Origin names the exact knob it came from — the environment variable's
	// name, or the config file's path. It is empty for SourceFlag, where the
	// caller already knows which flag it passed.
	Origin string
}

// String renders the assignment as the provider/id pair a human typed.
func (a Assignment) String() string { return a.Provider + "/" + a.Model }

// MalformedAssignmentError is a provider/id a human wrote and got wrong. It
// is deliberately not ErrNoReviewer: an unparseable value is a mistake to
// report, not a tier to skip, and classifying it as "nothing was assigned"
// would send the run down the silent native-fallback path with the one
// diagnostic that explains it thrown away (specs 13).
type MalformedAssignmentError struct {
	// Source is the layer the value came from.
	Source Source
	// Origin names the variable or file, when the layer has one.
	Origin string
	// Value is what could not be parsed, verbatim.
	Value string
}

func (e *MalformedAssignmentError) Error() string {
	where := e.Origin
	if where == "" {
		where = string(e.Source)
	}
	return fmt.Sprintf("malformed model assignment %q from %s: want provider/id, with both halves non-empty", e.Value, where)
}

// Assigner is the assignment half of the resolution chain: a tier name in,
// a provider and model out, decided by the three layers specs 06 fixes,
// highest first. It knows nothing about catalogs, credentials or families —
// whether the model it names can actually be reached is [Resolver]'s question.
//
// The zero Assigner assigns nothing, which is the honest answer for a machine
// nobody has set up.
type Assigner struct {
	// Explicit is a provider/id supplied by the caller — what --model will
	// carry. It bypasses tiers entirely, for every tier. Empty means absent.
	Explicit string
	// Getenv is how the environment layer is read. It is injected rather
	// than called as os.Getenv here so the chain is testable without
	// t.Setenv ordering games — and so a nil Getenv means *no environment*
	// rather than the developer's own, which keeps a test that forgets to
	// script one deterministic instead of quietly machine-dependent.
	Getenv func(name string) string
	// Config is the decoded tier assignment file, or nil when there is none.
	// An absent file is not an error: it is the default state before a human
	// has set anything up (specs 05).
	Config *config.Config
}

// Assign resolves tier through the chain, highest layer first:
//
//  1. Explicit — bypasses tiers entirely;
//  2. EXTERNAL_REVIEWER_TIER_<TIER> — overrides that one tier;
//  3. the tier's table in Config.
//
// A tier outside the fixed vocabulary ([config.Tiers]) consults neither the
// environment nor the file. There is no mechanism to extend the vocabulary
// from the config file (specs 05), and an environment variable named after an
// invented tier would be exactly that mechanism by another route.
//
// Nothing assigned is a *NoReviewerError with [ReasonUnassigned] — exit 1,
// the silent native fallback. A value that is not provider/id is a
// *MalformedAssignmentError instead, which is not ErrNoReviewer at any wrap
// depth.
func (a Assigner) Assign(tier string) (Assignment, error) {
	if a.Explicit != "" {
		return parseAssignment(a.Explicit, SourceFlag, "")
	}

	if !slices.Contains(config.Tiers, tier) {
		return Assignment{}, &NoReviewerError{Reason: ReasonUnassigned}
	}

	name := TierEnvPrefix + strings.ToUpper(tier)
	if value := a.getenv(name); value != "" {
		return parseAssignment(value, SourceEnvironment, name)
	}

	if a.Config != nil {
		if assigned, ok := a.Config.Assignments[tier]; ok {
			return Assignment{
				Provider: assigned.Provider,
				Model:    assigned.Model,
				Source:   SourceConfig,
				Origin:   a.Config.Path,
			}, nil
		}
	}

	return Assignment{}, &NoReviewerError{Reason: ReasonUnassigned}
}

func (a Assigner) getenv(name string) string {
	if a.Getenv == nil {
		return ""
	}
	return a.Getenv(name)
}

// parseAssignment splits a provider/id pair at the **first** slash, so a
// reseller id keeps its own vendor segment: openrouter/anthropic/claude-… is
// the provider openrouter serving the model anthropic/claude-…. Both halves
// must be non-empty; anything else names the value it could not parse.
func parseAssignment(value string, source Source, origin string) (Assignment, error) {
	provider, model, found := strings.Cut(value, "/")
	if !found || provider == "" || model == "" {
		return Assignment{}, &MalformedAssignmentError{Source: source, Origin: origin, Value: value}
	}
	return Assignment{Provider: provider, Model: model, Source: source, Origin: origin}, nil
}
