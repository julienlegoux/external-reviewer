// Package reviewer owns the foreign model that performs a review: resolving
// a provider and model id to a reachable, credentialed model before any
// request is sent, and the registry of providers that resolution runs
// against.
package reviewer

import (
	"context"
	"errors"
	"fmt"

	"github.com/julienlegoux/kern-link/ai"
	"github.com/julienlegoux/kern-link/ai/auth"
	"github.com/julienlegoux/kern-link/ai/providers"

	"github.com/julienlegoux/external-reviewer/internal/family"
)

// ErrNoReviewer classifies a run that never reached a model: exit code 1,
// SPECS' silent-fallback case that needs no line explaining it to the caller
// — a model absent from the catalog, a provider with no credential
// configured. It is deliberately *not* returned for a credential that exists
// and does not work: that is exit 2, because something is wrong that a human
// must fix. Wrap it with fmt.Errorf to add context; callers inspect it with
// errors.Is, so any wrap depth still resolves to exit 1.
var ErrNoReviewer = errors.New("no reviewer reached")

// Reason is which rule ended the run at ErrNoReviewer. The sentinel alone
// says only "exit 1"; a user whose correctly spelled tier resolves to nothing
// needs to be told *which* rule excluded it, and a caller that has to parse
// that out of a message string will get it wrong. The `model` stderr line and
// the tiers command (issues 05 and 06) both switch on this.
type Reason string

const (
	// ReasonUnassigned — no layer of the chain assigns this tier.
	ReasonUnassigned Reason = "unassigned"
	// ReasonNotInCatalog — the assigned model is absent from kern-link's
	// catalog, after a refresh if the provider has one.
	ReasonNotInCatalog Reason = "not-in-catalog"
	// ReasonFamilyExcluded — the model's vendor is on the exclusion list.
	ReasonFamilyExcluded Reason = "family-excluded"
	// ReasonFamilyUnknown — the classifier could not tell who made the model,
	// and an unknown family is excluded by every rule (specs 04). This is a
	// distinct reason rather than a shade of the one above because the fix is
	// different: an excluded family was excluded on purpose, an unknown one is
	// a gap in the classifier's data tables.
	ReasonFamilyUnknown Reason = "family-unknown"
	// ReasonUnconfigured — GetAuth returned (nil, nil): this machine has no
	// credential for the provider. An *absent* credential is exit 1; a
	// *broken* one is exit 2 and never arrives here (specs 13).
	ReasonUnconfigured Reason = "unconfigured"
)

// NoReviewerError is ErrNoReviewer with the rule that produced it named. It
// unwraps to the sentinel, so every existing errors.Is check still classifies
// it as exit 1 at any wrap depth.
type NoReviewerError struct {
	Reason   Reason
	Provider string
	ModelID  string
	// Family is the classification that refused the model, set only for the
	// two family reasons. It is family.Unknown for ReasonFamilyUnknown, which
	// prints as "unknown".
	Family family.Family
}

func (e *NoReviewerError) Error() string {
	switch e.Reason {
	case ReasonUnassigned:
		return "no model is assigned"
	case ReasonNotInCatalog:
		return fmt.Sprintf("%s/%s is not in the model catalog", e.Provider, e.ModelID)
	case ReasonFamilyExcluded:
		return fmt.Sprintf("%s/%s is a %s model, and the %s family is excluded from reviewing", e.Provider, e.ModelID, e.Family, e.Family)
	case ReasonFamilyUnknown:
		return fmt.Sprintf("the vendor family of %s/%s could not be determined, and an unknown family is never allowed to review", e.Provider, e.ModelID)
	case ReasonUnconfigured:
		return fmt.Sprintf("%s has no credential configured on this machine", e.Provider)
	default:
		return fmt.Sprintf("%s/%s: %s", e.Provider, e.ModelID, e.Reason)
	}
}

// Unwrap is what keeps ErrNoReviewer the single classification callers test.
func (e *NoReviewerError) Unwrap() error { return ErrNoReviewer }

// Resolution is what a successful pre-flight yields. It carries the model
// and the source label of the credential that reached it — and deliberately
// nothing else. kern-link resolves credentials wholly and re-resolves them
// per request, so an AuthResult has no reason to travel any further into
// this binary; keeping it out of this struct is what makes "no credential
// value can reach stdout or stderr" a property of the type rather than a
// habit of every call site.
type Resolution struct {
	Model *ai.Model
	// AuthSource is AuthResult.Source: a label like "OAuth" or
	// "ANTHROPIC_API_KEY", never a secret.
	AuthSource string
	// Assignment is which layer of the precedence chain named this model, and
	// the variable or file it named it in — what the `model` stderr line and
	// the `tiers` command print, because "why did this tier resolve to that?"
	// cannot be answered from the provider and id alone. It is set by
	// [Chain.ResolveTier]; the zero value means the caller named the pair
	// itself and already knows.
	Assignment Assignment
}

