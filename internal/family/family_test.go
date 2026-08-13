package family_test

import (
	"testing"

	"github.com/julienlegoux/external-reviewer/internal/family"
)

// A provider that serves exactly one vendor's models answers on its own, and
// the id is never consulted — which is also the first half of the
// substring-safety rule: a model called not-anthropic-clone is its provider's,
// not Anthropic's.
func TestClassify_AVendorProviderAnswersWithoutReadingTheId(t *testing.T) {
	cases := []struct {
		provider string
		id       string
		want     family.Family
	}{
		{"anthropic", "claude-opus-4-5", family.Anthropic},
		{"openai", "gpt-5.5", "openai"},
		{"openai-codex", "gpt-5.5", "openai"},
		{"azure-openai-responses", "gpt-5-codex", "openai"},
		{"google", "gemini-3.1-pro-preview", "google"},
		{"google-vertex", "gemini-3.1-pro-preview", "google"},
		{"mistral", "devstral-2512", "mistral"},
		{"deepseek", "deepseek-v4-pro", "deepseek"},
		{"xai", "grok-4.3", "xai"},
		{"moonshotai", "kimi-k2-thinking", "moonshotai"},
		{"kimi-coding", "kimi-for-coding", "moonshotai"},
		{"zai", "glm-5", "zai"},
		{"minimax", "MiniMax-M3", "minimax"},
		{"xiaomi", "mimo-v2-pro", "xiaomi"},
		{"xiaomi-token-plan-sgp", "mimo-v2-pro", "xiaomi"},
		{"ant-ling", "Ling-2.6-1T", "inclusionai"},

		// The substring trap, from the provider side: an id that mentions
		// another vendor cannot move a vendor provider's model out of its
		// own family.
		{"openai", "not-anthropic-clone", "openai"},
		{"openai", "anthropic.claude-opus-4-5", "openai"},
	}

	for _, c := range cases {
		if got := family.Classify(c.provider, c.id); got != c.want {
			t.Errorf("Classify(%q, %q) = %q, want %q", c.provider, c.id, got, c.want)
		}
	}
}

// A reseller or aggregator carries the vendor in the id's first segment, in
// both spellings the catalog uses: `vendor.model` on amazon-bedrock and
// `vendor/model` on the gateways.
func TestClassify_AResellerYieldsTheVendorSegmentOfTheId(t *testing.T) {
	cases := []struct {
		provider string
		id       string
		want     family.Family
	}{
		{"openrouter", "anthropic/claude-opus-4-5", family.Anthropic},
		{"openrouter", "google/gemini-3.1-pro", "google"},
		{"openrouter", "x-ai/grok-4.3", "xai"},
		{"openrouter", "meta-llama/llama-3.1-70b-instruct", "meta"},
		{"vercel-ai-gateway", "anthropic/claude-opus-4-5", family.Anthropic},
		{"vercel-ai-gateway", "alibaba/qwen-3-14b", "qwen"},
		{"amazon-bedrock", "anthropic.claude-opus-4-5-20251101-v1:0", family.Anthropic},
		{"amazon-bedrock", "amazon.nova-2-lite-v1:0", "amazon"},
		{"nvidia", "moonshotai/kimi-k2-thinking", "moonshotai"},
		{"together", "zai-org/GLM-5", "zai"},
		{"huggingface", "XiaomiMiMo/MiMo-V2-Flash", "xiaomi"},

		// OpenRouter's floating aliases carry a leading `~` on the vendor
		// segment and are the same vendor's models.
		{"openrouter", "~anthropic/claude-opus-latest", family.Anthropic},
		{"openrouter", "~openai/gpt-latest", "openai"},
	}

	for _, c := range cases {
		if got := family.Classify(c.provider, c.id); got != c.want {
			t.Errorf("Classify(%q, %q) = %q, want %q", c.provider, c.id, got, c.want)
		}
	}
}

func TestClassify_StripsARegionalQualifierBeforeTheVendorSegment(t *testing.T) {
	cases := []struct {
		id   string
		want family.Family
	}{
		{"us.anthropic.claude-opus-4-5-20251101-v1:0", family.Anthropic},
		{"eu.anthropic.claude-sonnet-4-5-20250929-v1:0", family.Anthropic},
		{"apac.anthropic.claude-sonnet-4-5-20250929-v1:0", family.Anthropic},
		{"global.anthropic.claude-opus-4-5-20251101-v1:0", family.Anthropic},
		{"au.anthropic.claude-sonnet-4-5-20250929-v1:0", family.Anthropic},
		{"jp.anthropic.claude-sonnet-4-5-20250929-v1:0", family.Anthropic},
		{"us.meta.llama3-1-70b-instruct-v1:0", "meta"},

		// Only one qualifier is stripped, and only when something follows it.
		{"us", family.Unknown},
		{"us.us.anthropic.claude-opus-4-5", family.Unknown},
	}

	for _, c := range cases {
		if got := family.Classify("amazon-bedrock", c.id); got != c.want {
			t.Errorf("Classify(%q, %q) = %q, want %q", "amazon-bedrock", c.id, got, c.want)
		}
	}
}

