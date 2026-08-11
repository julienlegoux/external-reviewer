package reviewer_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/julienlegoux/kern-link/ai"
	"github.com/julienlegoux/kern-link/ai/catalog"

	"github.com/julienlegoux/external-reviewer/internal/fauxtest"
	"github.com/julienlegoux/external-reviewer/internal/reviewer"
)

// recordingModels notes the order of the three catalog calls resolution
// makes. It is a decorator rather than a fake so the calls still run against
// the real registry underneath.
type recordingModels struct {
	ai.Models
	calls *[]string
}

func (r recordingModels) Refresh(ctx context.Context, provider string) error {
	*r.calls = append(*r.calls, "Refresh")
	return r.Models.Refresh(ctx, provider)
}

func (r recordingModels) GetModel(provider, id string) *ai.Model {
	*r.calls = append(*r.calls, "GetModel")
	return r.Models.GetModel(provider, id)
}

func (r recordingModels) GetAuth(ctx context.Context, model *ai.Model) (*ai.AuthResult, error) {
	*r.calls = append(*r.calls, "GetAuth")
	return r.Models.GetAuth(ctx, model)
}

// dynamicRegistry builds a provider that reports CanRefreshModels — the
// openrouter/vercel-ai-gateway/nvidia/github-copilot shape. initial is what
// it serves before a refresh (empty for the providers that hold nothing
// until refreshed); refreshed is what RefreshModels installs, or an error
// when refreshErr is set.
func dynamicRegistry(providerID string, initial, refreshed []string, refreshErr error, auth ai.ProviderAuth) ai.MutableModels {
	toModels := func(ids []string) []*ai.Model {
		out := make([]*ai.Model, 0, len(ids))
		for _, id := range ids {
			out = append(out, &ai.Model{ID: id, Name: id, Provider: providerID})
		}
		return out
	}

	provider := ai.CreateProvider(ai.CreateProviderOptions{
		ID:     providerID,
		Auth:   auth,
		Models: toModels(initial),
		RefreshModels: func(context.Context) ([]*ai.Model, error) {
			if refreshErr != nil {
				return nil, refreshErr
			}
			return toModels(refreshed), nil
		},
		Api: ai.StreamFuncs{},
	})

	models := ai.CreateModels(nil)
	models.SetProvider(provider)
	return models
}

func TestResolve_ReachableCredentialedModel_ResolvesWithItsAuthSource(t *testing.T) {
	models, _ := fauxtest.NewRegistry(t, fauxtest.RegistryOptions{
		ProviderID: "openai-codex", ModelIDs: []string{"gpt-5.5"}, Auth: fauxtest.CredentialedAuth("OAuth"),
	})

	resolution, err := reviewer.Resolver{Models: models, ProviderID: "openai-codex", ModelID: "gpt-5.5"}.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve() error = %v, want nil", err)
	}
	if resolution.Model == nil {
		t.Fatal("Resolve() returned no model")
	}
	if resolution.Model.ID != "gpt-5.5" || resolution.Model.Provider != "openai-codex" {
		t.Errorf("resolved model = %s/%s, want openai-codex/gpt-5.5", resolution.Model.Provider, resolution.Model.ID)
	}
	if resolution.AuthSource != "OAuth" {
		t.Errorf("AuthSource = %q, want OAuth", resolution.AuthSource)
	}
}

// TestResolve_ResolutionCarriesNoCredential is the structural half of the
// security boundary: whatever an AuthResult holds, the value resolution hands
// back must contain nothing but the model and the source label. A rendering
// of the whole struct is the widest net a test can cast over "no credential
// escaped".
func TestResolve_ResolutionCarriesNoCredential(t *testing.T) {
	models, _ := fauxtest.NewRegistry(t, fauxtest.RegistryOptions{
		ProviderID: "openai-codex", ModelIDs: []string{"gpt-5.5"}, Auth: fauxtest.CredentialedAuth("OAuth"),
	})

	resolution, err := reviewer.Resolver{Models: models, ProviderID: "openai-codex", ModelID: "gpt-5.5"}.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve() error = %v, want nil", err)
	}
	if rendered := fmt.Sprintf("%+v", resolution); strings.Contains(rendered, fauxtest.Secret) {
		t.Errorf("Resolution renders a credential value: %s", rendered)
	}
}

