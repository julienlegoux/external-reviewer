package cli_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/julienlegoux/kern-link/ai"

	"github.com/julienlegoux/external-reviewer/internal/cli"
	"github.com/julienlegoux/external-reviewer/internal/fauxtest"
)

// errFakeRefreshFailure is the scripted error a dynamic provider's
// RefreshModels returns in the refresh-failure test.
var errFakeRefreshFailure = errors.New("catalog unreachable")

// providerFixture describes one provider's shape for a models registry: its
// models, its auth strategy, and — for the dynamic-provider criteria — a
// RefreshModels func. A nil Refresh makes CanRefreshModels() false, the
// static-provider shape every ordinary catalog entry has.
type providerFixture struct {
	Models  []*ai.Model
	Auth    ai.ProviderAuth
	Refresh func(context.Context) ([]*ai.Model, error)
}

// modelsRegistry builds an offline registry serving several providers at
// once, each shaped by its own providerFixture — full control over auth,
// price, context window and refreshability, which the shared catalogRegistry
// helper (one auth strategy for every provider) cannot express.
func modelsRegistry(t *testing.T, providers map[string]providerFixture) ai.MutableModels {
	t.Helper()

	models := ai.CreateModels(nil)
	for id, fixture := range providers {
		for _, m := range fixture.Models {
			m.Provider = id
		}
		models.SetProvider(ai.CreateProvider(ai.CreateProviderOptions{
			ID:            id,
			Auth:          fixture.Auth,
			Models:        fixture.Models,
			RefreshModels: fixture.Refresh,
			Api:           ai.StreamFuncs{},
		}))
	}
	return models
}

// priced builds a model with an explicit price sheet and context window, so
// a test can prove those fields — rather than faux's own defaults — are what
// reaches the row.
func priced(id string, input, output float64, contextWindow int) *ai.Model {
	return &ai.Model{ID: id, Name: id, Cost: ai.ModelCost{Input: input, Output: output}, ContextWindow: contextWindow}
}

// unpriced builds a model with no price sheet and no context window at all —
// the catalog-carries-nothing case the placeholder criterion is about.
func unpriced(id string) *ai.Model {
	return &ai.Model{ID: id, Name: id}
}

// refreshCountingModels wraps a registry to record how many times Refresh
// was called, and for which provider — the assertion surface for "--refresh
// calls Refresh exactly once per provider reporting CanRefreshModels(), and
// never for one that does not."
type refreshCountingModels struct {
	ai.Models
	calls map[string]int
}

func newRefreshCountingModels(models ai.Models) *refreshCountingModels {
	return &refreshCountingModels{Models: models, calls: map[string]int{}}
}

func (r *refreshCountingModels) Refresh(ctx context.Context, provider string) error {
	r.calls[provider]++
	return r.Models.Refresh(ctx, provider)
}

// runModels invokes the `models` subcommand against models, returning the
// exit code and both streams.
func runModels(t *testing.T, models ai.Models, flags ...string) (int, string, string) {
	t.Helper()

	argv := append([]string{"models"}, flags...)
	var stdout, stderr bytes.Buffer
	code := cli.RunForTest(argv, strings.NewReader(""), &stdout, &stderr, models)
	return code, stdout.String(), stderr.String()
}

// TestRun_Models_TwoProviders_OnlyCredentialedListed is written failing
// first: with a registry holding two providers, one credentialed and one
// not, models lists only the credentialed provider's models, exit 0.
func TestRun_Models_TwoProviders_OnlyCredentialedListed(t *testing.T) {
	models := modelsRegistry(t, map[string]providerFixture{
		"openai-codex": {Models: []*ai.Model{priced("model-a", 1, 2, 128000)}, Auth: fauxtest.CredentialedAuth("OAuth")},
		"google":       {Models: []*ai.Model{priced("model-b", 3, 4, 64000)}, Auth: fauxtest.UnconfiguredAuth()},
	})

	code, stdout, stderr := runModels(t, models)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	if !strings.Contains(stdout, "openai-codex/model-a") {
		t.Errorf("stdout = %q, want the credentialed provider's model listed", stdout)
	}
	if strings.Contains(stdout, "google") {
		t.Errorf("stdout = %q, want the uncredentialed provider's model absent", stdout)
	}
}

