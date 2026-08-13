package reviewer_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/julienlegoux/kern-link/ai"
	"github.com/julienlegoux/kern-link/ai/providers/faux"

	"github.com/julienlegoux/external-reviewer/internal/family"
	"github.com/julienlegoux/external-reviewer/internal/fauxtest"
	"github.com/julienlegoux/external-reviewer/internal/reviewer"
)

// catalogOf builds an offline registry serving several providers at once —
// what a machine with three tiers assigned to three different providers
// actually looks like. fauxtest.NewRegistry serves one provider, which is all
// the pre-flight tests need; the chain tests need to prove that a tier
// resolves to *its own* model out of several reachable ones.
func catalogOf(t *testing.T, auth ai.ProviderAuth, catalog map[string][]string) ai.MutableModels {
	t.Helper()

	models := ai.CreateModels(nil)
	for provider, ids := range catalog {
		definitions := make([]faux.ModelDefinition, 0, len(ids))
		for _, id := range ids {
			definitions = append(definitions, faux.ModelDefinition{ID: id})
		}
		handle := faux.New(&faux.Options{Provider: provider, Models: definitions})
		models.SetProvider(fauxtest.NewAuthProvider(handle.Provider, auth))
	}
	return models
}

// threeTierCatalog serves every model threeTiers assigns, so a tier that
// resolves to the wrong one resolves to something that exists — which is the
// only way this assertion can fail for the reason it is testing.
func threeTierCatalog(t *testing.T) ai.MutableModels {
	t.Helper()

	return catalogOf(t, fauxtest.CredentialedAuth("OAuth"), map[string][]string{
		"google":       {"gemini-3.1-flash-lite"},
		"openai-codex": {"gpt-5.5"},
		"openrouter":   {"openai/gpt-5.5"},
	})
}

func TestResolveTier_ConfigAssignsEveryTier_EachResolvesToItsOwnModel(t *testing.T) {
	chain := reviewer.Chain{
		Models:   threeTierCatalog(t),
		Assigner: reviewer.Assigner{Config: tierConfig(t, threeTiers)},
	}

	tests := []struct{ tier, want string }{
		{"light", "google/gemini-3.1-flash-lite"},
		{"standard", "openai-codex/gpt-5.5"},
		{"heavy", "openrouter/openai/gpt-5.5"},
	}
	for _, tc := range tests {
		t.Run(tc.tier, func(t *testing.T) {
			resolution, err := chain.ResolveTier(context.Background(), tc.tier)
			if err != nil {
				t.Fatalf("ResolveTier(%q) error = %v, want nil", tc.tier, err)
			}
			if resolution.Model == nil {
				t.Fatalf("ResolveTier(%q) returned no model", tc.tier)
			}
			if got := resolution.Model.Provider + "/" + resolution.Model.ID; got != tc.want {
				t.Errorf("ResolveTier(%q) = %s, want %s and no other", tc.tier, got, tc.want)
			}
			if resolution.AuthSource != "OAuth" {
				t.Errorf("AuthSource = %q, want OAuth", resolution.AuthSource)
			}
			if resolution.Assignment.Source != reviewer.SourceConfig {
				t.Errorf("Assignment.Source = %q, want %q", resolution.Assignment.Source, reviewer.SourceConfig)
			}
		})
	}
}

// TestResolveTier_ExplicitModel_BypassesTheConfigEntirely asserts the top
// layer end to end, against a config that assigns standard to something else
// and a catalog that serves both — so the wrong answer is a reachable model
// rather than an error.
func TestResolveTier_ExplicitModel_BypassesTheConfigEntirely(t *testing.T) {
	chain := reviewer.Chain{
		Models: threeTierCatalog(t),
		Assigner: reviewer.Assigner{
			Explicit: "google/gemini-3.1-flash-lite",
			Config:   tierConfig(t, threeTiers),
		},
	}

	resolution, err := chain.ResolveTier(context.Background(), "standard")
	if err != nil {
		t.Fatalf("ResolveTier() error = %v, want nil", err)
	}
	if resolution.Model.Provider != "google" || resolution.Model.ID != "gemini-3.1-flash-lite" {
		t.Errorf("ResolveTier() = %s/%s, want google/gemini-3.1-flash-lite — the config assigns openai-codex/gpt-5.5",
			resolution.Model.Provider, resolution.Model.ID)
	}
	if resolution.Assignment.Source != reviewer.SourceFlag {
		t.Errorf("Assignment.Source = %q, want %q", resolution.Assignment.Source, reviewer.SourceFlag)
	}
}