func TestResolve_ModelAbsentFromCatalog_IsNoReviewer(t *testing.T) {
	models, _ := fauxtest.NewRegistry(t, fauxtest.RegistryOptions{
		ProviderID: "openai-codex", ModelIDs: []string{"some-other-model"}, Auth: fauxtest.CredentialedAuth("OAuth"),
	})

	_, err := reviewer.Resolver{Models: models, ProviderID: "openai-codex", ModelID: "gpt-5.5"}.Resolve(context.Background())
	if !errors.Is(err, reviewer.ErrNoReviewer) {
		t.Fatalf("Resolve() error = %v, want one matching ErrNoReviewer", err)
	}
}

func TestResolve_ProviderAbsentFromRegistry_IsNoReviewer(t *testing.T) {
	models, _ := fauxtest.NewRegistry(t, fauxtest.RegistryOptions{
		ProviderID: "some-other-provider", ModelIDs: []string{"gpt-5.5"}, Auth: fauxtest.CredentialedAuth("OAuth"),
	})

	_, err := reviewer.Resolver{Models: models, ProviderID: "openai-codex", ModelID: "gpt-5.5"}.Resolve(context.Background())
	if !errors.Is(err, reviewer.ErrNoReviewer) {
		t.Fatalf("Resolve() error = %v, want one matching ErrNoReviewer", err)
	}
}

// TestResolve_UnconfiguredProvider_IsNoReviewer covers the sharp edge of the
// exit-code taxonomy: an *absent* credential is a reviewer that was never
// reached, so it is ErrNoReviewer, not a failure.
func TestResolve_UnconfiguredProvider_IsNoReviewer(t *testing.T) {
	models, _ := fauxtest.NewRegistry(t, fauxtest.RegistryOptions{
		ProviderID: "openai-codex", ModelIDs: []string{"gpt-5.5"}, Auth: fauxtest.UnconfiguredAuth(),
	})

	_, err := reviewer.Resolver{Models: models, ProviderID: "openai-codex", ModelID: "gpt-5.5"}.Resolve(context.Background())
	if !errors.Is(err, reviewer.ErrNoReviewer) {
		t.Fatalf("Resolve() error = %v, want one matching ErrNoReviewer", err)
	}
}

// TestResolve_BrokenCredential_IsNotNoReviewer is the other side of that
// edge: a credential that exists and does not work is something a human must
// fix, so it must not be classified as "no reviewer".
func TestResolve_BrokenCredential_IsNotNoReviewer(t *testing.T) {
	tests := []struct {
		name     string
		auth     func(t *testing.T) (ai.ProviderAuth, ai.CredentialStore)
		wantCode ai.ModelsErrorCode
	}{
		{
			name: "expired oauth whose refresh fails",
			auth: func(t *testing.T) (ai.ProviderAuth, ai.CredentialStore) {
				t.Helper()
				return fauxtest.ExpiredOAuthAuth(t, "openai-codex")
			},
			wantCode: ai.ModelsErrorOAuth,
		},
		{
			name: "api key resolution failure",
			auth: func(*testing.T) (ai.ProviderAuth, ai.CredentialStore) {
				return fauxtest.BrokenAPIKeyAuth(), nil
			},
			wantCode: ai.ModelsErrorAuth,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			auth, credentials := tc.auth(t)
			models, _ := fauxtest.NewRegistry(t, fauxtest.RegistryOptions{
				ProviderID: "openai-codex", ModelIDs: []string{"gpt-5.5"}, Auth: auth, Credentials: credentials,
			})

			_, err := reviewer.Resolver{Models: models, ProviderID: "openai-codex", ModelID: "gpt-5.5"}.Resolve(context.Background())
			if err == nil {
				t.Fatal("Resolve() error = nil, want a broken-credential error")
			}
			if errors.Is(err, reviewer.ErrNoReviewer) {
				t.Errorf("Resolve() error matches ErrNoReviewer, want a reached-and-failed error: %v", err)
			}
			var modelsErr *ai.ModelsError
			if !errors.As(err, &modelsErr) {
				t.Fatalf("Resolve() error = %v, want it to wrap *ai.ModelsError", err)
			}
			if modelsErr.ErrCode != tc.wantCode {
				t.Errorf("ModelsError code = %q, want %q", modelsErr.ErrCode, tc.wantCode)
			}
		})
	}
}