// TestRun_Models_All_ListsBothAndMarksFilteredRows: --all lists both
// providers' models, and marks the row the default view filters out
// (uncredentialed) rather than printing it indistinguishably from a row that
// is actually reachable.
func TestRun_Models_All_ListsBothAndMarksFilteredRows(t *testing.T) {
	models := modelsRegistry(t, map[string]providerFixture{
		"openai-codex": {Models: []*ai.Model{priced("model-a", 1, 2, 128000)}, Auth: fauxtest.CredentialedAuth("OAuth")},
		"google":       {Models: []*ai.Model{priced("model-b", 3, 4, 64000)}, Auth: fauxtest.UnconfiguredAuth()},
	})

	code, stdout, stderr := runModels(t, models, "--all")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	if !strings.Contains(stdout, "openai-codex/model-a") {
		t.Errorf("stdout = %q, want the credentialed provider's model listed", stdout)
	}
	if !strings.Contains(stdout, "google/model-b") {
		t.Errorf("stdout = %q, want --all to list the uncredentialed provider's model too", stdout)
	}
	unreachableLine := lineNaming(stdout, "google/model-b")
	if !strings.Contains(unreachableLine, "uncredentialed") {
		t.Errorf("row for google/model-b = %q, want it marked uncredentialed", unreachableLine)
	}
	reachableLine := lineNaming(stdout, "openai-codex/model-a")
	if strings.Contains(reachableLine, "uncredentialed") {
		t.Errorf("row for openai-codex/model-a = %q, want it not marked uncredentialed", reachableLine)
	}
}

// TestRun_Models_AnthropicFamily_NeverAppearsWithoutAll asserts on the full
// stdout, including a reseller id (openrouter-style provider serving an
// anthropic/-prefixed id) — the failure this whole epic exists to prevent
// must never surface through the models command's default view either.
func TestRun_Models_AnthropicFamily_NeverAppearsWithoutAll(t *testing.T) {
	models := modelsRegistry(t, map[string]providerFixture{
		"openrouter": {
			Models: []*ai.Model{priced("anthropic/claude-sonnet-4.5", 3, 15, 200000), priced("openai/gpt-5.5", 1, 2, 128000)},
			Auth:   fauxtest.CredentialedAuth("OAuth"),
		},
	})

	code, stdout, stderr := runModels(t, models)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	if strings.Contains(stdout, "anthropic") {
		t.Errorf("stdout = %q, want no anthropic-family row without --all", stdout)
	}
	if !strings.Contains(stdout, "openrouter/openai/gpt-5.5") {
		t.Errorf("stdout = %q, want the non-anthropic model still listed", stdout)
	}

	allCode, allStdout, allStderr := runModels(t, models, "--all")
	if allCode != 0 {
		t.Fatalf("--all: exit code = %d, want 0 (stderr: %q)", allCode, allStderr)
	}
	if !strings.Contains(allStdout, "anthropic/claude-sonnet-4.5") {
		t.Errorf("--all stdout = %q, want the anthropic-family row present", allStdout)
	}
}

// TestRun_Models_Refresh_CallsOnlyRefreshableProvidersExactlyOnce: --refresh
// calls Refresh exactly once per provider reporting CanRefreshModels(), and
// never for one that does not.
func TestRun_Models_Refresh_CallsOnlyRefreshableProvidersExactlyOnce(t *testing.T) {
	base := modelsRegistry(t, map[string]providerFixture{
		"openai-codex": {
			Auth: fauxtest.CredentialedAuth("OAuth"),
			Refresh: func(context.Context) ([]*ai.Model, error) {
				return []*ai.Model{priced("model-a", 1, 2, 128000)}, nil
			},
		},
		"google": {Models: []*ai.Model{priced("model-b", 1, 2, 128000)}, Auth: fauxtest.CredentialedAuth("OAuth")},
	})
	counting := newRefreshCountingModels(base)

	code, _, stderr := runModels(t, counting, "--refresh")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	if got := counting.calls["openai-codex"]; got != 1 {
		t.Errorf("Refresh(openai-codex) called %d times, want exactly 1", got)
	}
	if got := counting.calls["google"]; got != 0 {
		t.Errorf("Refresh(google) called %d times, want 0 — it does not report CanRefreshModels()", got)
	}
}

// TestRun_Models_WithoutRefreshFlag_NeverCallsRefresh: without --refresh, no
// Refresh call is made at all, even for a dynamic provider.
func TestRun_Models_WithoutRefreshFlag_NeverCallsRefresh(t *testing.T) {
	base := modelsRegistry(t, map[string]providerFixture{
		"openai-codex": {
			Auth: fauxtest.CredentialedAuth("OAuth"),
			Refresh: func(context.Context) ([]*ai.Model, error) {
				return []*ai.Model{priced("model-a", 1, 2, 128000)}, nil
			},
		},
	})
	counting := newRefreshCountingModels(base)

	code, _, stderr := runModels(t, counting)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	if got := counting.calls["openai-codex"]; got != 0 {
		t.Errorf("Refresh(openai-codex) called %d times without --refresh, want 0", got)
	}
}

