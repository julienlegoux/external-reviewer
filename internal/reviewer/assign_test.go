package reviewer_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/julienlegoux/external-reviewer/internal/config"
	"github.com/julienlegoux/external-reviewer/internal/reviewer"
)

// threeTiers is the file a human actually writes: every tier assigned, one of
// them on a reseller so the id carries its vendor segment.
const threeTiers = `
[tiers.light]
provider = "google"
model    = "gemini-3.1-flash-lite"

[tiers.standard]
provider = "openai-codex"
model    = "gpt-5.5"

[tiers.heavy]
provider = "openrouter"
model    = "openai/gpt-5.5"
`

// tierConfig writes a tier assignment file and decodes it through
// internal/config — the real decoder over a real file, so a fixture cannot
// drift into a shape the config package would never produce.
func tierConfig(t *testing.T, content string) *config.Config {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writing config fixture: %v", err)
	}
	cfg, err := config.LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile(%s) error = %v, want nil", path, err)
	}
	return cfg
}

// envLookup is the injected environment every test in this file reads
// through. Nothing here calls os.Getenv or t.Setenv, so the precedence
// assertions are independent of the developer's own environment and of one
// another's ordering.
func envLookup(vars map[string]string) func(string) string {
	return func(name string) string { return vars[name] }
}

func TestAssign_ConfigFile_AssignsEachTierItsOwnModel(t *testing.T) {
	cfg := tierConfig(t, threeTiers)
	assigner := reviewer.Assigner{Config: cfg}

	tests := []struct {
		tier         string
		wantProvider string
		wantModel    string
	}{
		{"light", "google", "gemini-3.1-flash-lite"},
		{"standard", "openai-codex", "gpt-5.5"},
		{"heavy", "openrouter", "openai/gpt-5.5"},
	}
	for _, tc := range tests {
		t.Run(tc.tier, func(t *testing.T) {
			assignment, err := assigner.Assign(tc.tier)
			if err != nil {
				t.Fatalf("Assign(%q) error = %v, want nil", tc.tier, err)
			}
			if assignment.Provider != tc.wantProvider || assignment.Model != tc.wantModel {
				t.Errorf("Assign(%q) = %s/%s, want %s/%s",
					tc.tier, assignment.Provider, assignment.Model, tc.wantProvider, tc.wantModel)
			}
			if assignment.Source != reviewer.SourceConfig {
				t.Errorf("Source = %q, want %q", assignment.Source, reviewer.SourceConfig)
			}
			if assignment.Origin != cfg.Path {
				t.Errorf("Origin = %q, want the config path %q", assignment.Origin, cfg.Path)
			}
		})
	}
}

// TestAssign_ExplicitModel_BypassesEveryLowerLayer is the top of the chain:
// what --model will carry answers for any tier, including one the config and
// the environment both assign to something else.
func TestAssign_ExplicitModel_BypassesEveryLowerLayer(t *testing.T) {
	assigner := reviewer.Assigner{
		Explicit: "google/gemini-3.1-pro-preview",
		Getenv:   envLookup(map[string]string{"EXTERNAL_REVIEWER_TIER_STANDARD": "mistral/mistral-large"}),
		Config:   tierConfig(t, threeTiers),
	}

	assignment, err := assigner.Assign("standard")
	if err != nil {
		t.Fatalf("Assign() error = %v, want nil", err)
	}
	if assignment.Provider != "google" || assignment.Model != "gemini-3.1-pro-preview" {
		t.Errorf("Assign() = %s/%s, want google/gemini-3.1-pro-preview", assignment.Provider, assignment.Model)
	}
	if assignment.Source != reviewer.SourceFlag {
		t.Errorf("Source = %q, want %q", assignment.Source, reviewer.SourceFlag)
	}
}

// TestAssign_TierEnvironmentVariable_OverridesOnlyThatTier: the per-tier
// variables are surgical. One set variable must not disturb the two tiers it
// does not name, in the same run and the same Assigner.
func TestAssign_TierEnvironmentVariable_OverridesOnlyThatTier(t *testing.T) {
	assigner := reviewer.Assigner{
		Getenv: envLookup(map[string]string{"EXTERNAL_REVIEWER_TIER_STANDARD": "mistral/mistral-large"}),
		Config: tierConfig(t, threeTiers),
	}

	standard, err := assigner.Assign("standard")
	if err != nil {
		t.Fatalf("Assign(standard) error = %v, want nil", err)
	}
	if standard.Provider != "mistral" || standard.Model != "mistral-large" {
		t.Errorf("Assign(standard) = %s/%s, want mistral/mistral-large", standard.Provider, standard.Model)
	}
	if standard.Source != reviewer.SourceEnvironment {
		t.Errorf("Source = %q, want %q", standard.Source, reviewer.SourceEnvironment)
	}
	if standard.Origin != "EXTERNAL_REVIEWER_TIER_STANDARD" {
		t.Errorf("Origin = %q, want EXTERNAL_REVIEWER_TIER_STANDARD", standard.Origin)
	}

	for _, tc := range []struct{ tier, want string }{
		{"light", "google/gemini-3.1-flash-lite"},
		{"heavy", "openrouter/openai/gpt-5.5"},
	} {
		assignment, err := assigner.Assign(tc.tier)
		if err != nil {
			t.Fatalf("Assign(%q) error = %v, want nil", tc.tier, err)
		}
		if got := assignment.Provider + "/" + assignment.Model; got != tc.want {
			t.Errorf("Assign(%q) = %s, want %s — the standard override must leave it alone", tc.tier, got, tc.want)
		}
		if assignment.Source != reviewer.SourceConfig {
			t.Errorf("Assign(%q) Source = %q, want %q", tc.tier, assignment.Source, reviewer.SourceConfig)
		}
	}
}