// The vendor segment matches whole or not at all. A name that merely contains
// a vendor's is a different vendor, and treating it as the same one is how a
// model gets silently reclassified into an allowed family.
func TestClassify_AVendorSegmentMatchesWholeOrNotAtAll(t *testing.T) {
	cases := []struct {
		provider string
		id       string
	}{
		{"openrouter", "anthropicx.foo"},
		{"openrouter", "anthropicx/claude-opus-4-5"},
		{"openrouter", "not-anthropic-clone/model-1"},
		{"openrouter", "anthropic-clone/model-1"},
		{"amazon-bedrock", "anthropicx.claude-opus-4-5"},
		{"amazon-bedrock", "xanthropic.claude-opus-4-5"},

		// The vendor segment is the id's first, never a later one: a model
		// whose *name* mentions a vendor is not that vendor's.
		{"openrouter", "someone/anthropic-clone"},
	}

	for _, c := range cases {
		got := family.Classify(c.provider, c.id)
		if got == family.Anthropic {
			t.Errorf("Classify(%q, %q) = %q, want anything but %q", c.provider, c.id, got, family.Anthropic)
		}
		if got.Known() {
			t.Errorf("Classify(%q, %q) = %q, want an unknown family", c.provider, c.id, got)
		}
	}
}

func TestClassify_FoldsCase(t *testing.T) {
	cases := []struct {
		provider string
		id       string
		want     family.Family
	}{
		{"amazon-bedrock", "Anthropic.Claude-Opus-4-5-20251101-v1:0", family.Anthropic},
		{"AMAZON-BEDROCK", "ANTHROPIC.CLAUDE-OPUS-4-5", family.Anthropic},
		{"amazon-bedrock", "US.Anthropic.Claude-Opus-4-5", family.Anthropic},
		{"OpenRouter", "Anthropic/Claude-Opus-4-5", family.Anthropic},
		{"Anthropic", "Claude-Opus-4-5", family.Anthropic},
		{"Together", "ZAI-Org/GLM-5", "zai"},
	}

	for _, c := range cases {
		if got := family.Classify(c.provider, c.id); got != c.want {
			t.Errorf("Classify(%q, %q) = %q, want %q", c.provider, c.id, got, c.want)
		}
	}
}

// Rule 3, the one the whole design rests on: anything the two tables do not
// answer confidently is unknown, and unknown is refused.
func TestClassify_AnythingTheTablesDoNotAnswerIsUnknown(t *testing.T) {
	cases := []struct {
		provider string
		id       string
	}{
		{"some-new-provider", "a-model"},
		{"some-new-provider", "anthropic/claude-opus-4-5"},
		{"", ""},
		{"openrouter", ""},
		{"openrouter", "auto"},
		{"openrouter", "openrouter/auto"},
		{"openrouter", "newvendor/some-model"},
		{"github-copilot", "claude-sonnet-4.5"},
		{"fireworks", "accounts/fireworks/models/deepseek-v4-pro"},
	}

	for _, c := range cases {
		got := family.Classify(c.provider, c.id)
		if got.Known() {
			t.Errorf("Classify(%q, %q) = %q, want unknown", c.provider, c.id, got)
		}
		if got != family.Unknown {
			t.Errorf("Classify(%q, %q) = %q, want the Unknown zero value", c.provider, c.id, got)
		}
		if family.DefaultExclusion().Allows(c.provider, c.id) {
			t.Errorf("DefaultExclusion().Allows(%q, %q) = true, want an unknown family refused", c.provider, c.id)
		}
	}
}

func TestFamily_UnknownPrintsAsUnknown(t *testing.T) {
	if got := family.Unknown.String(); got != "unknown" {
		t.Errorf("Unknown.String() = %q, want %q", got, "unknown")
	}
	if got := family.Anthropic.String(); got != "anthropic" {
		t.Errorf("Anthropic.String() = %q, want %q", got, "anthropic")
	}
	if family.Unknown.Known() {
		t.Error("Unknown.Known() = true, want false")
	}
	if !family.Anthropic.Known() {
		t.Error("Anthropic.Known() = false, want true")
	}
}

func TestLookup_ResolvesEverySpellingOfAFamilyAndRefusesTheRest(t *testing.T) {
	known := []struct {
		name string
		want family.Family
	}{
		{"anthropic", family.Anthropic},
		{"Anthropic", family.Anthropic},
		{"  anthropic  ", family.Anthropic},
		{"x-ai", "xai"},
		{"mistralai", "mistral"},
		{"meta-llama", "meta"},
	}
	for _, c := range known {
		got, ok := family.Lookup(c.name)
		if !ok || got != c.want {
			t.Errorf("Lookup(%q) = %q, %t, want %q, true", c.name, got, ok, c.want)
		}
	}

	for _, name := range []string{"", "anthropci", "not-a-vendor", "unknown"} {
		if got, ok := family.Lookup(name); ok {
			t.Errorf("Lookup(%q) = %q, true, want false", name, got)
		}
	}
}