// TestResolve_DynamicProvider_RefreshesBeforeLookup pins the one ordering
// SPECS calls out. It fails if the order is inverted twice over: the recorded
// call sequence puts Refresh before GetModel, and the provider holds no
// models until refreshed, so a GetModel-first implementation resolves to
// "model not found" and falls back silently — the exact bug the order exists
// to prevent.
func TestResolve_DynamicProvider_RefreshesBeforeLookup(t *testing.T) {
	models := dynamicRegistry("openrouter", nil, []string{"gpt-5.5"}, nil, fauxtest.CredentialedAuth("OPENROUTER_API_KEY"))

	var calls []string
	resolution, err := reviewer.Resolver{
		Models:     recordingModels{Models: models, calls: &calls},
		ProviderID: "openrouter",
		ModelID:    "gpt-5.5",
	}.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve() error = %v, want nil — a dynamic provider holds no models until refreshed", err)
	}
	if resolution.Model == nil || resolution.Model.ID != "gpt-5.5" {
		t.Fatalf("resolved model = %+v, want gpt-5.5", resolution.Model)
	}

	want := []string{"Refresh", "GetModel", "GetAuth"}
	if strings.Join(calls, ",") != strings.Join(want, ",") {
		t.Errorf("call order = %v, want %v", calls, want)
	}
}

// TestResolve_StaticProvider_IsNotRefreshed is what keeps the assertion above
// honest: refreshing unconditionally would also pass it. Static providers are
// the overwhelming majority and a refresh on them is a wasted round trip.
func TestResolve_StaticProvider_IsNotRefreshed(t *testing.T) {
	models, _ := fauxtest.NewRegistry(t, fauxtest.RegistryOptions{
		ProviderID: "openai-codex", ModelIDs: []string{"gpt-5.5"}, Auth: fauxtest.CredentialedAuth("OAuth"),
	})

	var calls []string
	if _, err := (reviewer.Resolver{
		Models:     recordingModels{Models: models, calls: &calls},
		ProviderID: "openai-codex",
		ModelID:    "gpt-5.5",
	}).Resolve(context.Background()); err != nil {
		t.Fatalf("Resolve() error = %v, want nil", err)
	}

	for _, call := range calls {
		if call == "Refresh" {
			t.Fatalf("call order = %v, want no Refresh on a static provider", calls)
		}
	}
}

// TestResolve_RefreshFailure_WarnsAndUsesLastKnownModels: kern-link keeps a
// dynamic provider's last-known list when a refresh fails, so a transient
// catalog outage must not end a run that can still be served. The failure is
// worth a warn line, never a swallowed error.
func TestResolve_RefreshFailure_WarnsAndUsesLastKnownModels(t *testing.T) {
	models := dynamicRegistry("openrouter", []string{"gpt-5.5"}, nil, errors.New("catalog unreachable"), fauxtest.CredentialedAuth("OPENROUTER_API_KEY"))

	var warnings []string
	resolution, err := reviewer.Resolver{
		Models:     models,
		ProviderID: "openrouter",
		ModelID:    "gpt-5.5",
		Warn:       func(message string) { warnings = append(warnings, message) },
	}.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve() error = %v, want nil — the last-known list still serves this model", err)
	}
	if resolution.Model == nil {
		t.Fatal("Resolve() returned no model")
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "catalog unreachable") {
		t.Errorf("warnings = %v, want one naming the refresh failure", warnings)
	}
}

// TestResolve_RefreshFailureWithNoLastKnownModels_IsNoReviewer: the same
// outage on a provider that has nothing cached leaves no reviewer to reach,
// which is exit 1 rather than a failure.
func TestResolve_RefreshFailureWithNoLastKnownModels_IsNoReviewer(t *testing.T) {
	models := dynamicRegistry("openrouter", nil, nil, errors.New("catalog unreachable"), fauxtest.CredentialedAuth("OPENROUTER_API_KEY"))

	_, err := reviewer.Resolver{Models: models, ProviderID: "openrouter", ModelID: "gpt-5.5"}.Resolve(context.Background())
	if !errors.Is(err, reviewer.ErrNoReviewer) {
		t.Fatalf("Resolve() error = %v, want one matching ErrNoReviewer", err)
	}
}

// TestDefaults_NameOneHardCodedModel guards the walking skeleton's single
// reviewer: Epic 1 ships no tier vocabulary, so these constants are the whole
// of model selection, and they must name a provider and id kern-link's
// embedded catalog actually carries.
func TestDefaults_NameOneHardCodedModel(t *testing.T) {
	if reviewer.DefaultProviderID == "" || reviewer.DefaultModelID == "" {
		t.Fatalf("default model = %q/%q, want both set", reviewer.DefaultProviderID, reviewer.DefaultModelID)
	}
	if strings.Contains(reviewer.DefaultProviderID, "anthropic") {
		t.Errorf("default provider = %q, want one outside the Anthropic family", reviewer.DefaultProviderID)
	}
	if catalog.BuiltinModel(reviewer.DefaultProviderID, reviewer.DefaultModelID) == nil {
		t.Errorf("%s/%s is absent from kern-link's embedded catalog, so it can never resolve",
			reviewer.DefaultProviderID, reviewer.DefaultModelID)
	}
}
