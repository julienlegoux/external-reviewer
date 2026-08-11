// Package fauxtest is the one importable offline test harness for
// external-reviewer: kern-link's in-process faux provider wired behind a
// scripted auth strategy, and the registry builder both internal/cli's and
// internal/reviewer's black-box tests resolve and stream against. Nothing
// here touches the network or a real credential store.
//
// It is a non-_test.go package so it can be imported from both cli_test and
// reviewer_test — Go test files cannot import one another across packages —
// but it exists purely to be imported by tests: no product file imports it.
package fauxtest

import (
	"testing"

	"github.com/julienlegoux/kern-link/ai"
	"github.com/julienlegoux/kern-link/ai/providers/faux"
)

// AuthProvider decorates a provider with a scripted auth strategy, leaving
// its identity and model list alone. It is how a test drives GetAuth's four
// documented outcomes — a credential, (nil, nil), a ModelsError with code
// "oauth", one with code "auth" — without a provider that can reach
// anything.
type AuthProvider struct {
	ai.Provider
	auth ai.ProviderAuth
}

// NewAuthProvider wraps base so its Auth method returns auth instead of
// base's own, unconditionally.
func NewAuthProvider(base ai.Provider, auth ai.ProviderAuth) AuthProvider {
	return AuthProvider{Provider: base, auth: auth}
}

// Auth returns the scripted auth strategy rather than base's.
func (p AuthProvider) Auth() ai.ProviderAuth { return p.auth }

// RegistryOptions configures NewRegistry's offline provider.
type RegistryOptions struct {
	// ProviderID is the id the faux provider serves under.
	ProviderID string
	// ModelIDs are the models the provider serves.
	ModelIDs []string
	// Auth is the scripted auth strategy GetAuth resolves through.
	Auth ai.ProviderAuth
	// Credentials is the credential store GetAuth reads from; nil uses
	// kern-link's default in-memory store.
	Credentials ai.CredentialStore
	// TokensPerSecond throttles response streaming so a run takes
	// measurable wall clock; 0 (the default) streams as fast as the
	// machine allows.
	TokensPerSecond float64
	// Priced adds a price sheet to every model, so ai.CalculateCost (or the
	// adapter's own service-tier-scaled figure) has something to multiply.
	// Tests that don't assert on cost leave this false.
	Priced bool
}

// NewRegistry builds the offline MutableModels a test resolves against and
// streams from: kern-link's in-process faux provider under
// opts.ProviderID, serving opts.ModelIDs, with auth scripted through
// opts.Auth. No network, no bill, no credential store on disk. The returned
// *faux.Handle is the scripting surface — what the model answers, and how
// fast.
func NewRegistry(t testing.TB, opts RegistryOptions) (ai.MutableModels, *faux.Handle) {
	t.Helper()

	definitions := make([]faux.ModelDefinition, 0, len(opts.ModelIDs))
	for _, id := range opts.ModelIDs {
		def := faux.ModelDefinition{ID: id}
		if opts.Priced {
			def.Cost = &ai.ModelCost{Input: 100, Output: 200}
		}
		definitions = append(definitions, def)
	}
	handle := faux.New(&faux.Options{
		Provider:        opts.ProviderID,
		Models:          definitions,
		TokensPerSecond: opts.TokensPerSecond,
	})

	models := ai.CreateModels(&ai.CreateModelsOptions{Credentials: opts.Credentials})
	models.SetProvider(NewAuthProvider(handle.Provider, opts.Auth))
	return models, handle
}
