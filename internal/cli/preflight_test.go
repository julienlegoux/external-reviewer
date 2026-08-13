package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/julienlegoux/kern-link/ai"
	"github.com/julienlegoux/kern-link/ai/providers/faux"

	"github.com/julienlegoux/external-reviewer/internal/cli"
	"github.com/julienlegoux/external-reviewer/internal/fauxtest"
)

// fixtureProvider and fixtureModel are the provider and model this package's
// offline registry serves, and the pair selection_test.go's package-wide tier
// assignment fixture points `standard` at. They live in the test package
// because the binary itself no longer names a provider anywhere: tier
// resolution replaced the walking skeleton's hard-coded reviewer, and the one
// place a provider id may still be written down in non-test source is the
// family classifier's data table (TestBinary_NamesNoProviderOutsideTheFamilyTables).
const (
	fixtureProvider = "openai-codex"
	fixtureModel    = "gpt-5.5"
)

// registryWith builds the offline registry a run resolves against and
// streams from: kern-link's in-process faux provider under the fixture
// provider id, serving modelIDs, with auth scripted. No network, no bill, no
// credential store on disk. The handle it returns is the scripting surface —
// what the model answers, and how fast.
//
// The models carry a price sheet (fauxtest.RegistryOptions.Priced) so
// ai.CalculateCost has something to multiply: a run that reports $0.0000 for
// every turn cannot tell cost accounting that works from cost accounting
// that was never wired.
func registryWith(t *testing.T, auth ai.ProviderAuth, credentials ai.CredentialStore, tokensPerSecond float64, modelIDs ...string) (ai.Models, *faux.Handle) {
	t.Helper()
	return fauxtest.NewRegistry(t, fauxtest.RegistryOptions{
		ProviderID:      fixtureProvider,
		ModelIDs:        modelIDs,
		Auth:            auth,
		Credentials:     credentials,
		TokensPerSecond: tokensPerSecond,
		Priced:          true,
	})
}

// registry is registryWith for the tests that care about pre-flight rather
// than about the round trip: one unremarkable answer is scripted, because
// every path that gets past resolution now goes on to call the model.
func registry(t *testing.T, auth ai.ProviderAuth, credentials ai.CredentialStore, modelIDs ...string) ai.Models {
	t.Helper()
	models, handle := registryWith(t, auth, credentials, 0, modelIDs...)
	handle.SetResponses(faux.Step(faux.TextMessage(scriptedAnswer, nil)))
	return models
}

// scriptedAnswer is what the default registry's reviewer replies. Tests that
// assert on the report's exact bytes script their own.
const scriptedAnswer = "no findings\n"

// runReview invokes a syntactically valid review against models, returning
// the exit code and both streams.
func runReview(t *testing.T, models ai.Models) (int, string, string) {
	t.Helper()

	var stdout, stderr bytes.Buffer
	code := cli.RunForTest([]string{"review", "--allow", ".", "--prompt", "review this", t.TempDir()}, strings.NewReader(""), &stdout, &stderr, models)
	return code, stdout.String(), stderr.String()
}

// TestRun_ResolvableCredentialedModel_ProceedsPastPreflight is the only
// success path pre-flight has: the model is in the catalog and a credential
// reaches it, so the run continues into the round trip and terminates
// normally with the reviewer's answer on stdout.
func TestRun_ResolvableCredentialedModel_ProceedsPastPreflight(t *testing.T) {
	code, stdout, stderr := runReview(t, registry(t, fauxtest.CredentialedAuth("OAuth"), nil, fixtureModel))

	if code != 0 {
		t.Errorf("exit code = %d, want 0 (stderr: %q)", code, stderr)
	}
	if stdout != scriptedAnswer {
		t.Errorf("stdout = %q, want the reviewer's answer %q", stdout, scriptedAnswer)
	}
	// "stop" is kern-link's own StopReasonStop — faux.TextMessage's default
	// when no AssistantMessageOptions.StopReason is scripted (see
	// roundtrip_test.go's TestRun_SuccessfulTurn_DoneLineRendersModelsOwnStopReason
	// for the explicit no-translation-table proof).
	if fields := fauxtest.ParseDoneLine(t, stderr); fields.Stop != "stop" {
		t.Errorf("done stop = %q, want stop (the model's own reason)", fields.Stop)
	}
	if !strings.Contains(stderr, fixtureProvider+"/"+fixtureModel) {
		t.Errorf("stderr = %q, want it to name the resolved model", stderr)
	}
	if !strings.Contains(stderr, "OAuth") {
		t.Errorf("stderr = %q, want it to carry the auth source label", stderr)
	}
}

