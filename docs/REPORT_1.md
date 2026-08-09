# Issue Review Report 1

## Scope

- Reviewed: `docs/epics/epic-1-walking-skeleton/issues/` (5 issue files + `index.md`) and
  `docs/epics/epic-2-read-only-agentic-loop/issues/` (7 issue files + `index.md`) — the
  twelve issues `create-issues` cut for Epics 1 and 2.
- Reviewed against: `docs/epics/epic-1-walking-skeleton/EPIC_1.md` and
  `docs/epics/epic-2-read-only-agentic-loop/EPIC_2.md`, with `docs/planning/SPECS.md` and
  `docs/planning/CONVENTIONS.md` as the governing standards, and `docs/epics/index.md` /
  `docs/epics/log.md` for cross-epic context. `docs/planning/SCOPE.md` was not re-read:
  the epic is the contract here, and a scope item an epic dropped is `review-epics`' finding.
  `EPIC_3.md` was read only where Epics 1 and 2 defer work to it.
- GitHub verification: **verified**. `gh` authenticated as `julienlegoux`. Issues #4–#8
  exist on milestone 1 and #9–#15 on milestone 2; every title matches its local
  `title:` frontmatter; all twelve are `OPEN`, matching `status: open`; every `gh_issue`
  and `resource` URL resolves to the right issue; and all twelve are registered as native
  sub-issues of #1 and #2 respectively, so both epic progress bars are honest.
- External review: `openrouter/~openai/gpt-latest`, two batched passes (coverage and
  invented scope; ordering, dependency metadata and sizing). `google/gemini-3.1-pro-preview`
  was tried first and is not authenticated on this machine. Its leads were verified against
  the files before entering this report; several were rejected as correct-by-design.

## Findings

### P1 — `git_read status` is exempted from confinement, contradicting the epic's own scope

- Location: `docs/epics/epic-2-read-only-agentic-loop/issues/06-git-read-tool.md:45`
  — "`status` — unaffected."
- Source: `docs/epics/epic-2-read-only-agentic-loop/EPIC_2.md:32` — "Four native,
  in-process read-only tools, **all scoped by the allow-list**"; `docs/planning/SPECS.md:124`
  repeats it verbatim; `docs/planning/SPECS.md:333` — "What leaves the machine is the
  contents of the allowed subtrees."
- Problem: the issue confines three of the four subcommands (`log`/`diff` by pathspec,
  `show` by path validation) and explicitly exempts the fourth. `git status` reports the
  working-tree state of the *whole* repository, so with `--allow docs` the model receives
  the names and modification state of every path outside `docs/` — including paths the
  sensitive-file floor exists to hide (`.env`, `*.pem`, `credentials*`). SPECS is silent on
  `status`, which is likely how the exemption got in; the epic is not silent.
- Impact: a hole in the product's central guarantee, shipped deliberately and untested.
  The epic's acceptance criteria cover `show`, `log` and `diff`
  (`EPIC_2.md:80-81`) but never `status`, so nothing catches it. It is also the same class
  of hole the issue's own Summary was written to close — "a hole that would otherwise make
  every file-level control decorative" (`06-git-read-tool.md:21-23`).
- Recommendation: scope `status` with the allowed subtrees as pathspecs
  (`git status -- <paths>`), exactly as `log` and `diff` are, and add an acceptance
  criterion asserting that a change in a disallowed subtree does not appear. If leaving
  `status` unscoped is genuinely intended, it needs a SPECS decision recording why, not a
  one-line exemption inside an issue.

### P1 — the Windows reserved-device acceptance criterion cannot pass on `ubuntu-latest`

- Location: `docs/epics/epic-2-read-only-agentic-loop/issues/01-allow-list-and-root-confinement.md:79-81`
  — "On Windows, a Windows reserved device name (`NUL`, `COM1`) is refused. Asserted by a
  test that runs on both matrix OSes and **asserts refusal on both**."
- Source: `docs/planning/SPECS.md:117-118` — "and **on Windows** reserved device names are
  rejected"; `docs/planning/CONVENTIONS.md:77-79` — no `//go:build` divergence without a
  test that runs on both OSes.
