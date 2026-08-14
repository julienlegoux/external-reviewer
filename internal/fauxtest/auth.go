package fauxtest

import (
	"context"
	"errors"
	"testing"

	"github.com/julienlegoux/kern-link/ai"
)

// Secret is the credential value every scripted provider in this package
// hands back. No assertion may ever find it outside kern-link: it stands in
// for the API key or bearer token a real AuthResult carries, and the point
// of the resolution seam is that only AuthResult.Source escapes it.
// A planted fake credential is the point here — this value exists to prove
// it never leaves kern-link.
//
//nolint:gosec // G101: deliberately credential-shaped test data.
const Secret = "sk-do-not-print-me-0123456789"

// CredentialedAuth resolves successfully, carrying Secret in every field an
// AuthResult can hide one in, behind a Source label that is safe to print.
func CredentialedAuth(source string) ai.ProviderAuth {
	return ai.ProviderAuth{APIKey: &ai.APIKeyAuth{
		Name: "Faux API key",
		Resolve: func(context.Context, ai.APIKeyResolveInput) (*ai.AuthResult, error) {
			bearer := "Bearer " + Secret
			return &ai.AuthResult{
				Auth: ai.ModelAuth{
					APIKey:  Secret,
					Headers: ai.ProviderHeaders{"authorization": &bearer},
				},
				Env:    ai.ProviderEnv{"FAUX_API_KEY": Secret},
				Source: source,
			}, nil
		},
	}}
}

// UnconfiguredAuth is the provider-unconfigured case: kern-link documents
// GetAuth as returning (nil, nil) when a provider is unknown or has no
// credential, and a nil Resolve result is how a provider reports that.
func UnconfiguredAuth() ai.ProviderAuth {
	return ai.ProviderAuth{APIKey: &ai.APIKeyAuth{
		Name: "Faux API key",
		Resolve: func(context.Context, ai.APIKeyResolveInput) (*ai.AuthResult, error) {
			return nil, nil
		},
	}}
}

// BrokenAPIKeyAuth fails api-key resolution, which kern-link reports as a
// ModelsError with code "auth".
func BrokenAPIKeyAuth() ai.ProviderAuth {
	return ai.ProviderAuth{APIKey: &ai.APIKeyAuth{
		Name: "Faux API key",
		Resolve: func(context.Context, ai.APIKeyResolveInput) (*ai.AuthResult, error) {
			return nil, errors.New("credential store unreadable")
		},
	}}
}

// ExpiredOAuthAuth stores an already-expired OAuth credential under
// providerID whose refresh fails, which kern-link reports as a ModelsError
// with code "oauth" — the broken-credential case a human has to fix by
// logging in again. The stored credential carries Secret, so anything that
// logs a credential shows up in a no-leak assertion.
func ExpiredOAuthAuth(t testing.TB, providerID string) (ai.ProviderAuth, ai.CredentialStore) {
	t.Helper()

	store := ai.NewInMemoryCredentialStore()
	if _, err := store.Modify(context.Background(), providerID, func(ai.Credential) (ai.Credential, error) {
		return &ai.OAuthCredential{Refresh: Secret, Access: Secret, Expires: 0}, nil
	}); err != nil {
		t.Fatalf("seeding credential store: %v", err)
	}

	auth := ai.ProviderAuth{OAuth: &ai.OAuthAuth{
		Name: "Faux OAuth",
		Refresh: func(context.Context, *ai.OAuthCredential) (*ai.OAuthCredential, error) {
			return nil, errors.New("refresh token rejected")
		},
		ToAuth: func(context.Context, *ai.OAuthCredential) (ai.ModelAuth, error) {
			return ai.ModelAuth{APIKey: Secret}, nil
		},
	}}
	return auth, store
}
