# Log

## 2026-08-13

* **Issue 05 done**: [05 — Select the reviewer from the command line with --tier, --model and --exclude-family](/epic-3-reviewer-selection-integration/issues/05-review-tier-flags-and-exit-codes.md)
  ([#64](https://github.com/julienlegoux/external-reviewer/issues/64)) — [PR #86](https://github.com/julienlegoux/external-reviewer/pull/86)
  merged into `develop`; issue #64 already closed. Reconciled by issue 06's run — the
  issue file and `issues/index.md` still read `pr-open`.

* **Epic 3, issue 05 pr-open**: [Select the reviewer from the command line with --tier, --model and --exclude-family](/epic-3-reviewer-selection-integration/issues/05-review-tier-flags-and-exit-codes.md)
  ([#64](https://github.com/julienlegoux/external-reviewer/issues/64)) — [PR #86](https://github.com/julienlegoux/external-reviewer/pull/86)
  opened against `develop` from branch `issue-64-cli-reviewer-selection`. `reviewer.Chain`
  now sits behind `--tier` / `--model` / `--exclude-family`, and `DefaultProviderID` /
  `DefaultModelID` are gone: the grep criterion holds, so the only provider named in
  non-test source is the family classifier's own data table. Four usage errors, one of
  them beyond the issue's criteria — an unrecognised `--exclude-family` name is refused,
  because `family.NewExclusion` keeps it verbatim and it would then exclude nothing at
  all. Issue 04's **third** failure class, `*MalformedAssignmentError`, had no slot in the
  contract and is split by its `Source` rather than folded into a neighbour: exit 2 either
  way, `stop=usage` from `--model` and `stop=failed` from the environment or the config
  file. `Chain.ResolveTier` now returns its assignment on the failure path too, which is
  what the exit-1 diagnostic reads. One drift record: the exit-1 path writes a single
  `warn` line where SPECS says "silent" — stdout stays empty and no `error:` line is
  written, so both criteria hold, but exit 1 now covers three assignment layers and five
  reasons and is otherwise undebuggable. Also gives `internal/cli` a `TestMain` that pins
  `EXTERNAL_REVIEWER_CONFIG` and clears the tier variables, which the suite had been
  reading from the developer's own profile. 1173 changed lines of Go against a ~1000-line
  `L` ceiling; `-race` ran through the remote fallback, `gcc` being blocked again exactly
  as drift record 02 describes.

* **Issue 04 done**: [04 — Resolve a tier to a reachable, allowed model](/epic-3-reviewer-selection-integration/issues/04-tier-resolution-chain.md)
  ([#63](https://github.com/julienlegoux/external-reviewer/issues/63)) — [PR #85](https://github.com/julienlegoux/external-reviewer/pull/85)
  merged into `develop`; issue #63 already closed, its stale `status: pr-open` label
  removed here. Reconciled by issue 05's run — the issue file and `issues/index.md` still
  read `pr-open`.

* **Issue 09 done**: [09 — Set the loop caps from Epic 2's measurements](/epic-3-reviewer-selection-integration/issues/09-loop-caps-from-measurements.md)
  ([#68](https://github.com/julienlegoux/external-reviewer/issues/68)) — [PR #83](https://github.com/julienlegoux/external-reviewer/pull/83)
  merged into `develop`; issue #68 already closed. Reconciled here while merging
  `origin/develop` into issue 04's branch to resolve a conflict — the issue file and
  `issues/index.md` still read `pr-open`.

* **Epic 3, issue 04 pr-open**: [Resolve a tier to a reachable, allowed model](/epic-3-reviewer-selection-integration/issues/04-tier-resolution-chain.md)
  ([#63](https://github.com/julienlegoux/external-reviewer/issues/63)) — [PR #85](https://github.com/julienlegoux/external-reviewer/pull/85)
  opened against `develop`. The four-layer precedence chain as `reviewer.Assigner`, the
  family gate sequenced between `GetModel` and `GetAuth` inside `Resolver`, and
  `reviewer.Chain` composing them; `NoReviewerError` now carries which rule refused, and a
  malformed `provider/id` is a third error class that is deliberately not `ErrNoReviewer`.
  Refresh-before-lookup verified by mutation. One drift record written — a `github-copilot`
  tier resolves to no reviewer because all 25 of that provider's ids are bare, ten of them
  Anthropic's, so the fix is a specs 04 rule rather than a widening of this PR.

* **Issue 09 PR opened**: [09 — Set the loop caps from Epic 2's measurements](/epic-3-reviewer-selection-integration/issues/09-loop-caps-from-measurements.md)
  ([#68](https://github.com/julienlegoux/external-reviewer/issues/68)) opened as
  [PR #83](https://github.com/julienlegoux/external-reviewer/pull/83) from branch
  `issue-09-epic3-caps`. `cli.Run` wires a non-zero `reviewer.Bounds` on every real
  invocation: `MaxTurns: 30` (SCOPE-fixed [25, 40]), `MaxElapsed: 20 * time.Minute`
  (SCOPE-fixed [15m, 25m]), `MaxCost` stays 0 — sized from Epic 2's MEASUREMENTS, read
  knowing issue 01 (PR #81, merged) had already closed the `git_read` gap that inflated
  the twelve-turn run. Also fixes the correctness gap the issue named:
  `resolveAndReview`'s empty-report check no longer reclassifies a bounds stop with no
  prose yet as `failed` — it now checks `state.StopReason` first, verified red against
  the pre-fix code (exit 2, `stop=failed`) before the fix went in. **No flag** exposes
  any of the three ceilings — they are a safety net sized well above every measured run,
  not a knob the invocation template has asked to turn, and the seam already makes
  adding one later an ordinary PR rather than a restructuring; reasoning recorded in the
  PR body since the issue explicitly permits answering "no". 126 changed lines against
  the ~500-line S target.

* **Issue 02 done**: [02 — Add the fail-closed model family classifier](/epic-3-reviewer-selection-integration/issues/02-model-family-classifier.md)
  ([#61](https://github.com/julienlegoux/external-reviewer/issues/61)) — [PR #82](https://github.com/julienlegoux/external-reviewer/pull/82)
  merged into `develop`; issue #61 already closed. Reconciled here while merging
  `origin/develop` into issue 09's branch to resolve a conflict — the issue file and
  `issues/index.md` still read `pr-open`. Issue 04's run reconciled the same fact
  independently, and additionally removed the stale `status: pr-open` label that the merge
  had left on the closed issues [#61](https://github.com/julienlegoux/external-reviewer/issues/61)
  and [#62](https://github.com/julienlegoux/external-reviewer/issues/62).

* **Epic 3, issue 02 pr-open**: [Add the fail-closed model family classifier](/epic-3-reviewer-selection-integration/issues/02-model-family-classifier.md)
  ([#61](https://github.com/julienlegoux/external-reviewer/issues/61)) — [PR #82](https://github.com/julienlegoux/external-reviewer/pull/82)
  opened against `develop`. New `internal/family` package: `(Provider, ID)` classified
  against two data tables, unknown as the `Family` zero value and refused by every
  `Exclusion`, proved over all 1042 models of the 35 embedded providers. Two drift records
  written — `azure-openai-responses` classified by rule 1 rather than the reseller rule, and
  the native `-race` command not running because `gcc.exe` draws an Application Control
  fail-fast. That second record carries the root cause behind the `gcc` refusal issue 03
  reported the same day: `0xC0000602` on a one-line C file, so no Go build flag can
  re-roll it.

* **Issue 03 done**: [03 — Read the machine-local tier assignment TOML](/epic-3-reviewer-selection-integration/issues/03-tier-assignment-config-file.md)
  ([#62](https://github.com/julienlegoux/external-reviewer/issues/62)) — [PR #80](https://github.com/julienlegoux/external-reviewer/pull/80)
  merged into `develop`; issue #62 already closed. Reconciled independently by issue 09's
  and issue 02's runs — the issue file and `issues/index.md` still read `pr-open`.

* **Issue 01 done**: [01 — Make a path-scoped diff reachable through git_read](/epic-3-reviewer-selection-integration/issues/01-git-read-path-scoped-diff.md)
  ([#60](https://github.com/julienlegoux/external-reviewer/issues/60)) — [PR #81](https://github.com/julienlegoux/external-reviewer/pull/81)
  merged into `develop`; issue #60 already closed. Reconciled independently by issue 09's,
  issue 02's and issue 03's runs — the issue file and `issues/index.md` still read
  `pr-open`.

* **PR opened**: Epic 3 issue 03 — [Read the machine-local tier assignment TOML](https://github.com/julienlegoux/external-reviewer/issues/62)
  (#62) — [PR #80](https://github.com/julienlegoux/external-reviewer/pull/80) against
  `develop`, from branch `issue-03-epic3-reviewer-selection`. Adds `internal/config`:
  `Locate()`/`LoadFile()`/`Load()` find and decode the hand-written, never-committed
  tier assignment TOML — `EXTERNAL_REVIEWER_CONFIG` or `os.UserConfigDir()`, then
  `[tiers.<name>]` tables into `provider`/`model` assignments verbatim, with
  `MetaData.Undecoded()` driving the unrecognised-key warning in SPECS' exact wording.
  Also adds the project's second and last non-stdlib dependency,
  `github.com/BurntSushi/toml v1.6.0`. 441 changed lines against a ~500-line M target.

  Two things found by the red-green loop rather than assumed: printing the OS path in
  the malformed-TOML error with `%q` would have escaped a Windows path's backslashes,
  contradicting CONVENTIONS' "printed as the OS renders it" — fixed to `%s`. And a
  typo'd key (e.g. `models` for `model`) alongside the resulting missing `model` would
  otherwise fire two warnings for one mistake; the missing-required-key warning is now
  suppressed for a tier that already carries an unrecognised-key warning, matching the
  issue's "exactly one warning" criterion.

  **No native `-race` run this session**: `gcc.exe` itself is currently refused by
  Smart App Control (exit 127, no output, even on `gcc --version`), a step beyond the
  per-test-binary hash blocking DRIFT already documents. Verified instead through the
  documented fallback, `scripts/test-remote.sh` — full suite green, `-race` included.
  Not promoted to a new drift record: DRIFT already documents this class of problem and
  its fallback; this run exercised the fallback rather than proving anything wrong.

* **Epic 3, issue 01 PR opened**: [01 — Make a path-scoped diff reachable through git_read](/epic-3-reviewer-selection-integration/issues/01-git-read-path-scoped-diff.md)
  ([#60](https://github.com/julienlegoux/external-reviewer/issues/60)) opened as
  [PR #81](https://github.com/julienlegoux/external-reviewer/pull/81) from branch
  `issue-60-git-read-path-scoped-diff`. `git_read` gains a **`paths`** parameter: every
  entry resolved through `Scope.Resolve` before the command is built, under the same three
  rules and the same wording as a `<rev>:<path>` object, then used as the pathspecs in
  place of the granted subtrees — a narrowing of the grant, never a widening, and refused
  alongside an object because git takes pathspecs or an object and not both. Verified by
  mutation: dropping the resolution makes the out-of-allow-list test fail carrying the
  ungranted subtree's own diff. SPECS' tool table, specs 09 and specs 10's pathspec bullet
  amended to match. No drift record — this is the gap specs 10 left, not a departure from
  it.

* **Retirement**: Epic 0 (Epic 3 plan repair) retired - 4 issues, milestone 5 closed.
  Consumed reports docs/reviews/2026-08-13-issues-epic-3.md. Fixed: repaired Epic 3's plan
  against all eight review findings — the scope EPIC_3 never named for its own issues 01
  and 08, issue 10's cross-repository `gh_issue` resolving to the wrong repository, three
  missing dependency edges, the uniform M sizing and its boilerplate PR-size note, issue
  07's missing silent-empty-catalog criterion, two stale outward-facing surfaces, and the
  `README.md` the plan promised but no issue created.

  **What survives on GitHub**: issues [#71](https://github.com/julienlegoux/external-reviewer/issues/71),
  [#72](https://github.com/julienlegoux/external-reviewer/issues/72),
  [#73](https://github.com/julienlegoux/external-reviewer/issues/73) and
  [#74](https://github.com/julienlegoux/external-reviewer/issues/74) (issues 01–04),
  tracking issue [#70](https://github.com/julienlegoux/external-reviewer/issues/70), merged
  as [PR #75](https://github.com/julienlegoux/external-reviewer/pull/75) (01),
  [PR #77](https://github.com/julienlegoux/external-reviewer/pull/77) (02),
  [PR #78](https://github.com/julienlegoux/external-reviewer/pull/78) (03) and
  [PR #76](https://github.com/julienlegoux/external-reviewer/pull/76) (04), plus the
  reconcile [PR #79](https://github.com/julienlegoux/external-reviewer/pull/79).

  **Recorded, not scheduled — upstream.** `_shared/pipeline-interfaces.md`'s issue schema
  has no way to express a cross-repository `gh_issue` — it defines the field as an
  unqualified integer, which is what let Epic 3's issue 10 point at the wrong repository;
  repair B worked around it locally with a qualified value. The schema lives in
  `julienlegoux/skills` and this epic could not reach it — still to be filed there.

  The folder `docs/epics/epic-0-epic-3-plan-repair/` is gone; milestone 5 and issues
  #70–#74 survive on GitHub and every link that pointed into the folder was repointed
  there first. The epic-0 slot is free for the next triage cycle.

* **Issue 04 done**: [04 — Add the README stating v1's OpenAI-only validation boundary](https://github.com/julienlegoux/external-reviewer/issues/74)
  ([#74](https://github.com/julienlegoux/external-reviewer/issues/74)) — [PR #76](https://github.com/julienlegoux/external-reviewer/pull/76) merged into
  `develop`; issue #74 already closed. Reconciled here — `04-readme-validation-boundary.md`
  and `issues/index.md` still read `pr-open`.

* **Issue 03 done**: [03 — Re-grade Epic 3's issue sizes and sync the recorded surfaces](https://github.com/julienlegoux/external-reviewer/issues/73)
  ([#73](https://github.com/julienlegoux/external-reviewer/issues/73)) — [PR #78](https://github.com/julienlegoux/external-reviewer/pull/78) merged into
  `develop`; issue #73 already closed. Reconciled here — `03-epic-3-sizes-and-surfaces.md`
  and `issues/index.md` still read `pr-open`.

* **Issue 03 PR opened**: [03 — Re-grade Epic 3's issue sizes and sync the recorded surfaces](https://github.com/julienlegoux/external-reviewer/issues/73)
  ([#73](https://github.com/julienlegoux/external-reviewer/issues/73)) opened as
  [PR #78](https://github.com/julienlegoux/external-reviewer/pull/78) from branch
  `issue-03-epic0-work`. **Re-graded against Epic 2's measured shapes** — a new package
  plus its tool and its tests graded `L` five times out of seven there — so Epic 3's 02
  (model family classifier), 04 (tier resolution chain) and 05 (CLI tier flags and exit
  codes) move to `L`; 09 (loop caps) drops to `S`, matching its own Summary's "a data
  change plus one correctness gap, not a restructuring"; 01, 03, 06, 07, 08 and 10 stay
  `M`; 11 was already `S`. Every boilerplate PR-size note replaced with one naming that
  issue's own bulk and split line. Also fixed `docs/epics/index.md`'s stale
  "`create-issues` deliberately deferred" clause and rewrote GitHub #60's body to the same
  rendering `#61`–`#69` carry, keeping the original discovery transcript as its Summary.

* **Issue 02 done**: [02 — Repair Epic 3's cross-repository pointer, dependency edges and the models criterion](https://github.com/julienlegoux/external-reviewer/issues/72)
  (#72) — [PR #77](https://github.com/julienlegoux/external-reviewer/pull/77) merged into
  `develop`; issue #72 already closed. Reconciled here — `02-epic-3-issue-corrections.md`
  and `issues/index.md` still read `pr-open`.

* **Issue 01 done**: [01 Amend EPIC_3 with the scope its issues will
  ship](https://github.com/julienlegoux/external-reviewer/issues/71) (#71) —
  [PR #75](https://github.com/julienlegoux/external-reviewer/pull/75) merged into
  `develop`.

* **PR opened**: [01 Amend EPIC_3 with the scope its issues will
  ship](https://github.com/julienlegoux/external-reviewer/issues/71) (#71) —
  [PR #75](https://github.com/julienlegoux/external-reviewer/pull/75) against
  `develop`. `EPIC_3.md`'s Scope and Acceptance criteria gain the `git_read` `paths`
  parameter and the JSON request object / `--system` / `version`, already cut into
  issues 01 and 08 but never named in the epic; `SPECS.md:275`'s stop-reason miscount
  ("five" for a list of six) is fixed alongside. 4 files, 20 insertions / 6 deletions
  against an S target.

* **Started**: [01 Amend EPIC_3 with the scope its issues will
  ship](https://github.com/julienlegoux/external-reviewer/issues/71) (#71) —
  branch `issue-01-epic0-plan-repair`.

* **Issues created**: [Epic 0: Epic 3 plan repair](https://github.com/julienlegoux/external-reviewer/issues/70)
  (#70) cut into four issues on milestone 5, all four native sub-issues of #70:
  [01](https://github.com/julienlegoux/external-reviewer/issues/71)
  ([#71](https://github.com/julienlegoux/external-reviewer/issues/71)),
  [02](https://github.com/julienlegoux/external-reviewer/issues/72)
  ([#72](https://github.com/julienlegoux/external-reviewer/issues/72)),
  [03](https://github.com/julienlegoux/external-reviewer/issues/73)
  ([#73](https://github.com/julienlegoux/external-reviewer/issues/73)) and
  [04](https://github.com/julienlegoux/external-reviewer/issues/74)
  ([#74](https://github.com/julienlegoux/external-reviewer/issues/74)).

  **Seven repairs, four issues — cut by which files each opens rather than by repair.**
  Repairs B, C and E all edit files under
  `epic-3-reviewer-selection-integration/issues/`, and repair D sweeps all ten of them for
  `size` and the PR-size note; splitting them one repair per PR would have put four branches
  on the same ten files. So the correctness edits are issue 02 and the sizing sweep is issue
  03, ordered behind it — the only dependency in this epic. Repairs D and F ride together
  because both are recorded surfaces (`issues/index.md`, `docs/epics/index.md`, `log.md`,
  GitHub #60), and #60's body must be rendered from an issue file that issue 02 has already
  corrected. Repair A (issue 01, `EPIC_3.md` + `SPECS.md`) and repair G (issue 04,
  `README.md`) open files nothing else in the epic touches, so all three of 01, 02 and 04
  can run in parallel.

  **Sizes: S, S, M, S — nothing needed the L ceiling, and this one is measured.** The whole
  epic is planning artifacts plus one new `README.md`; no `.go` file is touched. Issue 03 is
  M for spread rather than depth — thirteen files touched shallowly plus a GitHub body
  rewritten from one of them. Every `## PR size note` names that issue's own bulk and its
  own split line, which is the defect issue 03 exists to fix in Epic 3 and would have been
  hypocritical to reproduce here.

* **Issue 02 PR opened**: [02 — Repair Epic 3's cross-repository pointer, dependency edges and the models criterion](https://github.com/julienlegoux/external-reviewer/issues/72)
  ([#72](https://github.com/julienlegoux/external-reviewer/issues/72)) opened as
  [PR #77](https://github.com/julienlegoux/external-reviewer/pull/77) from branch
  `issue-72-epic-3-issue-corrections`, qualifying issue 10's cross-repository
  `gh_issue`, adding the three missing dependency edges (11 → 1, 9; 07 → 6), and
  giving issue 07 a criterion for the silent-empty catalog.

* **Creation**: Established [Epic 0: Epic 3 plan repair](https://github.com/julienlegoux/external-reviewer/issues/70)
  by converting [the Epic 3 issue review](../reviews/2026-08-13-issues-epic-3.md) —
  milestone 5, issue [#70](https://github.com/julienlegoux/external-reviewer/issues/70).
  The
  remediation lane, which runs before [Epic 3](/epic-3-reviewer-selection-integration/EPIC_3.md)
  is implemented and is retired once every issue is `done` and its milestone closed.
  **Consumed reports**: `docs/reviews/2026-08-13-issues-epic-3.md`.

  **8 findings extracted, 0 dropped as stale, 7 repairs.** Staleness was checked rather
  than assumed: no commit touched `docs/epics/` or `docs/planning/` between the report and
  this conversion — the last is `72d9350`, which is the tree the report reviewed — every
  cited `file:line` was re-verified against the working tree, `gh issue list --state all`
  covers none of the eight, and DRIFT has accepted none of them. Nothing was recorded
  `won't-fix`.

  The compression is small because the findings are mostly one-of-a-kind rather than one
  defect replicated: only two pairs group (the two dependency-graph findings, and the two
  outward-facing surfaces). Where it does bite is the sizing sweep — one repair covering
  ten issue files and the index — and the seventh repair is not a finding at all but the
  report's own first open question, promoted: `specs/05:118-122` and `EPIC_3.md:130-132`
  both promise a README that states v1's OpenAI-only validation boundary, and this
  repository has no `README.md`.

  **Two riders were folded in rather than filed**, both edits rather than decisions, both
  in files a repair already opens: `SPECS.md:275` introduces the non-model stop reasons as
  "one of five fixed CLI words" and lists six, and issue 10's invocation template says
  nothing about whether issue 09 exposes a cap on the command line.

  **One thing this lane cannot reach.** `_shared/pipeline-interfaces.md` defines `gh_issue`
  as an unqualified integer, with no way to express a cross-repository issue — which is how
  issue 10 came to point `gh_issue: 37` at a merged Epic-0 issue in *this* repository while
  meaning `julienlegoux/skills#37`. Repair B works around it locally with a qualified value;
  the schema lives in `julienlegoux/skills` and gets filed there once this lane lands.

  `docs/REPORT_1.md` and `docs/REPORT_2.md` were **not** re-read: the first was applied
  2026-08-09 as a revision to the twelve issues of Epics 1 and 2, the second consumed by
  the previous Epic 0 (retirement notice, 2026-08-11).

* **Issues created**: [Epic 3: Reviewer selection and pipeline integration](/epic-3-reviewer-selection-integration/EPIC_3.md)
  (#3) cut into eleven issues, sized uniformly S or M at cut time — a grading later shown
  to be wrong against Epic 2's measured shapes and corrected by
  [Epic 0 issue 03](https://github.com/julienlegoux/external-reviewer/issues/73) to
  three L, six M and two S. In build
  order: [01](/epic-3-reviewer-selection-integration/issues/01-git-read-path-scoped-diff.md)
  ([#60](https://github.com/julienlegoux/external-reviewer/issues/60), filed at Epic 2's
  close and adopted as this epic's first issue rather than created again),
  [02](/epic-3-reviewer-selection-integration/issues/02-model-family-classifier.md)
  ([#61](https://github.com/julienlegoux/external-reviewer/issues/61)),
  [03](/epic-3-reviewer-selection-integration/issues/03-tier-assignment-config-file.md)
  ([#62](https://github.com/julienlegoux/external-reviewer/issues/62)),
  [04](/epic-3-reviewer-selection-integration/issues/04-tier-resolution-chain.md)
  ([#63](https://github.com/julienlegoux/external-reviewer/issues/63)),
  [05](/epic-3-reviewer-selection-integration/issues/05-review-tier-flags-and-exit-codes.md)
  ([#64](https://github.com/julienlegoux/external-reviewer/issues/64)),
  [06](/epic-3-reviewer-selection-integration/issues/06-tiers-command.md)
  ([#65](https://github.com/julienlegoux/external-reviewer/issues/65)),
  [07](/epic-3-reviewer-selection-integration/issues/07-models-command.md)
  ([#66](https://github.com/julienlegoux/external-reviewer/issues/66)),
  [08](/epic-3-reviewer-selection-integration/issues/08-request-object-and-version.md)
  ([#67](https://github.com/julienlegoux/external-reviewer/issues/67)),
  [09](/epic-3-reviewer-selection-integration/issues/09-loop-caps-from-measurements.md)
  ([#68](https://github.com/julienlegoux/external-reviewer/issues/68)),
  [10](/epic-3-reviewer-selection-integration/issues/10-review-interfaces-external-reviewer-swap.md)
  ([julienlegoux/skills#37](https://github.com/julienlegoux/skills/issues/37)) and
  [11](/epic-3-reviewer-selection-integration/issues/11-success-criteria-verification-run.md)
  ([#69](https://github.com/julienlegoux/external-reviewer/issues/69)). All eleven are
  native sub-issues of #3, **including the cross-repository one** — GitHub accepted a
  sub-issue owned by another repository of the same account, so the epic's progress bar
  tracks the skills-repo work too.

  Three things the cut decided that the epic did not state. **Issue 01 comes first**
  because [MEASUREMENTS](/epic-2-read-only-agentic-loop/MEASUREMENTS.md) says the
  `git_read` gap inflated the turn counts issue 09's caps are sized from (12 turns, versus
  a predicted 6–8 with it closed) — sizing a ceiling against a tool that wastes turns bakes
  the waste in. **Issue 08 is in scope by necessity**: the caller-owned system prompt and
  its JSON request channel were deferred by
  [Epic 1 issue 02](/epic-1-walking-skeleton/issues/02-review-invocation-parsing.md) to
  "the epics that need them", and issue 10 cannot write an invocation template without one;
  `version` rides along as the last unimplemented item of SPECS' grammar. And **issue 09
  carries a correctness gap found while reading the code**: at a bounds stop `Loop.Run`
  returns the accumulated report, which is empty when the run never wrote prose, and
  `resolveAndReview` turns an empty report into a `failed` at exit 2 — while
  [SPECS § Interfaces](../planning/SPECS.md) says a bounded run is exit 0 with what it had.

## 2026-08-12

* **Epic closed**: [Epic 2: Read-only agentic loop](/epic-2-read-only-agentic-loop/EPIC_2.md)
  (#2) — all seven issues `done`, milestone 2 closed with fifteen closed issues, tracking
  issue closed. Issue 07 was still recorded `pr-open` though [PR #59](https://github.com/julienlegoux/external-reviewer/pull/59)
  had merged, so `issues/07-hand-run-measurements.md` and `issues/index.md` were
  reconciled to `done` here. The epic shipped both its deliverables: the multi-turn loop
  with `list`, `read_file`, `search` and `git_read` all confined by one `*os.Root` plus
  the allow-list and the sensitive-file floor, and
  [MEASUREMENTS](/epic-2-read-only-agentic-loop/MEASUREMENTS.md), which Epic 3 reads to
  size its caps. Its four drift records were promoted into
  [DRIFT](../planning/DRIFT.md) as three entries, all triaged `accepted` — the fourth
  (issue 05's) was already `resolved` in its own record and folded into the Epic 0 entry
  it corrects. The one defect the hand runs exposed and this document could not fix is
  now [#60](https://github.com/julienlegoux/external-reviewer/issues/60) on Epic 3's
  milestone.

* **PR open**: Epic 2 issue 07 — [Run a real review by hand and record the measurements](https://github.com/julienlegoux/external-reviewer/issues/15) (#15),
  [PR #59](https://github.com/julienlegoux/external-reviewer/pull/59). The epic's last
  issue, and the only one whose deliverable is a document rather than code: no `.go` file
  is touched. 239 insertions against an `S` estimate.

* **New document**: [Epic 2 — Hand-run measurements](/epic-2-read-only-agentic-loop/MEASUREMENTS.md),
  the epic's second deliverable and Epic 3's only input for the loop caps. Four hand runs
  on `openai-codex/gpt-5.5` against this repository — a documentation directory, `docs/`
  plus `internal/`, and this epic's own merged diff — of which three completed on the
  model's own stop reason and one died on a transient provider error. Turns 4/6/12, wall
  clock 1m57s/2m12s/3m59s, peak single-turn prompt 99,163 tokens, $3.27 total at catalog
  prices on a subscription credential that bills none of it. **Context overflow was not
  hit**: the per-answer tool caps bind first, one capped `git_read` answer costing ~31k
  tokens. All four tools were exercised by a real model, and two designed behaviours were
  observed rather than argued — a failing turn ends the run with its `done` line intact,
  and a refused tool call is only a turn the model re-plans around. The runs also exposed
  one gap of their own: a path-scoped `diff` is unreachable through `git_read`, since the
  model may not write `--` and a bare pathspec is parsed as a revision ahead of the
  appended one.

* **Done**: Epic 2 issue 06 — [Add the confined git_read tool](https://github.com/julienlegoux/external-reviewer/issues/14) (#14)
  merged into `develop` as [PR #55](https://github.com/julienlegoux/external-reviewer/pull/55);
  issue closed. Reconciled the bundle's own bookkeeping — `issues/06-git-read-tool.md`
  and `issues/index.md` still read `pr-open` though PR #55 and GitHub issue #14 were
  already merged and closed. Epic 2 now stands at six of its seven issues `done`, with
  issue 07 the sole `open` issue remaining.

* **Merge**: `origin/develop` merged into Epic 2 issue 06's branch, twice — for issue
  05's [PR #53](https://github.com/julienlegoux/external-reviewer/pull/53) and then for
  issue 04's [PR #54](https://github.com/julienlegoux/external-reviewer/pull/54), the two
  siblings that landed while #55 was open. `internal/tools/run.go` is now the union of all
  four registrations — `List`, `ReadFile`, `Search`, `GitRead` — with no placeholder
  comments left, and `issues/index.md`, this file and `drift/index.md` keep every issue's
  and every record's own line. Issue 04's rename of `intArgument` to `wholeNumberArgument`
  is why a package-level identifier check ran on the merged tree rather than trusting a
  clean text merge: `go build ./...` and `go vet ./...` (which type-checks `_test.go` too)
  are both clean, so `git_read.go` collides with neither sibling.

* **Done**: Epic 2 issue 04 — [Add the batched read_file tool](https://github.com/julienlegoux/external-reviewer/issues/12) (#12)
  merged into `develop` as [PR #54](https://github.com/julienlegoux/external-reviewer/pull/54);
  issue closed. Reconciled by issue 06's run during that second merge — the bundle still
  read `pr-open`. All four of the epic's tools now exist, and their tests run together for
  the first time on issue 06's branch.

* **PR open**: Epic 2 issue 06 — [Add the confined git_read tool](https://github.com/julienlegoux/external-reviewer/issues/14) (#14),
  [PR #55](https://github.com/julienlegoux/external-reviewer/pull/55). History is the one
  surface `*os.Root` cannot guard, so `git_read` makes its confinement in argument space
  before the command exists: the subcommand against the fixed `log`/`diff`/`show`/`status`
  allowlist, every `<rev>:<path>` object — including the index forms `:<path>` and
  `:<stage>:<path>`, in every subcommand's arguments — through `confine.Scope.Resolve`,
  and the granted subtrees appended as pathspecs a literal `--` may not reach past.
  `repo.Git` exports issue 02's exec helper rather than adding a second exec, now
  capturing git's stderr so a non-zero exit is a tool error the model can act on.
  **Oversize**: ~1008 changed lines against a ~700-line `L` estimate; implementation ~385
  and on target, the overrun is 578 lines of git-fixture tests — the fourth `L` in this
  epic to overrun on tests alone.

  **Drift recorded**:
  [04 — git_read scopes status too, and validates `<rev>:<path>` in every subcommand](/epic-2-read-only-agentic-loop/drift/04-git-read-scopes-status-and-every-object-argument.md).
  specs 10 says "`status` is unaffected" and confines only `show`'s object form; measured
  against real git, an unscoped `status` names every path in the repository and
  `git diff <blob> <blob>` prints two files' contents.

* **Started**: Epic 2 issue 06 — [Add the confined git_read tool](https://github.com/julienlegoux/external-reviewer/issues/14) (#14) —
  branch `issue-06-git-read-tool`.

* **Merge**: `origin/develop` merged into Epic 2 issue 04's branch to pick up issue
  05's concurrently-merged [PR #53](https://github.com/julienlegoux/external-reviewer/pull/53).
  Resolved `internal/tools/run.go` as a union — `ReadFile(deps.Scope)` and
  `Search(deps.Scope, deps.Files)` both registered in `tools.NewRunRegistry`, issue
  06's placeholder comment left intact — and this file and `issues/index.md` likewise,
  reconciling issue 05 to `done` in the same pass issue 04's own run already gave
  issue 03.

* **PR open**: Epic 2 issue 04 — [Add the batched read_file tool](https://github.com/julienlegoux/external-reviewer/issues/12) (#12),
  [PR #54](https://github.com/julienlegoux/external-reviewer/pull/54). `read_file`
  takes a list of repo-relative paths plus `offset`/`limit`, every path resolved
  through issue 01's `confine.Scope`; a refusal, a missing path, a directory or
  non-UTF-8 content each render as an inline message under that path's own header
  while the rest of the batch still returns, and only a malformed call (a bad
  `paths`/`offset`/`limit` shape) is a tool error. Content is streamed line-by-line
  via `bufio.Scanner` rather than read whole and trimmed, with the scan still
  counting every line so the truncation marker's total is always real. Registers as
  the one line issue 03's frozen seam reserved for it in `tools.NewRunRegistry`. 689
  changed lines against a ~400-line M target (~700 split line) — under the split,
  driven by test volume like every issue before it in this epic. No drift recorded;
  the native `go test ./...` flake on `internal/confine` observed during this run
  matches the machine's already-drift-recorded Application Control behavior under
  concurrent sibling load, not a new finding.

* **Done**: Epic 2 issue 05 — [Add the search tool with surrounding context lines](https://github.com/julienlegoux/external-reviewer/issues/13) (#13)
  merged into `develop` as [PR #53](https://github.com/julienlegoux/external-reviewer/pull/53);
  issue closed. Reconciled by issue 06's run, while merging `develop` into its branch —
  the bundle still read `pr-open`.

* **PR open**: Epic 2 issue 05 — [Add the search tool with surrounding context lines](https://github.com/julienlegoux/external-reviewer/issues/13) (#13),
  [PR #53](https://github.com/julienlegoux/external-reviewer/pull/53). In-process RE2 over
  the same enumeration `list` reads from, streamed line by line through the `Scope`: a
  match renders `path:line:text`, a context line `path-line-text`, non-adjacent windows
  are separated by `--`, and overlapping windows merge rather than repeat their shared
  lines. One result is bounded by four numbers at once — 200 matches, 10 context lines
  either side, 500 characters per line, 2 MB per file with binary files sniffed out — and
  truncation carries both real numbers, so matches past the cap are counted but not kept.
  **Oversize**: 1324 changed lines against a ~650-line `L` estimate; the implementation is
  457 and on target, the overrun is 763 lines of tests, split along the line the issue
  itself nominated (the context rendering and its edge cases).

  **Drift recorded**: the native half of the test command stopped working on the
  development machine, whatever `GOTMPDIR` says, so this issue's whole red-green cycle ran
  through `scripts/test-remote.sh` with `-race`.

* **Started**: Epic 2 issue 05 — [Add the search tool with surrounding context lines](https://github.com/julienlegoux/external-reviewer/issues/13) (#13) —
  branch `issue-05-search-tool`.

* **Done**: Epic 2 issue 03 — [Drive the multi-turn loop with tool dispatch and the bounds seam](https://github.com/julienlegoux/external-reviewer/issues/11) (#11)
  merged into `develop` as [PR #52](https://github.com/julienlegoux/external-reviewer/pull/52);
  issue closed. The dispatch seam it froze — `reviewer.ToolSet`, `tools.Deps` and the
  single registration point `tools.NewRunRegistry` — is what issues 04, 05 and 06 plug
  into, one file plus one line each.

* **PR open**: Epic 2 issue 03 — [Drive the multi-turn loop with tool dispatch and the bounds seam](https://github.com/julienlegoux/external-reviewer/issues/11) (#11),
  [PR #52](https://github.com/julienlegoux/external-reviewer/pull/52). The binary stops
  being a prompt-pipe: `internal/reviewer.Loop` declares the registry's tools on every
  request, dispatches each `ai.ToolCall` and feeds the answer back as a
  `ToolResultMessage`, terminating in the asserted order bound → cancellation → stop
  reason. `cli.recordTurn`'s accumulation moves into the loop, where the bound check can
  read it. `reviewer.Bounds` ships with every field unset and is threaded from the
  invocation through `reviewRequest`, so Epic 3 sets numbers rather than restructuring.
  **Oversize**: ~1465 changed lines against a ~650-line `L` estimate; the implementation
  is ~570 and on target, the overrun is 855 lines of scripted conversations — the split
  line the issue's own size note nominated, and the third `L` in a row to overrun on
  tests alone.

  **The seam issues 04–06 build on is frozen here**: `reviewer.ToolSet`
  (`Declarations() []ai.Tool` + `Dispatch(ctx, ai.ToolCall) (string, bool)`), which
  `*tools.Registry` satisfies, so `internal/reviewer` never imports `internal/tools`.
  Adding a tool is a new file in `internal/tools` plus one line in `tools.NewRunRegistry`,
  the single registration point, over a `tools.Deps` that already carries `Scope`,
  `RepoPath`, `Files` and `Warn`.

  **No drift recorded.** Two amendments went the other way — SPECS moves `bounds` into
  the closed `stop=` set it had reserved for this epic, and its transcript example now
  shows a `tool` line the generic renderer can actually produce.

* **Started**: Epic 2 issue 03 — [Drive the multi-turn loop with tool dispatch and the bounds seam](https://github.com/julienlegoux/external-reviewer/issues/11) (#11) —
  branch `issue-03-multi-turn-loop-and-dispatch`.

* **Done**: Epic 2 issue 02 — [Enumerate the repository through git ls-files and add the list tool](https://github.com/julienlegoux/external-reviewer/issues/10) (#10)
  merged into `develop` as [PR #51](https://github.com/julienlegoux/external-reviewer/pull/51);
  issue closed. Reconciled by issue 03's run — the issue file still read `pr-open` though
  GitHub issue #10 was already closed. `internal/repo` and `internal/tools` exist but have
  no `internal/cli` caller yet; wiring them in is issue 03's job.

* **PR open**: Epic 2 issue 02 — [Enumerate the repository through git ls-files and add the list tool](https://github.com/julienlegoux/external-reviewer/issues/10) (#10),
  [PR #51](https://github.com/julienlegoux/external-reviewer/pull/51). Adds
  `internal/repo` — `git ls-files -z` enumeration with a walk as its fallback, and the
  one `exec` in this codebase, argv-only with the machine's git configuration
  neutralised — plus `internal/tools` with the registry and `list`. **Oversize**: 1654
  changed lines against a ~700-line `L` estimate; the implementation is 645 and the
  overrun is tests, split cleanly along the line the issue itself nominated.

  **Drift recorded**: one entry — enumeration goes through the `Scope`'s own `ReadDir`
  and `Stat`, not `Root.FS()` as SPECS names it, because an `fs.FS` over the root
  carries neither the allow-list nor the floor and `fs.WalkDir` cannot start at `"."`
  when a single subtree is granted. This settles the seam issue 01 left open.

* **Done**: Epic 2 issue 01 — [Confine the run with --allow and os.OpenRoot](https://github.com/julienlegoux/external-reviewer/issues/9) (#9)
  merged into `develop` as [PR #50](https://github.com/julienlegoux/external-reviewer/pull/50);
  issue closed. `internal/confine` is now the boundary every later tool reads through.

* **PR open**: Epic 2 issue 01 — [Confine the run with --allow and os.OpenRoot](https://github.com/julienlegoux/external-reviewer/issues/9) (#9),
  [PR #50](https://github.com/julienlegoux/external-reviewer/pull/50). Adds
  `internal/confine` — one `*os.Root` and the allow-list travelling as one `*Scope`, the
  three refusal rules and the non-configurable sensitive-file floor — plus the required
  repeatable `--allow` flag, and amends Epic 1 issue 02's success-case tests, whose
  grammar this issue deliberately tightens. **Oversize**: 1360 changed lines against a
  ~700-line `L` estimate, entirely tests and doc comments; the workable split line was the
  allow-list validation half, not the floor the issue nominated.

  **Drift recorded**: one entry — wire-path spellings are validated syntactically before
  `os.Root` sees them, which SPECS delegates wholly to `os.Root` and warns against
  hand-rolling. Layered before rather than in place of it, because the allow-list test,
  the two-OS absolute-path criterion and issue 06's no-I/O validation each need a cleaned
  relative path `os.Root` cannot supply.

## 2026-08-11

* **Retirement**: Epic 0 (Skeleton hardening) retired - 12 issues, milestone 4 closed.
  Consumed reports docs/REPORT_2.md. Fixed: the four surviving criterion-bearing mutants
  and the two terminations that escaped the done-line seam, plus the three contracts Epic
  2 extends — the stop-reason vocabulary in SPECS, the write-API guard in CONVENTIONS and
  `.golangci.yml`, and Epic 2's own issues 01 and 03 amended with Epic 1's structural
  warnings.

  `docs/REPORT_1.md` was **not** consumed by this lane — it was applied earlier as a
  revision to the twelve issues of Epics 1 and 2, recorded below on 2026-08-09.

  The folder `docs/epics/epic-0-skeleton-hardening/` is gone; milestone 4 and issues
  #22–#34 survive on GitHub and every link that pointed into the folder was repointed
  there first. The epic-0 slot is free for the next triage cycle.

* **Closed**: [Epic 0: Skeleton hardening](https://github.com/julienlegoux/external-reviewer/issues/22) (#22) —
  all **12 issues** `done` on 12 merged PRs (#35–#46, plus reconcile PR #47), milestone 4
  closed, tracking issue #22 closed. Every issue's bookkeeping was already reconciled by
  PR #47; the close verified it against GitHub rather than trusting it, and found no
  divergence.

  **Drift promoted**: one entry, swept from the run's log and verification files rather
  than from drift records — this epic's implementers wrote none. The decided test command
  (`go test ./... -race`) does not run on the development machine: a Windows Application
  Control policy blocks `go test`'s temp binaries, and `-race` needs a cgo C compiler that
  neither the Windows toolchain nor the `docker-desktop` WSL distro has, so the suite is
  cross-built and run under WSL and CI is the sole authority for `-race` and
  `windows-latest`. Triaged **`fix-now`** →
  [#48](https://github.com/julienlegoux/external-reviewer/issues/48), which should land
  before Epic 2's loop, that being the concurrency the race detector exists for. Recorded
  in [DRIFT](../planning/DRIFT.md).

  Three PR size overruns (#40, #44, #45 — up to 871 changed lines against a ~250-line `M`
  estimate, all under the `L` ceiling) were considered and deliberately **not** promoted:
  estimation calibration for test-heavy hardening issues, not the code contradicting a
  decided standard. They stay recorded per-PR below.

  **Environment cleaned**: 13 agent worktrees removed, and 13 issue/reconcile branches
  deleted locally and on `origin` — every one verified clean, fully pushed, and backed by
  a MERGED PR — plus 13 local `worktree-agent-*` scaffolding refs. `develop` is checked
  out, clean and pushed. [Epic 2](/epic-2-read-only-agentic-loop/EPIC_2.md) is unblocked:
  its issues already carry the two structural warnings issue 03 amended into them.

* **Merged**: [11 bound the turn and fix its cost and cancellation
  precedence](https://github.com/julienlegoux/external-reviewer/issues/33) (#33) —
  [PR #45](https://github.com/julienlegoux/external-reviewer/pull/45) merged into
  `develop`.

* **Merged**: [09 read the task prompt under a context, a byte bound and real
  validation](https://github.com/julienlegoux/external-reviewer/issues/31)
  (#31) — [PR #46](https://github.com/julienlegoux/external-reviewer/pull/46)
  merged into `develop`.

* **PR opened**: [11 bound the turn and fix its cost and cancellation
  precedence](https://github.com/julienlegoux/external-reviewer/issues/33) (#33) —
  [PR #45](https://github.com/julienlegoux/external-reviewer/pull/45) against
  `develop`. `Conversation.Next` keeps the adapter's service-tier-adjusted
  `Usage.Cost` and computes from the price sheet only where the adapter
  reported none; a `DefaultStreamTimeout` of ten minutes bounds the round trip
  from this side and is handed to the adapter as well, with a turn that ends on
  it reporting the new `reviewer.ErrStreamTimeout` — deliberately not a context
  error, so it takes SPECS' existing `failed` branch and adds no `stop=` word;
  and a second, non-cancellable read of `stream.Result` makes a message that is
  already there beat a cancellation that landed beside it, with `endedBecause`
  stating the remaining order once. `internal/cli`'s `asInterrupted` is removed
  as redundant with what `Next` now returns, leaving `interruptedError` the one
  decision point. Deleting the reviewer-side guard reproduces issue 05's
  intermittent `windows-latest` failure (`stop=failed` instead of
  `interrupted`) 3 runs out of 3. ~871 changed lines against a ~250 M target,
  386 of them the new test file. Evidence, including what was argued rather
  than observed, in the issue's verification log on
  [PR #45](https://github.com/julienlegoux/external-reviewer/pull/45).

* **Merged**: [12 escape provider-controlled text on stderr and finish diag's
  polish](https://github.com/julienlegoux/external-reviewer/issues/34)
  (#34) — [PR #44](https://github.com/julienlegoux/external-reviewer/pull/44)
  merged into `develop`.

* **PR opened**: [09 read the task prompt under a context, a byte bound and real
  validation](https://github.com/julienlegoux/external-reviewer/issues/31)
  (#31) — [PR #46](https://github.com/julienlegoux/external-reviewer/pull/46) against
  `develop`. `runReviewCommand`'s stdin read moves into `readPromptFromStdin(ctx,
  stdin)`: the read runs on its own goroutine feeding a buffered channel, selected
  against `ctx.Done()`, so a `SIGINT` (or any cancellation) reaching a blocked read
  now terminates through the ordinary `done`-line seam — `stop=interrupted` — instead
  of hanging forever; the goroutine left behind when `ctx` wins is a documented,
  bounded leak for the process's remaining lifetime. The read is also bounded to 1
  MiB via `io.LimitReader`, with a named `errPromptTooLarge` sentinel classification
  inspects through `errors.Is`. The emptiness test moves to
  `strings.TrimSpace(task) == ""`, catching `--prompt " "` and `echo |`'s lone
  newline, while the prompt actually sent to the model is never rewritten. The
  `--prompt ""` flag-presence check (`fs.Visit`) is unchanged and pinned against the
  `promptSet := *prompt != ""` collapse by a dedicated test, verified red by hand
  under the mutation. The real-`SIGINT` acceptance criterion was verified by hand
  against a real Linux kernel via `wsl -d docker-desktop` (issue 10's own environment
  for its SIGPIPE criterion), transcript on
  [PR #46](https://github.com/julienlegoux/external-reviewer/pull/46).
  No new stop-reason word — `usage` and `interrupted` already covered every path. 365
  insertions / 9 deletions (~374 changed lines) against a ~250 M estimate, under the
  ~500 target.

* **Merged**: [06 kill the four surviving mutants on the request side of the
  round trip](https://github.com/julienlegoux/external-reviewer/issues/28)
  (#28) — [PR #43](https://github.com/julienlegoux/external-reviewer/pull/43)
  merged into `develop`. Reconciled independently by issue 11's and issue 12's
  runs — the issue file still read `pr-open` though GitHub issue #28 was
  already closed.

* **PR opened**: [12 escape provider-controlled text on stderr and finish diag's
  polish](https://github.com/julienlegoux/external-reviewer/issues/34) (#34) —
  [PR #44](https://github.com/julienlegoux/external-reviewer/pull/44) against `develop`.
  `diag.escapeControlChars`, applied inside `WriteWarn` and `WriteError`, renders every
  control character — `\n`/`\r` and the ESC byte an ANSI/OSC sequence opens with — as its
  Go escape rather than letting it reach the terminal raw or split the line; a provider
  error body carrying a newline and a well-formed `done … stop=ok` line can no longer
  forge a second done line ahead of the real one. The false "kern-link redacts
  diagnostics upstream" comment is removed everywhere it appeared, including
  `docs/planning/SPECS.md` itself. `internal/diag/diag_test.go`'s tests are table-driven,
  `formatElapsed` gains an hour unit, and the duplicated done-line/turn-line transcript
  parsers move into `internal/fauxtest` as `ParseDoneLine`/`ParseTurnLine`, issue 05's
  shared harness. 706 changed lines against a ~280-line M estimate — over by roughly
  2.5x but under the 1000-line L ceiling, so opened as-is; the growth is almost entirely
  the table-driven rewrite (307 lines) and the two new integration tests proving the
  injection is actually stopped, both of which the issue's own size note anticipated as
  the bulk of the diff, just underestimated. Reconciled issue 05 to `done` at the start
  of this run (already merged, issue file still read `pr-open`) and merged `origin/develop`
  once to pick up issue 10's concurrently-merged PR #42, resolving a duplicate reconcile
  of issue 05 as a union and re-homing issue 10's new `stdout_test.go` onto the same
  shared parser.

* **Started**: [12 escape provider-controlled text on stderr and finish diag's
  polish](https://github.com/julienlegoux/external-reviewer/issues/34) (#34) —
  branch `issue-12-transcript-parsing-fauxtest`.

* **Merged**: [10 survive a failing or closed stdout without escaping the
  done-line seam](https://github.com/julienlegoux/external-reviewer/issues/32)
  (#32) — [PR #42](https://github.com/julienlegoux/external-reviewer/pull/42)
  merged into `develop`.

* **PR opened**: [06 kill the four surviving mutants on the request side of
  the round trip](https://github.com/julienlegoux/external-reviewer/issues/28)
  (#28) — [PR #43](https://github.com/julienlegoux/external-reviewer/pull/43)
  against `develop`. `internal/reviewer/run_test.go` (new, `package
  reviewer_test`) asserts the task prompt reaches the model verbatim from
  inside a scripted `faux.StepFunc`, that the unexported `messages`
  accumulator holds the user and assistant messages in order after a turn
  (observed black-box by inspecting the *second* turn's incoming context),
  and that `FinalText` concatenates text blocks in order while filtering
  thinking blocks and tool calls. `internal/cli/roundtrip_test.go` gains the
  same filtering claim asserted through `Run`, plus a cache-token test built
  on a hand-rolled `ai.Provider` (`resolve_test.go`'s `dynamicRegistry`
  pattern) since `faux`'s own cache simulation always reports zero without
  `SessionID` reuse, which `Conversation.Next` never does. All four named
  mutations applied by hand, watched red, reverted, watched green. Also
  reconciles issue 05 to `done` (PR #40 merged into `develop` while this
  issue was blocked on it). 263 changed lines against a ~300 M target.

* **Started**: [06 kill the four surviving mutants on the request side of
  the round trip](https://github.com/julienlegoux/external-reviewer/issues/28)
  (#28) — branch `issue-06-work`.

* **PR opened**: [10 survive a failing or closed stdout without escaping the done-line
  seam](https://github.com/julienlegoux/external-reviewer/issues/32) (#32) —
  [PR #42](https://github.com/julienlegoux/external-reviewer/pull/42) against
  `develop`. `Run` registers for `syscall.SIGPIPE`, so a broken reader on fd 1
  returns `EPIPE` instead of killing the process — `review … | head -20` now exits
  `2` with a `done` line rather than `141` with the transcript cut off.
  `syscall.SIGPIPE` exists on Windows too, so no `//go:build` divergence ships and
  the one path is exercised on both matrix OSes by a child-process test that selects
  its expectation at runtime. The report's single write is classified rather than
  all-or-nothing: a write that fails after N bytes names the report incomplete, with
  the bytes written and the report's length, through `diag.WriteError`. SPECS
  § Interfaces and CONVENTIONS § Error handling both said "stdout is empty on any
  failure", which a partial write makes false; both now state the three-part
  guarantee the code holds. No new stop-reason word — `failed` already covered it.
  408 insertions / 16 deletions (~424 changed lines) against a ~250 M estimate, the
  overrun being the two-OS subprocess harness.

* **Started**: [10 survive a failing or closed stdout without escaping the done-line
  seam](https://github.com/julienlegoux/external-reviewer/issues/32) (#32) —
  branch `issue-32-stdout-write-failures`.

* **Started**: [11 bound the turn and fix its cost and cancellation
  precedence](https://github.com/julienlegoux/external-reviewer/issues/33) (#33) —
  branch `issue-33-turn-hardening`.

* **Merged**: [05 extract one importable faux harness and retire the mutable
  package-level seams](https://github.com/julienlegoux/external-reviewer/issues/27)
  (#27) — [PR #40](https://github.com/julienlegoux/external-reviewer/pull/40)
  merged into `develop`. Reconciled independently by both issue 10's and issue
  12's runs — the issue file still read `pr-open` though GitHub issue #27 was
  already closed and the merge predates both issue 08's and issue 10's own PRs.

* **Merged**: [08 handle the repository path as an OS path and emit wire
  paths on stderr](https://github.com/julienlegoux/external-reviewer/issues/30)
  (#30) — [PR #41](https://github.com/julienlegoux/external-reviewer/pull/41)
  merged into `develop`.

* **PR opened**: [08 handle the repository path as an OS path and emit wire
  paths on stderr](https://github.com/julienlegoux/external-reviewer/issues/30)
  (#30) — [PR #41](https://github.com/julienlegoux/external-reviewer/pull/41)
  against `develop`. `runReviewCommand` now runs the positional repository
  path through `filepath.Clean` before `os.Stat`, and every diagnostic that
  reaches stderr converts it to a `filepath.ToSlash` wire form with the OS's
  own error sentence dropped entirely — the message states only what was
  attempted, lowercase and unpunctuated; classification stays on the wrapped
  error via `errors.Is`/`errors.As`. `TestRun_Review_PathDiagnostics_AreWireForm`
  is table-driven over the nonexistent-path and not-a-directory cases and
  builds its expectation from `filepath.ToSlash` of the same native path,
  with no `runtime.GOOS` branch; run against the pre-fix source it
  reproduced the exact reported bug (double-escaped backslashes, the OS's
  capitalised sentence) on this Windows dev machine. 84 changed lines (22
  production, 62 test) against a ~200 M target.

* **Started**: [08 handle the repository path as an OS path and emit wire
  paths on stderr](https://github.com/julienlegoux/external-reviewer/issues/30)
  (#30) — branch `issue-08-path-boundary`.

* **Merged**: [07 write one prefixed error: line from diag on every exit-2
  path](https://github.com/julienlegoux/external-reviewer/issues/29) (#29) —
  [PR #39](https://github.com/julienlegoux/external-reviewer/pull/39) merged into
  `develop`.

* **Fix**: [05 extract one importable faux harness and retire the mutable
  package-level seams](https://github.com/julienlegoux/external-reviewer/issues/27)
  (#27) — [PR #40](https://github.com/julienlegoux/external-reviewer/pull/40).
  A windows-latest `go test ./... -race` run caught
  `TestRun_CancelledMidStream_ExitsTwo` classifying a mid-stream cancellation
  as `stop=failed` instead of `interrupted` — the identical commit had passed
  the same job moments earlier, so this was a genuine race rather than a bad
  assertion: kern-link's `Stream.Result(ctx)` selects between its result
  channel and `ctx.Done()`, and when the provider's own goroutine wins with a
  `StopReasonAborted` message before `Result`'s select runs, the resulting
  error carries no wrapped cancellation cause for `errors.Is` to find.
  `asInterrupted` (`internal/cli/review.go`) wraps a failing turn's error
  with `ctx.Err()` whenever `ctx` has actually ended, regardless of which
  race produced the error's exact shape; a nil `turnErr` — a complete,
  successful message — is never touched. Pinned by
  `TestAsInterrupted_ReclassifiesFailingTurnWhenContextEnded`
  (`run_test.go`), which constructs both of the helper's inputs by hand
  rather than racing a real stream. The full cancellation-*precedence*
  restructuring — preferring a complete message over a late `ctx.Err()`,
  making the guard reachable from `internal/reviewer`'s own suite — is left
  to [issue 11](https://github.com/julienlegoux/external-reviewer/issues/33),
  which owns it; this fix stays at the classification seam only.

* **PR opened**: [05 extract one importable faux harness and retire the mutable
  package-level seams](https://github.com/julienlegoux/external-reviewer/issues/27)
  (#27) — [PR #40](https://github.com/julienlegoux/external-reviewer/pull/40)
  against `develop`. `internal/fauxtest` (new, non-`_test.go`, imported by no
  product file) carries the auth-provider decorator, the four auth shapes and
  one registry builder with options, replacing three divergent copies across
  `internal/reviewer/resolve_test.go`, `internal/cli/preflight_test.go` and
  `internal/cli/roundtrip_test.go`. `var performReview` and `var models`
  (`internal/cli/review.go`) are retired: the registry now reaches
  `resolveAndReview` as an explicit parameter threaded from `run`'s inner seam,
  with `export_test.go` exposing `RunForTest`/`RunWithModelsForTest`
  constructors instead of setters. `exit_test.go`'s two stubbed terminations
  are re-homed onto the faux-driven tests that already cover the same exit
  codes for real; the two-level `ErrNoReviewer` wrap survives as a new test
  asserted directly against `classify`. No behaviour change — every prior
  assertion still exists, in the same or a more direct form. 325 insertions /
  374 deletions across 11 files (~700 changed lines, including this run's
  issue-04 reconcile and issue-05 bookkeeping) against a ~500-line M target.
  Merged `origin/develop` twice more since — issue 07's PR #39, then issue
  08's PR #41 — resolving real conflicts in `internal/cli`'s `run.go`,
  `review.go` and their tests each time: the resolution keeps issue 07's
  `diag.WriteError` routing and interruption-check deletions, issue 08's
  OS-path/wire-path boundary in `runReviewCommand`, and issue 05's explicit
  registry threading and `asInterrupted` fix all alongside each other.

* **Started**: [05 extract one importable faux harness and retire the mutable
  package-level seams](https://github.com/julienlegoux/external-reviewer/issues/27)
  (#27) — branch `issue-05-fixture-package`.

* **PR opened**: [07 write one prefixed error: line from diag on every exit-2
  path](https://github.com/julienlegoux/external-reviewer/issues/29) (#29) —
  [PR #39](https://github.com/julienlegoux/external-reviewer/pull/39) against
  `develop`. `diag.WriteError` becomes the one writer every hand-rolled
  `fmt.Fprint*` in `internal/cli` routed through, including the previously
  silent empty-argv path; `fs.SetOutput(io.Discard)` and a `flag.ErrHelp`
  split stop `review --help`/`review -h` from exiting 2 and stop flag's own
  usage dump from reaching stderr; the three "was this interrupted?" checks
  collapse to the one in `runReview`, deleting the top-of-`run()`
  short-circuit and `reviewer.Conversation.Next`'s belt-and-braces check
  (both confirmed redundant against the kern-link v0.1.1 pin by running the
  full suite with each removed). 294 insertions / 55 deletions (~349 changed
  lines) against a ~350 M target.

* **Started**: [07 write one prefixed error: line from diag on every exit-2
  path](https://github.com/julienlegoux/external-reviewer/issues/29) (#29) —
  branch `issue-07-error-line-seam`.

* **Merged**: [04 enforce gofmt in CI, normalise line endings, and lint on both
  OSes](https://github.com/julienlegoux/external-reviewer/issues/26) (#26) —
  [PR #38](https://github.com/julienlegoux/external-reviewer/pull/38) merged into
  `develop`.

* **PR opened**: [04 enforce gofmt in CI, normalise line endings, and lint on both
  OSes](https://github.com/julienlegoux/external-reviewer/issues/26) (#26) —
  [PR #38](https://github.com/julienlegoux/external-reviewer/pull/38) against
  `develop`. A top-level `formatters:` block (`gofmt`) in `.golangci.yml`,
  `.gitattributes` (`* text=auto eol=lf`) with `git add --renormalize .` a
  verified no-op against the LF-only index, and the `lint` job joined onto the
  `[ubuntu-latest, windows-latest]` matrix. 7 files changed, 23 insertions / 7
  deletions (~30 changed lines) against a ~40-line target.

* **Started**: [04 enforce gofmt in CI, normalise line endings, and lint on both
  OSes](https://github.com/julienlegoux/external-reviewer/issues/26) (#26) —
  branch `issue-26-ci-formatting-gate`.

* **Merged**: [02 widen the write-API guard to every write operation CONVENTIONS
  names](https://github.com/julienlegoux/external-reviewer/issues/24) (#24) —
  [PR #36](https://github.com/julienlegoux/external-reviewer/pull/36) merged into
  `develop`.

* **Merged**: [01 settle the done line's stop-reason vocabulary and document it in
  SPECS](https://github.com/julienlegoux/external-reviewer/issues/23) (#23) —
  [PR #37](https://github.com/julienlegoux/external-reviewer/pull/37) merged into
  `develop`.

* **PR opened**: [02 widen the write-API guard to every write operation CONVENTIONS
  names](https://github.com/julienlegoux/external-reviewer/issues/24) (#24) —
  [PR #36](https://github.com/julienlegoux/external-reviewer/pull/36) against `develop`.
  CONVENTIONS' enumeration and `.golangci.yml`'s `forbidigo.forbid` list both gain
  `os.Chmod`, `os.Chtimes`, `os.Truncate`, `os.Link`, `os.Root.Chmod`,
  `(*os.File).Write`, `(*os.File).WriteString` and `(*os.File).Truncate`, plus
  `os.Root.Link` named in CONVENTIONS for the first time; a probe file
  (`testdata/probe/probe.go`, skipped by `golangci-lint`'s default `./...`
  expansion) verifies all nine now fire and the repository itself still reports zero
  findings. 53 insertions / 6 deletions across 5 files (~59 changed lines) against a
  ~60-line target.

* **Started**: [02 widen the write-API guard to every write operation CONVENTIONS
  names](https://github.com/julienlegoux/external-reviewer/issues/24) (#24) —
  branch `issue-24-write-api-guard-coverage`.

* **Merged**: [03 carry Epic 1's structural warnings into Epic 2's
  issues](https://github.com/julienlegoux/external-reviewer/issues/25) (#25) —
  [PR #35](https://github.com/julienlegoux/external-reviewer/pull/35) merged into
  `develop`.

* **Amended**: [Epic 2](/epic-2-read-only-agentic-loop/EPIC_2.md) issue
  [03 — multi-turn loop and dispatch](/epic-2-read-only-agentic-loop/issues/03-multi-turn-loop-and-dispatch.md)
  (#11) and issue
  [01 — allow-list and root confinement](/epic-2-read-only-agentic-loop/issues/01-allow-list-and-root-confinement.md)
  (#9), carrying two structural warnings Epic 1's review turned up
  ([report 2](../REPORT_2.md) § "Accounting lives on the wrong side of the seam
  Epic 2's loop needs" and its `os.Stat` row) into the issues that walk into them.
  Issue 03 now states that the accumulated turn/cost totals `bounds.check(state)`
  needs live in `internal/cli` (`recordTurn`, `review.go:88-105`) while the loop
  lands in `internal/reviewer` (`Conversation.Next`, `run.go:61-98`), and that
  `internal/reviewer` wins — the accumulation and `diag.WriteTool`
  (`diag.go:40-43`) both move into the loop. Issue 01 now states that its
  `os.OpenRoot` confinement must not build on the existing symlink-following,
  TOCTOU-prone `os.Stat` at `review.go:171`. Both GitHub issue bodies re-synced;
  neither issue's size, status or `gh_issue` changed. Both of DRIFT's Epic 1
  entries naming this confinement work as their revisit trigger stay open —
  amending the issues is not discharging them, only Epic 2's own PR can do that.

* **PR opened**: [03 carry Epic 1's structural warnings into Epic 2's
  issues](https://github.com/julienlegoux/external-reviewer/issues/25) (#25) —
  [PR #35](https://github.com/julienlegoux/external-reviewer/pull/35) against
  `develop`. 56 changed lines against an S-sized ~500 target, all under
  `docs/epics/`: two new Scope bullets on Epic 2 issue 03, one on issue 01, both
  GitHub bodies re-synced.

* **Started**: [03 carry Epic 1's structural warnings into Epic 2's
  issues](https://github.com/julienlegoux/external-reviewer/issues/25) (#25) —
  branch `issue-25-epic-2-plan-amendments`.

* **Creation**: Cut [Epic 0: Skeleton hardening](https://github.com/julienlegoux/external-reviewer/issues/22) into
  **12 issues** on milestone 4, all linked as native sub-issues of
  [#22](https://github.com/julienlegoux/external-reviewer/issues/22) — none sized `L`.
  The epic's 15 scope groups compress to 12 PRs because three pairs are one change:
  a whitespace-only prompt, an unbounded `io.ReadAll` and an uninterruptible stdin read
  are one rewrite of the same call; cost, timeout and cancellation precedence are three
  repairs inside one 40-line `Conversation.Next`; and the escaping of provider-controlled
  text lands beside `internal/diag`'s own polish.

  **Order follows the epic's own rule** — plan and standard repairs first, since each one
  changes the contract the code is judged against
  ([#23](https://github.com/julienlegoux/external-reviewer/issues/23) the stop-reason
  vocabulary, [#24](https://github.com/julienlegoux/external-reviewer/issues/24) the
  write-API guard, [#25](https://github.com/julienlegoux/external-reviewer/issues/25) the
  amendments to Epic 2's own issues), then the CI formatting gate
  ([#26](https://github.com/julienlegoux/external-reviewer/issues/26)) because it protects
  every PR after it, then the code. Two deviations from the epic's narrative order were
  deliberate: the shared `faux` harness
  ([#27](https://github.com/julienlegoux/external-reviewer/issues/27)) moves *ahead* of the
  test work rather than after it, since every remaining issue scripts `faux` and the point
  of the harness is that they build on one copy; and `#24` blocks `#26` only because both
  edit `.golangci.yml`.

  **Three issues carry a judgment the implementer must make explicit rather than
  discover.** `#24`'s probe file cannot be committed where CI lints it — it would fail the
  repository's own gate — so the verification is a recorded manual run. `#32`'s invariant
  ("stdout is empty on any failure") cannot be discharged by retracting bytes already
  written, so the issue asks which of the three defensible readings ships and requires
  SPECS to be amended in the same PR if the wording has to change. And `#31`'s Ctrl-C
  criterion is hand-verified, since delivering a real `SIGINT` is not portable to the
  Windows runner — the constraint issue 03 of Epic 1 already documented.

  Labels follow the repository's existing scheme rather than a new one: `bug` for the
  defect repairs, `documentation` for the plan amendments, `enhancement` for the harness
  extraction.

* **PR opened**:
  [01 — Settle the done line's stop-reason vocabulary and document it in SPECS](https://github.com/julienlegoux/external-reviewer/issues/23)
  (issue [#23](https://github.com/julienlegoux/external-reviewer/issues/23)) as
  [PR #37](https://github.com/julienlegoux/external-reviewer/pull/37) against `develop`.

## 2026-08-10

* **Creation**: Established
  [Epic 0: Skeleton hardening](https://github.com/julienlegoux/external-reviewer/issues/22) by converting
  [implementation review report 2](../REPORT_2.md) — milestone 4, issue
  [#22](https://github.com/julienlegoux/external-reviewer/issues/22). The remediation
  lane, which runs before [Epic 2](/epic-2-read-only-agentic-loop/EPIC_2.md) resumes and
  is retired once every issue is `done` and its milestone closed.

  **36 findings extracted, 0 dropped as stale.** No non-docs commit exists between the
  reviewed range (`9bb1745^1..84f3d21`) and this conversion, and every cited `file:line`
  was re-verified against the working tree; no GitHub issue covered any finding, and
  DRIFT's two entries were the ones the report had already excluded. Report 2 marked most
  findings `fix-now` but filed no issues for them at the user's direction, so they came
  back into the triage as the report intended. **34 were grouped into 15 repairs**, one
  became drift, and **2 were recorded `won't-fix`**: the invisible red phase across Epic
  1's five PRs (unverifiable retroactively, no code change discharges it — real feedback
  for `implement-issue` instead), and `diag.WriteTool`/`reviewRequest.RepoPath` (dead by
  design, callers due in Epic 2).

  Grouping is by the repair, not the finding, which is where the compression came from:
  the four surviving criterion-bearing mutants are one `run_test.go`; the two P0s are
  among them; the four shapes of the `error:` line, the silent empty-argv path, the three
  independent "was this interrupted?" checks and `review --help` exiting `2` are one
  consolidation through `diag`.

  **Three questions were decided rather than deferred.** The `done` line's `stop=` field
  carries **both** vocabularies — the model's own reason on the success path, the CLI word
  on every termination the model never reaches — because they answer different questions
  and a usage error can never be `end_turn`; that makes SPECS' worked example literally
  true and Epic 2's first acceptance criterion satisfiable without amending either.
  SPECS will document the closed set, since it is a union that grows per epic and a
  calling skill cannot parse what is written nowhere. And CONVENTIONS is the source of
  truth for the forbidden write-API list with `.golangci.yml` mirroring it, which turns
  an aspiration into a checkable criterion — the current list lets eight write operations
  through, including `(*os.File).Write`, and Epic 2 is the epic that starts opening file
  handles.

  Plan and standard repairs are ordered before the code fixes, and the CI formatting gate
  before the rest of the code, so it protects every PR after it.

## 2026-08-09

* **Epic closed**: [Epic 1: Walking
  skeleton](/epic-1-walking-skeleton/EPIC_1.md) (#1) — all five issues `done` and
  merged into `develop` (PRs #16–#20, plus reconcile #21), tracking issue closed,
  milestone 1 closed. Both drift records promoted into
  [DRIFT](../planning/DRIFT.md) and triaged `accepted`, with SCOPE, `scope/09` and
  SPECS amended so the standards stop contradicting the code: the `kern-link` pin is
  now v0.1.1, and the no-files guarantee now says what it actually covers. Six agent
  worktrees removed and every merged branch deleted local and remote. Epic 2 is
  unblocked.

* **Merged**: [05 call the model once and return its markdown on
  stdout](/epic-1-walking-skeleton/issues/05-single-turn-model-call.md) (#8) —
  [PR #20](https://github.com/julienlegoux/external-reviewer/pull/20) merged into
  `develop`.

* **PR opened**: [05 call the model once and return its markdown on
  stdout](/epic-1-walking-skeleton/issues/05-single-turn-model-call.md) (#8) —
  [PR #20](https://github.com/julienlegoux/external-reviewer/pull/20) against
  `develop`. `internal/reviewer.Conversation` (the accumulating `[]ai.Message`
  and one `StreamSimple` + `Result` round trip, with `ai.CalculateCost` per
  turn), the reviewer's markdown written to stdout verbatim once at the end,
  and every way one turn can end without a usable report classified as exit
  `2`. The walking skeleton walks: a hand run against the real
  `openai-codex/gpt-5.5` returned markdown on stdout and exited `0`, retiring
  the first half of the project's largest risk. 784 changed lines against a
  ~450 target — 220 of production code, 447 of tests for the six termination
  paths and the untouched-tree assertion, and 117 of docs. Drift:
  [the no-files guarantee excludes kern-link's credential
  store](/epic-1-walking-skeleton/drift/05-credential-store-writes.md).

* **Started**: [05 call the model once and return its markdown on
  stdout](/epic-1-walking-skeleton/issues/05-single-turn-model-call.md) (#8) —
  branch `issue-8-single-turn-model-call`.

* **Merged**: [04 resolve the hard-coded reviewer model and preflight
  auth](/epic-1-walking-skeleton/issues/04-model-resolution-and-auth.md) (#7) —
  [PR #19](https://github.com/julienlegoux/external-reviewer/pull/19) merged into
  `develop`.

* **PR opened**: [04 resolve the hard-coded reviewer model and preflight
  auth](/epic-1-walking-skeleton/issues/04-model-resolution-and-auth.md)
  (#7) — [PR #19](https://github.com/julienlegoux/external-reviewer/pull/19)
  against `develop`. `internal/reviewer` (the `Resolver` running
  `Refresh` → `GetModel` → `GetAuth`, the hard-coded `openai-codex/gpt-5.5`,
  and `ErrNoReviewer` moved to where it is produced), the pre-flight wired
  into the review seam, and a `model` diagnostics line carrying only
  `AuthResult.Source`. 1144 changed lines against a ~300 target — 220 of
  production code, the rest tests for the eight acceptance criteria,
  generated `go.sum`, and the drift record. Drift:
  [kern-link v0.2.0 does not exist](/epic-1-walking-skeleton/drift/04-kern-link-version-pin.md);
  pinned at v0.1.1, with no API-name divergence.

* **Started**: [04 resolve the hard-coded reviewer model and preflight
  auth](/epic-1-walking-skeleton/issues/04-model-resolution-and-auth.md)
  (#7) — branch `issue-7-model-resolution-and-auth`.

* **Merged**: [03 emit run diagnostics on stderr and classify exit
  codes](/epic-1-walking-skeleton/issues/03-diagnostics-and-exit-codes.md) (#6) —
  [PR #18](https://github.com/julienlegoux/external-reviewer/pull/18) merged into
  `develop`.

* **PR opened**: [03 emit run diagnostics on stderr and classify exit
  codes](/epic-1-walking-skeleton/issues/03-diagnostics-and-exit-codes.md)
  (#6) — [PR #18](https://github.com/julienlegoux/external-reviewer/pull/18)
  against `develop`. `internal/diag` (the prefixed stderr writer and the run
  `State` the `done` line renders from), `internal/cli/exit.go`
  (`ErrNoReviewer` and `errors.Is`-based classification), and
  `signal.NotifyContext` wired for `SIGINT` over an inner `run(ctx, …)`, so
  every termination path — success, no reviewer, failure, interruption —
  emits exactly one `done` line via a single `defer`. 530 changed lines
  (226 production, 304 tests) against a ~400 target, driven mostly by the
  three termination-path tests and full `internal/diag` coverage.

* **Started**: [03 emit run diagnostics on stderr and classify exit
  codes](/epic-1-walking-skeleton/issues/03-diagnostics-and-exit-codes.md)
  (#6) — branch `issue-6-diagnostics-and-exit-codes`.

* **Merged**: [02 parse the review invocation and exit 2 on usage
  errors](/epic-1-walking-skeleton/issues/02-review-invocation-parsing.md) (#5) —
  [PR #17](https://github.com/julienlegoux/external-reviewer/pull/17) merged into
  `develop`.

* **PR opened**: [02 parse the review invocation and exit 2 on usage
  errors](/epic-1-walking-skeleton/issues/02-review-invocation-parsing.md) (#5) —
  [PR #17](https://github.com/julienlegoux/external-reviewer/pull/17) against
  `develop`. `review [--prompt <text>] <repo-path>` and `help`/`--help`/`-h`
  over stdlib `flag`, `ContinueOnError` throughout; every malformed
  invocation exits `2` with stdout empty and a reason on stderr. 301 changed
  lines in `internal/cli/` against a ~350 target.

* **Started**: [02 parse the review invocation and exit 2 on usage
  errors](/epic-1-walking-skeleton/issues/02-review-invocation-parsing.md) (#5) —
  branch `issue-5-review-invocation-parsing`.

* **Merged**: [01 scaffold the Go module, the run() seam and
  CI](/epic-1-walking-skeleton/issues/01-scaffold-module-and-ci.md) (#4) —
  [PR #16](https://github.com/julienlegoux/external-reviewer/pull/16) merged into
  `develop`.

* **PR opened**: [01 scaffold the Go module, the run() seam and
  CI](/epic-1-walking-skeleton/issues/01-scaffold-module-and-ci.md) (#4) —
  [PR #16](https://github.com/julienlegoux/external-reviewer/pull/16) against
  `develop`. `go.mod`, `main.go`, `internal/cli.Run`, `.golangci.yml` (v2 schema,
  `forbidigo` guard on the write half of the filesystem API and
  `exec.Command`/`exec.CommandContext`), and `.github/workflows/ci.yml`
  (`test` on ubuntu-latest/windows-latest, `lint` pinned to golangci-lint
  `v2.12.2`). 184 changed lines against a ~150 target.

* **Started**: [01 scaffold the Go module, the run() seam and
  CI](/epic-1-walking-skeleton/issues/01-scaffold-module-and-ci.md) (#4) — branch
  `issue-4-scaffold-module-and-ci`.

* **Revision**: Applied [issue review report 1](../REPORT_1.md) to the twelve issues of
  Epics 1 and 2, and pushed the corrected bodies to #4–#15.

  Two P1s. **`git_read status` was exempted from confinement** while the other three
  subcommands were scoped — an unscoped `status` reports the working-tree state of the
  whole repository, so it handed the model every path name the allow-list and the
  sensitive-file floor exist to hide. It is now pathspec-scoped like `log` and `diff`,
  with its own acceptance criterion. And **the Windows reserved-device criterion was
  unsatisfiable** — it asked one test to assert refusal on both matrix OSes, where `NUL`
  and `COM1` are ordinary Linux filenames; the cheapest way to green it would have been a
  hand-rolled blocklist, which is precisely what `os.Root` was chosen to avoid. The
  expectation is now selected at runtime from `runtime.GOOS`.

  Three things were **decided** rather than merely corrected: the `done` line carries the
  accumulated token counts (SPECS § Interfaces amended to match, [planning
  log](../planning/log.md)); `--allow` grants **subtrees only**, a single-file grant being
  a usage error, since every downstream mechanism reads it as a subtree; and the caps
  `list` and `git_read` announce truncation against are stated numbers — 200 paths and
  2000 lines — rather than constants each implementer would have invented differently.

  **Sizing was compressed and is now honest.** Eleven of twelve issues carried a
  byte-identical size note telling an M issue to target 500 lines and split past 1000,
  which authorised an L-sized PR on every one of them. Six issues were re-declared
  (epic-1/01 → S; epic-2/01, 02, 03, 05, 06 → L) and every note now names its own target
  and its own split line. Only epic-2/02 has a real split line — enumeration and the
  `exec` helper, then the registry and `list`.

  Two findings were closed **without changing the files**. Cross-epic blockers are
  invisible in `depends_on` because the schema scopes it to one epic; the guard is
  `EPIC_2.md`'s epic-level dependency, and the fix — if it recurs — belongs in the schema.
  And no issue states the branch model (`develop`, `issue-<n>-<slug>`, `Closes #<n>`):
  the pipeline skills own it — `implement-epic` selects `develop` as the integration
  branch, `implement-issue` reads CONVENTIONS § Git & PRs — so repeating it twelve times
  would duplicate a rule that already has an enforcer.

* **Creation**: Cut
  [Epic 2](/epic-2-read-only-agentic-loop/EPIC_2.md) into seven issues — issues #9
  through #15 on milestone 2, each linked as a native sub-issue of #2.

  The order is boundary → loop → tools → measurement:
  [01 the allow-list and the `os.Root` boundary](/epic-2-read-only-agentic-loop/issues/01-allow-list-and-root-confinement.md),
  [02 enumeration and `list`](/epic-2-read-only-agentic-loop/issues/02-enumeration-and-list-tool.md),
  [03 the multi-turn loop and dispatch](/epic-2-read-only-agentic-loop/issues/03-multi-turn-loop-and-dispatch.md),
  [04 `read_file`](/epic-2-read-only-agentic-loop/issues/04-read-file-tool.md),
  [05 `search`](/epic-2-read-only-agentic-loop/issues/05-search-tool.md),
  [06 `git_read`](/epic-2-read-only-agentic-loop/issues/06-git-read-tool.md),
  [07 the hand-run measurements](/epic-2-read-only-agentic-loop/issues/07-hand-run-measurements.md).

  Three judgments worth recording. **The loop lands at 03 with only one tool wired**,
  rather than after all four: every later tool PR is then small and assertable end-to-end
  through a working loop, instead of three tools and a loop integrating in one review.
  **Enumeration and `list` ship together** (02) because enumeration alone has no
  observable behaviour, and because that PR carries the one `exec` helper the forbidigo
  guard tolerates — which 06 then reuses rather than re-deriving. And **the measurement
  is an issue, not a checklist item**: it is the epic's second deliverable and Epic 3's
  loop caps have no input without it, so it gets a PR, a document in the bundle
  (`MEASUREMENTS.md`) and an acceptance criterion that no `.go` file changes in it.

  Still greenfield — Epic 1's PRs are unmerged — so every issue's "Relevant files /
  areas" again states that its paths follow the layout SPECS fixes rather than a verified
  tree.

* **Creation**: Cut
  [Epic 1](/epic-1-walking-skeleton/EPIC_1.md) into five issues in a strict
  dependency chain — issues #4 through #8 on milestone 1, each linked as a native
  sub-issue of #1.

  The split is by layer rather than by feature, so each PR closes one seam:
  [01 scaffolding and CI](/epic-1-walking-skeleton/issues/01-scaffold-module-and-ci.md),
  [02 the invocation grammar](/epic-1-walking-skeleton/issues/02-review-invocation-parsing.md),
  [03 diagnostics and exit codes](/epic-1-walking-skeleton/issues/03-diagnostics-and-exit-codes.md),
  [04 model resolution and auth pre-flight](/epic-1-walking-skeleton/issues/04-model-resolution-and-auth.md),
  [05 the single round trip](/epic-1-walking-skeleton/issues/05-single-turn-model-call.md).

  Two judgments worth recording. **The output contract is built before there is anything
  to output** (03 precedes 04 and 05): the `done`-on-every-path guarantee and the
  `1`-versus-`2` classification are what every later epic depends on, and retrofitting
  them around a working call is how a termination path gets missed. And **`kern-link`
  enters the module at 04, not at 01**, because a require line with no import is removed
  again by `go mod tidy` — 01's `go.mod` names the module and Go 1.26 only.

  The repository is greenfield (only `docs/` and `.git`), so every issue's "Relevant
  files / areas" states that its paths follow the layout SPECS fixes rather than a
  verified tree.

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