- Problem: the bullet contradicts itself in two sentences. Reserved device names are a
  Windows concept; on Linux `NUL` and `COM1` are ordinary filenames and `os.Root` will
  open them. A test that asserts refusal on both matrix OSes fails on `ubuntu-latest` by
  construction. The convention it cites forbids *build-tag divergence*, not
  platform-dependent expectations.
- Impact: the acceptance criterion is unsatisfiable, and the cheapest way to make CI green
  is the wrong one — adding a hand-rolled reserved-name blocklist so Linux refuses too.
  That makes a legitimately named Linux file permanently unreadable and reintroduces
  exactly the hand-rolled path validation `os.Root` was chosen to avoid
  (`SPECS.md:115-120`).
- Recommendation: keep the single test running on both OSes, but assert the
  platform-appropriate outcome — refused on Windows, resolved on Linux — with the
  expectation selected at runtime via `runtime.GOOS`, not a build tag. Reword the criterion
  so the "on both" applies to the test running, not to the refusal.

### P2 — `--allow` is pointed at the wrong file, and the Epic 1 tests it invalidates are unnamed

- Location: `docs/epics/epic-2-read-only-agentic-loop/issues/01-allow-list-and-root-confinement.md:103`
  — "`internal/cli/run.go` — the `--allow` flag and the startup `os.OpenRoot`".
- Source: `docs/epics/epic-1-walking-skeleton/issues/02-review-invocation-parsing.md:82-84`
  — `internal/cli/run.go` owns *subcommand dispatch*; `internal/cli/review.go` owns "the
  `review` FlagSet and its parsed request struct"; its tests live in
  `internal/cli/review_test.go`.
- Problem: two errors in one line. The flag belongs on the `review` FlagSet in
  `review.go`, not in `run.go` — the issue's own Scope says so
  (`01-allow-list-and-root-confinement.md:33`, "on the `review` `FlagSet` from issue 02 of
  Epic 1") while its Relevant files says otherwise. And `internal/cli/review_test.go` is
  not listed at all, even though this PR *inverts* an assertion already in it: Epic 1
  issue 02 asserts `review --prompt "review this" <tempdir>` parses successfully
  (`02-review-invocation-parsing.md:60-61`), and this PR makes that same invocation exit
  `2` (`01-allow-list-and-root-confinement.md:67-69`).
- Impact: the implementer's first CI run fails on a pre-existing test that the issue never
  told them they were allowed to change. Under the strict red-green rule
  (`CONVENTIONS.md:99-103`) an unexplained pre-existing failure is ambiguous — the wrong
  resolution is to make `--allow` optional and satisfy both.
- Recommendation: correct the path to `internal/cli/review.go`, add
  `internal/cli/review_test.go` to Relevant files, and add one Scope line stating that
  Epic 1 issue 02's success-case tests are amended to pass `--allow` because the grammar
  deliberately tightens here.

### P2 — `list` and `git_read` must announce truncation against a cap no issue defines

- Location: `docs/epics/epic-2-read-only-agentic-loop/issues/02-enumeration-and-list-tool.md:46-48`
  and `:84-85`; `docs/epics/epic-2-read-only-agentic-loop/issues/06-git-read-tool.md:46-47`
  and `:90`.
- Source: `EPIC_2.md:50` — "Truncation **always announced**"; `docs/planning/SPECS.md:139-140`
  — "Caps are generous — 5000 lines per file, 200 matches", which sizes `read_file` and
  `search` and nothing else.
- Problem: `read_file` states its numbers (`04-read-file-tool.md:30-31`: default 2000,
  capped at 5000) and `search` states its own (`05-search-tool.md:31-32`: default 100,
  capped at 200). `list` shows `200` only inside an illustrative truncation string, and its
  acceptance criterion says "a pattern matching more than **the cap**" without ever naming
  it. `git_read` never names a number at all — "truncation announced when it hits the cap".
- Impact: two of the four tools have an acceptance criterion that cannot be written as a
  test without the implementer inventing the constant, and two independent PRs will invent
  different ones. Truncation is the epic's answer to the concept's central objection about
  silent truncation, so the caps are contract, not tuning.
- Recommendation: state the cap in each Scope the way `read_file` and `search` do — a path
  count for `list`, a byte or line count for `git_read` output — and reference it from the
  acceptance criterion. If the numbers should be decided rather than assumed, they belong
  in SPECS § Reading the repository beside the existing two.

### P2 — half the issues declare a `size` their own content contradicts, under an identical boilerplate size note

- Location: every issue's `size:` field and `## PR size note`. Eleven of twelve notes are
  byte-identical: "Target ~500 changed lines; if this grows past ~1000, split it before
  opening the PR" (`epic-1/issues/01:99`, `02:98`, `03:95`, `04:99`, `05:102`;
  `epic-2/issues/01:121`, `02:121`, `03:125`, `04:104`, `05:108`, `06:121`).
