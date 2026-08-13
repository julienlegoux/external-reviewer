# Issue Review Report

## Scope

- Reviewed: `docs/epics/epic-3-reviewer-selection-integration/issues/*.md` — eleven issue
  files (01–11) — and `issues/index.md`
- Reviewed against: `docs/epics/epic-3-reviewer-selection-integration/EPIC_3.md`, with
  `docs/planning/SPECS.md`, `docs/planning/SCOPE.md`, `docs/planning/CONVENTIONS.md`,
  `docs/planning/DRIFT.md`, the cited `specs/` and `scope/` decisions, and
  `docs/epics/index.md` / `docs/epics/log.md` read for context. The current Go tree was
  read to check the issues' claims about existing code.
- GitHub verification: verified — `gh` authenticated against both
  `julienlegoux/external-reviewer` and `julienlegoux/skills`
- External review: attempted and not usable — `opencode` is installed, but the two
  delegated passes (`google/gemini-3.1-pro-preview`, after `google/gemini-3.1-pro-preview`
  and `openai/gpt-5.6` first failed on credentials and on a Codex account restriction) had
  not returned after ~20 minutes and the review was completed natively. This is exactly the
  fallback the contract prescribes, but it means no second-family model looked at these
  files.

## Findings

### P2 — Issues 01 and 08 build behaviour EPIC_3 never states

- Location: `issues/01-git-read-path-scoped-diff.md:50-72` (a new `paths` wire parameter on
  `git_read`, plus SPECS and specs 09 amendments);
  `issues/08-request-object-and-version.md:43-61` (stdin for `review` becomes a JSON
  `{"system","task"}` object, `--system`, and a `version` subcommand)
- Source: `EPIC_3.md:29-59` (`## Scope` — seven bullets, none of which is a tool
  parameter, a prompt channel or `version`) and `EPIC_3.md:72-95` (`## Acceptance
  criteria`, likewise silent on both)
- Problem: both additions are legitimate and traceable — `SPECS.md:189-196` already
  specifies `--system` and `version` in the `review` grammar, and `docs/epics/log.md:34-46`
  records why each was pulled in — but EPIC_3 itself was never amended. The epic a later
  reader opens does not contain the scope its own issues will ship.
- Impact: `close-epic` and `review-implementation` grade the merged diff against the
  **epic's** acceptance criteria. Two issues' worth of shipped behaviour — a new wire
  parameter and a breaking change to how `review` reads stdin — will read there as
  unexplained scope. Nothing in EPIC_3 explains why issue 01 is the epic's first issue
  either.
- Recommendation: fold the two additions into `EPIC_3.md`'s `## Scope` and
  `## Acceptance criteria`, reusing the reasoning already written at
  `docs/epics/log.md:34-46`. Do not weaken the issues; the epic is the file that is wrong.

### P2 — `gh_issue: 37` on the cross-repo issue names a different, already-merged issue in this repo

- Location: `issues/10-review-interfaces-external-reviewer-swap.md:14` — `gh_issue: 37`,
  with `resource:` at line 7 carrying the only repository qualifier
  (`https://github.com/julienlegoux/skills/issues/37`)
- Source: `_shared/pipeline-interfaces.md` § Issue file — `gh_issue` is an unqualified
  integer, and § Issue status lifecycle makes it the handle every status mirror and the
  final close are written against
- Problem: verified — `gh issue view 37 --repo julienlegoux/external-reviewer` returns
  *"Settle the done line's stop-reason vocabulary and document it in SPECS"*, state
  **MERGED**. The number is live in this repo and points at unrelated, finished work.
- Impact: `implement-issue` and `implement-epic` mirror `open → in-progress → pr-open →
  done` onto `gh_issue` and close it at `done`. Run from this repository — which is where
  the bundle lives and where those skills are invoked — they would comment on and attempt
  to close a merged Epic-0 issue, and the skills-repo issue would never advance. The epic's
  progress bar (which does track `julienlegoux/skills#37` as a native sub-issue, verified)
  would then disagree with the file.
- Recommendation: qualify the field so no consumer can resolve it locally —
  `gh_issue: julienlegoux/skills#37`, or an explicit `repo:` extension field — and repeat
  it in the ⚠ block at `issues/10-…md:21-27`. This is also a gap in
  `pipeline-interfaces.md`, which has no way to express a cross-repository issue; worth
  raising there rather than solving twice.