// Resolver is the pre-flight that turns a provider and model id into a
// reachable, credentialed model. Epic 3 replaces the hard-coded ProviderID
// and ModelID with tier resolution; the sequence Resolve runs stays put.
type Resolver struct {
	Models     ai.Models
	ProviderID string
	ModelID    string
	// Exclusion is the family rule the gate applies. A nil Exclusion is the
	// product default — Anthropic excluded — rather than the zero value,
	// because the zero Exclusion excludes no family by name and a call site
	// that simply does not set the field must land on the fail-closed side of
	// that difference (specs 04).
	Exclusion *family.Exclusion
	// Warn receives non-fatal diagnostics — currently only a failed catalog
	// refresh, which leaves the provider's last-known model list in place.
	// Optional; a nil Warn discards them.
	Warn func(message string)
}

// Resolve runs the pre-flight in the order SPECS fixes, and can end a run
// before a single request is sent.
//
// Refresh comes first, and only for providers that report
// CanRefreshModels — openrouter, vercel-ai-gateway, nvidia and
// github-copilot hold no models at all until refreshed, so looking up the
// model first would make a correctly configured provider resolve to "model
// not found" and fall back silently, which is precisely the failure the
// ordering exists to prevent. A refresh that fails is not fatal on its own:
// kern-link keeps the last-known list, so the lookup below decides.
//
// The family gate sits between the catalog lookup and the credential check,
// and that placement is load-bearing in both directions: it wants the model
// the catalog actually returned, and an excluded model must never cause a
// credential to be resolved for it — GetAuth reaches kern-link's credential
// store and can rewrite what it finds there.
//
// The two failure classes it can return are the ones the exit-code
// classification turns on, produced here as typed errors so a single switch
// at the CLI boundary picks the code rather than each call site:
// ErrNoReviewer for a reviewer that was never reached (exit 1) — as a
// *NoReviewerError carrying which rule refused it — and an error wrapping
// *ai.ModelsError for a credential that exists and is broken, code "oauth" or
// "auth" (exit 2).
func (r Resolver) Resolve(ctx context.Context) (Resolution, error) {
	if provider := r.Models.GetProvider(r.ProviderID); provider != nil && provider.CanRefreshModels() {
		if err := r.Models.Refresh(ctx, r.ProviderID); err != nil {
			r.warn(fmt.Sprintf("model catalog refresh failed for %s, using the last known model list: %v", r.ProviderID, err))
		}
	}

	model := r.Models.GetModel(r.ProviderID, r.ModelID)
	if model == nil {
		return Resolution{}, r.noReviewer(ReasonNotInCatalog, family.Unknown)
	}

	if vendor := family.Classify(r.ProviderID, model.ID); !r.exclusion().AllowsFamily(vendor) {
		reason := ReasonFamilyExcluded
		if !vendor.Known() {
			reason = ReasonFamilyUnknown
		}
		return Resolution{}, r.noReviewer(reason, vendor)
	}

	result, err := r.Models.GetAuth(ctx, model)
	if err != nil {
		return Resolution{}, fmt.Errorf("resolving credentials for %s/%s: %w", r.ProviderID, r.ModelID, err)
	}
	if result == nil {
		return Resolution{}, r.noReviewer(ReasonUnconfigured, family.Unknown)
	}

	return Resolution{Model: model, AuthSource: result.Source}, nil
}

// exclusion is the rule the gate applies: the caller's when it set one, the
// product default otherwise. Never the zero Exclusion, which would exclude no
// family by name.
func (r Resolver) exclusion() family.Exclusion {
	if r.Exclusion == nil {
		return family.DefaultExclusion()
	}
	return *r.Exclusion
}

func (r Resolver) noReviewer(reason Reason, vendor family.Family) error {
	return &NoReviewerError{Reason: reason, Provider: r.ProviderID, ModelID: r.ModelID, Family: vendor}
}

func (r Resolver) warn(message string) {
	if r.Warn != nil {
		r.Warn(message)
	}
}

// DefaultModels builds the registry resolution runs against in production:
// every provider kern-link ships, over its own cross-process-locked
// credential store at ~/.pi/agent/auth.json, plus the models defined locally
// because the pinned release's embedded catalog predates them (see
// catalog.go). This binary reads no credential itself, stores none and
// refreshes none — it hands kern-link the store's location and asks nothing
// else.
func DefaultModels() (ai.MutableModels, error) {
	path, err := auth.DefaultPath()
	if err != nil {
		return nil, fmt.Errorf("locating the credential store: %w", err)
	}
	models := providers.Models(&ai.CreateModelsOptions{
		Credentials: auth.NewFileCredentialStore(path),
	})
	if err := RegisterLocalModels(models); err != nil {
		return nil, err
	}
	return models, nil
}
