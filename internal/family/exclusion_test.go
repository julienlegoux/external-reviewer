package family_test

import (
	"slices"
	"testing"

	"github.com/julienlegoux/external-reviewer/internal/family"
)

func TestDefaultExclusion_ExcludesAnthropicAndOnlyAnthropic(t *testing.T) {
	if got := family.DefaultExcluded(); !slices.Equal(got, []family.Family{family.Anthropic}) {
		t.Fatalf("DefaultExcluded() = %v, want [anthropic]", got)
	}

	exclusion := family.DefaultExclusion()
	refused := []struct{ provider, id string }{
		{"anthropic", "claude-opus-4-5"},
		{"amazon-bedrock", "us.anthropic.claude-opus-4-5-20251101-v1:0"},
		{"openrouter", "anthropic/claude-opus-4-5"},
		{"vercel-ai-gateway", "anthropic/claude-sonnet-4-5"},
	}
	for _, c := range refused {
		if exclusion.Allows(c.provider, c.id) {
			t.Errorf("DefaultExclusion().Allows(%q, %q) = true, want false", c.provider, c.id)
		}
	}

	allowed := []struct{ provider, id string }{
		{"openai-codex", "gpt-5.5"},
		{"openrouter", "google/gemini-3.1-pro"},
		{"amazon-bedrock", "us.meta.llama3-1-70b-instruct-v1:0"},
		{"xai", "grok-4.3"},
	}
	for _, c := range allowed {
		if !exclusion.Allows(c.provider, c.id) {
			t.Errorf("DefaultExclusion().Allows(%q, %q) = false, want true", c.provider, c.id)
		}
	}
}

func TestNewExclusion_TakesTheCallersListInAnySpelling(t *testing.T) {
	exclusion := family.NewExclusion("Anthropic", "x-ai", "  google  ")

	refused := []struct{ provider, id string }{
		{"anthropic", "claude-opus-4-5"},
		{"xai", "grok-4.3"},
		{"openrouter", "x-ai/grok-4.3"},
		{"google-vertex", "gemini-3.1-pro-preview"},
	}
	for _, c := range refused {
		if exclusion.Allows(c.provider, c.id) {
			t.Errorf("Allows(%q, %q) = true, want false", c.provider, c.id)
		}
	}

	if !exclusion.Allows("openai-codex", "gpt-5.5") {
		t.Error(`Allows("openai-codex", "gpt-5.5") = false, want true`)
	}

	if got, want := exclusion.Excluded(), []family.Family{family.Anthropic, "google", "xai"}; !slices.Equal(got, want) {
		t.Errorf("Excluded() = %v, want %v", got, want)
	}
}

// An empty list excludes nothing — and still refuses an unknown family. That
// half is not a list entry and cannot be switched off: a false exclusion costs
// a native review, a false inclusion costs Claude reviewing Claude while
// reporting that it did not.
func TestExclusion_RefusesAnUnknownFamilyWhateverTheList(t *testing.T) {
	exclusions := map[string]family.Exclusion{
		"zero value":  {},
		"empty list":  family.NewExclusion(),
		"other lists": family.NewExclusion("google"),
		"default":     family.DefaultExclusion(),
	}

	for name, exclusion := range exclusions {
		if exclusion.AllowsFamily(family.Unknown) {
			t.Errorf("%s: AllowsFamily(Unknown) = true, want false", name)
		}
		if exclusion.Allows("some-new-provider", "anthropic/claude-opus-4-5") {
			t.Errorf("%s: Allows on an unrecognised provider = true, want false", name)
		}
		// The known families an empty list leaves alone are still allowed,
		// so the refusal above is the unknown family and not a blanket no.
		if name == "zero value" || name == "empty list" {
			if !exclusion.Allows("anthropic", "claude-opus-4-5") {
				t.Errorf("%s: Allows a known family = false, want true", name)
			}
		}
	}
}

// A name that is not a family is kept verbatim and never matches, which is why
// a caller taking the list from a human checks it with Lookup first — a typo
// that silently excluded nothing is the fail-open direction of this rule.
func TestNewExclusion_AnUnknownNameExcludesNothing(t *testing.T) {
	exclusion := family.NewExclusion("anthropci")

	if !exclusion.Allows("anthropic", "claude-opus-4-5") {
		t.Error(`Allows("anthropic", "claude-opus-4-5") = false, want true — a misspelt name must not excuse a real family`)
	}
	if _, ok := family.Lookup("anthropci"); ok {
		t.Error(`Lookup("anthropci") reported a known family, so the caller has no way to warn`)
	}
}

func TestExclusion_AllowsFamilyAndAllowsAgreeOnEveryModel(t *testing.T) {
	exclusion := family.DefaultExclusion()

	cases := []struct{ provider, id string }{
		{"anthropic", "claude-opus-4-5"},
		{"openai-codex", "gpt-5.5"},
		{"openrouter", "anthropic/claude-opus-4-5"},
		{"openrouter", "openrouter/auto"},
	}
	for _, c := range cases {
		want := exclusion.AllowsFamily(family.Classify(c.provider, c.id))
		if got := exclusion.Allows(c.provider, c.id); got != want {
			t.Errorf("Allows(%q, %q) = %t, AllowsFamily(Classify(...)) = %t", c.provider, c.id, got, want)
		}
	}
}
