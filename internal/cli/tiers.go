package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/julienlegoux/kern-link/ai"

	"github.com/julienlegoux/external-reviewer/internal/config"
	"github.com/julienlegoux/external-reviewer/internal/diag"
	"github.com/julienlegoux/external-reviewer/internal/family"
	"github.com/julienlegoux/external-reviewer/internal/reviewer"
)

// tiersAnsweredStopReason is the `stop=` word the done line carries when the
// tiers command completes: neither of the six words SPECS already fixes fits
// a query that never tries to reach a model at all — not even `no_reviewer`,
// which means *this run's own* reviewer was never reached, whereas `tiers`
// reporting that no tier resolves is the query correctly answered. SPECS
// calls the stop-reason set "a union that only grows" and gives `bounds`
// (Epic 2) as the precedent for adding a word rather than repurposing one of
// the existing six.
const tiersAnsweredStopReason = "answered"

// runTiersCommand implements `external-reviewer tiers`: one row per tier
// naming its assignment, where that assignment came from, and — when it does
// not resolve — which rule refused it. Unlike review, the report *is* the
// command's answer (specs 12), so it goes on stdout; every warning and
// diagnostic stays on stderr.
//
// Exit 0 covers every answered query, including "no tier resolves" and "a
// tier's credential is broken" — both are true answers about this machine
// right now, not failures of the query itself. Exit 2 is reserved for the
// query never being answerable at all: a malformed invocation, malformed
// TOML, or a credential store that cannot even be located.
func runTiersCommand(ctx context.Context, args []string, stdout, stderr io.Writer, state *diag.State, registry ai.Models) error {
	fs := flag.NewFlagSet("tiers", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	excludeFamily := fs.String("exclude-family", defaultExcludeFamily, "comma-separated model families that may not review")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printUsage(stdout)
			state.StopReason = "help"
			return nil
		}
		diag.WriteError(stderr, err.Error())
		state.StopReason = usageStopReason
		return fmt.Errorf("parsing tiers flags: %w", err)
	}
	if extra := fs.Args(); len(extra) > 0 {
		return usageError(stderr, state, fmt.Sprintf("unexpected extra arguments: %s", strings.Join(extra, " ")))
	}

	exclusion, err := parseExclusion(*excludeFamily)
	if err != nil {
		return usageError(stderr, state, err.Error())
	}

	registry, err = ensureModels(registry)
	if err != nil {
		return tiersFailed(stderr, state, err.Error())
	}

	warn := func(message string) { diag.WriteWarn(stderr, message) }
	cfg, err := config.Load()
	if err != nil {
		return tiersFailed(stderr, state, err.Error())
	}
	for _, warning := range cfg.Warnings {
		warn(warning)
	}

	assigner := reviewer.Assigner{Getenv: os.Getenv, Config: cfg}
	rows := make([]tierRow, 0, len(config.Tiers))
	for _, tier := range config.Tiers {
		row, err := resolveTierRow(ctx, registry, assigner, exclusion, tier)
		if err != nil {
			return tiersFailed(stderr, state, err.Error())
		}
		rows = append(rows, row)
	}

	writeTiersReport(stdout, cfg.Path, rows)
	state.StopReason = tiersAnsweredStopReason
	return nil
}

// tiersFailed is usageError's sibling for the tiers command's other exit-2
// case: the query itself could not be answered (malformed TOML, a
// credential store that cannot be located, a hand-typed assignment that is
// not provider/id), as opposed to a malformed invocation. Both write one
// error: line and return the error classify turns into exit 2; only the
// done line's stop reason differs, so a human reading the transcript can
// tell "you typed this wrong" from "the machine is not set up right".
func tiersFailed(stderr io.Writer, state *diag.State, message string) error {
	diag.WriteError(stderr, message)
	state.StopReason = "failed"
	return errors.New(message)
}

// tierRow is one line of the tiers report: a tier's assignment, where it
// came from, and its current status on this machine.
type tierRow struct {
	Tier   string
	Model  string // "provider/id", or "-" when nothing is assigned
	Source string // "config", an EXTERNAL_REVIEWER_TIER_<TIER> name, or "-"
	Status string
}

