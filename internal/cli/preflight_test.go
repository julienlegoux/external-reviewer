package cli_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/julienlegoux/kern-link/ai"
	"github.com/julienlegoux/kern-link/ai/providers/faux"

	"github.com/julienlegoux/external-reviewer/internal/cli"
	"github.com/julienlegoux/external-reviewer/internal/reviewer"
)

// preflightSecret is the credential value every scripted provider below
// hands back. Finding it on stdout or stderr is the failure this file exists
// to make impossible.
//
//nolint:gosec // G101: deliberately credential-shaped test data.
const preflightSecret = "sk-do-not-print-me-0123456789"

// authProvider decorates a provider with a scripted auth strategy, leaving
// its identity and model list alone — the seam that drives GetAuth's
// documented outcomes without a provider able to reach anything.
type authProvider struct {
	ai.Provider
	auth ai.ProviderAuth
}

func (p authProvider) Auth() ai.ProviderAuth { return p.auth }

// registry builds the offline registry a run resolves against: kern-link's
// in-process faux provider under the hard-coded provider id, serving
// modelIDs, with auth scripted. No network, no bill, no credential store on
// disk.
func registry(t *testing.T, auth ai.ProviderAuth, credentials ai.CredentialStore, modelIDs ...string) ai.Models {
	t.Helper()

	definitions := make([]faux.ModelDefinition, 0, len(modelIDs))
	for _, id := range modelIDs {
		definitions = append(definitions, faux.ModelDefinition{ID: id})
	}
	handle := faux.New(&faux.Options{Provider: reviewer.DefaultProviderID, Models: definitions})

	models := ai.CreateModels(&ai.CreateModelsOptions{Credentials: credentials})
	models.SetProvider(authProvider{Provider: handle.Provider, auth: auth})
	return models
}

// credentialedAuth resolves successfully, hiding preflightSecret in every
// field an AuthResult can hide one in, behind a Source label that is safe to
// print.
func credentialedAuth(source string) ai.ProviderAuth {
	return ai.ProviderAuth{APIKey: &ai.APIKeyAuth{
		Name: "Faux API key",
		Resolve: func(context.Context, ai.APIKeyResolveInput) (*ai.AuthResult, error) {
			bearer := "Bearer " + preflightSecret
			return &ai.AuthResult{
				Auth: ai.ModelAuth{
					APIKey:  preflightSecret,
					Headers: ai.ProviderHeaders{"authorization": &bearer},
				},
				Env:    ai.ProviderEnv{"FAUX_API_KEY": preflightSecret},
				Source: source,
			}, nil
		},
	}}
}

// unconfiguredAuth is the provider-unconfigured case kern-link reports as
// GetAuth returning (nil, nil).
func unconfiguredAuth() ai.ProviderAuth {
	return ai.ProviderAuth{APIKey: &ai.APIKeyAuth{
		Name: "Faux API key",
		Resolve: func(context.Context, ai.APIKeyResolveInput) (*ai.AuthResult, error) {
			return nil, nil
		},
	}}
}

// brokenAPIKeyAuth fails api-key resolution: a ModelsError with code "auth".
func brokenAPIKeyAuth() ai.ProviderAuth {
	return ai.ProviderAuth{APIKey: &ai.APIKeyAuth{
		Name: "Faux API key",
		Resolve: func(context.Context, ai.APIKeyResolveInput) (*ai.AuthResult, error) {
			return nil, errors.New("credential store unreadable")
		},
	}}
}

// expiredOAuthAuth stores an expired OAuth credential whose refresh fails: a
// ModelsError with code "oauth". The stored credential carries
// preflightSecret, so anything that logs a credential shows up in the
// no-leak assertions below.
func expiredOAuthAuth(t *testing.T) (ai.ProviderAuth, ai.CredentialStore) {
	t.Helper()

	store := ai.NewInMemoryCredentialStore()
	if _, err := store.Modify(context.Background(), reviewer.DefaultProviderID, func(ai.Credential) (ai.Credential, error) {
		return &ai.OAuthCredential{Refresh: preflightSecret, Access: preflightSecret, Expires: 0}, nil
	}); err != nil {
		t.Fatalf("seeding credential store: %v", err)
	}

	auth := ai.ProviderAuth{OAuth: &ai.OAuthAuth{
		Name: "Faux OAuth",
		Refresh: func(context.Context, *ai.OAuthCredential) (*ai.OAuthCredential, error) {
			return nil, errors.New("refresh token rejected")
		},
		ToAuth: func(context.Context, *ai.OAuthCredential) (ai.ModelAuth, error) {
			return ai.ModelAuth{APIKey: preflightSecret}, nil
		},
	}}
	return auth, store
}