### P2 — Issue 11 does not depend on issue 09, though its criteria cannot be answered without it

- Location: `issues/11-success-criteria-verification-run.md:14` — `depends_on: [10]`
- Source: the same file's own criteria and scope: `:86-88` (*"The document states whether
  any cap bit, and if none did, how close the run came to each"*) and `:52-56` (*"did the
  caps set in issue 09 leave enough headroom, and did closing the `git_read` gap
  (issue 01) move the turn count the way MEASUREMENTS predicted"*). Issue 09 is what sets
  `Bounds` (`issues/09-loop-caps-from-measurements.md:89`); issue 01 is what closes the gap
  (`issues/01-…md:50-59`). Issue 09 declares *"Blocks: None strictly"* at `:132`.
- Problem: two of issue 11's acceptance criteria are about caps and a closed gap that
  issue 11 does not require to have landed.
- Impact: `implement-epic` picks the next unblocked issue from `depends_on`. With 10
  merged and 09 still open, issue 11 is eligible — and would measure a binary whose
  `Bounds` is still the zero value (`internal/cli/run.go:54`). The cap questions then have
  no answer, and the run — a real, quota-consuming, hand-supervised review — has to be
  done again.
- Recommendation: `depends_on: [1, 9, 10]` on issue 11, and change issue 09's `:132`
  "Blocks" line to name issue 11 rather than "None strictly".

### P2 — `models` hides every dynamic provider unless `--refresh` is passed, and no criterion catches it