// resolveTierRow runs the whole of selection for one tier — the precedence
// chain, then the pre-flight — and turns the outcome into a row rather than
// a run's exit code. The two error classes selection can return terminate
// the whole query (a malformed assignment matches "the query could not be
// answered", the same class as malformed TOML); every other outcome,
// including a broken credential, is a fact about this one tier and becomes a
// row instead.
func resolveTierRow(ctx context.Context, registry ai.Models, assigner reviewer.Assigner, exclusion family.Exclusion, tier string) (tierRow, error) {
	assignment, err := assigner.Assign(tier)
	if err != nil {
		var noReviewer *reviewer.NoReviewerError
		if errors.As(err, &noReviewer) && noReviewer.Reason == reviewer.ReasonUnassigned {
			return tierRow{Tier: tier, Model: "-", Source: "-", Status: "unassigned"}, nil
		}
		// A *MalformedAssignmentError only ever comes from the environment
		// or the config file here — tiers has no --model flag to bypass
		// assignment with — so there is no "the caller mistyped argv" case
		// to split out the way review's malformedFlagAssignment does.
		return tierRow{}, fmt.Errorf("tier %q: %w", tier, err)
	}

	row := tierRow{Tier: tier, Model: assignment.String(), Source: tierSource(assignment)}

	resolution, err := reviewer.Resolver{
		Models:     registry,
		ProviderID: assignment.Provider,
		ModelID:    assignment.Model,
		Exclusion:  &exclusion,
	}.Resolve(ctx)
	switch err {
	case nil:
		row.Status = fmt.Sprintf("reachable (auth=%s)", resolution.AuthSource)
		return row, nil
	default:
		var noReviewer *reviewer.NoReviewerError
		if errors.As(err, &noReviewer) {
			row.Status = tierStatus(noReviewer)
			return row, nil
		}
		var modelsErr *ai.ModelsError
		if errors.As(err, &modelsErr) && (modelsErr.Code() == "oauth" || modelsErr.Code() == "auth") {
			row.Status = "credential broken"
			return row, nil
		}
		// Neither typed class — an unexpected failure (e.g. a cancelled
		// context) is a fact about the whole query, not this one tier.
		return tierRow{}, fmt.Errorf("tier %q: %w", tier, err)
	}
}

// tierSource names the knob that produced this row's assignment, in the
// vocabulary the row can show at a glance: "config" for the tier assignment
// file (whose path is already printed once, ahead of the table) or the
// EXTERNAL_REVIEWER_TIER_<TIER> variable's own name.
func tierSource(assignment reviewer.Assignment) string {
	if assignment.Source == reviewer.SourceEnvironment {
		return assignment.Origin
	}
	return string(assignment.Source)
}

// tierStatus renders the rule that refused a tier, switching on the typed
// Reason rather than the error's message text — the same discipline
// selectReviewer's noReviewerDiagnostic follows, for the same reason: a
// caller (here, this function) must not have to parse a sentence apart to
// tell one rule from another.
func tierStatus(e *reviewer.NoReviewerError) string {
	switch e.Reason {
	case reviewer.ReasonNotInCatalog:
		return "not in the catalog"
	case reviewer.ReasonFamilyExcluded, reviewer.ReasonFamilyUnknown:
		return fmt.Sprintf("excluded by family: %s", e.Family)
	case reviewer.ReasonUnconfigured:
		return "provider unconfigured"
	default:
		return string(e.Reason)
	}
}

// writeTiersReport writes the command's whole answer to stdout: the config
// path that was looked at, whether or not anything is there, then one
// aligned row per tier via text/tabwriter — no dependency beyond the
// standard library, and no ANSI or colour (the caller is a script, specs
// 12).
func writeTiersReport(stdout io.Writer, configPath string, rows []tierRow) {
	_, _ = fmt.Fprintf(stdout, "config  %s\n\n", configPath)
	tw := tabwriter.NewWriter(stdout, 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "tier\tmodel\tsource\tstatus")
	for _, row := range rows {
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", row.Tier, row.Model, row.Source, row.Status)
	}
	_ = tw.Flush()
}