// TestRun_Models_RefreshFails_WarnsAndStillPrintsLastKnownRows: a refresh
// that fails emits a warn line and still prints the rows the last-known
// catalog holds; exit stays 0.
func TestRun_Models_RefreshFails_WarnsAndStillPrintsLastKnownRows(t *testing.T) {
	models := modelsRegistry(t, map[string]providerFixture{
		"openai-codex": {
			Models: []*ai.Model{priced("model-a", 1, 2, 128000)},
			Auth:   fauxtest.CredentialedAuth("OAuth"),
			Refresh: func(context.Context) ([]*ai.Model, error) {
				return nil, errFakeRefreshFailure
			},
		},
	})

	code, stdout, stderr := runModels(t, models, "--refresh")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	if !strings.Contains(stderr, "warn") {
		t.Errorf("stderr = %q, want a warn line for the failed refresh", stderr)
	}
	if !strings.Contains(stdout, "openai-codex/model-a") {
		t.Errorf("stdout = %q, want the last-known catalog still printed", stdout)
	}
}

// TestRun_Models_CredentialedEmptyCatalogProvider_NamedAsNeedingRefresh: a
// credentialed provider reporting CanRefreshModels() whose catalog is empty
// is named on stdout as needing --refresh, rather than rendering as an
// absent row.
func TestRun_Models_CredentialedEmptyCatalogProvider_NamedAsNeedingRefresh(t *testing.T) {
	models := modelsRegistry(t, map[string]providerFixture{
		"openai-codex": {
			Auth: fauxtest.CredentialedAuth("OAuth"),
			Refresh: func(context.Context) ([]*ai.Model, error) {
				return []*ai.Model{priced("model-a", 1, 2, 128000)}, nil
			},
		},
	})

	code, stdout, stderr := runModels(t, models)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	if !strings.Contains(stdout, "openai-codex") {
		t.Errorf("stdout = %q, want the empty-catalog provider named", stdout)
	}
	if !strings.Contains(stdout, "--refresh") {
		t.Errorf("stdout = %q, want it to say --refresh would help", stdout)
	}
}

// TestRun_Models_UnknownProvider_ExitsTwo: --provider <unknown> is a usage
// error, because a typo silently printing nothing is indistinguishable from
// a provider with no models.
func TestRun_Models_UnknownProvider_ExitsTwo(t *testing.T) {
	models := modelsRegistry(t, map[string]providerFixture{
		"openai-codex": {Models: []*ai.Model{priced("model-a", 1, 2, 128000)}, Auth: fauxtest.CredentialedAuth("OAuth")},
	})

	code, stdout, stderr := runModels(t, models, "--provider", "bogus")

	assertUsageError(t, code, stdout, stderr)
	if !strings.Contains(stderr, "bogus") {
		t.Errorf("stderr = %q, want it to quote the unknown provider id", stderr)
	}
}

// TestRun_Models_ProviderFlag_RestrictsToThatProvider proves --provider
// actually filters, rather than merely being accepted.
func TestRun_Models_ProviderFlag_RestrictsToThatProvider(t *testing.T) {
	models := modelsRegistry(t, map[string]providerFixture{
		"openai-codex": {Models: []*ai.Model{priced("model-a", 1, 2, 128000)}, Auth: fauxtest.CredentialedAuth("OAuth")},
		"google":       {Models: []*ai.Model{priced("model-b", 1, 2, 128000)}, Auth: fauxtest.CredentialedAuth("OAuth")},
	})

	code, stdout, stderr := runModels(t, models, "--provider", "openai-codex")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	if !strings.Contains(stdout, "openai-codex/model-a") {
		t.Errorf("stdout = %q, want openai-codex's model listed", stdout)
	}
	if strings.Contains(stdout, "google") {
		t.Errorf("stdout = %q, want google absent", stdout)
	}
}

// TestRun_Models_SortedDeterministically: rows are sorted by provider then
// id, and running the same command twice against an unchanged registry
// produces byte-identical stdout.
func TestRun_Models_SortedDeterministically(t *testing.T) {
	models := modelsRegistry(t, map[string]providerFixture{
		"xai":      {Models: []*ai.Model{priced("b-model", 1, 2, 1000), priced("a-model", 1, 2, 1000)}, Auth: fauxtest.CredentialedAuth("OAuth")},
		"deepseek": {Models: []*ai.Model{priced("m", 1, 2, 1000)}, Auth: fauxtest.CredentialedAuth("OAuth")},
	})

	_, first, _ := runModels(t, models)
	_, second, _ := runModels(t, models)

	if first != second {
		t.Errorf("two runs against an unchanged registry produced different stdout:\nfirst:  %q\nsecond: %q", first, second)
	}

	aaaIndex := strings.Index(first, "deepseek/m")
	zzzAIndex := strings.Index(first, "xai/a-model")
	zzzBIndex := strings.Index(first, "xai/b-model")
	if aaaIndex < 0 || zzzAIndex < 0 || zzzBIndex < 0 {
		t.Fatalf("stdout is missing an expected row: %q", first)
	}
	if aaaIndex >= zzzAIndex || zzzAIndex >= zzzBIndex {
		t.Errorf("stdout is not sorted by provider then id: %q", first)
	}
}