// TestAssign_MalformedValue_IsDistinctFromNoReviewer: a value a human typed
// wrong is a mistake to report, not a tier to skip. Reporting it as "no
// assignment" would send the run down the silent-fallback path with the
// diagnostic that explains it thrown away.
func TestAssign_MalformedValue_IsDistinctFromNoReviewer(t *testing.T) {
	malformed := []string{"no-slash", "/id", "provider/", "/", "   "}

	for _, value := range malformed {
		t.Run("flag "+value, func(t *testing.T) {
			_, err := reviewer.Assigner{Explicit: value}.Assign("standard")
			assertMalformed(t, err, value, reviewer.SourceFlag)
		})
		t.Run("environment "+value, func(t *testing.T) {
			_, err := reviewer.Assigner{
				Getenv: envLookup(map[string]string{"EXTERNAL_REVIEWER_TIER_STANDARD": value}),
			}.Assign("standard")
			assertMalformed(t, err, value, reviewer.SourceEnvironment)
		})
	}
}

func assertMalformed(t *testing.T, err error, value string, want reviewer.Source) {
	t.Helper()

	if err == nil {
		t.Fatalf("Assign(%q) error = nil, want a malformed-assignment error", value)
	}
	if errors.Is(err, reviewer.ErrNoReviewer) {
		t.Errorf("Assign(%q) error matches ErrNoReviewer, want a distinct malformed error: %v", value, err)
	}
	var malformed *reviewer.MalformedAssignmentError
	if !errors.As(err, &malformed) {
		t.Fatalf("Assign(%q) error = %v, want a *MalformedAssignmentError", value, err)
	}
	if malformed.Value != value {
		t.Errorf("MalformedAssignmentError.Value = %q, want %q", malformed.Value, value)
	}
	if malformed.Source != want {
		t.Errorf("MalformedAssignmentError.Source = %q, want %q", malformed.Source, want)
	}
	// The message is what a human reads to find the knob they mistyped; the
	// value they typed has to be in it.
	if !strings.Contains(malformed.Error(), value) {
		t.Errorf("MalformedAssignmentError message = %q, want it to name %q", malformed.Error(), value)
	}
}

// TestAssign_ModelIDContainingSlashes_SplitsAtTheFirstOne: reseller ids carry
// their vendor segment, so provider/id is split once and the rest is the id
// verbatim.
func TestAssign_ModelIDContainingSlashes_SplitsAtTheFirstOne(t *testing.T) {
	assignment, err := reviewer.Assigner{Explicit: "openrouter/anthropic/claude-sonnet-4.5"}.Assign("standard")
	if err != nil {
		t.Fatalf("Assign() error = %v, want nil", err)
	}
	if assignment.Provider != "openrouter" || assignment.Model != "anthropic/claude-sonnet-4.5" {
		t.Errorf("Assign() = %s/%s, want openrouter/anthropic/claude-sonnet-4.5",
			assignment.Provider, assignment.Model)
	}
}

func TestAssign_NothingAssignsTheTier_IsNoReviewer(t *testing.T) {
	tests := []struct {
		name     string
		assigner reviewer.Assigner
	}{
		{"no config at all", reviewer.Assigner{}},
		{"a config that assigns other tiers", reviewer.Assigner{Config: tierConfig(t, "[tiers.light]\nprovider = \"google\"\nmodel = \"gemini-3.1-flash-lite\"\n")}},
		{"an empty environment variable", reviewer.Assigner{Getenv: envLookup(map[string]string{"EXTERNAL_REVIEWER_TIER_STANDARD": ""})}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.assigner.Assign("standard")
			if !errors.Is(err, reviewer.ErrNoReviewer) {
				t.Fatalf("Assign() error = %v, want one matching ErrNoReviewer", err)
			}
			var noReviewer *reviewer.NoReviewerError
			if !errors.As(err, &noReviewer) {
				t.Fatalf("Assign() error = %v, want a *NoReviewerError naming the rule", err)
			}
			if noReviewer.Reason != reviewer.ReasonUnassigned {
				t.Errorf("Reason = %q, want %q", noReviewer.Reason, reviewer.ReasonUnassigned)
			}
		})
	}
}

// TestAssign_TierOutsideTheVocabulary_ConsultsNothing keeps the environment
// from becoming the mechanism specs 05 refused the config file: a tier name
// nobody decided cannot be brought into existence by exporting a variable
// named after it.
func TestAssign_TierOutsideTheVocabulary_ConsultsNothing(t *testing.T) {
	_, err := reviewer.Assigner{
		Getenv: envLookup(map[string]string{"EXTERNAL_REVIEWER_TIER_ENORMOUS": "google/gemini-3.1-pro-preview"}),
	}.Assign("enormous")

	if !errors.Is(err, reviewer.ErrNoReviewer) {
		t.Fatalf("Assign(enormous) error = %v, want one matching ErrNoReviewer", err)
	}
}

// TestAssign_NilGetenv_SkipsTheEnvironmentLayer pins the injected-lookup
// contract: the zero Assigner reads no environment at all rather than falling
// back to os.Getenv, so a test that forgets to script one cannot pick up the
// developer's own variables.
func TestAssign_NilGetenv_SkipsTheEnvironmentLayer(t *testing.T) {
	t.Setenv("EXTERNAL_REVIEWER_TIER_STANDARD", "mistral/mistral-large")

	assignment, err := reviewer.Assigner{Config: tierConfig(t, threeTiers)}.Assign("standard")
	if err != nil {
		t.Fatalf("Assign() error = %v, want nil", err)
	}
	if assignment.Source != reviewer.SourceConfig {
		t.Errorf("Source = %q, want %q — a nil Getenv reads no environment", assignment.Source, reviewer.SourceConfig)
	}
}
