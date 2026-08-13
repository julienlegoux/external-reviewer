package family

import (
	"slices"
	"strings"
)

// defaultExcluded is the exclusion list this product ships with, expressed
// once, here: Anthropic, because the work under review is Claude's and a
// review of it by Claude is the failure this binary exists to prevent
// (scope 06 — Family exclusion rule).
var defaultExcluded = []Family{Anthropic}

// DefaultExcluded returns the families excluded when the caller names none.
func DefaultExcluded() []Family { return slices.Clone(defaultExcluded) }

// Exclusion is the rule a caller builds once from a list of family names and
// then asks about each model: "may this one review?" is one call with one
// answer, rather than a predicate every call site reassembles out of Classify.
//
// The zero Exclusion excludes no family by name — and still refuses an unknown
// one. That half is not a list entry and cannot be switched off.
type Exclusion struct {
	excluded map[Family]bool
}

// DefaultExclusion is the rule with the default list: Anthropic excluded.
func DefaultExclusion() Exclusion { return NewExclusion(namesOf(defaultExcluded)...) }

// NewExclusion builds the rule from family names in any spelling the catalog
// uses — case and surrounding space are folded, and an alias resolves to its
// family, so excluding "x-ai" excludes grok wherever it is resold.
//
// A name that is no family is kept verbatim and therefore never matches:
// NewExclusion is not the place to reject a typo, because refusing to build
// the rule would be a worse outcome than the caller's own diagnostic. Callers
// taking names from a human check them with Lookup first.
func NewExclusion(names ...string) Exclusion {
	excluded := make(map[Family]bool, len(names))
	for _, name := range names {
		normalized := normalize(name)
		if normalized == "" {
			continue
		}
		if vendor, ok := Lookup(normalized); ok {
			excluded[vendor] = true
			continue
		}
		excluded[Family(normalized)] = true
	}
	return Exclusion{excluded: excluded}
}

// Allows reports whether the model identified by provider and id may perform a
// review — classification and the rule in one call.
func (e Exclusion) Allows(provider, id string) bool {
	return e.AllowsFamily(Classify(provider, id))
}

// AllowsFamily reports whether f may perform a review. Unknown never may:
// that is the fail-closed default, and it holds for the zero Exclusion, an
// empty list and any list the caller passes.
func (e Exclusion) AllowsFamily(f Family) bool {
	if !f.Known() {
		return false
	}
	return !e.excluded[f]
}

// Excluded returns the families this rule refuses by name, sorted, for the
// diagnostics that explain why a tier resolved to no reviewer. The unknown
// family is not among them: it is refused by every rule and belongs to none.
func (e Exclusion) Excluded() []Family {
	families := make([]Family, 0, len(e.excluded))
	for f := range e.excluded {
		families = append(families, f)
	}
	slices.SortFunc(families, func(a, b Family) int { return strings.Compare(a.String(), b.String()) })
	return families
}

func namesOf(families []Family) []string {
	names := make([]string, len(families))
	for i, f := range families {
		names[i] = string(f)
	}
	return names
}