// TestResolveTier_TierEnvironmentVariable_OverridesOnlyThatTier is the same
// surgical property as the Assigner test, asserted through a real catalog
// lookup and credential check: light still resolves from the file in the same
// run, through the same chain value.
func TestResolveTier_TierEnvironmentVariable_OverridesOnlyThatTier(t *testing.T) {
	chain := reviewer.Chain{
		Models: threeTierCatalog(t),
		Assigner: reviewer.Assigner{
			Getenv: envLookup(map[string]string{"EXTERNAL_REVIEWER_TIER_STANDARD": "openrouter/openai/gpt-5.5"}),
			Config: tierConfig(t, threeTiers),
		},
	}

	standard, err := chain.ResolveTier(context.Background(), "standard")
	if err != nil {
		t.Fatalf("ResolveTier(standard) error = %v, want nil", err)
	}
	if standard.Model.Provider != "openrouter" {
		t.Errorf("ResolveTier(standard) provider = %q, want openrouter", standard.Model.Provider)
	}
	if standard.Assignment.Origin != "EXTERNAL_REVIEWER_TIER_STANDARD" {
		t.Errorf("Assignment.Origin = %q, want EXTERNAL_REVIEWER_TIER_STANDARD", standard.Assignment.Origin)
	}

	light, err := chain.ResolveTier(context.Background(), "light")
	if err != nil {
		t.Fatalf("ResolveTier(light) error = %v, want nil", err)
	}
	if light.Model.Provider != "google" || light.Assignment.Source != reviewer.SourceConfig {
		t.Errorf("ResolveTier(light) = %s from %s, want google from the config file",
			light.Model.Provider, light.Assignment.Source)
	}
}

// TestResolveTier_DynamicProvider_RefreshesBeforeLookup is the epic's
// acceptance criterion at the level a caller actually uses: a correctly
// configured tier on a provider that holds no models until refreshed
// resolves. Verified by mutation — deleting the Refresh call in Resolve makes
// this fail with "openrouter/openai/gpt-5.5 is not in the model catalog",
// which is the silent native fallback the ordering exists to prevent.
func TestResolveTier_DynamicProvider_RefreshesBeforeLookup(t *testing.T) {
	models := dynamicRegistry("openrouter", nil, []string{"openai/gpt-5.5"}, nil, fauxtest.CredentialedAuth("OPENROUTER_API_KEY"))

	var calls []string
	chain := reviewer.Chain{
		Models: recordingModels{Models: models, calls: &calls},
		Assigner: reviewer.Assigner{Config: tierConfig(t, `
[tiers.standard]
provider = "openrouter"
model    = "openai/gpt-5.5"
`)},
	}

	resolution, err := chain.ResolveTier(context.Background(), "standard")
	if err != nil {
		t.Fatalf("ResolveTier() error = %v, want nil — a dynamic provider holds no models until refreshed", err)
	}
	if resolution.Model == nil || resolution.Model.ID != "openai/gpt-5.5" {
		t.Fatalf("resolved model = %+v, want openai/gpt-5.5", resolution.Model)
	}
	if want := "Refresh,GetModel,GetAuth"; strings.Join(calls, ",") != want {
		t.Errorf("call order = %v, want %v", calls, want)
	}
}