// TestRun_Models_PriceAndContextWindow_FromCatalog_PlaceholderWhenAbsent:
// price and context window come from the catalog's own fields; a model whose
// catalog entry carries no price renders a placeholder rather than 0, so
// free and unknown are not confused.
func TestRun_Models_PriceAndContextWindow_FromCatalog_PlaceholderWhenAbsent(t *testing.T) {
	models := modelsRegistry(t, map[string]providerFixture{
		"openai-codex": {Models: []*ai.Model{priced("priced-model", 1.5, 7.25, 131072), unpriced("bare-model")}, Auth: fauxtest.CredentialedAuth("OAuth")},
	})

	code, stdout, stderr := runModels(t, models)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}

	pricedLine := lineNaming(stdout, "openai-codex/priced-model")
	if !strings.Contains(pricedLine, "131072") {
		t.Errorf("row for priced-model = %q, want the context window 131072", pricedLine)
	}
	if !strings.Contains(pricedLine, "1.50") || !strings.Contains(pricedLine, "7.25") {
		t.Errorf("row for priced-model = %q, want both catalog prices", pricedLine)
	}

	bareLine := lineNaming(stdout, "openai-codex/bare-model")
	if strings.Contains(bareLine, "$0.00") {
		t.Errorf("row for bare-model = %q, want a placeholder rather than a literal 0 price", bareLine)
	}
	fields := strings.Fields(bareLine)
	if len(fields) < 3 || fields[1] != "-" || fields[2] != "-" {
		t.Errorf("row for bare-model = %q, want the price columns to render as a placeholder (\"-\")", bareLine)
	}
}

// TestRun_Models_NoCredentialValueReachesEitherStream is the security
// boundary this command shares with tiers: only what a row needs to say may
// travel, whatever state a provider's credential is in.
func TestRun_Models_NoCredentialValueReachesEitherStream(t *testing.T) {
	models := modelsRegistry(t, map[string]providerFixture{
		"openai-codex": {Models: []*ai.Model{priced("model-a", 1, 2, 128000)}, Auth: fauxtest.CredentialedAuth("OAuth")},
	})

	_, stdout, stderr := runModels(t, models, "--all")
	if strings.Contains(stdout, fauxtest.Secret) {
		t.Errorf("stdout leaked a credential value: %q", stdout)
	}
	if strings.Contains(stderr, fauxtest.Secret) {
		t.Errorf("stderr leaked a credential value: %q", stderr)
	}
}

// TestRun_Models_UnknownFlag_ExitsTwo and
// TestRun_Models_ExtraPositional_ExitsTwo are the malformed-invocation half
// of the exit-code contract, matching tiers' own usage-error shape.
func TestRun_Models_UnknownFlag_ExitsTwo(t *testing.T) {
	models := modelsRegistry(t, map[string]providerFixture{
		"openai-codex": {Models: []*ai.Model{priced("model-a", 1, 2, 128000)}, Auth: fauxtest.CredentialedAuth("OAuth")},
	})

	code, stdout, stderr := runModels(t, models, "--nope")

	assertUsageError(t, code, stdout, stderr)
}

func TestRun_Models_ExtraPositional_ExitsTwo(t *testing.T) {
	models := modelsRegistry(t, map[string]providerFixture{
		"openai-codex": {Models: []*ai.Model{priced("model-a", 1, 2, 128000)}, Auth: fauxtest.CredentialedAuth("OAuth")},
	})

	code, stdout, stderr := runModels(t, models, "extra-positional")

	assertUsageError(t, code, stdout, stderr)
}

// TestUsage_DocumentsModelsCommand: usageText must promise the command that
// now exists.
func TestUsage_DocumentsModelsCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := cli.Run([]string{"help"}, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("help exit code = %d, want 0", code)
	}
	usage := stdout.String()

	if !strings.Contains(usage, "\n  models") {
		t.Errorf("usage text does not document the models command:\n%s", usage)
	}
}

// lineNaming returns the one line of stdout containing needle, or "" when
// there is none — the assertion surface for "what does this model's own row
// say?", mirroring selection_test.go's modelLine and tiers_test.go's
// tierLine.
func lineNaming(stdout, needle string) string {
	for _, line := range strings.Split(stdout, "\n") {
		if strings.Contains(line, needle) {
			return line
		}
	}
	return ""
}