// TestRun_NotReached_ExitsOne covers both halves of SPECS' silent-fallback
// case: a model that is not in the catalog, and a provider with no
// credential configured. Neither is worth a line to the caller, but both
// must still leave stdout empty and emit the done line. The in/out=0
// assertion is exit_test.go's former TestRun_NoReviewerReached_ExitsOne,
// re-homed here now that this test drives the same exit-1 path for real
// through resolution rather than through a stubbed reviewer.
func TestRun_NotReached_ExitsOne(t *testing.T) {
	tests := []struct {
		name   string
		models func(t *testing.T) ai.Models
	}{
		{
			name: "model absent from the catalog",
			models: func(t *testing.T) ai.Models {
				t.Helper()
				return registry(t, fauxtest.CredentialedAuth("OAuth"), nil, "some-other-model")
			},
		},
		{
			name: "provider unconfigured, GetAuth returns (nil, nil)",
			models: func(t *testing.T) ai.Models {
				t.Helper()
				return registry(t, fauxtest.UnconfiguredAuth(), nil, fixtureModel)
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
			fields := fauxtest.ParseDoneLine(t, stderr)
			if fields.Stop != "no_reviewer" {
				t.Errorf("done stop = %q, want no_reviewer", fields.Stop)
			}
			if fields.In != "0" || fields.Out != "0" {
				t.Errorf("done in/out = %q/%q, want 0/0 (a path that never reached a model)", fields.In, fields.Out)
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
				auth, credentials := fauxtest.ExpiredOAuthAuth(t, fixtureProvider)
				return registry(t, auth, credentials, fixtureModel)
			},
		},
		{
			name: "ModelsError code auth",
			models: func(t *testing.T) ai.Models {
				t.Helper()
				return registry(t, fauxtest.BrokenAPIKeyAuth(), nil, fixtureModel)
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
			if fields := fauxtest.ParseDoneLine(t, stderr); fields.Stop != "failed" {
				t.Errorf("done stop = %q, want failed", fields.Stop)
			}
			// Exactly one, not merely at least one: the exit-code contract
			// issue 05 lands states the count, because a caller that copies
			// every error: line into its report must not get two for one
			// broken credential — nor zero, which is the exit-1 shape.
			if n := countErrorLines(stderr); n != 1 {
				t.Errorf("stderr = %q, want exactly one error: line, found %d", stderr, n)
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
	oauthAuth, oauthCredentials := fauxtest.ExpiredOAuthAuth(t, fixtureProvider)

	tests := []struct {
		name   string
		models ai.Models
	}{
		{"resolved successfully", registry(t, fauxtest.CredentialedAuth("OAuth"), nil, fixtureModel)},
		{"broken oauth credential", registry(t, oauthAuth, oauthCredentials, fixtureModel)},
		{"broken api key", registry(t, fauxtest.BrokenAPIKeyAuth(), nil, fixtureModel)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, stdout, stderr := runReview(t, tc.models)

			if strings.Contains(stdout, fauxtest.Secret) {
				t.Errorf("stdout leaked a credential value: %q", stdout)
			}
			if strings.Contains(stderr, fauxtest.Secret) {
				t.Errorf("stderr leaked a credential value: %q", stderr)
			}
		})
	}
}