- Source: the size bands the pipeline defines — S under 200 changed lines, M 200–500,
  L 500–1000, "L is the ceiling, not the target".
- Problem: two things at once. First, the note contradicts the field it accompanies: an
  issue declared `size: M` (200–500) is told to *target* 500 and to split only past 1000,
  which authorises an L-sized PR on every M issue. Second, the sizes themselves are
  compressed — eleven of twelve are M — and do not survive contact with the content.
  `epic-1/issues/01` is a `go.mod`, an eight-line `main.go`, two YAML files and one test:
  S, not M. `epic-2/issues/01` (resolver + floor + three refusal forms + traversal,
  absolute-path, symlink, device-name, eight-filename and wire-path tests), `02`
  (enumeration + exec helper + fallback + `list` + registry), `03` (loop + dispatch +
  bounds + termination-order + instrumentation), `05` and `06` each read as L. The external
  pass estimated 650–750 lines for four of them independently and reached the same
  conclusion on the same six issues.
- Impact: sizing is the finding a developer feels first. An M label on a 750-line PR sets
  the wrong expectation for the author and the reviewer, and the boilerplate note removes
  the only signal that would have caught it — `implement-issue` reads "target ~500" on a
  PR that should have been split.
- Recommendation: re-declare `size: S` on `epic-1/issues/01`, `size: L` on `epic-2/issues/01`,
  `02`, `03`, `05` and `06`, and replace each boilerplate note with a real one naming that
  issue's own target and its natural split line. `epic-2/issues/02` is the one genuine
  split candidate (enumeration + exec helper, then registry + `list`); the external pass
  and this review agree that `03` and `06` are cohesive and should stay whole at L.
  `epic-2/issues/07`'s note is already tailored and is the model to follow.

### P2 — every cross-reference in the twelve GitHub issue bodies is a dead link

- Location: GitHub issues #4–#15. Each body is a verbatim copy of its `.md` file, so it
  carries the bundle's link forms — e.g. #9 ends with
  "Blocked by: … see [Epic 1](/epic-1-walking-skeleton/EPIC_1.md)"
  (`epic-2/issues/01-allow-list-and-root-confinement.md:115-116`) and cites
  `[SPECS § Reading the repository](../../../planning/SPECS.md)` (`:106-110`).
- Source: the bundle link rules make `/epic-1-…` bundle-relative and `../../../planning/…`
  cross-bundle — both correct *inside* `docs/epics/`. On github.com an issue body resolves
  `/epic-1-walking-skeleton/EPIC_1.md` against the site root, and the `../../../` form has
  no file to be relative to.
- Problem: the links are right for the file and wrong for the issue. Every reference to
  SPECS, CONVENTIONS, the parent epic and the blocking issues 404 on the surface a
  developer actually works from. Separately, blocking relationships are rendered as file
  links rather than as `#4`–`#15`, so GitHub cannot show them and the developer cannot
  click through to the prerequisite.
- Impact: the twelve issues are complete on disk and hollow on GitHub. Sub-issue linkage
  to #1/#2 is correct, so the epic view works; sibling dependencies and every governing
  decision do not.
- Recommendation: when a body is pushed to GitHub, rewrite links to repo-rooted blob paths
  (`docs/planning/SPECS.md`) and render `Blocked by` / `Blocks` as issue numbers alongside
  the file link. The local files stay exactly as they are — the transform belongs at the
  push boundary, not in the bundle.

