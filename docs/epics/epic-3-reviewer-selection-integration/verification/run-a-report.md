# Epic 3 issue review

## 1. Coverage

I found no EPIC_3 Scope bullet or Acceptance criterion that lands in no issue.

Brief mapping checked:

- Weight tiers — EPIC_3.md:31-33 → issue 05:36-46, issue 10:63-66.
- Machine-local TOML assignment — EPIC_3.md:34-38 → issue 03:40-63, issue 04:41-49.
- Resolution order and dynamic refresh — EPIC_3.md:39-44 → issue 04:41-64, 97-103.
- Family classification/exclusion — EPIC_3.md:45-50 → issues 02, 04, 05, 07.
- `tiers` / `models` discovery — EPIC_3.md:51-54 → issues 06 and 07.
- Loop caps — EPIC_3.md:55-57 → issue 09.
- `review-interfaces.md` swap — EPIC_3.md:58-59 → issue 10.
- `git_read.paths` — EPIC_3.md:60-62 → issue 01.
- JSON request / `--system` / `version` — EPIC_3.md:63-67 → issue 08.

Acceptance criteria EPIC_3.md:84-109 are likewise covered, mainly by issues 01, 02, 04, 05, 06, 08, 10 and 11.

I did not find an issue claiming work outside the epic/planning documents. The apparently “extra” items — `models`, `version`, `--system`, `git_read.paths` — are all now named in EPIC_3.md:51-67 and/or SPECS.md:191-199.

## 2. Boundaries and ordering

### Lead: issue 10 appears to need issue 09, but does not declare it

- Source being honoured: issue 09 owns the cap-flag decision: `docs/epics/epic-3-reviewer-selection-integration/issues/09-loop-caps-from-measurements.md:56-61`.
- Issue that may fail to honour it: issue 10 says its invocation template changes if issue 09 adds a ceiling flag, but its `depends_on` omits 09: `docs/epics/epic-3-reviewer-selection-integration/issues/10-review-interfaces-external-reviewer-swap.md:55-59` and `:15`.

What would confirm it: check whether issue 10’s PR was opened after issue 09’s “no flag” decision was already merged. The log suggests that happened, but the issue dependency graph still doesn’t encode the need.

Otherwise, ordering is sane: no lower-numbered issue depends on a higher-numbered one, and the main seams are owned clearly.

## 3. Internal consistency

### Lead: `provider/model` wording may drift from the decided `provider/id` contract

- Source being honoured: SPECS says the model line is `provider/id` verbatim from `kern-link`’s catalog: `docs/planning/SPECS.md:252-255`; the tier assignment also says `provider` and `model` are `kern-link` identifiers verbatim: `docs/planning/SPECS.md:323-337`.
- Issue wording that may fail to honour it: issue 10 asks the report Scope to name `provider/model`: `docs/epics/epic-3-reviewer-selection-integration/issues/10-review-interfaces-external-reviewer-swap.md:81-82`; issue 11 repeats `provider/model`: `docs/epics/epic-3-reviewer-selection-integration/issues/11-success-criteria-verification-run.md:39-40` and `:75-77`.

What would confirm it: inspect the skills template / verification document and see whether it records the actual resolved `provider/id` string from stderr, not a display name or ambiguous “model” label.

Everything else I checked matches the sources closely: config path resolution, TOML schema, fail-closed family rule, absent-vs-broken credential split, JSON request object, and stdout/stderr contracts all line up with SCOPE/SPECS/CONVENTIONS/DRIFT.

## 4. Sizing

Most sizes look plausible, but the log now gives evidence that several were under-estimated.

### Lead: issues 06, 07, and 08 are recorded as M but landed well above the M band

- Issue 06 size: `docs/epics/epic-3-reviewer-selection-integration/issues/06-tiers-command.md:11`; log says 679 changed lines against ~500 M target: `docs/epics/log.md:116-118`.
- Issue 07 size: `docs/epics/epic-3-reviewer-selection-integration/issues/07-models-command.md:11`; log says 726 changed lines against ~500 M target: `docs/epics/log.md:62-64`.
- Issue 08 size: `docs/epics/epic-3-reviewer-selection-integration/issues/08-request-object-and-version.md:11`; log says 1024 changed lines against predicted M: `docs/epics/log.md:94-96`.

What would confirm it: compare the merged PR stats. The log already suggests these should have been L, or split, under the stated bands.

### Lead: issue 05 is L but exceeded even the L ceiling

- Issue size: `docs/epics/epic-3-reviewer-selection-integration/issues/05-review-tier-flags-and-exit-codes.md:11`.
- Log evidence: 1173 changed lines against ~1000-line L ceiling: `docs/epics/log.md:149-151`.

What would confirm it: inspect PR #86 diff stats and whether the issue’s own split line at issue 05:130-138 would have produced a cleaner split.

Sizes that looked right: issue 01 M, issue 02 L, issue 03 M, issue 04 L, issue 09 S, issue 10 M, and issue 11 S are plausible against their stated acceptance criteria and/or logged outcomes.