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
)

// DefaultProviderID and DefaultModelID are the walking skeleton's single
// reviewer, hard-coded because Epic 1 ships no tier vocabulary, no --model
// flag and no config file. Both are kern-link's own identifiers verbatim,
// and the model is chosen by hand outside the Anthropic family so a review
// of Claude's work is never sent back to Claude — the family classifier that
// makes that a rule rather than a choice arrives in Epic 3.
const (
	DefaultProviderID = "openai-codex"
	DefaultModelID    = "gpt-5.5"
)

// ErrNoReviewer classifies a run that never reached a model: exit code 1,
// SPECS' silent-fallback case that needs no line explaining it to the caller
// — a model absent from the catalog, a provider with no credential
// configured. It is deliberately *not* returned for a credential that exists
// and does not work: that is exit 2, because something is wrong that a human
// must fix. Wrap it with fmt.Errorf to add context; callers inspect it with
// errors.Is, so any wrap depth still resolves to exit 1.
var ErrNoReviewer = errors.New("no reviewer reached")

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
}

// Resolver is the pre-flight that turns a provider and model id into a
// reachable, credentialed model. Epic 3 replaces the hard-coded ProviderID
// and ModelID with tier resolution; the sequence Resolve runs stays put.
type Resolver struct {
	Models     ai.Models
	ProviderID string
	ModelID    string
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
// The two failure classes it can return are the ones the exit-code
// classification turns on, produced here as typed errors so a single switch
// at the CLI boundary picks the code rather than each call site:
// ErrNoReviewer for a reviewer that was never reached (exit 1), and an error
// wrapping *ai.ModelsError for a credential that exists and is broken —
// code "oauth" or "auth" (exit 2).
func (r Resolver) Resolve(ctx context.Context) (Resolution, error) {
	if provider := r.Models.GetProvider(r.ProviderID); provider != nil && provider.CanRefreshModels() {
		if err := r.Models.Refresh(ctx, r.ProviderID); err != nil {
			r.warn(fmt.Sprintf("model catalog refresh failed for %s, using the last known model list: %v", r.ProviderID, err))
		}
	}

	model := r.Models.GetModel(r.ProviderID, r.ModelID)
	if model == nil {
		return Resolution{}, fmt.Errorf("looking up model %s/%s in the catalog: %w", r.ProviderID, r.ModelID, ErrNoReviewer)
	}

	result, err := r.Models.GetAuth(ctx, model)
	if err != nil {
		return Resolution{}, fmt.Errorf("resolving credentials for %s/%s: %w", r.ProviderID, r.ModelID, err)
	}
	if result == nil {
		return Resolution{}, fmt.Errorf("resolving credentials for %s/%s: provider unconfigured: %w", r.ProviderID, r.ModelID, ErrNoReviewer)
	}

	return Resolution{Model: model, AuthSource: result.Source}, nil
}

func (r Resolver) warn(message string) {
	if r.Warn != nil {
		r.Warn(message)
	}
}

// DefaultModels builds the registry resolution runs against in production:
// every provider kern-link ships, over its own cross-process-locked
// credential store at ~/.pi/agent/auth.json. This binary reads no credential
// itself, stores none and refreshes none — it hands kern-link the store's
// location and asks nothing else.
func DefaultModels() (ai.MutableModels, error) {
	path, err := auth.DefaultPath()
	if err != nil {
		return nil, fmt.Errorf("locating the credential store: %w", err)
	}
	return providers.Models(&ai.CreateModelsOptions{
		Credentials: auth.NewFileCredentialStore(path),
	}), nil
}