### P2 — issue 07 updates an `index.md` that does not exist

- Location: `docs/epics/epic-2-read-only-agentic-loop/issues/07-hand-run-measurements.md:81`
  — "The epic folder's `index.md` and `docs/epics/log.md` are updated per the bundle
  rules" — against its own Relevant files at `:89`, which names
  `docs/epics/epic-2-read-only-agentic-loop/issues/index.md`.
- Source: the epic folder `docs/epics/epic-2-read-only-agentic-loop/` contains only
  `EPIC_2.md` and `issues/`; there is no `index.md` at that level, and the bundle rules do
  not reserve one there. The bundle root index is `docs/epics/index.md`.
- Problem: the acceptance criterion and the file list point at two different targets, and
  the criterion's target does not exist. The file list's target is wrong too: `MEASUREMENTS.md`
  is not an issue, so it does not belong in `issues/index.md`.
- Impact: the implementer either creates a stray epic-level `index.md` that nothing else in
  the bundle expects, files a reference document in the issues index, or silently skips the
  step — and `MEASUREMENTS.md`, the document Epic 3 is explicitly told to read
  (`07-hand-run-measurements.md:49-51`), ends up listed in no index at all.
- Recommendation: name `docs/epics/index.md` in both places, with a bullet in the mechanical
  form the bundle rules require.

### P2 — no issue tells the implementer to branch from and target `develop`

- Location: all twelve issue files. `develop`, `issue-<n>-<slug>` and `Closes #<n>` appear
  in none of them (verified by grep across `docs/epics/*/issues/*.md`).
- Source: `docs/planning/CONVENTIONS.md:85-95` — "`develop` is the integration trunk;
  `main` is the install surface … Issue branches cut from `develop` and merge back into it
  … One branch per issue, named `issue-<n>-<slug>` … A PR closes exactly one issue via
  `Closes #<n>`."
- Problem: the issues reflect the other conventions well — test-first, black-box packages,
  `-race`, `golangci-lint`, the two-OS matrix, conventional commit prefixes
  (`epic-1/issues/01:74`) — but not the branch model. There is no `CONTRIBUTING.md`, no
  `.github/PULL_REQUEST_TEMPLATE.md` and no `CLAUDE.md` in this repository to supply it,
  so the issue body is the only channel.
- Impact: `main` is the default branch and is literally what `go install …@latest`
  resolves to. A PR opened against the default branch merges a half-finished epic onto the
  install surface, and the failure is silent until someone installs.
- Recommendation: add one line to each issue — base branch `develop`, branch name
  `issue-<n>-<slug>`, `Closes #<n>` in the PR body — or, better, add
  `.github/PULL_REQUEST_TEMPLATE.md` and a `CONTRIBUTING.md` note in `epic-1/issues/01`
  and have the remaining eleven point at it.

### P3 — cross-epic blockers are invisible in `depends_on`

- Location: `epic-2/issues/01-allow-list-and-root-confinement.md:14` (`depends_on: []`)
  against `:112-116` ("Blocked by: Epic 1 issues 02 … and 03");
  `epic-2/issues/03-multi-turn-loop-and-dispatch.md:14` (`depends_on: [2]`) against
  `:116-117` (also blocked by Epic 1 issue 05).
- Source: the issue schema scopes `depends_on` to "other issue numbers in this epic", so
  the arrays are not malformed — the schema simply cannot express a cross-epic edge.
- Problem: a tool reading only frontmatter sees Epic 2 issue 01 as unblocked and
  immediately startable. The prose is correct; nothing machine-readable is.
- Impact: low in practice — `EPIC_2.md:93-95` declares the dependency on Epic 1 at epic
  level, and a supervisor that reads the epic before its issues will not start early. It
  is worth knowing that the guard is the epic file, not the issue metadata.
- Recommendation: no change to the files. If this recurs across epics, the schema is where
  to fix it, not the issues.

### P3 — the issues indexes use `./`-relative links where the bundle rules want bundle-relative

- Location: `docs/epics/epic-1-walking-skeleton/issues/index.md:5-9` and
  `docs/epics/epic-2-read-only-agentic-loop/issues/index.md:6-12` — e.g.
  `[…](./01-scaffold-module-and-ci.md)`.
