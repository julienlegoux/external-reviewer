package family

// The three tables below are the classifier's whole data surface. They are
// derived from kern-link v0.1.1's embedded catalog and verified against all of
// it by the catalog-wide test — adding a family is a line here plus a line in
// that test's record, never a key in a config file, because a file on a user's
// disk that can supply a family is the one mechanism able to weaken the
// fail-closed rule silently (specs 05 — Tier assignment schema).

// vendorProviders maps a provider that serves exactly one vendor's models to
// that vendor. Membership is that property, not corporate ownership: a
// provider is here when every model it can ever serve is one vendor's, so the
// id needs no reading — which is what makes openai-codex (OpenAI's Codex
// endpoint), azure-openai-responses (Azure's OpenAI service) and kimi-coding
// (Moonshot's coding plan) belong beside anthropic and google.
//
// The regional and plan variants are listed one by one rather than matched by
// prefix, so that a future provider called xiaomi-token-plan-relay has to be
// classified by hand instead of inheriting a family from a string that happens
// to start the same way.
var vendorProviders = map[string]Family{
	"anthropic": Anthropic,

	"openai":                 "openai",
	"openai-codex":           "openai",
	"azure-openai-responses": "openai",

	"google":        "google",
	"google-vertex": "google",

	"mistral":  "mistral",
	"deepseek": "deepseek",
	"xai":      "xai",

	"moonshotai":    "moonshotai",
	"moonshotai-cn": "moonshotai",
	"kimi-coding":   "moonshotai",

	"zai":           "zai",
	"zai-coding-cn": "zai",

	"minimax":    "minimax",
	"minimax-cn": "minimax",

	"xiaomi":                "xiaomi",
	"xiaomi-token-plan-ams": "xiaomi",
	"xiaomi-token-plan-cn":  "xiaomi",
	"xiaomi-token-plan-sgp": "xiaomi",

	"ant-ling": "inclusionai",
}

// resellers is the set of providers that serve several vendors' models. Their
// ids carry the vendor as their first segment — and where an id does not (a
// bare model name on github-copilot, opencode or cloudflare-ai-gateway, or
// fireworks' accounts/fireworks/models/… form), rule 3 answers Unknown and the
// model is excluded. That is the fail-closed cost, paid where it belongs:
// those providers resolve to no reviewer rather than to a guess.
//
// A provider in neither table is Unknown for every id, including an id that
// names a vendor. Recognising a provider is what makes its id shape
// trustworthy, so a new one is classified deliberately or not at all.
var resellers = map[string]bool{
	"amazon-bedrock":        true,
	"openrouter":            true,
	"vercel-ai-gateway":     true,
	"github-copilot":        true,
	"groq":                  true,
	"together":              true,
	"fireworks":             true,
	"nvidia":                true,
	"huggingface":           true,
	"cerebras":              true,
	"cloudflare-ai-gateway": true,
	"cloudflare-workers-ai": true,
	"opencode":              true,
	"opencode-go":           true,
}

// vendorAliases lists every family, with the other spellings that appear as an
// id's vendor segment across the catalog: openrouter writes x-ai and z-ai
// where amazon-bedrock writes xai and zai, huggingface writes zai-org and
// XiaomiMiMo, vercel-ai-gateway writes alibaba where bedrock writes qwen.
// Folding them here is what lets one exclusion list name a vendor once and
// have it hold across every provider that resells it.
var vendorAliases = map[Family][]string{
	Anthropic:     nil,
	"openai":      nil,
	"google":      nil,
	"amazon":      nil,
	"nvidia":      nil,
	"cohere":      nil,
	"ai21":        nil,
	"writer":      nil,
	"inception":   nil,
	"arcee-ai":    nil,
	"inclusionai": nil,
	"kwaipilot":   nil,
	"liquid":      nil,
	"poolside":    nil,
	"rekaai":      nil,
	"relace":      nil,
	"sakana":      nil,
	"sao10k":      nil,
	"thedrummer":  nil,
	"tencent":     nil,
	"upstage":     nil,
	"meituan":     nil,
	"interfaze":   nil,
	"essentialai": nil,

	"meta":       {"meta-llama"},
	"mistral":    {"mistralai"},
	"deepseek":   {"deepseek-ai"},
	"xai":        {"x-ai"},
	"moonshotai": {"moonshot"},
	"zai":        {"z-ai", "zai-org"},
	"minimax":    {"minimaxai"},
	"xiaomi":     {"xiaomimimo"},
	"qwen":       {"alibaba"},
	"bytedance":  {"bytedance-seed"},
	"stepfun":    {"stepfun-ai"},
	"ibm":        {"ibm-granite"},
}

// vendorSegments is vendorAliases inverted: every spelling, canonical name
// included, to its family. Lookups that miss return the zero value — Unknown —
// which is how rule 3 is the table's own behaviour rather than a branch.
var vendorSegments = invertAliases(vendorAliases)

func invertAliases(aliases map[Family][]string) map[string]Family {
	segments := make(map[string]Family, len(aliases)*2)
	for vendor, spellings := range aliases {
		segments[string(vendor)] = vendor
		for _, spelling := range spellings {
			segments[spelling] = vendor
		}
	}
	return segments
}

// regionalQualifiers are the deployment regions amazon-bedrock prefixes onto an
// id — us.anthropic.claude-…, global.anthropic.claude-… — which say where the
// model runs and nothing about who made it. Exactly one is stripped, and only
// when a segment follows it, so us.us.anthropic.… stays Unknown rather than
// being unwrapped until something matches.
var regionalQualifiers = map[string]bool{
	"us":     true,
	"eu":     true,
	"apac":   true,
	"global": true,
	"au":     true,
	"jp":     true,
}
