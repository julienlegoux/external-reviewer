package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/julienlegoux/kern-link/ai"

	"github.com/julienlegoux/external-reviewer/internal/diag"
	"github.com/julienlegoux/external-reviewer/internal/family"
)

// runModelsCommand implements `external-reviewer models`: the other half of
// discovery-as-a-query. Where tiers answers "what is configured", models
// answers "what could be" — the intersection of the catalog, what this
// machine's credentials reach, and what the family rule allows, one row per
// model with its price and context window (specs 05).
//
// Like tiers, the report is the command's answer and belongs on stdout;
// every warning goes to stderr. Exit 0 covers every answered query,
// including an empty result — nothing reachable is a true fact about this
// machine, not a failure of the query. Exit 2 is reserved for a malformed
// invocation (an unknown flag, an unrecognised --provider) or a registry
// that could not even be built.
func runModelsCommand(ctx context.Context, args []string, stdout, stderr io.Writer, state *diag.State, registry ai.Models) error {
	fs := flag.NewFlagSet("models", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	providerFlag := fs.String("provider", "", "restrict the listing to one provider id")
	refresh := fs.Bool("refresh", false, "refresh dynamic providers' catalogs before listing (costs network calls)")
	all := fs.Bool("all", false, "show the whole catalog, including rows the credential and family filters would drop")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printUsage(stdout)
			state.StopReason = "help"
			return nil
		}
		diag.WriteError(stderr, err.Error())
		state.StopReason = usageStopReason
		return fmt.Errorf("parsing models flags: %w", err)
	}
	if extra := fs.Args(); len(extra) > 0 {
		return usageError(stderr, state, fmt.Sprintf("unexpected extra arguments: %s", strings.Join(extra, " ")))
	}

	registry, err := ensureModels(registry)
	if err != nil {
		return modelsFailed(stderr, state, err.Error())
	}

	providers := registry.GetProviders()
	if *providerFlag != "" {
		provider := registry.GetProvider(*providerFlag)
		if provider == nil {
			return usageError(stderr, state, fmt.Sprintf("unknown provider %q", *providerFlag))
		}
		providers = []ai.Provider{provider}
	}

	warn := func(message string) { diag.WriteWarn(stderr, message) }
	if *refresh {
		for _, provider := range providers {
			if !provider.CanRefreshModels() {
				continue
			}
			if err := registry.Refresh(ctx, provider.ID()); err != nil {
				warn(fmt.Sprintf("model catalog refresh failed for %s, using the last known model list: %v", provider.ID(), err))
			}
		}
	}

	rows := buildModelRows(ctx, registry, providers, *all)
	writeModelsReport(stdout, rows)
	state.StopReason = queryAnsweredStopReason
	return nil
}

// modelsFailed is usageError's sibling for the one other exit-2 case a
// models query can hit: the registry itself could not be built (a
// credential store that cannot be located), which is a broken machine
// rather than a malformed invocation.
func modelsFailed(stderr io.Writer, state *diag.State, message string) error {
	diag.WriteError(stderr, message)
	state.StopReason = "failed"
	return errors.New(message)
}

// modelRow is one line of the models report: a model's identity, its price
// sheet and context window from the catalog, and — under --all — the reason
// the default view would have dropped it.
type modelRow struct {
	Provider string
	ID       string
	Input    string
	Output   string
	Context  string
	Note     string
}

