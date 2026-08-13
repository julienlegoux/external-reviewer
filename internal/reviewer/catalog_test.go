package reviewer_test

import (
	"context"
	"testing"

	"github.com/julienlegoux/kern-link/ai"
	"github.com/julienlegoux/kern-link/ai/providers"

	"github.com/julienlegoux/external-reviewer/internal/family"
	"github.com/julienlegoux/external-reviewer/internal/reviewer"
)

// localProvider is the provider every locally defined model is served by
// today. It is written out rather than read off the definitions so that a
// definition moved to another provider fails these tests instead of silently
// carrying them along.
const localProvider = "openai-codex"

// wantLocal is the price sheet and shape each locally registered model must
// reach the registry with — the user's own definitions, transcribed once so
// that a typo in reviewer.LocalModels is a failing test rather than a wrong
// number on an invoice.
var wantLocal = map[string]struct {
	Name          string
	Cost          ai.ModelCost
	ContextWindow int
	MaxTokens     int
}{
	"gpt-5.6-sol": {
		Name:          "GPT-5.6 Sol",
		Cost:          ai.ModelCost{Input: 5, Output: 30, CacheRead: 0.5, CacheWrite: 6.25},
		ContextWindow: 272000,
		MaxTokens:     128000,
	},
	"gpt-5.6-terra": {
		Name:          "GPT-5.6 Terra",
		Cost:          ai.ModelCost{Input: 2, Output: 12, CacheRead: 0.2, CacheWrite: 2.5},
		ContextWindow: 272000,
		MaxTokens:     128000,
	},
	"gpt-5.6-luna": {
		Name:          "GPT-5.6 Luna",
		Cost:          ai.ModelCost{Input: 0.2, Output: 1.2, CacheRead: 0.02, CacheWrite: 0.25},
		ContextWindow: 272000,
		MaxTokens:     128000,
	},
}

// The registration's whole point: after it runs against the registry the
// binary really builds, each locally defined model is an ordinary catalog
// lookup away, carrying the price sheet and context window it was defined
// with. Anything less and `models` lists a row with a "-" price, or nothing
// at all.
func TestRegisterLocalModels_EachLocalModelIsReachableWithItsOwnPriceSheet(t *testing.T) {
	registry := providers.Models(nil)
	if err := reviewer.RegisterLocalModels(registry); err != nil {
		t.Fatalf("RegisterLocalModels() = %v, want no error", err)
	}

	if len(reviewer.LocalModels()) != len(wantLocal) {
		t.Fatalf("LocalModels() has %d entries, want %d — record the new definition in wantLocal", len(reviewer.LocalModels()), len(wantLocal))
	}

	for id, want := range wantLocal {
		model := registry.GetModel(localProvider, id)
		if model == nil {
			t.Errorf("GetModel(%q, %q) = nil, want the registered model", localProvider, id)
			continue
		}
		if model.Name != want.Name {
			t.Errorf("%s: Name = %q, want %q", id, model.Name, want.Name)
		}
		if model.Provider != localProvider {
			t.Errorf("%s: Provider = %q, want %q", id, model.Provider, localProvider)
		}
		if model.Api != ai.ApiOpenAICodexResponses {
			t.Errorf("%s: Api = %q, want %q", id, model.Api, ai.ApiOpenAICodexResponses)
		}
		if model.BaseURL != "https://chatgpt.com/backend-api" {
			t.Errorf("%s: BaseURL = %q, want the Codex backend", id, model.BaseURL)
		}
		if !model.Reasoning {
			t.Errorf("%s: Reasoning = false, want true", id)
		}
		if !model.SupportsImageInput() {
			t.Errorf("%s: SupportsImageInput() = false, want true — text and image were both declared", id)
		}
		if model.Cost != want.Cost {
			t.Errorf("%s: Cost = %+v, want %+v", id, model.Cost, want.Cost)
		}
		if model.ContextWindow != want.ContextWindow {
			t.Errorf("%s: ContextWindow = %d, want %d", id, model.ContextWindow, want.ContextWindow)
		}
		if model.MaxTokens != want.MaxTokens {
			t.Errorf("%s: MaxTokens = %d, want %d", id, model.MaxTokens, want.MaxTokens)
		}
	}
}

