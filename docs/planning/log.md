# Log

## 2026-08-11

* **Update**: appended a fourth [DRIFT](/DRIFT.md) entry at the close of
  [Epic 0](https://github.com/julienlegoux/external-reviewer/issues/22). The epic's implementers wrote
  no drift records, so the entry was swept out of the run's own log and verification
  files: [CONVENTIONS § Testing](/CONVENTIONS.md) decides `go test ./... -race`, and
  that command does not run on the development machine at all — a Windows Application
  Control policy blocks `go test`'s temp binaries, and `-race` needs a cgo C compiler
  that neither the Windows toolchain nor the `docker-desktop` WSL distro has. The suite
  is cross-built and run under WSL instead, leaving `-race` and `windows-latest` to CI.
  Triaged **`fix-now`** rather than `accepted`, so CONVENTIONS is deliberately left
  saying the right thing while
  [#48](https://github.com/julienlegoux/external-reviewer/issues/48) makes the machine
  match it; the entry is discharged when that issue closes, not by rewording the
  standard.

  Not promoted: three PRs overran their `M` size estimates (#40, #44, #45 — up to 871
  lines against ~250, all under the `L` ceiling). Estimation calibration for
  test-heavy hardening issues, not the code contradicting a decided standard, and
  already recorded per-PR in [the epics log](../epics/log.md).

## 2026-08-10

* **Update**: appended a third [DRIFT](/DRIFT.md) entry and corrected the criterion it
  contradicts. Epic 1's no-files test compares the tree's file set and every file mtime
  but deliberately not **directory** mtimes — on Windows two consecutive walks of an
  untouched tree already report those moved, so asserting on them tests the platform
  rather than the binary. The narrowing was documented in the test and nowhere else, which
  is how [implementation review report 2](../REPORT_2.md) found it; issue 05's criterion
  now states what is actually asserted and why the exemption costs the guarantee nothing.
  Triaged `accepted` on the conversion of report 2 into
  [Epic 0](https://github.com/julienlegoux/external-reviewer/issues/22).

## 2026-08-09

* **Established**: [DRIFT](/DRIFT.md), promoted at the close of Epic 1 from the two drift
  records its implementers wrote. Both entries were triaged `accepted`, which is the
  disposition that obliges the standards to move, so they did:

  * **The `kern-link` pin was ahead of its own repository.** [SPECS](/SPECS.md) named
    v0.2.0, a tag that has never been cut — `go get` fails outright and the proxy knows
    only v0.1.0 and v0.1.1. The table now says **v0.1.1**, the version the code actually
    builds against and which carries every symbol SPECS names. A pin nobody can install
    is not a standard, and bumping it once v0.2.0 exists is an ordinary version-bump PR.

  * **"The binary writes no files" was true of the binary and false of the run.** Epic 1's
    hand run showed `kern-link` rewriting `~/.pi/agent/auth.json` on OAuth rotation, and
    creating an `auth.json.lock` sidecar whenever it reads an existing store — on the
    binary's behalf, outside the repository. The only way to keep the literal wording
    would be to read credentials here instead, which contradicts this document and is
    worse for security. [SCOPE](/SCOPE.md),
    [scope/09](/scope/09-report-output-contract.md) and SPECS now state the guarantee
    that is real and enforced: **nothing written inside the repository under review, and
    no output file** — credential storage belongs to `kern-link`.

* **Update**: amended [SPECS](/SPECS.md) § Interfaces — the `done` line now carries the
  accumulated `in=`/`out=` token counts. The document already called tokens one of the
  line's load-bearing fields while its example omitted them, which left Epic 1's
  diagnostics issue unable to say whether `done` must print them and Epic 2's measurement
  issue reading them off the `turn` lines instead. Decided on the review of
  [issue review report 1](../REPORT_1.md).

* **Update**: wrote [CONVENTIONS](/CONVENTIONS.md) — the personal baseline filtered to Go,
  with an 8-item [deviations ledger](/conventions/index.md), all accepted at triage.
  Planning is now complete: `split-epics` can cut SCOPE.

* **The read-only guarantee is now something CI fails on.** SPECS states the product's
  central property as a fact about the source — the write half of the filesystem API is
  not in the codebase — which is a claim about every future commit, made in a repo whose
  commits will largely be written by agents that never read SPECS. A `forbidigo` block in
  the already-pinned linter turns it into a check, with `exec.Command` on the list for the
  same reason ripgrep was rejected: a subprocess reads on its own authority, outside the
  root handle ([01](/conventions/01-write-api-guard.md)).

* **Two path vocabularies, because the dev machine and CI disagree.** SCOPE's "nothing may
  assume POSIX" constraint had no day-to-day form. It now does: *OS paths* use `filepath`
  and never cross the model boundary; *wire paths* are `/`-separated and repo-relative
  everywhere. Naming them is the point — a reviewer can see the bug in a diff instead of
  waiting for a Linux runner ([02](/conventions/02-cross-platform-path-rules.md)).

* **`main`/`develop` earns itself here.** Usually a matter of taste; on this project
  distribution is `go install …@latest` on an untagged module, which resolves to the
  **default branch's** newest commit — so `main` is literally what gets installed, and an
  epic mid-flight on it is a broken install
  ([03](/conventions/03-branch-model.md)).

* **Three baseline rules were too weak for what SPECS decided.** The baseline treats a new
  dependency as a one-line PR note, but SPECS said "two, and no others" — so a third module
  reopens a specs decision instead ([07](/conventions/07-dependency-freeze.md)). Go's
  document-every-export convention was dropped inside `internal/`, where there is no godoc
  reader and the rule mostly produces restated names
  ([08](/conventions/08-doc-comment-policy.md)). And "never swallow an error" was given its
  one inversion: a failing *tool* is a value returned into the loop, not an error
  propagated up the stack — following the baseline literally would end reviews that should
  have continued ([06](/conventions/06-go-error-idiom.md)).

* **Kept out of the ledger deliberately**: PR and issue sizing (owned by `create-issues`
  and `split-epics` — a second statement of it here would drift from the skills that
  enforce it), and anything SPECS already decided. Four verdicts were flagged as
  **promotion candidates** for the baseline itself, their rationale being personal rather
  than project-specific: cross-platform path rules, the `main`/`develop` split, black-box
  Go test packages, and the generic half of the Go error idiom.

* **Update**: wrote [SPECS](/SPECS.md) from the completed ledger — 17 decided, 3 N/A. It
  carries a **Departures from SCOPE** section listing every place the two now disagree and
  why, because a developer who reads SCOPE and builds from it is the person that drift
  actually costs.

* **Access is an allow-list, and the boundary had a hole.** The recommendation was a root
  plus a deny-list of sensitive filenames; the user rejected the shape — everything should
  be forbidden except the directories explicitly granted, passed as parameters. `--allow`
  is now required on every `review`. The objection that this pre-selects the reviewer's
  evidence was raised and dismissed on the user's distinction: granting access to
  directories is not handing over chosen files, and inside them the reviewer still chooses
  everything. Designing that also surfaced a real defect in the previous design —
  `git show HEAD:.env` read anything in repository history through the one tool that never
  touches the confined root, making file-level confinement decorative. `git_read` is now
  pathspec-scoped with `show`'s path validated.

* **Ripgrep reconsidered, rejected on a better argument.** Not speed — a subprocess reading
  the filesystem on its own authority sits outside `os.Root`, downgrading kernel-enforced
  confinement to after-the-fact path validation. What `rg` would have brought for free, and
  the first recommendation had quietly dropped, was gitignore awareness; `git ls-files`
  supplies exactly that from the implementation that defines it, through a dependency
  `git_read` already needs.

* **A subscription changes what "measure" means.** The credentialed provider is
  `openai-codex` over OAuth, so per-run cost is flat. Result caps survive but their
  justification changes from spend to the context window; of the three quantities the
  bounds seam watches, turns and wall clock are the meaningful ones. Rippled into SCOPE's
  success criterion 3 and risks 4 and 5, and into three specs decisions.

* **Update**: ran `define-specs` and enumerated a 20-item
  [decision ledger](/specs/index.md) — 17 open, 3 N/A (auth, background work, data
  migrations, none of which this program has). Two intake findings shaped it: `kern-link`
  carries **no family field** on `ai.Model`, so the product's central safety property has
  to be built here ([04](/specs/04-model-family-classification.md)); and its `docs/usage.md`
  now documents the full multi-turn tool loop, so risk 1 is weaker than SCOPE states.

* **Where SCOPE has been amended, and why.** Kept deliberately visible: SCOPE drifting
  from SPECS costs a user nothing and costs a developer an afternoon of building the
  wrong thing. Two inline amendments, both factual corrections rather than reversals —
  the credentialed family is **OpenAI over subscription OAuth**, not Google
  ([risk 2](/scope/18-risks-and-assumptions.md)), and the "kern-link's example covers a
  single round trip" constraint was already stale when SCOPE was written. Each carries a
  pointer to the specs decision that corrected it. A third divergence is recorded only in
  the specs ledger and deliberately not in SCOPE: see the implementer note in
  [08](/specs/08-system-prompt-and-task-assembly.md).

* **The caller owns the system prompt.** The recommendation was a `const` prompt inside
  the binary, on the argument that tool instructions belong beside the tools. Rejected:
  an adjustment to how the reviewer is instructed must be a change to a *skill*, not a
  new build of the binary. `--system-file` was then proposed and also rejected — a
  prompt generated in the calling turn has no file, and a temp file per invocation is
  machinery standing in for an argument. Settled on a JSON request object on stdin
  (`system` + `task`, `DisallowUnknownFields`, no default prompt ever), with
  `--system`/`--prompt` as by-hand shorthand. The binary keeps ownership of the tool
  descriptions and schemas, which cannot drift from their implementations.

* **Discovery is a query, not a document.** `kern-link` can be asked at runtime what
  exists and what is reachable — `GetModels`, `Refresh` for dynamic providers, `GetAuth`
  without sending a request — so a `models` subcommand lists catalog ∩ credentialed ∩
  family-allowed with price and context window. This also forced a resolution rule:
  dynamic providers (`openrouter`, `vercel-ai-gateway`, `nvidia`, `github-copilot`) hold
  no models until `Refresh` runs, so refreshing precedes resolution — otherwise a
  correctly configured tier resolves to "model not found" and falls back silently.

* **Generic mechanism, one validated provider.** The binary names no provider anywhere
  except the family classifier's data table, so all ~35 remain reachable; but v1 is
  *validated* against `openai-codex` alone, stated as an honest boundary rather than an
  implied promise. A rescue-only `family` override in the config was proposed for unusual
  providers and dropped — it was the one mechanism able to weaken the fail-closed family
  rule, and it lived on a user's disk where being wrong is silent.

* **Update**: ran `define-scope` and wrote [SCOPE](/SCOPE.md) from an 18-item
  [decision ledger](/scope/index.md) — 16 decided, 2 marked N/A because
  [CONCEPT](/CONCEPT.md) already settled them (the problem, and the CLI delivery form).
  Premise confirmed before enumerating: a Go CLI on `kern-link`, author-first,
  replacing the pipeline's `opencode` dependency.

* **v1 runs unbounded, and measures instead of capping.** The recommendation was three
  hard caps (turns, cost, wall clock) plus a coverage statement, on the argument that
  the binary spends the user's own credit unsupervised. The user rejected it on the
  ground that nobody knows yet how many turns or how long a review takes, so any cap
  chosen now is a guess — and a guess set too low silently truncates legitimate
  reviews. The counter-argument turned out not to apply yet: through Milestones 1–2 the
  author launches by hand and can interrupt; the unsupervised case only arrives at
  Milestone 3, by which point the numbers exist. So Milestone 2 ships instrumentation
  (turns, cost, elapsed time on stderr) and Milestone 3 inherits caps sized from it.
  Rippled into six decisions; the honest cost is recorded as two risks — context
  overflow is now unmitigated in v1, and a runaway hand-run is accepted deliberately.

* **The skills-side edit is inside v1.** Milestone 3 removes the `opencode` branch from
  `_shared/review-interfaces.md` rather than adding a second delegation path beside it:
  keeping both would maintain two contracts forever for compatibility with nobody, since
  v1 has exactly one user and that user will have the new binary. The native fallback
  stays unchanged. Consequence flagged for `split-epics`: Milestone 3's issues land in
  the **lx skills repository**, not this one.

* **Weight tiers, not roles.** The concept fixed the split (skills ship the vocabulary,
  the machine holds the assignment) but not the vocabulary itself. Chose `light` /
  `standard` / `heavy`, reusing the grading `implement-epic` already applies — a role
  taxonomy like "security-reviewer" would push a vendor-shaped judgment back into the
  shipped artifact, which is exactly what the concept moved onto the user's machine.

* **The binary enforces the family exclusion, not the caller.** The concept settled the
  principle; the open question was the mechanism. A claim that depends on every caller
  remembering to check is not a claim, and enforcing centrally codes the
  `amazon-bedrock` trap once, next to the catalogue where family metadata lives.

* **Creation**: established this bundle and wrote [CONCEPT](/CONCEPT.md) from a
  working session that started out designing an LLM gateway and ended somewhere else.

* **The product is a called binary, not a gateway.** The starting memo priced three
  topologies and recommended the cheapest; the session set out to build the most
  expensive one anyway, "as simply as possible". Simplifying it far enough inverted
  the answer. A gateway's real work is relaying a tool loop between Claude Code and a
  foreign model, translating tool calls in both directions mid-stream — and a reviewer
  does not need that relayed, because the loop can run inside the binary and speak no
  Anthropic protocol at all. The project was renamed accordingly (see below). The
  gateway survives as a named non-goal with one trigger: foreign models used as real
  subagents in a session, implementers rather than reviewers.

* **The reviewer reads the repository itself.** The cheaper design has the calling
  session assemble the evidence and paste it into one prompt. Rejected: the party under
  review would decide what the reviewer is allowed to see, copying its own blind spots
  into the selection, and would truncate silently once the evidence outgrew the
  context. A read-only loop inside the binary costs little because `kern-link` already
  carries the tool-call protocol.

* **"External" is about model family, not provider.** Verified against `kern-link`'s
  own catalogue: the `amazon-bedrock` provider serves both Amazon and Anthropic models,
  so selecting on "provider is not Anthropic" would send Claude's work to Claude. The
  distinction is dormant while only one foreign family is configured, but the concept
  holds it explicitly so a later configuration cannot lose it.

* **Model judgment is written by the user, on the user's machine.** Considered and
  rejected: shipping a maintained list of models and their strengths inside the skills.
  It rots on a two-month cycle, recommends models the reader may have no account for,
  and keeps its confident tone after it stops being true. What ships is the procedure
  and the vocabulary; what stays local is the assignment. The vocabulary is not new —
  `implement-epic` already grades work into tiers and picks horsepower from that
  grading.

* **Catalogue staleness lands on `kern-link`.** It is versioned, tracks an upstream and
  has a sync procedure, which makes it a better home for facts that expire than a file
  in each user's repository. Its embedded catalogue was behind the models the user
  intends to use at the time of writing; the fix is upstream, not a local override.

* **Named `external-reviewer`.** The working name was `claude-gateway`, kept until a
  check found three problems: `claude gateway` is a first-party Claude Code subcommand
  (the enterprise auth/telemetry gateway), the repository name is already taken twice
  on GitHub, and every comparable project in that crowded category routes Claude Code
  to other providers — precisely what this one decided not to do. The name would have
  advertised the rejected design.

* **Parked, deliberately**: how a stale catalogue signals that it needs revisiting;
  whether `review-implementation` joins and whether an epic's merged diff is a
  comfortable surface; how a report states its own coverage; the installation path.
  Recorded in the concept's open questions rather than answered.