- Location: `issues/07-models-command.md:46-47` (*"`--refresh` is opt-in, because it costs
  network calls"*) and `:74` (*"Without `--refresh`, no `Refresh` call is made at all"*) —
  with no criterion covering what the user then sees
- Source: `EPIC_3.md:41-44` — dynamic providers *"hold no models until then, so skipping it
  makes a correctly configured tier fall back silently"*; and
  `docs/planning/specs/05-tier-assignment-schema.md:110-113`, which states the same
  mechanism (`GetModel` returns nil until `Refresh` runs) in the same breath as making
  `--refresh` opt-in
- Problem: `models` is the command journeys 3 and 4 exist around — it is how a user picks
  what to assign to a tier. For `openrouter`, `vercel-ai-gateway`, `nvidia` and
  `github-copilot` it prints **nothing** by default, and issue 07's criteria assert only
  that no `Refresh` happens, never that the emptiness explains itself.
- Impact: the precise silent-empty failure the epic names, reproduced in the command built
  to prevent it. A user with an `openrouter` credential runs `models`, sees no openrouter
  rows, and concludes the provider is unreachable.
- Recommendation: add a criterion — a credentialed provider reporting
  `CanRefreshModels()` whose catalog is empty is named on stderr as needing `--refresh`
  (or renders a row saying so), asserted with a registry holding one such provider.

### P2 — Every code issue is graded `M` on identical boilerplate, against an Epic 2 precedent of `L` for the same shapes

- Location: `size: M` at line 11 of issues 01–10, each closing with the same sentence —
  *"Target ~500 changed lines; if this grows past ~1000, split it before opening the PR"*
  (`02:126`, `04:144`, `05:131`, `06:119`, `07:110`, and so on)
- Source: `_shared/pipeline-interfaces.md` § Issue file — S ≈ under 200, M ≈ 200–500,
  L ≈ 500–1000, *"L is the ceiling, not the target"*; and
  `docs/epics/epic-2-read-only-agentic-loop/issues/index.md:6-11`, where a new package plus
  its tool and tests was graded **L** five times out of seven
- Problem: "~500" is the top of M, not a point inside it, so the note and the field
  disagree in spirit. Measured against the Epic 2 precedent, three issues look like L work:
  02 (a new package, two data tables, and a table-driven test over the whole embedded
  catalog), 04 (the four-layer chain, the family gate, eleven criteria, `Resolver`
  reworked), 05 (three flags, deletion of the hard-coded reviewer, and the full exit-code
  contract re-driven through `Run`). Issue 09 runs the other way — it describes itself as
  *"a data change plus one correctness gap, not a restructuring"* (`09:30`) and is likelier
  S.
- Impact: `size` is what a supervisor budgets and sequences against, and a uniform value
  carries no signal at all. `docs/epics/log.md:6` records *"all sized S or M — nothing
  needed the L ceiling"* as though that were measured rather than assumed.
- Recommendation: re-grade 02, 04 and 05 to L (or split 05 into the flag surface and the
  exit-code contract), 09 to S, and replace the boilerplate PR-size note with a per-issue
  one naming what the bulk actually is.

### P3 — Issues 06, 07 and 08 all rewrite the same subcommand switch and `usageText` with no ordering between 07 and the others

- Location: `issues/06-tiers-command.md:95-96`, `issues/07-models-command.md:89-90`,
  `issues/08-request-object-and-version.md:111-113` — all three name
  `internal/cli/run.go` (the subcommand switch) and `internal/cli/usage.go` (`usageText`).
  `depends_on` is `[4, 5]`, `[2, 4]` and `[5]` respectively (line 14 of each), so 07 is
  ordered against neither 06 nor 08.
- Source: `issues/08-…md:126-129`, which applies exactly this reasoning between 05 and 08 —
  *"both rewrite `runReviewCommand`'s flag and prompt handling, and landing them in
  parallel produces a conflict neither PR's tests would catch"*
- Problem: the sequencing rule the cut invented for one pair is not applied to the other
  pairs that share the same two files for the same reason.
- Impact: mechanical merge conflicts in `usage.go` and `run.go` when 06, 07 and 08 are
  implemented concurrently — trivially resolvable, but they will land on whichever agent is
  slowest, which is not who introduced them.
- Recommendation: chain 07 behind 06 (`depends_on: [2, 4, 6]`), or state in all three
  bodies that a `usageText` conflict is expected and how to resolve it.

### P3 — `docs/epics/index.md` still reports Epic 3's issues as not cut

- Location: `docs/epics/index.md:12` — *"open, #3 — spans two repositories; `create-issues`
  deliberately deferred"*
- Source: `_shared/bundle-interfaces.md` § The two bundles — index bullets are mechanical
  and track the target's frontmatter. The issues exist and are recorded at
  `docs/epics/log.md:5`.
- Impact: the epics bundle root is the one surface a later reader scans for where the
  project stands, and it says the epic has not been broken down.
- Recommendation: drop the deferral clause; keep the cross-repository half, which is still
  true.

### P3 — Issue 01's GitHub body is the only one that carries no scope or acceptance criteria

- Location: `julienlegoux/external-reviewer#60` — its body is the original text filed at
  Epic 2's close, with a footer appended pointing at
  `issues/01-git-read-path-scoped-diff.md`
- Source: `_shared/pipeline-interfaces.md` § Issue status lifecycle, and the observed
  practice across this epic — verified that #61–#69 and `julienlegoux/skills#37` each carry
  a full `## Acceptance criteria` section, and #60 does not
- Problem: the issue was adopted rather than created (`docs/epics/log.md:8-10` says so), and
  adopting it skipped the body rewrite the other ten got.
- Impact: anyone reading the epic's sub-issue list on GitHub gets criteria for ten issues
  and a prose description for one. The footer link mitigates it, which is why this is P3
  rather than higher.
- Recommendation: replace #60's body with the same rendering the other ten use, keeping the
  original discovery text as the Summary it already is.

## Coverage notes

- **Every acceptance criterion in EPIC_3 traces to at least one issue.** The three v1
  success criteria (`EPIC_3.md:76-83`) land on issue 11, with criterion 1's mechanism in
  issue 10. The six mechanics beneath them (`:87-95`) map to: excluded family → no
  reviewer, exit 1 → issues 04 and 05; unknown family excluded → 02 and 04; dynamic
  provider resolves, refresh first → 04 (verified by mutation, `04:97-100`); absent
  credential exit 1 / broken exit 2 → 04 and 05; `tiers` plus the unrecognised-key `warn`
  line → 03 and 06; the catalog-wide classifier table → 02. Nothing is missing.
- **Every `## Scope` bullet is covered**, one to one: tiers → 05, the TOML → 03, the
  resolution order → 03 + 04, family classification and exclusion → 02 + 04 + 05,
  discovery → 06 + 07, loop caps → 09, the swap → 10.
- **The dependency graph is acyclic and strictly forward.** `01:[]`, `02:[]`, `03:[]`,
  `04:[2,3]`, `05:[4]`, `06:[4,5]`, `07:[2,4]`, `08:[5]`, `09:[1]`, `10:[5,6,8]`,
  `11:[10]` — no issue depends on a higher number. The single defect is the one recorded
  above on issue 11.
- **GitHub state matches the files.** Ten issues (#60–#69) exist, all `OPEN`, all on
  milestone 3, all labelled `enhancement`; `julienlegoux/skills#37` exists and is open.
  All eleven are native sub-issues of tracking issue #3, **including the cross-repository
  one** — GitHub accepted it, so the epic's progress bar does track the skills work. Every
  file's `status: open`, `gh_issue` and `resource` agree with what GitHub reports; no
  `gh_pr` is set anywhere, which is correct before implementation.
- **The bundle mechanics are right.** `issues/index.md` carries no frontmatter (correct for
  a non-root index), its bullets use bundle-relative absolute links and the same mechanical
  form Epic 2 used, cross-bundle links into `docs/planning/` are plain relative paths, and
  `docs/epics/log.md:5-46` carries a full creation entry.
- **The issues' claims about existing code check out.** Spot-verified: `usageText`
  documents only `--allow` and `--prompt` (`internal/cli/usage.go:12-24`, as issue 05
  says); `cli.Run` passes `reviewer.Bounds{}` — the zero value — at
  `internal/cli/run.go:54`, as issue 09 says; `reviewer.DefaultProviderID`/`DefaultModelID`
  are used at `internal/cli/review.go:105-106`, as issue 05 says; and `Resolver.Resolve`
  already refreshes before `GetModel` (`internal/reviewer/resolve.go:82-88`), so issue 04's
  family gate really does slot between lines 88 and 93. Every non-new path in every
  "Relevant files" section exists in the tree (18 checked, 18 present); the two invented
  paths (`internal/family/`, `internal/config/`) each say in place that they follow the
  layout rather than being verified.
- **The issues transcribe their governing decisions faithfully.** Issue 02's three
  classification rules, the regional-qualifier strip and the segment-exact matching match
  `specs/04:55-74` word for word in substance; issue 03's precedence and the exact warning
  wording match `specs/05` and `specs/06:39-49`; `stop=no_reviewer` and `stop=bounds` are
  both in SPECS' closed set (`SPECS.md:275-282`). `CONVENTIONS.md`'s testing rules —
  black-box `package x_test`, table-driven, the `faux` provider, and the literal
  `go test -race -ldflags=-s ./... -count=1` command — appear in every code issue's
  criteria.
- **Verification-by-mutation is asked for where it matters**, not decoratively: issues 01,
  02 and 04 each name a specific deletion that must make a specific test fail. That is what
  makes the fail-closed family rule an assertable property rather than a claim.

## Open questions

- **The README the validation boundary is supposed to live in does not exist and no issue
  creates it.** `specs/05-tier-assignment-schema.md:118-122` says v1's OpenAI-only
  validation is *"stated in the README so it reads as an honest boundary rather than an
  implied promise"*; `EPIC_3.md:130-132` repeats the claim as a Note without making it
  scope or a criterion, and the repository has no `README.md` at all. Whether that belongs
  in this epic is a question about the epic, not about the issues — flagged here so it is
  not lost.
- **Issue 09 may add a cap flag that issue 10's invocation template will not know about.**
  Issue 09 owns the decision *"whether any of the three ceilings is exposed on the command
  line"* (`09:57-62`), and issue 10 — which writes the copy-pasteable command line into
  `review-interfaces.md` — depends on 05, 06 and 08 but not 09. Harmless if the caller is
  expected to take the defaults; worth stating in issue 10 either way.
- Not a finding against the issues, but noticed while checking them: `SPECS.md:275`
  introduces the non-model stop reasons as *"one of five fixed CLI words"* and then lists
  six (`usage`, `help`, `no_reviewer`, `interrupted`, `failed`, `bounds`). Issues 05 and 09
  both rest on that closed set.