- Source: the bundle link rules — within a bundle, a bundle-relative absolute path with a
  leading `/`. `docs/epics/index.md:10-12` follows it, and so does every `Dependencies`
  link inside the issue files themselves (`/epic-1-walking-skeleton/issues/02-….md`).
- Problem: the two issues indexes are the only files in the bundle using the `./` form.
- Impact: nothing breaks — the links resolve for a human and on GitHub — but the bundle is
  now internally inconsistent about its own link form, which is how the convention erodes.
- Recommendation: rewrite the twelve bullets to
  `/epic-<n>-<slug>/issues/<nn>-<slug>.md`. The bullet text (`- M, open, [#4](…)`) is
  already in the mechanical form the rules require and should stay.

### P3 — "issue 01" is ambiguous inside Epic 2

- Location: `epic-2/issues/01-allow-list-and-root-confinement.md:90` — "so the issue-01
  forbidigo guard stays clean", meaning **Epic 1** issue 01, written inside **Epic 2**
  issue 01.
- Source: the same file uses bare "issue 01" for itself elsewhere (`:41`, `:89` of
  `04-read-file-tool.md`, `:39` of `05-search-tool.md`), and Epic 1 cross-references are
  written out in full everywhere else (`epic-2/issues/03:117`, "Epic 1 issue 05").
- Problem: one reference in the set resolves to the wrong issue on a plain reading.
- Impact: a reader chasing the forbidigo guard looks in the file they are already in.
- Recommendation: write "Epic 1 issue 01" at `:90`, and reserve bare "issue NN" for
  same-epic references throughout.

### P3 — the `done` line's token field is left undecided

- Location: `epic-1/issues/03-diagnostics-and-exit-codes.md:30-32` — the example
  `done    turns=7  $0.0918  2m14s  stop=end_turn` carries no token field, immediately
  followed by "Its load-bearing fields are turns, **tokens** and elapsed time" — against
  the acceptance criterion at `:61-63`, which requires only "the turn count, the elapsed
  time and the stop reason".
- Source: `docs/planning/SPECS.md:207` and `:211` contain the same tension; the issue
  copies it faithfully rather than resolving it.
- Problem: the implementer cannot tell whether `done` must carry tokens. The acceptance
  criterion says no, the scope bullet says they are load-bearing.
- Impact: small but real downstream — `epic-2/issues/07` records "in/out tokens" per run
  from the transcript (`:71-73`), and `EPIC_2.md:89-90` makes tokens part of the epic's
  measurement deliverable. If `done` omits them the author reads them off the last `turn`
  line instead, which is fine only if that is the intent.
- Recommendation: decide it in the issue — either add a token field to the `done` shape and
  to the acceptance criterion, or state that tokens live on the `turn` lines and `done`
  aggregates turns, cost, elapsed and stop reason only. Worth amending SPECS § Interfaces
  in the same breath.

### P3 — only `read_file` is required to accept backslash-separated wire paths

- Location: `epic-2/issues/04-read-file-tool.md:75-76` — "including when the caller passed
  a path with backslashes".
- Source: `docs/planning/CONVENTIONS.md:71-76` — wire paths "are `path`-package,
  `/`-separated and repo-relative on every platform", converted at the boundary with
  `FromSlash`/`ToSlash`; `EPIC_2.md:32-34` says the same.
