# Log

## 2026-08-09

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
