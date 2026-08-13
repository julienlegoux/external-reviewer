package family_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/julienlegoux/external-reviewer/internal/family"
	"github.com/julienlegoux/kern-link/ai/catalog"
)

// anthropicIDPrefixes spells out, literally, every way an id in the embedded
// catalog declares Anthropic as its vendor — the reseller `.` form with and
// without each regional qualifier, the aggregator `/` form, and OpenRouter's
// floating-alias `~` form. They are written out rather than derived from the
// classifier's own tables so that this test still fails when those tables are
// wrong.
var anthropicIDPrefixes = []string{
	"anthropic.",
	"anthropic/",
	"~anthropic/",
	"us.anthropic.",
	"eu.anthropic.",
	"apac.anthropic.",
	"global.anthropic.",
	"au.anthropic.",
	"jp.anthropic.",
}

// The executable form of the concept's central warning: amazon-bedrock and the
// aggregators serve Anthropic's models, so a filter that went by provider
// would send Claude's work back to Claude. Every id in the embedded catalog
// that declares Anthropic as its vendor classifies as anthropic, and the
// default exclusion refuses it.
func TestClassify_NoAnthropicIDInTheEmbeddedCatalogClassifiesAsAnythingElse(t *testing.T) {
	exclusion := family.DefaultExclusion()
	checked := 0

	for _, provider := range catalog.Providers() {
		for _, model := range catalog.BuiltinModels(provider) {
			id := strings.ToLower(model.ID)
			if !hasAnyPrefix(id, anthropicIDPrefixes) {
				continue
			}
			checked++
			if got := family.Classify(provider, model.ID); got != family.Anthropic {
				t.Errorf("Classify(%q, %q) = %q, want %q", provider, model.ID, got, family.Anthropic)
			}
			if exclusion.Allows(provider, model.ID) {
				t.Errorf("DefaultExclusion().Allows(%q, %q) = true, want an Anthropic model refused", provider, model.ID)
			}
		}
	}

	// Without this the test passes vacuously the day kern-link renames the
	// vendor segment, which is the exact change it exists to catch.
	if checked < 20 {
		t.Errorf("only %d embedded ids declare Anthropic as their vendor, want at least 20 — the id shapes this test names have changed", checked)
	}
}

