package cli

import (
	"errors"

	"github.com/julienlegoux/external-reviewer/internal/reviewer"
)

// ErrNoReviewer classifies a run that never reached a model: exit code 1,
// SPECS' silent-fallback case that needs no line explaining it to the
// caller — no config, no tier entry, a model absent from the catalog, a
// family exclusion, an unconfigured provider. Wrap it with fmt.Errorf to add
// context; classify inspects it with errors.Is, so any wrap depth still
// resolves to exit 1.
//
// The sentinel is defined where it is produced — resolution, in
// internal/reviewer — and re-exported here because the exit-code contract is
// this package's. One value, so errors.Is answers the same question wherever
// it is asked.
var ErrNoReviewer = reviewer.ErrNoReviewer

// classify maps a run's terminal error to the process exit code SPECS
// fixes: 0 usable, 1 no reviewer ever reached (ErrNoReviewer, at any wrap
// depth), 2 reached and unusable — every other non-nil error, including
// usage errors and interruption. It inspects err with errors.Is only and
// never compares error text.
func classify(err error) int {
	if err == nil {
		return 0
	}
	if errors.Is(err, ErrNoReviewer) {
		return 1
	}
	return 2
}
