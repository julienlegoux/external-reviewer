package reviewer

import (
	"fmt"
	"slices"

	"github.com/julienlegoux/kern-link/ai"
)

// The definitions below are model entries kern-link v0.1.1's embedded catalog
// predates. They are a departure from SPECS' dependency table and CONVENTIONS'
// pinning policy, which make that catalog the one source of model definitions,
// and the departure is recorded — with the condition that retires it — in
// docs/epics/epic-3-reviewer-selection-integration/drift/06-gpt-5-6-models-registered-locally.md.
//
// Adding another is one line in localModels plus its definition; removing the
// whole set, the day a kern-link release ships these ids, is deleting this
// file and its test. Two definitions of one model that disagree on price or
// context window are worse than either, so the release is a deletion rather
// than a merge — RegisterLocalModels drops any local definition the catalog
// has since grown so that the intervening state is inert rather than wrong.

var gpt56Sol = &ai.Model{
	ID:               "gpt-5.6-sol",
	Name:             "GPT-5.6 Sol",
	Api:              ai.ApiOpenAICodexResponses,
	Provider:         "openai-codex",
	BaseURL:          "https://chatgpt.com/backend-api",
	Reasoning:        true,
	ThinkingLevelMap: ai.ThinkingLevelMap{ai.ThinkingMinimal: thinking("low"), ai.ThinkingXHigh: thinking("xhigh")},
	Input:            []ai.Modality{ai.ModalityText, ai.ModalityImage},
	Cost:             ai.ModelCost{Input: 5, Output: 30, CacheRead: 0.5, CacheWrite: 6.25},
	ContextWindow:    272000,
	MaxTokens:        128000,
}

var gpt56Terra = &ai.Model{
	ID:               "gpt-5.6-terra",
	Name:             "GPT-5.6 Terra",
	Api:              ai.ApiOpenAICodexResponses,
	Provider:         "openai-codex",
	BaseURL:          "https://chatgpt.com/backend-api",
	Reasoning:        true,
	ThinkingLevelMap: ai.ThinkingLevelMap{ai.ThinkingMinimal: thinking("low"), ai.ThinkingXHigh: thinking("xhigh")},
	Input:            []ai.Modality{ai.ModalityText, ai.ModalityImage},
	Cost:             ai.ModelCost{Input: 2, Output: 12, CacheRead: 0.2, CacheWrite: 2.5},
	ContextWindow:    272000,
	MaxTokens:        128000,
}

var gpt56Luna = &ai.Model{
	ID:               "gpt-5.6-luna",
	Name:             "GPT-5.6 Luna",
	Api:              ai.ApiOpenAICodexResponses,
	Provider:         "openai-codex",
	BaseURL:          "https://chatgpt.com/backend-api",
	Reasoning:        true,
	ThinkingLevelMap: ai.ThinkingLevelMap{ai.ThinkingMinimal: thinking("low"), ai.ThinkingXHigh: thinking("xhigh")},
	Input:            []ai.Modality{ai.ModalityText, ai.ModalityImage},
	Cost:             ai.ModelCost{Input: 0.2, Output: 1.2, CacheRead: 0.02, CacheWrite: 0.25},
	ContextWindow:    272000,
	MaxTokens:        128000,
}

// localModels is the whole set, and the only list that grows: one line per
// model.
var localModels = []*ai.Model{
	gpt56Sol,
	gpt56Terra,
	gpt56Luna,
}

// LocalModels returns the locally defined models, in registration order. The
// pointers are the registry's own — a caller reads them and does not write
// them, exactly as it treats anything kern-link's catalog hands back.
func LocalModels() []*ai.Model { return slices.Clone(localModels) }

// RegisterLocalModels adds every locally defined model to models, through the
// one mechanism kern-link v0.1.1 offers for changing a registry's contents:
// ai.MutableModels.SetProvider, which upserts a provider by id. A provider's
// model list is not writable, so each affected provider is replaced in place
// by a decorator that delegates every method — id, name, base URL, headers,
// auth, refresh, streaming — to the real one and appends the local models to
// what its GetModels returns. Delegating rather than rebuilding is what keeps
// the credential strategy and the wire adapter kern-link wired, and what makes
// a dynamic provider's refreshed list still carry the additions.
//
// A local model whose id the provider already serves is dropped: the catalog's
// own definition wins, always. A provider absent from models is an error
// rather than a skipped entry — a definition aimed at a provider the registry
// does not hold can never resolve, and it must fail where a human sees it.
func RegisterLocalModels(models ai.MutableModels) error {
	byProvider := map[string][]*ai.Model{}
	order := []string{}
	for _, model := range localModels {
		if _, seen := byProvider[model.Provider]; !seen {
			order = append(order, model.Provider)
		}
		byProvider[model.Provider] = append(byProvider[model.Provider], model)
	}

	for _, id := range order {
		provider := models.GetProvider(id)
		if provider == nil {
			return fmt.Errorf("registering locally defined models: provider %q is not in the registry", id)
		}
		models.SetProvider(augmentedProvider{Provider: provider, extra: byProvider[id]})
	}
	return nil
}

// augmentedProvider is one built-in provider serving its own catalog plus the
// locally defined models. Everything but GetModels is the embedded
// Provider's.
type augmentedProvider struct {
	ai.Provider
	extra []*ai.Model
}

// GetModels returns the provider's own models first — so the catalog's
// definition is the one a lookup finds — followed by the local models it does
// not already carry.
func (p augmentedProvider) GetModels() []*ai.Model {
	own := p.Provider.GetModels()
	models := make([]*ai.Model, 0, len(own)+len(p.extra))
	models = append(models, own...)
	for _, model := range p.extra {
		if slices.ContainsFunc(own, func(existing *ai.Model) bool { return existing.ID == model.ID }) {
			continue
		}
		models = append(models, model)
	}
	return models
}

func thinking(level string) *string { return &level }