// embeddedFamilies is what the whole embedded catalog classifies to today, one
// line per provider. `unknown` is a legitimate entry: an id shape neither table
// answers confidently is excluded, which costs a native review and never a
// review of Claude's work by Claude.
//
// This is the loud half of the catalog sweep. A kern-link sync that adds a
// provider, drops one, or introduces an id shape whose vendor segment is not in
// the classifier's table changes one of these lines and fails here, naming the
// models — rather than silently degrading them to "excluded" and leaving a tier
// mysteriously unresolvable.
var embeddedFamilies = map[string][]family.Family{
	"amazon-bedrock":         {"amazon", "anthropic", "deepseek", "google", "meta", "minimax", "mistral", "moonshotai", "nvidia", "openai", "qwen", "writer", "xai", "zai"},
	"ant-ling":               {"inclusionai"},
	"anthropic":              {"anthropic"},
	"azure-openai-responses": {"openai"},
	"cerebras":               {family.Unknown},
	"cloudflare-ai-gateway":  {family.Unknown},
	"cloudflare-workers-ai":  {family.Unknown},
	"deepseek":               {"deepseek"},
	"fireworks":              {family.Unknown},
	"github-copilot":         {family.Unknown},
	"google":                 {"google"},
	"google-vertex":          {"google"},
	"groq":                   {"meta", "openai", "qwen", family.Unknown},
	"huggingface":            {"deepseek", "google", "meta", "minimax", "moonshotai", "openai", "qwen", "stepfun", "xiaomi", "zai"},
	"kimi-coding":            {"moonshotai"},
	"minimax":                {"minimax"},
	"minimax-cn":             {"minimax"},
	"mistral":                {"mistral"},
	"moonshotai":             {"moonshotai"},
	"moonshotai-cn":          {"moonshotai"},
	"nvidia":                 {"meta", "minimax", "mistral", "moonshotai", "nvidia", "openai", "qwen", "stepfun", "zai"},
	"openai":                 {"openai"},
	"openai-codex":           {"openai"},
	"opencode":               {family.Unknown},
	"opencode-go":            {family.Unknown},
	"openrouter":             {"ai21", "amazon", "anthropic", "arcee-ai", "bytedance", "cohere", "deepseek", "google", "ibm", "inception", "inclusionai", "kwaipilot", "liquid", "meta", "minimax", "mistral", "moonshotai", "nvidia", "openai", "poolside", "qwen", "rekaai", "relace", "sakana", "sao10k", "stepfun", "tencent", "thedrummer", "upstage", "xai", "xiaomi", "zai", family.Unknown},
	"together":               {"deepseek", "essentialai", "google", "meta", "minimax", "moonshotai", "nvidia", "openai", "qwen", "zai"},
	"vercel-ai-gateway":      {"amazon", "anthropic", "arcee-ai", "bytedance", "cohere", "deepseek", "google", "inception", "interfaze", "kwaipilot", "meituan", "meta", "minimax", "mistral", "moonshotai", "nvidia", "openai", "qwen", "sakana", "stepfun", "xai", "xiaomi", "zai"},
	"xai":                    {"xai"},
	"xiaomi":                 {"xiaomi"},
	"xiaomi-token-plan-ams":  {"xiaomi"},
	"xiaomi-token-plan-cn":   {"xiaomi"},
	"xiaomi-token-plan-sgp":  {"xiaomi"},
	"zai":                    {"zai"},
	"zai-coding-cn":          {"zai"},
}

func TestClassify_EveryEmbeddedModelClassifiesToTheRecordedFamilies(t *testing.T) {
	providers := catalog.Providers()
	if len(providers) == 0 {
		t.Fatal("kern-link's embedded catalog reported no providers")
	}

	for _, provider := range providers {
		want, recorded := embeddedFamilies[provider]
		if !recorded {
			t.Errorf("provider %q is in kern-link's embedded catalog and not in embeddedFamilies: classify its ids and record the line", provider)
			continue
		}

		got := map[family.Family][]string{}
		for _, model := range catalog.BuiltinModels(provider) {
			classified := family.Classify(provider, model.ID)
			got[classified] = append(got[classified], model.ID)
		}

		for _, unexpected := range difference(keys(got), want) {
			t.Errorf("%s classifies %d model(s) as %s, which is not recorded: %s",
				provider, len(got[unexpected]), unexpected, examples(got[unexpected]))
		}
		for _, missing := range difference(want, keys(got)) {
			t.Errorf("%s no longer classifies any model as %s, which is still recorded", provider, missing)
		}
	}

	for provider := range embeddedFamilies {
		if !slices.Contains(providers, provider) {
			t.Errorf("provider %q is recorded in embeddedFamilies and no longer in kern-link's embedded catalog", provider)
		}
	}
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

func keys(m map[family.Family][]string) []family.Family {
	out := make([]family.Family, 0, len(m))
	for f := range m {
		out = append(out, f)
	}
	return out
}

// difference returns the members of a that are absent from b, sorted by how
// they print so that "unknown" reads as a family rather than as an empty cell.
func difference(a, b []family.Family) []family.Family {
	out := []family.Family{}
	for _, f := range a {
		if !slices.Contains(b, f) {
			out = append(out, f)
		}
	}
	slices.SortFunc(out, func(x, y family.Family) int { return strings.Compare(x.String(), y.String()) })
	return out
}

func examples(ids []string) string {
	const most = 5
	shown := ids
	suffix := ""
	if len(shown) > most {
		shown, suffix = shown[:most], fmt.Sprintf(", and %d more", len(ids)-most)
	}
	return strings.Join(shown, ", ") + suffix
}