// The thinking map is the one field whose Go shape differs from the
// definition it was transcribed from — kern-link v0.1.1 types the values as
// *string — so the values are asserted rather than assumed.
func TestRegisterLocalModels_TheThinkingLevelMapCarriesTheDefinedValues(t *testing.T) {
	registry := providers.Models(nil)
	if err := reviewer.RegisterLocalModels(registry); err != nil {
		t.Fatalf("RegisterLocalModels() = %v, want no error", err)
	}

	want := map[ai.ModelThinkingLevel]string{ai.ThinkingMinimal: "low", ai.ThinkingXHigh: "xhigh"}
	for id := range wantLocal {
		model := registry.GetModel(localProvider, id)
		if model == nil {
			t.Fatalf("GetModel(%q, %q) = nil, want the registered model", localProvider, id)
		}
		if len(model.ThinkingLevelMap) != len(want) {
			t.Errorf("%s: ThinkingLevelMap has %d entries, want %d", id, len(model.ThinkingLevelMap), len(want))
		}
		for level, value := range want {
			got, ok := model.ThinkingLevelMap[level]
			if !ok || got == nil {
				t.Errorf("%s: ThinkingLevelMap[%q] is unset, want %q", id, level, value)
				continue
			}
			if *got != value {
				t.Errorf("%s: ThinkingLevelMap[%q] = %q, want %q", id, level, *got, value)
			}
		}
	}
}

// The registration is an addition. Every model kern-link's embedded catalog
// already serves under this provider stays reachable, and no other provider
// is disturbed — the decorator replaces one entry in place, keeping its id,
// name, base URL and refreshability.
func TestRegisterLocalModels_TheEmbeddedCatalogIsNotShadowed(t *testing.T) {
	before := providers.Models(nil)
	registry := providers.Models(nil)
	if err := reviewer.RegisterLocalModels(registry); err != nil {
		t.Fatalf("RegisterLocalModels() = %v, want no error", err)
	}

	original, augmented := before.GetProvider(localProvider), registry.GetProvider(localProvider)
	if original == nil || augmented == nil {
		t.Fatalf("GetProvider(%q) = (%v, %v), want both present", localProvider, original, augmented)
	}
	if augmented.Name() != original.Name() || augmented.BaseURL() != original.BaseURL() {
		t.Errorf("provider identity changed: (%q, %q), want (%q, %q)",
			augmented.Name(), augmented.BaseURL(), original.Name(), original.BaseURL())
	}
	if augmented.CanRefreshModels() != original.CanRefreshModels() {
		t.Errorf("CanRefreshModels() = %v, want %v", augmented.CanRefreshModels(), original.CanRefreshModels())
	}

	for _, model := range original.GetModels() {
		if registry.GetModel(localProvider, model.ID) == nil {
			t.Errorf("GetModel(%q, %q) = nil after registration, want the embedded model still reachable", localProvider, model.ID)
		}
	}
	if got, want := len(augmented.GetModels()), len(original.GetModels())+len(wantLocal); got != want {
		t.Errorf("the augmented provider serves %d models, want %d", got, want)
	}
	if got, want := len(registry.GetProviders()), len(before.GetProviders()); got != want {
		t.Errorf("the registry holds %d providers, want %d — registration must replace in place, not append", got, want)
	}
}

