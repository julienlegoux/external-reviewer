# Log

## 2026-08-11

* **PR opened**: [04 enforce gofmt in CI, normalise line endings, and lint on both
  OSes](/epic-0-skeleton-hardening/issues/04-ci-formatting-gate.md) (#26) —
  [PR #38](https://github.com/julienlegoux/external-reviewer/pull/38) against
  `develop`. A top-level `formatters:` block (`gofmt`) in `.golangci.yml`,
  `.gitattributes` (`* text=auto eol=lf`) with `git add --renormalize .` a
  verified no-op against the LF-only index, and the `lint` job joined onto the
  `[ubuntu-latest, windows-latest]` matrix. 7 files changed, 23 insertions / 7
  deletions (~30 changed lines) against a ~40-line target.

* **Started**: [04 enforce gofmt in CI, normalise line endings, and lint on both
  OSes](/epic-0-skeleton-hardening/issues/04-ci-formatting-gate.md) (#26) —
  branch `issue-26-ci-formatting-gate`.

* **Merged**: [02 widen the write-API guard to every write operation CONVENTIONS
  names](/epic-0-skeleton-hardening/issues/02-write-api-guard-coverage.md) (#24) —
  [PR #36](https://github.com/julienlegoux/external-reviewer/pull/36) merged into
  `develop`.

* **PR opened**: [02 widen the write-API guard to every write operation CONVENTIONS
  names](/epic-0-skeleton-hardening/issues/02-write-api-guard-coverage.md) (#24) —
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
  names](/epic-0-skeleton-hardening/issues/02-write-api-guard-coverage.md) (#24) —
  branch `issue-24-write-api-guard-coverage`.

* **Merged**: [03 carry Epic 1's structural warnings into Epic 2's
  issues](/epic-0-skeleton-hardening/issues/03-epic-2-plan-amendments.md) (#25) —
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
  issues](/epic-0-skeleton-hardening/issues/03-epic-2-plan-amendments.md) (#25) —
  [PR #35](https://github.com/julienlegoux/external-reviewer/pull/35) against
  `develop`. 56 changed lines against an S-sized ~500 target, all under
  `docs/epics/`: two new Scope bullets on Epic 2 issue 03, one on issue 01, both
  GitHub bodies re-synced.

* **Started**: [03 carry Epic 1's structural warnings into Epic 2's
  issues](/epic-0-skeleton-hardening/issues/03-epic-2-plan-amendments.md) (#25) —
  branch `issue-25-epic-2-plan-amendments`.

* **Creation**: Cut [Epic 0: Skeleton hardening](/epic-0-skeleton-hardening/EPIC_0.md) into
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
  [01 — Settle the done line's stop-reason vocabulary and document it in SPECS](/epic-0-skeleton-hardening/issues/01-stop-reason-vocabulary.md)
  (issue [#23](https://github.com/julienlegoux/external-reviewer/issues/23)) as
  [PR #37](https://github.com/julienlegoux/external-reviewer/pull/37) against `develop`.

## 2026-08-10

* **Creation**: Established
  [Epic 0: Skeleton hardening](/epic-0-skeleton-hardening/EPIC_0.md) by converting
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