// TestResolveTier_RefreshFailure_IsNotFatalOnItsOwn: a transient catalog
// outage leaves the last-known list in place, so a tier that can still be
// served is still served — with the failure on stderr rather than swallowed.
func TestResolveTier_RefreshFailure_IsNotFatalOnItsOwn(t *testing.T) {
	models := dynamicRegistry("openrouter", []string{"openai/gpt-5.5"}, nil, errors.New("catalog unreachable"), fauxtest.CredentialedAuth("OPENROUTER_API_KEY"))

	var warnings []string
	chain := reviewer.Chain{
		Models: models,
		Warn:   func(message string) { warnings = append(warnings, message) },
		Assigner: reviewer.Assigner{Config: tierConfig(t, `
[tiers.standard]
provider = "openrouter"
model    = "openai/gpt-5.5"
`)},
	}

	if _, err := chain.ResolveTier(context.Background(), "standard"); err != nil {
		t.Fatalf("ResolveTier() error = %v, want nil — the last-known list still serves this model", err)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "catalog unreachable") {
		t.Errorf("warnings = %v, want one naming the refresh failure", warnings)
	}
}

// TestResolveTier_ExcludedFamilyTier_ResolvesToNothing: a tier assigned an
// Anthropic-family model resolves to no reviewer *for that tier*. It does not
// fall back to another tier, to another model, or to the same family reached
// through a different provider — the catalog here serves three alternatives
// that all satisfy "some reachable model exists".
func TestResolveTier_ExcludedFamilyTier_ResolvesToNothing(t *testing.T) {
	models := catalogOf(t, fauxtest.CredentialedAuth("OAuth"), map[string][]string{
		"openrouter":   {"anthropic/claude-sonnet-4.5", "openai/gpt-5.5"},
		"openai-codex": {"gpt-5.5"},
		"google":       {"gemini-3.1-flash-lite"},
	})
	chain := reviewer.Chain{
		Models: models,
		Assigner: reviewer.Assigner{Config: tierConfig(t, `
[tiers.light]
provider = "google"
model    = "gemini-3.1-flash-lite"

[tiers.standard]
provider = "openrouter"
model    = "anthropic/claude-sonnet-4.5"
`)},
	}

	resolution, err := chain.ResolveTier(context.Background(), "standard")
	if !errors.Is(err, reviewer.ErrNoReviewer) {
		t.Fatalf("ResolveTier(standard) error = %v, want one matching ErrNoReviewer", err)
	}
	if resolution.Model != nil {
		t.Fatalf("ResolveTier(standard) returned model %+v, want none", resolution.Model)
	}
	var noReviewer *reviewer.NoReviewerError
	if !errors.As(err, &noReviewer) || noReviewer.Family != family.Anthropic {
		t.Errorf("error = %v, want a *NoReviewerError naming the anthropic family", err)
	}

	// The other tier is untouched: exclusion is per assignment, not a mode
	// the whole chain falls into.
	if _, err := chain.ResolveTier(context.Background(), "light"); err != nil {
		t.Errorf("ResolveTier(light) error = %v, want nil", err)
	}
}

// TestResolveTier_UnknownFamilyTier_SaysWhichRuleRefusedIt is the gap issue 02
// measured and left open, made legible rather than silent: every github-copilot
// id in the embedded catalog is a bare model name, so a tier assigned to that
// provider resolves to nothing. The user gets told which rule did it.
func TestResolveTier_UnknownFamilyTier_SaysWhichRuleRefusedIt(t *testing.T) {
	models := dynamicRegistry("github-copilot", nil, []string{"gpt-5"}, nil, fauxtest.CredentialedAuth("GITHUB_TOKEN"))
	chain := reviewer.Chain{
		Models: models,
		Assigner: reviewer.Assigner{Config: tierConfig(t, `
[tiers.standard]
provider = "github-copilot"
model    = "gpt-5"
`)},
	}

	_, err := chain.ResolveTier(context.Background(), "standard")
	var noReviewer *reviewer.NoReviewerError
	if !errors.As(err, &noReviewer) {
		t.Fatalf("ResolveTier() error = %v, want a *NoReviewerError", err)
	}
	if noReviewer.Reason != reviewer.ReasonFamilyUnknown {
		t.Errorf("Reason = %q, want %q — the refresh worked and the model was found", noReviewer.Reason, reviewer.ReasonFamilyUnknown)
	}
	if !strings.Contains(err.Error(), "github-copilot/gpt-5") {
		t.Errorf("error = %q, want it to name the model it refused", err)
	}
}

