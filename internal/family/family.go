// Package family owns the notion kern-link's catalog does not carry: which
// vendor made a model, and whether that vendor may review this repository's
// work. ai.Model records ID, Name, Api and Provider and nothing about who
// built the thing, while amazon-bedrock, openrouter and vercel-ai-gateway all
// serve Anthropic's models — so a filter that went by provider would send
// Claude's work back to Claude (specs 04 — Model family classification).
//
// Classification is a pure function over (Provider, ID) against two data
// tables, and it fails closed: what neither table answers confidently is
// Unknown, and Unknown is refused by every Exclusion whatever the caller's
// list says. The asymmetry is the whole design. A false exclusion costs a
// native review, which SCOPE already calls a normal outcome; a false inclusion
// costs Claude reviewing Claude while reporting that it did not, which is the
// one failure this product cannot detect after the fact.
//
// The package depends on nothing — not on kern-link, not on this binary's
// config, flags or resolution — so the table above is testable offline against
// the whole embedded catalog.
package family

import "strings"

// Family is the vendor that made a model, as a lowercase name: "anthropic",
// "openai", "google". It is the unit selection is expressed in, never the
// provider, because one provider serves several vendors.
type Family string

const (
	// Unknown is the family of a model neither table answers for, and it is
	// deliberately the zero value: a Family nobody set is refused, and a
	// missing key in the tables below yields it without a branch.
	Unknown Family = ""

	// Anthropic is the family the default exclusion names — the one this
	// product exists to keep out of its own review.
	Anthropic Family = "anthropic"
)

// Known reports whether f is a family the classifier recognised.
func (f Family) Known() bool { return f != Unknown }

// String prints Unknown as "unknown" rather than as the empty string it is,
// so a diagnostic line naming a family never has a hole in it.
func (f Family) String() string {
	if f == Unknown {
		return "unknown"
	}
	return string(f)
}

// Classify returns the family of the model identified by provider and id, both
// kern-link's own identifiers verbatim, in the order specs 04 fixes:
//
//  1. the provider serves exactly one vendor's models — that vendor is the
//     family, and the id is never read;
//  2. the provider is a reseller or aggregator — the family is the id's vendor
//     segment, everything before the first "." or "/", after stripping a
//     leading regional qualifier;
//  3. neither answers confidently — Unknown.
//
// Matching folds case and is segment-exact, never substring: anthropic.claude-…
// matches on the whole segment "anthropic", so a model called
// not-anthropic-clone or a vendor called anthropicx is not silently
// reclassified into Anthropic's family — nor out of it.
func Classify(provider, id string) Family {
	normalized := normalize(provider)
	if vendor, ok := vendorProviders[normalized]; ok {
		return vendor
	}
	if !resellers[normalized] {
		return Unknown
	}
	return vendorSegment(id)
}

// Lookup resolves one spelling of a family name — canonical or any alias the
// catalog uses — to its family. Callers that take family names from a human
// (a flag, a config file) check them here first: a name that is not a family
// silently excludes nothing, which is the fail-open direction of this rule and
// the one worth a warning.
func Lookup(name string) (Family, bool) {
	vendor, ok := vendorSegments[normalize(name)]
	return vendor, ok
}

// vendorSegment applies rule 2 to a reseller's id.
func vendorSegment(id string) Family {
	segment, rest := splitSegment(normalize(id))
	if regionalQualifiers[segment] && rest != "" {
		segment, _ = splitSegment(rest)
	}
	// OpenRouter spells its floating aliases ~vendor/model-latest; the "~"
	// marks the alias, not a different vendor.
	segment = strings.TrimPrefix(segment, "~")

	// A missing key is the zero value, which is Unknown: rule 3 is the map's
	// own behaviour, not a branch that can be forgotten.
	return vendorSegments[segment]
}

// splitSegment returns the id's first segment and whatever follows it. Both
// separators the catalog uses divide a vendor from a model name:
// amazon-bedrock writes anthropic.claude-…, the gateways write
// anthropic/claude-….
func splitSegment(id string) (segment, rest string) {
	if i := strings.IndexAny(id, "./"); i >= 0 {
		return id[:i], id[i+1:]
	}
	return id, ""
}

func normalize(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