- Problem: the wire contract is that `/` is the separator, full stop. Tolerating `\` means
  hand-normalising input beyond `filepath.FromSlash` (a no-op on Linux), which makes a
  legal Linux filename containing a backslash unreachable. None of the other three tool
  issues asks for it, so the four tools would differ on what they accept.
- Impact: minor, but an inconsistency between tools is exactly what a wire contract exists
  to prevent, and the model learns the loose behaviour from whichever tool it tries first.
- Recommendation: either drop the clause and let a backslash path be refused like any other
  non-conforming path, or lift it to all four tools and record it as the wire contract's
  documented leniency.

### P3 — `--allow` may name a file, while the epic and SPECS say "subtrees" throughout

- Location: `epic-2/issues/01-allow-list-and-root-confinement.md:51-52` — "An `--allow`
  value that does not resolve to an existing directory **or file** inside the root is a
  usage error".
- Source: `EPIC_2.md:38-47` and `docs/planning/SPECS.md:110-113` describe the allow-list
  only as subtrees; the same issue says "the union of allowed **subtrees**" at `:41`.
- Problem: single-file grants are a real widening of the confinement vocabulary, introduced
  in a subordinate clause about validation rather than as a decision. Every downstream
  mechanism reads it — the resolver's subtree test, `git_read`'s pathspecs
  (`06-git-read-tool.md:40-41`), `search`'s `path` parameter.
- Impact: an implementer who reads only the subtree wording writes a prefix test that
  rejects a file grant; one who reads `:51-52` supports it. The `git_read` pathspec path is
  where the two diverge visibly.
- Recommendation: state it once, deliberately, in Scope — files are grantable or they are
  not — and make the acceptance criteria cover whichever it is.

## Coverage notes

- **Epic coverage is complete on both epics.** Every scope item and every acceptance
  criterion in `EPIC_1.md` and `EPIC_2.md` maps to at least one issue; the independent
  external pass built the same mapping line by line and found nothing uncovered. The two
  criteria most often lost in a split survive here intact: Epic 1's "the `done` line
  appears on all three paths" is asserted in `epic-1/issues/03`, `04` and `05`
  separately, and Epic 2's hand-run measurement is a full issue with its own PR rather
  than a checklist item.
- **Ordering holds.** Both epics are workable strictly top to bottom, one PR each, with no
  forward references — confirmed independently by the external pass. The two non-obvious
  ordering calls are both right and both recorded in `docs/epics/log.md`: building the
  output contract before there is anything to output (`epic-1/issues/03` before `04`/`05`),
  and landing the loop with one tool wired (`epic-2/issues/03`) so the three later tool PRs
  are each small and assertable end-to-end. Adding `kern-link` at `epic-1/issues/04` rather
  than `01`, because `go mod tidy` would drop an unimported require, is the kind of detail
  that is only ever learned the hard way.
- **No invented scope of consequence.** Everything the external pass flagged as invented
  was verified and rejected except the items above: the exported `cli.Run` is a documented,
  reasoned deviation from SPECS' lowercase `run` (`epic-1/issues/01:36-38`); the drift
  record required at `epic-1/issues/04:69-71` is the pipeline's own mechanism at its own
  path; `help` on stdout at exit `0` follows from SPECS naming `help` as a subcommand; and
  the concrete tool defaults (`read_file` 2000/5000, `search` 100/200/context 2) are
  exactly the specificity an issue is supposed to add.
- **GitHub state is clean.** Twelve issues, right titles, right milestones, right
  sub-issue parents, `status: open` matching `OPEN` everywhere, `resource` URLs correct.
  `enhancement` on all twelve and `epic` on the three tracking issues is consistent; the
  repository defines no other label scheme to honour.
- **Deliberately out of scope and correctly so**: Epic 3 issues are not cut
  (`docs/epics/index.md:12`, `log.md:88-89`), and this review does not grade them.

## Open questions

- **Nothing claims the JSON request object on stdin or `--system`.** `epic-1/issues/02:49-51`
  defers "`{"system": …, "task": …}`" and `--system` to "the epics that need them", Epic 2
  never takes them, and `EPIC_3.md`'s scope does not list them either — yet the caller
  owning the system prompt is load-bearing in SPECS (`:182-196`) and Epic 3's
  `review-interfaces.md` swap has to send one. This is an epic-level gap rather than an
  issue-level one, so it is `review-epics`' or `create-issues`-for-Epic-3's to close, but
  it should not be closed by accident.
- **`version` is likewise unclaimed.** `epic-1/issues/02:48` sends it to Epic 3 with the
  discovery commands; `EPIC_3.md`'s scope names `tiers` and `models` but not `version`.
- **`epic-2/issues/07:75-76` requires that "at least one run exercises each of the four
  tools".** The model chooses its own tools, so this is steerable by the task prompt but
  not guaranteed. Worth deciding whether a run that never calls `git_read` fails the
  criterion or is simply re-run with a prompt that invites history.
