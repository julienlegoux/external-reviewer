# Log

## 2026-08-09

* **Creation**: Established [Epic 1: Walking skeleton](/epic-1-walking-skeleton/EPIC_1.md)
  — milestone 1, issue #1.

* **Creation**: Established
  [Epic 2: Read-only agentic loop](/epic-2-read-only-agentic-loop/EPIC_2.md)
  — milestone 2, issue #2.

* **Creation**: Established
  [Epic 3: Reviewer selection and pipeline integration](/epic-3-reviewer-selection-integration/EPIC_3.md)
  — milestone 3, issue #3.

* **Creation**: Established this bundle by splitting
  [SCOPE](../planning/SCOPE.md) on its `## Milestone N:` headings — three epics, in the
  plan's own order, each depending on the one before it. Boundaries were taken from the
  plan unchanged; no milestone was merged or subdivided.

  Three things were carried into the epic files rather than left to be rediscovered.
  **Epic 3 spans two repositories** — SCOPE's note addressed to `split-epics` puts the
  `_shared/review-interfaces.md` swap in the lx skills repo while the rest of the epic is
  Go code here, which is the one place the pipeline's single-repo assumption breaks; per
  [conventions decision 04](../planning/conventions/04-cross-repo-conventions.md) those
  issues follow the skills repo's own contract. **Epic 1 ships a subset of the CLI
  grammar SPECS specifies**, since the allow-list arrives in Epic 2 and the tier flags in
  Epic 3 — recorded so the subset does not read as a contradiction of SPECS. And
  **Epic 2's real deliverable is partly measurements**, not code: Epic 3's caps and
  SCOPE's third success criterion are both blocked on numbers that only exist once a
  review has actually been run by hand.

  Plan-wide constraints (Windows-first with no POSIX assumption, the user's own
  credentials, the non-goals, and the unenforceable leads-not-findings assumption) were
  placed in the Notes of each epic they actually touch, marked project-wide, rather than
  forced into one epic's "Out of scope".

* **Note**: `create-issues` for Epic 3 is deliberately deferred by the user's decision at
  split time — it will be cut separately, elsewhere. Epics 1 and 2 are ready to cut now.