func TestResolveTier_UnassignedTier_IsNoReviewer(t *testing.T) {
	chain := reviewer.Chain{Models: threeTierCatalog(t)}

	_, err := chain.ResolveTier(context.Background(), "standard")
	if !errors.Is(err, reviewer.ErrNoReviewer) {
		t.Fatalf("ResolveTier() error = %v, want one matching ErrNoReviewer", err)
	}
}

// TestResolveTier_MalformedAssignment_NeverReachesTheCatalog: an unparseable
// value stops the chain at the assignment layer, so nothing is looked up and
// no credential is resolved on the strength of a typo.
func TestResolveTier_MalformedAssignment_NeverReachesTheCatalog(t *testing.T) {
	var calls []string
	chain := reviewer.Chain{
		Models:   recordingModels{Models: threeTierCatalog(t), calls: &calls},
		Assigner: reviewer.Assigner{Getenv: envLookup(map[string]string{"EXTERNAL_REVIEWER_TIER_STANDARD": "gpt-5.5"})},
	}

	_, err := chain.ResolveTier(context.Background(), "standard")
	var malformed *reviewer.MalformedAssignmentError
	if !errors.As(err, &malformed) {
		t.Fatalf("ResolveTier() error = %v, want a *MalformedAssignmentError", err)
	}
	if errors.Is(err, reviewer.ErrNoReviewer) {
		t.Errorf("ResolveTier() error matches ErrNoReviewer, want a distinct malformed error: %v", err)
	}
	if len(calls) != 0 {
		t.Errorf("catalog calls = %v, want none", calls)
	}
}

// TestResolveTier_BrokenCredential_IsNotNoReviewer keeps the exit-2 class
// intact through the chain: it must not be flattened into ErrNoReviewer on the
// way out.
func TestResolveTier_BrokenCredential_IsNotNoReviewer(t *testing.T) {
	models := catalogOf(t, fauxtest.BrokenAPIKeyAuth(), map[string][]string{"openai-codex": {"gpt-5.5"}})
	chain := reviewer.Chain{
		Models:   models,
		Assigner: reviewer.Assigner{Explicit: "openai-codex/gpt-5.5"},
	}

	_, err := chain.ResolveTier(context.Background(), "")
	if err == nil {
		t.Fatal("ResolveTier() error = nil, want a broken-credential error")
	}
	if errors.Is(err, reviewer.ErrNoReviewer) {
		t.Errorf("ResolveTier() error matches ErrNoReviewer, want a reached-and-failed error: %v", err)
	}
	var modelsErr *ai.ModelsError
	if !errors.As(err, &modelsErr) {
		t.Fatalf("ResolveTier() error = %v, want it to wrap *ai.ModelsError", err)
	}
	if modelsErr.ErrCode != ai.ModelsErrorAuth {
		t.Errorf("ModelsError code = %q, want %q", modelsErr.ErrCode, ai.ModelsErrorAuth)
	}
}

// TestResolveTier_ResolutionCarriesNoCredential extends the structural half of
// the security boundary over the field this issue adds: an assignment is three
// strings a human typed, and rendering the whole Resolution must still turn up
// nothing a credential store put there.
func TestResolveTier_ResolutionCarriesNoCredential(t *testing.T) {
	chain := reviewer.Chain{
		Models:   threeTierCatalog(t),
		Assigner: reviewer.Assigner{Config: tierConfig(t, threeTiers)},
	}

	resolution, err := chain.ResolveTier(context.Background(), "standard")
	if err != nil {
		t.Fatalf("ResolveTier() error = %v, want nil", err)
	}
	if rendered := fmt.Sprintf("%+v", resolution); strings.Contains(rendered, fauxtest.Secret) {
		t.Errorf("Resolution renders a credential value: %s", rendered)
	}
}