// runReview invokes a syntactically valid review against models, returning
// the exit code and both streams.
func runReview(t *testing.T, models ai.Models) (int, string, string) {
	t.Helper()
	defer cli.SetModelsForTest(models)()

	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{"review", "--prompt", "review this", t.TempDir()}, strings.NewReader(""), &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

// TestRun_ResolvableCredentialedModel_ProceedsPastPreflight is the only
// success path pre-flight has: the model is in the catalog and a credential
// reaches it, so the run continues (to the round trip issue 05 adds) and
// terminates normally.
func TestRun_ResolvableCredentialedModel_ProceedsPastPreflight(t *testing.T) {
	code, stdout, stderr := runReview(t, registry(t, credentialedAuth("OAuth"), nil, reviewer.DefaultModelID))

	if code != 0 {
		t.Errorf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty until issue 05 wires the round trip", stdout)
	}
	if fields := parseDoneLine(t, stderr); fields.stop != "ok" {
		t.Errorf("done stop = %q, want ok", fields.stop)
	}
	if !strings.Contains(stderr, reviewer.DefaultProviderID+"/"+reviewer.DefaultModelID) {
		t.Errorf("stderr = %q, want it to name the resolved model", stderr)
	}
	if !strings.Contains(stderr, "OAuth") {
		t.Errorf("stderr = %q, want it to carry the auth source label", stderr)
	}
}

// TestRun_NotReached_ExitsOne covers both halves of SPECS' silent-fallback
// case: a model that is not in the catalog, and a provider with no
// credential configured. Neither is worth a line to the caller, but both
// must still leave stdout empty and emit the done line.
func TestRun_NotReached_ExitsOne(t *testing.T) {
	tests := []struct {
		name   string
		models func(t *testing.T) ai.Models
	}{
		{
			name: "model absent from the catalog",
			models: func(t *testing.T) ai.Models {
				t.Helper()
				return registry(t, credentialedAuth("OAuth"), nil, "some-other-model")
			},
		},
		{
			name: "provider unconfigured, GetAuth returns (nil, nil)",
			models: func(t *testing.T) ai.Models {
				t.Helper()
				return registry(t, unconfiguredAuth(), nil, reviewer.DefaultModelID)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, stdout, stderr := runReview(t, tc.models(t))

			if code != 1 {
				t.Errorf("exit code = %d, want 1 (stderr: %q)", code, stderr)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty on a failure", stdout)
			}
			if fields := parseDoneLine(t, stderr); fields.stop != "no_reviewer" {
				t.Errorf("done stop = %q, want no_reviewer", fields.stop)
			}
		})
	}
}

// TestRun_BrokenCredential_ExitsTwo is the sharp edge of the taxonomy: a
// credential that exists and does not work is something a human must fix, so
// it is exit 2 with the reason on stderr — not the silent 1 above.
func TestRun_BrokenCredential_ExitsTwo(t *testing.T) {
	tests := []struct {
		name   string
		models func(t *testing.T) ai.Models
	}{
		{
			name: "ModelsError code oauth",
			models: func(t *testing.T) ai.Models {
				t.Helper()
				auth, credentials := expiredOAuthAuth(t)
				return registry(t, auth, credentials, reviewer.DefaultModelID)
			},
		},
		{
			name: "ModelsError code auth",
			models: func(t *testing.T) ai.Models {
				t.Helper()
				return registry(t, brokenAPIKeyAuth(), nil, reviewer.DefaultModelID)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, stdout, stderr := runReview(t, tc.models(t))

			if code != 2 {
				t.Errorf("exit code = %d, want 2 (stderr: %q)", code, stderr)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty on a failure", stdout)
			}
			if fields := parseDoneLine(t, stderr); fields.stop != "failed" {
				t.Errorf("done stop = %q, want failed", fields.stop)
			}
			if !strings.Contains(stderr, "error:") {
				t.Errorf("stderr = %q, want the reason written to it", stderr)
			}
		})
	}
}

// TestRun_NoCredentialValueReachesTheStreams is the security boundary, not a
// nicety: this binary reads a private repository on the user's own
// credentials, and its stderr is a transcript a calling skill may keep. Only
// AuthResult.Source may travel — on every path, including the ones that fail
// while holding a credential.
func TestRun_NoCredentialValueReachesTheStreams(t *testing.T) {
	oauthAuth, oauthCredentials := expiredOAuthAuth(t)

	tests := []struct {
		name   string
		models ai.Models
	}{
		{"resolved successfully", registry(t, credentialedAuth("OAuth"), nil, reviewer.DefaultModelID)},
		{"broken oauth credential", registry(t, oauthAuth, oauthCredentials, reviewer.DefaultModelID)},
		{"broken api key", registry(t, brokenAPIKeyAuth(), nil, reviewer.DefaultModelID)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, stdout, stderr := runReview(t, tc.models)

			if strings.Contains(stdout, preflightSecret) {
				t.Errorf("stdout leaked a credential value: %q", stdout)
			}
			if strings.Contains(stderr, preflightSecret) {
				t.Errorf("stderr leaked a credential value: %q", stderr)
			}
		})
	}
}