// buildModelRows computes the whole answer: one row per model that passes
// the default filters (credentialed, family-allowed), plus — when all is
// true — the rows those filters would otherwise drop, marked with why; plus
// one row per credentialed, refreshable provider whose catalog is currently
// empty, naming it as needing --refresh rather than letting it render as an
// absent provider indistinguishable from one with no models at all.
//
// Credentialed-ness is resolved once per provider, from whichever model is
// available to probe with — the provider's own first model, or a bare
// provider-only stub when the catalog holds none yet — because GetAuth
// resolves against the provider's credential, not any one model's fields.
func buildModelRows(ctx context.Context, registry ai.Models, providers []ai.Provider, all bool) []modelRow {
	exclusion := family.DefaultExclusion()
	credentialed := make(map[string]bool, len(providers))
	isCredentialed := func(provider ai.Provider) bool {
		if v, ok := credentialed[provider.ID()]; ok {
			return v
		}
		v := providerCredentialed(ctx, registry, provider)
		credentialed[provider.ID()] = v
		return v
	}

	var rows []modelRow
	for _, provider := range providers {
		models := provider.GetModels()
		if len(models) == 0 {
			if provider.CanRefreshModels() && isCredentialed(provider) {
				rows = append(rows, modelRow{Provider: provider.ID(), ID: "(needs --refresh)", Input: "-", Output: "-", Context: "-"})
			}
			continue
		}
		for _, model := range models {
			vendor := family.Classify(provider.ID(), model.ID)
			allowedFamily := exclusion.AllowsFamily(vendor)
			cred := isCredentialed(provider)
			if !all && (!allowedFamily || !cred) {
				continue
			}
			rows = append(rows, modelRow{
				Provider: provider.ID(),
				ID:       model.ID,
				Input:    priceCell(model.Cost.Input),
				Output:   priceCell(model.Cost.Output),
				Context:  contextCell(model.ContextWindow),
				Note:     modelRowNote(cred, allowedFamily, vendor),
			})
		}
	}
	sortModelRows(rows)
	return rows
}

// providerCredentialed reports whether this machine's credentials reach
// provider, probing with one of its own models when it has any and a bare
// provider-only stub otherwise — GetAuth resolves the provider's own
// credential strategy, and no strategy in this codebase reads any other
// field of the model it is handed.
func providerCredentialed(ctx context.Context, registry ai.Models, provider ai.Provider) bool {
	probe := &ai.Model{Provider: provider.ID()}
	if models := provider.GetModels(); len(models) > 0 {
		probe = models[0]
	}
	result, err := registry.GetAuth(ctx, probe)
	return err == nil && result != nil
}

// modelRowNote names, in --all's vocabulary, why the default view would have
// dropped this row — empty when it would not have.
func modelRowNote(credentialed, allowedFamily bool, vendor family.Family) string {
	var reasons []string
	if !credentialed {
		reasons = append(reasons, "uncredentialed")
	}
	if !allowedFamily {
		reasons = append(reasons, fmt.Sprintf("excluded by family: %s", vendor))
	}
	return strings.Join(reasons, ", ")
}

// sortModelRows orders rows by provider then id, so a script can rely on
// where a given model lands and two runs against an unchanged registry
// produce byte-identical stdout.
func sortModelRows(rows []modelRow) {
	slices.SortFunc(rows, func(a, b modelRow) int {
		if a.Provider != b.Provider {
			return strings.Compare(a.Provider, b.Provider)
		}
		return strings.Compare(a.ID, b.ID)
	})
}

// priceCell renders a catalog price in $/million tokens, or "-" when the
// catalog carries none — a model whose entry has no price must never read
// as free, so 0 is never printed literally.
func priceCell(v float64) string {
	if v == 0 {
		return "-"
	}
	return fmt.Sprintf("$%.2f", v)
}

// contextCell renders a catalog context window, or "-" when the catalog
// carries none.
func contextCell(n int) string {
	if n == 0 {
		return "-"
	}
	return strconv.Itoa(n)
}

// writeModelsReport writes the command's whole answer to stdout: one aligned
// row per model via text/tabwriter, matching tiers' own rendering — no
// dependency beyond the standard library, no colour (specs 12).
func writeModelsReport(stdout io.Writer, rows []modelRow) {
	tw := tabwriter.NewWriter(stdout, 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "model\tinput/M\toutput/M\tcontext\tnote")
	for _, row := range rows {
		_, _ = fmt.Fprintf(tw, "%s/%s\t%s\t%s\t%s\t%s\n", row.Provider, row.ID, row.Input, row.Output, row.Context, row.Note)
	}
	_ = tw.Flush()
}