// The failure mode this registration exists to avoid is the silent one: a
// model the classifier cannot place is Unknown, Unknown is refused by every
// exclusion, and the tier resolves to no reviewer with the model looking
// perfectly well registered. Each local model must classify openai and be
// allowed under the shipped anthropic exclusion.
func TestLocalModels_ClassifyIntoTheOpenAIFamilyAndAreAllowed(t *testing.T) {
	exclusion := family.DefaultExclusion()
	for _, model := range reviewer.LocalModels() {
		if got := family.Classify(model.Provider, model.ID); got != "openai" {
			t.Errorf("Classify(%q, %q) = %q, want %q", model.Provider, model.ID, got, "openai")
		}
		if !exclusion.Allows(model.Provider, model.ID) {
			t.Errorf("DefaultExclusion().Allows(%q, %q) = false, want the model allowed to review", model.Provider, model.ID)
		}
	}
}

// A local definition never wins over the catalog's. The day a kern-link
// release ships one of these ids, the upstream entry is the one that answers
// and the local registration is inert — which is what makes deleting it an
// ordinary cleanup rather than a behaviour change.
func TestRegisterLocalModels_TheCatalogsOwnDefinitionWinsOnAnIDCollision(t *testing.T) {
	upstream := &ai.Model{
		ID:            "gpt-5.6-sol",
		Name:          "GPT-5.6 Sol (upstream)",
		Provider:      localProvider,
		Cost:          ai.ModelCost{Input: 7, Output: 42},
		ContextWindow: 400000,
	}
	registry := ai.CreateModels(nil)
	registry.SetProvider(ai.CreateProvider(ai.CreateProviderOptions{
		ID:     localProvider,
		Models: []*ai.Model{upstream},
		Api:    ai.StreamFuncs{},
	}))

	if err := reviewer.RegisterLocalModels(registry); err != nil {
		t.Fatalf("RegisterLocalModels() = %v, want no error", err)
	}

	model := registry.GetModel(localProvider, upstream.ID)
	if model == nil {
		t.Fatalf("GetModel(%q, %q) = nil, want the upstream model", localProvider, upstream.ID)
	}
	if model.Name != upstream.Name || model.ContextWindow != upstream.ContextWindow {
		t.Errorf("GetModel(%q, %q) = %+v, want the upstream definition kept", localProvider, upstream.ID, model)
	}
	if got, want := len(registry.GetProvider(localProvider).GetModels()), len(wantLocal); got != want {
		t.Errorf("the provider serves %d models, want %d — the colliding local definition must be dropped, not appended", got, want)
	}
}

// A provider no registry holds cannot serve a model, and a local definition
// aimed at one is a mistake that must be loud rather than a row that silently
// never appears.
func TestRegisterLocalModels_AnAbsentProviderIsAnError(t *testing.T) {
	err := reviewer.RegisterLocalModels(ai.CreateModels(nil))
	if err == nil {
		t.Fatalf("RegisterLocalModels() = nil, want an error naming the absent provider")
	}
}

// Registration must not disturb what the registry does with a model: the
// augmented provider is still the real one for auth resolution, which is what
// keeps a locally defined model reachable on the same credential as the rest
// of its provider's catalog.
func TestRegisterLocalModels_AuthStillResolvesThroughTheRealProvider(t *testing.T) {
	registry := providers.Models(nil)
	if err := reviewer.RegisterLocalModels(registry); err != nil {
		t.Fatalf("RegisterLocalModels() = %v, want no error", err)
	}

	model := registry.GetModel(localProvider, "gpt-5.6-sol")
	if model == nil {
		t.Fatalf("GetModel(%q, %q) = nil, want the registered model", localProvider, "gpt-5.6-sol")
	}
	// The in-memory credential store holds nothing, so this is the
	// unconfigured answer — (nil, nil) — rather than a failure. What is being
	// asserted is that the call reaches a provider at all.
	result, err := registry.GetAuth(context.Background(), model)
	if err != nil {
		t.Fatalf("GetAuth() = %v, want no error on an unconfigured store", err)
	}
	if result != nil {
		t.Errorf("GetAuth() = %+v, want (nil, nil) against an empty credential store", result)
	}
}
