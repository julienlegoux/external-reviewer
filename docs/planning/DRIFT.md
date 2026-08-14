---
type: Drift
title: "External Reviewer — Drift"
description: "Standards this codebase has drifted from, and why"
tags: [planning, drift]
timestamp: 2026-08-14T09:30:00Z
---

# Drift

Standards [SPECS](/SPECS.md), [SCOPE](/SCOPE.md) or [CONVENTIONS](/CONVENTIONS.md)
already decided, that the implementation could not follow — with the verified reason and
what was decided about it. Read this alongside those three documents: a decided standard
plus its live drift is what the code actually looks like.

Append-only, newest epic first. An entry that stops being true becomes
`resolved (<date>)` rather than disappearing.

## Epic 3: Reviewer selection and pipeline integration

### 02 — `azure-openai-responses` is classified by rule 1, not by the reseller rule specs 04 lists it under

- **Decided**: [specs 04 § rule 2](/specs/04-model-family-classification.md) names
  `azure-openai-responses` among the resellers, whose family is *"parsed from the id's vendor
  segment: everything before the first `.` or `/`"*. Issue 02's Scope repeats both lists verbatim.
- **Actual**: `internal/family/vendors.go` puts it in `vendorProviders`, mapped to `openai` —
  rule 1, the id never read. `vendorSegment` additionally strips a leading `~` from OpenRouter's
  nine floating aliases (`~anthropic/claude-opus-latest`, …), which specs 04 does not mention.
- **Because**: all 42 of the provider's embedded ids are bare OpenAI names (`gpt-4o`, `o3`,
  `gpt-5-codex`, …), so rule 2 yields `gpt-4` from `gpt-4.1` and the whole id from `o3` — neither
  is a vendor, rule 3 fires, and all 42 models become Unknown and therefore excluded. Rule 1's own
  list already contains a provider that is not a vendor by name (`openai-codex`), so its real
  membership test is *serves exactly one vendor's models*, which this provider meets. The
  departure can never admit a model of the wrong family: Azure OpenAI cannot serve Anthropic's.
- **Disposition**: accepted — specs 04 should move the provider to rule 1's list and state rule 1's
  membership test explicitly, so the next provider is placed by the test rather than by the
  example. The amendment can travel with the installation define-change below rather than as a
  separate PR, since both touch the same file.
- **Revisit when**: specs 04 is next touched, or kern-link gives this provider vendor-qualified
  ids, at which point both rules agree.
- **Evidence**:
  [drift record 02](../epics/epic-3-reviewer-selection-integration/drift/02-azure-openai-responses-is-a-vendor-provider.md),
  PR #82. `TestClassify_AVendorProviderAnswersWithoutReadingTheId`,
  `TestClassify_EveryEmbeddedModelClassifiesToTheRecordedFamilies`.

### 02, 04, 05 — bare model ids classify Unknown, so 160 of 1042 catalog models are refused and `github-copilot` cannot serve a tier at all

- **Decided**: [EPIC_3](../epics/epic-3-reviewer-selection-integration/EPIC_3.md),
  [SPECS § Architecture](/SPECS.md) and [specs 05](/specs/05-tier-assignment-schema.md) name four
  dynamic providers — `openrouter`, `vercel-ai-gateway`, `nvidia`, `github-copilot` — and promise
  that refreshing them before resolution is what makes *"a correctly configured tier resolve"*.
  Issue 04 carries that into an acceptance criterion.
- **Actual**: the refresh works for all four, and the family gate then refuses every model whose id
  carries no vendor segment. Measured against kern-link v0.1.1: `nvidia` 0/20 Unknown,
  `vercel-ai-gateway` 0/187, `openrouter` 4/258, **`github-copilot` 25/25**. Across the whole
  catalog, 160 of 1042 models, over eight providers (`github-copilot` 25,
  `cloudflare-ai-gateway` 38, `opencode` 49, `opencode-go` 13, `fireworks` 16,
  `cloudflare-workers-ai` 13, `cerebras` 3, `groq` 2).
- **Because**: rule 2 reads a vendor segment that these ids do not have, so rule 3's fail-closed
  default fires. `openrouter`'s four are routing meta-models (`auto`, `openrouter/free`, …) whose
  vendor is chosen per request — refusing those is the rule working, not a gap.
- **Disposition**: deferred — pending a `define-change` on installation and provider selection.
  **The product direction has since changed and is recorded here so that run does not rediscover
  it**: an installation skill will connect the user to providers and let them assign models to
  tiers, making model choice the user's responsibility rather than the binary's. Under that
  direction the fix is *not* the third table of bare-name prefixes two drift records proposed —
  it is inverting what `Unknown` means, from refused to allowed, which is one line in
  `Exclusion.Allows` rather than a classifier.
  That inversion trades a safety net for user responsibility, and the concrete cost is on record:
  **ten of `github-copilot`'s 25 ids are `claude-*`**, so a user assigning one to a tier gets
  Claude reviewing Claude — the one failure this product cannot detect after the fact. A middle
  position exists and should be considered by that define-change: keep the default `anthropic`
  exclusion, which catches every id that names itself, and make only `Unknown` permissive.
- **Revisit when**: the installation `define-change` decides the family rule's fate. Until then the
  gate stays fail-closed and the refusal stays legible — `ReasonFamilyUnknown` is a distinct reason
  from `ReasonFamilyExcluded`, and `tiers` reports it as `excluded by family: unknown`.
- **Evidence**:
  [drift record 04](../epics/epic-3-reviewer-selection-integration/drift/04-github-copilot-tiers-resolve-to-no-reviewer.md)
  (PR #85), which measured the per-provider table and lists all 25 ids;
  [drift record 02](../epics/epic-3-reviewer-selection-integration/drift/02-azure-openai-responses-is-a-vendor-provider.md)
  (PR #82), which first proposed and deferred the table;
  [drift record 05](../epics/epic-3-reviewer-selection-integration/drift/05-exit-1-is-silent-except-for-one-warn-line.md)
  (PR #86), which cites this as the live example that made exit 1 undebuggable.
  `TestResolveTier_UnknownFamilyTier_SaysWhichRuleRefusedIt`.

### 05 — the exit-1 path is silent for the caller's report, not for the run: it writes one `warn` line

- **Decided**: [SPECS § Interfaces](/SPECS.md) and
  [specs 13](/specs/13-error-handling-and-failure-classification.md) both describe it as
  *"`1`, a silent native fallback nobody needs told about"*. Issue 05 carries it into two criteria:
  stdout empty and stderr carrying no `error:` line, and no substitute model ever named.
- **Actual**: both criteria hold and are asserted on the full stderr text, but `selectReviewer`
  writes exactly one `warn` line before returning, switching on the typed `Reason` — naming the
  tier, the model, the rule that refused it and the layer that assigned it.
- **Because**: Epic 1 could call exit 1 silent honestly, with one hard-coded reviewer. After this
  epic there are three assignment layers and five refusal reasons, and the exit code carries none
  of the four facts needed to fix the state. The live case is not hypothetical: a `github-copilot`
  tier gives a correct configuration, a successful refresh, a found model — and exit 1. SPECS
  itself distinguishes the two writers: `error:` is what a calling skill copies into its report and
  must stay empty here ([scope 09](/scope/09-report-output-contract.md)); `warn` is
  transcript-only, and SPECS already uses it for exactly this shape of fact.
- **Disposition**: accepted — amend SPECS' and specs 13's wording to *a silent native fallback for
  the caller's report* rather than a silent run. The properties anyone relies on are asserted
  directly, so the amendment loosens no guarantee.
- **Revisit when**: the invocation contract stops keeping stderr. Settled in the affirmative for
  now — issue 10's template forbids `2>/dev/null` and points diagnosing users at this line, so it
  is read rather than dead output.
- **Evidence**:
  [drift record 05](../epics/epic-3-reviewer-selection-integration/drift/05-exit-1-is-silent-except-for-one-warn-line.md),
  PR #86. `TestRun_Review_NoConfigAndNoEnvironment_ExitsOneSilently`,
  `TestRun_Review_AnthropicTier_ResolvesToNothingAndNamesNoSubstitute`.

### 02, 08 — the native `-race` command cannot link: Application Control refuses `ld.exe`

- **Decided**: [CONVENTIONS § Testing](/CONVENTIONS.md) states the command without qualification —
  *"The whole command runs natively on the Windows development machine, `-race` included"* — and
  closes the point: *"cgo works: a WinLibs MinGW-W64 UCRT toolchain compiles and its unsigned
  output executes, so `-race` needs nothing remote."* The
  [Epic 0 entry below](#09-10-11--the-decided-test-command-does-not-run-on-the-development-machine--resolved-2026-08-11)
  records the same as resolved on 2026-08-11.
- **Actual**: no package builds with `-race` on 2026-08-13/14. `-race` ran on the remote through
  `scripts/test-remote.sh` for every issue in this epic; the fast local loop ran
  `go test -ldflags=-s ./... -count=1`, green, whole suite, without the detector.
- **Because**: the same per-binary hash rule the Epic 0 entry documents, applied to the C
  toolchain — and the symptom moved twice while the epic ran, which is why it was misdiagnosed
  three times. First `gcc.exe` itself drew `0xC0000602` (`STATUS_FAIL_FAST_EXCEPTION`), inherited
  by `cgo.exe` as a silent `exit status 2`. Then `gcc --version` began exiting 0 and printing
  `16.1.0` while a one-line C file still failed, at `collect2.exe: fatal error: CreateProcess: No
  such file or directory`. Measured at the close of the run, the refused image is **`ld.exe`**:
  `ld --version` returns `Permission denied`, exit `126`, while `gcc --version` succeeds. So the
  driver runs and the linker it spawns is refused. Re-rolling *Go* build flags cannot help — the
  block is on a binary the Go build only invokes.
- **Disposition**: accepted — CONVENTIONS § Testing must stop stating cgo as a settled fact and
  state a check instead, because the sentence has now been false twice for two different reasons
  and each time the symptom first read as a code failure. The check is **`ld --version`**:
  `Permission denied` / exit 126 means no linking, so no cgo and no `-race`, whatever
  `gcc --version` says. A green `go build -race` proves nothing while `runtime/cgo` can come from
  the build cache.
- **Revisit when**: `ld --version` succeeds — which happens when the toolchain is reinstalled or
  updated to an image the policy has not judged, not on any schedule. Verify with the one-line C
  file before trusting `-race` locally again.
- **Evidence**:
  [drift record 02-gcc](../epics/epic-3-reviewer-selection-integration/drift/02-gcc-is-blocked-so-race-ran-remotely.md)
  (PR #82, sharpened by issue 08 in PR #88), Epic 2's
  [drift record 03](../epics/epic-2-read-only-agentic-loop/drift/03-native-go-test-blocked-again.md),
  and the `ld --version` measurement above. This supersedes the Epic 0 entry's
  *"cgo is not blocked either"*, which was true of one gcc image and is not true of the machine's
  current one.

### — the three `gpt-5.6` models are defined in this repository, alongside kern-link's embedded catalog

- **Decided**: [SPECS § Stack](/SPECS.md) fixes one dependency for everything to do with models,
  and [specs 03](/specs/03-kern-link-dependency-policy.md) makes its **embedded catalog** the
  source of model definitions: what a model costs, how large its context window is and which wire
  protocol it speaks are facts this binary reads, never facts it states.
  [CONVENTIONS § Dependencies](/CONVENTIONS.md) pins the dependency and treats a version bump as an
  ordinary PR — the intended way new models arrive.
- **Actual**: `internal/reviewer/catalog.go` holds three `*ai.Model` literals — `gpt-5.6-sol`,
  `gpt-5.6-terra`, `gpt-5.6-luna`, all on `openai-codex` — and `RegisterLocalModels` adds them to
  the registry `DefaultModels` builds. `go.mod` is untouched at v0.1.1. The registration uses
  `ai.MutableModels.SetProvider`, the only mutation v0.1.1 offers: the built-in provider is
  replaced in place by a decorator embedding it, so id, name, base URL, headers, auth
  (`oauth.CodexOAuth`), refresh and streaming all delegate, and only `GetModels` appends. A local
  definition whose id the catalog already carries is dropped — the catalog always wins.
  Separately, issue 05's grep criterion `TestBinary_NamesNoProviderOutsideTheFamilyTables` now
  allows the literal `openai-codex` in two files instead of one.
- **Because**: the models exist and the release carrying them does not. v0.1.1's `openai-codex`
  catalog stops at `gpt-5.5`, and v0.1.1 is still the newest tag — the same fact
  [Epic 1's entry](#04--kern-link-is-pinned-at-v011-not-the-specified-v020) records for v0.2.0.
  There is nothing to bump to, and a tier naming `openai-codex/gpt-5.6-sol` would otherwise resolve
  to `ReasonNotInCatalog`, which is exit 1: current models on the user's account, and a binary that
  cannot reach them. A committed `replace` onto an edited checkout was rejected by
  [CONVENTIONS § Dependencies](/CONVENTIONS.md); rebuilding the provider was rejected as restating
  `oauth.CodexOAuth` and the codex stream functions here.
- **Disposition**: deferred — there is no available action until a release exists.
- **Revisit when**: **a kern-link release ships these ids — then delete the local registration
  rather than keeping both.** Two definitions of one model that disagree on price or context window
  are invisible: `models` would print one number, an invoice another, and nothing would fail. The
  collision rule makes the intervening state *inert* rather than wrong, which is not the same as an
  acceptable resting state. Also revisit if a fourth model is added there: three literals are a
  stopgap, a growing list is a private catalog, and that is a specs 03 decision.
- **Evidence**:
  [drift record 06](../epics/epic-3-reviewer-selection-integration/drift/06-gpt-5-6-models-registered-locally.md),
  PR #91. `internal/reviewer/catalog.go`, `catalog_test.go`;
  `TestLocalModels_ClassifyIntoTheOpenAIFamilyAndAreAllowed`.

## Epic 2: Read-only agentic loop

### 01 — wire-path spellings are validated before `os.Root` sees them, not only by it

- **Decided**: [SPECS § Reading the repository](/SPECS.md) delegated the whole
  requirement to the standard library — *"every read goes through that `*os.Root`"*, and
  it names what not to build, *"the list a hand-rolled `filepath.Clean` + prefix check
  would have to reproduce"*. [specs 10](/specs/10-repository-confinement.md) rejects
  `filepath.Abs` + `EvalSymlinks` + prefix comparison in the same terms, and issue 01's
  Scope made it an instruction for that PR: escapes are *"delegated to `*os.Root`, not
  re-implemented"*.
- **Actual**: `internal/confine.Scope.Resolve` runs `cleanWirePath` first, which refuses
  five spellings on its own authority — the empty path, a backslash, a leading `/`, a
  drive-letter second byte (`C:\Windows\win.ini`, `C:x`), and after `path.Clean` a `..`
  or `../` prefix. That last one is the `Clean`-plus-prefix shape SPECS names. Everything
  the kernel alone can see — symlinks out of the tree, absolute symlinks, Windows device
  names, a directory swapped for a link mid-resolution — is still `os.Root`'s answer, and
  is reported as the same `ErrOutsideRoot`.
- **Because**: three requirements in issue 01 cannot be met by delegation alone.
  `os.Root` answers *may this be opened*, never *which granted subtree is this in*, so
  `Scope.allows` has nothing to compare against without a canonical relative path —
  `docs/../../outside` passes a raw `HasPrefix(p, "docs/")`. Issue 01's criterion names
  `/etc/passwd` **and** `C:\Windows\win.ini` refused on both matrix OSes; `os.Root`
  refuses both on Windows (measured: `path escapes from parent`), but on Linux
  `C:\Windows\win.ini` is a legal single-segment filename that `os.Root` refuses only as
  a missing file. And `Resolve` must answer without I/O, because issue 06 validates
  `git show <rev>:<path>` for paths that exist only in history.
- **Disposition**: accepted — [SPECS § Reading the repository](/SPECS.md) and
  [specs 10](/specs/10-repository-confinement.md) were amended on 2026-08-12 to separate
  the two layers: the wire-path *vocabulary* is checked above the root, every
  kernel-visible escape is still the root's.
- **Revisit when**: `os.Root` exports a way to canonicalise a name against a root without
  opening it (which removes the reason for the `Clean`-and-prefix test), or a sentinel
  that distinguishes its escape refusal from other errors (which removes `openError`'s
  default branch, today reporting every non-absence, non-permission refusal as
  `ErrOutsideRoot` because Go exports nothing finer).
- **Evidence**:
  [drift record 01](../epics/epic-2-read-only-agentic-loop/drift/01-wire-path-validation-before-os-root.md),
  PR #50. `internal/confine/allow.go`, `internal/confine/root.go`;
  `TestScope_Resolve_RefusesPathsThatAreNotInsideTheRoot` covers the syntactic half,
  `TestScope_Open_RefusesSymlinksThatLeaveTheRoot` and
  `TestScope_Open_WindowsReservedDeviceNames` the delegated half.

### 02 — enumeration walks `Scope.ReadDir`, not `Root.FS()`

- **Decided**: [SPECS § Reading the repository](/SPECS.md) named the mechanism by its
  type — *"`Root.FS()` backs `list` and `search`, so one confinement mechanism serves all
  three file tools"* — and
  [specs 11](/specs/11-tool-implementation-strategy.md) spelled the walk out as
  `fs.WalkDir(root.FS(), ".")`, with the fallback as *"`fs.WalkDir` over the root with
  `.git/` skipped"*.
- **Actual**: `*os.Root` and any `fs.FS` over it never leave `internal/confine`. The walk
  in `internal/repo` holds a `Scope` and calls `Scope.ReadDir`/`Scope.Stat`, starts at
  `Scope.Allowed()` rather than at `"."`, and puts every child name back through
  `Resolve`. The confinement mechanism is unchanged — still the one `*os.Root` — but the
  handle is the root plus the allow-list and the floor, not the root alone.
- **Because**: `fs.WalkDir(root.FS(), ".")` cannot start where the grant is not `.` —
  `--allow docs` makes `"."` an `ErrOutsideAllowList`, and walking from `"."` anyway to
  filter afterwards would read the names of every ungranted directory, which is the same
  leak `git_read`'s pathspec confinement exists to prevent. An exported `fs.FS` also
  reads on the root's authority alone: any holder could `fs.WalkDir` into `secrets/` and
  `ReadFile` a `.env`, turning the boundary back into a convention every caller must
  remember. Going through `Resolve` additionally deletes the `.git/` special case — the
  floor already refuses it, so a `credentials/` directory is pruned by the same line.
- **Disposition**: accepted — [SPECS](/SPECS.md) and
  [specs 11](/specs/11-tool-implementation-strategy.md) were amended on 2026-08-12. The
  claim they were protecting (one confinement mechanism for all three file tools) is
  unchanged; only the type in the sentence was wrong.
- **Revisit when**: `os.Root` grows a `ReadDir`, or an `fs.FS` implementation that accepts
  a filter — either shrinks `Scope.ReadDir` to a call rather than an open-list-close. Also
  when a tool wants a genuine `fs.FS` for something the standard library only offers over
  one (`fs.Glob`, `fs.WalkDir` proper): the answer is then an `fs.FS` implemented *by*
  `Scope`, gated on every `Open`, never the root's own.
- **Evidence**:
  [drift record 02](../epics/epic-2-read-only-agentic-loop/drift/02-enumeration-does-not-use-root-fs.md),
  PR #51. `internal/confine/root.go`, `internal/repo/enumerate.go`;
  `TestScope_ReadDir_RefusesDirectoriesTheThreeRulesRefuse`,
  `TestFiles_TheFallbackWalkStaysInsideTheAllowedSubtrees`,
  `TestFiles_OutsideAGitRepositoryWalksTheFilesystemInstead`.

### 06 — `git_read` scopes `status` too, and validates `<rev>:<path>` in every subcommand

- **Decided**: [specs 10](/specs/10-repository-confinement.md) closed the
  `git show HEAD:.env` hole with three measures, the third of which was a statement that
  one subcommand needs none: `show`'s `<rev>:<path>` validated, `log` and `diff`
  pathspec-scoped, and *"`status` is unaffected"*.
- **Actual**: `internal/tools/git_read.go` applies both measures wider. `status` gets the
  same appended pathspecs as `log` and `diff`, and the `<rev>:<path>` form is validated in
  every subcommand's arguments, in all three of git's spellings — `<rev>:<path>`,
  `:<path>`, `:<stage>:<path>`.
- **Because**: two behaviours measured against real git, not reasoned from the docs. An
  unscoped `git status --porcelain` lists ` M secret/creds.txt` beside ` M docs/notes.md`
  — it is the one subcommand that hands over every path name in the repository, and their
  modification state, without opening a file. And `git diff HEAD:.env HEAD:docs/notes.md`
  prints both blobs' contents, so the object form reads files in `diff` exactly as in
  `show`; pathspecs cannot substitute, since `git show HEAD:.env -- docs` still prints
  `.env` and `git diff <blob> <blob> -- docs` is a usage error (exit 129).
- **Disposition**: accepted — [specs 10](/specs/10-repository-confinement.md)'s two
  bullets were amended on 2026-08-12. Nothing in the decision's reasoning changes, only
  the enumeration of where it applies, which was written before those two commands were
  measured.
- **Revisit when**: git changes how `--` and object arguments interact, or a subcommand is
  added to `git_read`'s allowed set — the object-form validation is per-subcommand and a
  new one inherits nothing.
- **Evidence**:
  [drift record 04](../epics/epic-2-read-only-agentic-loop/drift/04-git-read-scopes-status-and-every-object-argument.md),
  PR #55. `internal/tools/git_read.go` (`gitReadArgv`, `objectPath`, `objectRefusal`);
  `TestGitRead_ScopesStatusToTheGrantedSubtrees` and
  `TestGitRead_RefusesABlobPairOnTheFloor`, both verified by mutation — removing the
  appended pathspecs makes the scoping tests fail with the ungranted subtree's paths in
  the output.

Epic 2 also re-hit the Epic 0 entry below, and is what corrected it: issue 05 ran entirely
on the remote host believing the native half had failed as a class, and the experiment that
found the real rule — Smart App Control keys its verdict to the test binary's exact
SHA-256 — was run the same day. No separate entry; the correction is folded into
[Epic 0's entry](#09-10-11--the-decided-test-command-does-not-run-on-the-development-machine--resolved-2026-08-11),
whose evidence cites
[drift record 03](../epics/epic-2-read-only-agentic-loop/drift/03-native-go-test-blocked-again.md).

## Epic 0: Skeleton hardening

### 09, 10, 11 — the decided test command does not run on the development machine — resolved (2026-08-11)

- **Decided**: [CONVENTIONS § Testing](/CONVENTIONS.md) names the command without
  qualification — *"`go test ./... -race` is the command"* — resting on
  [specs 16](/specs/16-testing-infrastructure.md)'s standard-library-only testing
  stack. Strict red-green in the same section assumes the author can watch a test go
  red and green locally.
- **Actual**: the suite is cross-built `GOOS=linux GOARCH=amd64` on the Windows host
  and run under `wsl -d docker-desktop`, **without** `-race`, and CI is treated as the
  authority for both `-race` and `windows-latest`. Two acceptance criteria that need a
  real signal — issue 09's `SIGINT`, issue 10's `SIGPIPE` — were verified by hand the
  same way, against a Linux binary driven through a fifo, rather than by the suite.
- **Because**: two independent blockers, both verified during Epic 0. `go test` cannot
  execute its own temp binaries natively — a local Windows Application Control policy
  blocks them — and `-race` cannot be built at all, because it requires cgo and neither
  the Windows toolchain nor the ~136 MB `docker-desktop` distro has a C compiler. The
  cost is visible: issue 11 records four findings as *"argued rather than observed"*,
  among them all behaviour under `-race` and on `windows-latest`.
- **Disposition**: fix-now ([#48](https://github.com/julienlegoux/external-reviewer/issues/48))
  — a C toolchain on the host with the policy exempted, or a full WSL distro with
  `build-essential`. Epic 2 lands the multi-turn loop, which is the concurrency the
  race detector exists for; finding a data race in CI on an OS the author cannot
  reproduce on is the expensive version of this.
- **Resolved (2026-08-11)** by [#48](https://github.com/julienlegoux/external-reviewer/issues/48),
  through neither branch of the disposition above: the author's own Linux VPS already had
  gcc and a Go toolchain, so `scripts/test-remote.sh` syncs the working tree there over
  ssh and runs the command, and nothing was installed on the Windows host at all. The
  suite now completes with `-race` from the development machine, and the detector was
  observed firing — a deliberately racy probe test reported `WARNING: DATA RACE` and
  exit `1` — rather than assumed live. [CONVENTIONS § Testing](/CONVENTIONS.md) now says
  where the command runs.
- **Then the Windows half fell too, the same day, and without administrator rights** —
  but *not* for the reason recorded at the time. The 2026-08-11 conclusion was that
  pointing `GOTMPDIR` at `C:\dev\gotmp` instead of `%LOCALAPPDATA%\Temp` was enough.
  It was not: the suite passing that day was coincidence, and the setting has since been
  shown to make no difference at all. See the correction below.
- **Corrected 2026-08-12, by experiment rather than by inference.** Smart App Control's
  verdict is keyed to the test binary's **exact SHA-256, and nothing else**. Measured on
  `internal/confine`: the `go test -c` output (`04773FBE…`) was blocked; a byte-identical
  copy under a different name *and* directory was blocked; a deterministic rebuild of
  unchanged source was blocked; the same source built with `-ldflags=-s` (`E4DB25FC…`)
  ran. Not the package, not the path, not the filename, not `-race`, and not the size —
  `internal/diag`'s 10.8 MB race binary ran while `confine`'s 7.5 MB one was refused.
  - This is why the symptom read differently to different observers: **sticky** when Go's
    build cache returns the same binary run after run (`confine` failed 5/5 that way),
    **random, a different subset each run** when the source changes between runs so every
    hash is new. One rule, two workflows.
  - `GOTMPDIR` is irrelevant — the same package draws the same verdict under
    `C:\dev\gotmp` and under the default.
  - **cgo is not blocked either.** WinLibs MinGW-W64 UCRT `gcc` 16.1.0 compiles, and its
    unsigned output executes. The earlier `libiconv-2.dll` refusal was a different,
    older toolchain, and the conclusion drawn from it — that `-race` can never run
    natively here — no longer holds.
  - **Consequence**: `go test -race -ldflags=-s ./... -count=1` passes natively on
    Windows, every package. The fast red-green loop includes `-race`, and
    `scripts/test-remote.sh` drops to a fallback for Linux-specific reproduction.
- **Reopened 2026-08-13, superseded by
  [Epic 3's entry above](#02-08--the-native--race-command-cannot-link-application-control-refuses-ldexe).**
  The `-race` half of this resolution stopped being true: the linker `ld.exe` now draws the block,
  so nothing cgo-dependent builds natively. The `-ldflags=-s` finding below still holds for the
  non-race suite. Read Epic 3's entry for the current check (`ld --version`) before trusting the
  paragraphs beneath this line.
- **Revisit when**: a package starts failing again after a source change — that is a new
  hash drawing a block, so re-roll with build flags rather than debugging the code. Or
  when the machine's policy changes for unrelated reasons.
- **Evidence**: PR #45 (issue 11's verification log — the constraint stated in full, and
  the four findings it left argued rather than observed), PR #46 (issue 09's hand-run
  SIGINT verification, the WSL cross-build in practice), PR #42 (issue 10's SIGPIPE
  criterion, verified the same way). Epic 0's folder was retired at its close, so these
  cite the PRs the evidence merged in rather than the files. For the 2026-08-12
  correction: the hash experiment and the full green `-race` suite are recorded on
  [#48](https://github.com/julienlegoux/external-reviewer/issues/48#issuecomment-5265250369),
  and Epic 2's
  [drift record 03](../epics/epic-2-read-only-agentic-loop/drift/03-native-go-test-blocked-again.md)
  carries the observations that prompted it.

## Epic 1: Walking skeleton

### 04 — `kern-link` is pinned at v0.1.1, not the specified v0.2.0

- **Decided**: [SPECS § Stack](/SPECS.md)'s dependency table pins
  `github.com/julienlegoux/kern-link` at **v0.2.0**, and
  [CONVENTIONS § Dependencies](/CONVENTIONS.md) requires everything pinnable to be
  pinned. Issue 04's Scope repeats the version verbatim.
- **Actual**: `go.mod` requires `github.com/julienlegoux/kern-link v0.1.1`. No `replace`
  directive is committed and `go.work` stays gitignored, so the rest of the dependency
  policy holds. The embedded model catalog is v0.1.1's, which is why the hard-coded
  reviewer is `openai-codex/gpt-5.5` rather than SPECS' illustrative `gpt-5.1-codex`.
- **Because**: the tag does not exist. `go get …@v0.2.0` fails with
  `invalid version: unknown revision v0.2.0`; `go list -m -versions` returns
  `v0.1.0 v0.1.1`; `gh api repos/julienlegoux/kern-link/tags` agrees. A forward pin to a
  release that has not been cut, not a yanked version. Every symbol the resolution
  sequence needs is present in v0.1.1 under the exact names SPECS uses — there is **no
  API-name drift**, only a version one.
- **Disposition**: accepted — [SPECS § Stack](/SPECS.md) amended to v0.1.1 on
  2026-08-09, so the table stops naming a version that cannot be installed.
- **Revisit when**: `kern-link` v0.2.0 (or later) is tagged. From here that is an
  ordinary version-bump PR, which CONVENTIONS already classes as ordinary — not the
  discharge of drift.
- **Evidence**:
  [drift record 04](../epics/epic-1-walking-skeleton/drift/04-kern-link-version-pin.md),
  PR #19.

### 05 — "the binary writes no files" excludes `kern-link`'s credential store

- **Decided**: [SCOPE § Report output contract](/SCOPE.md) and
  [SPECS § Interfaces](/SPECS.md) stated it without qualification — **the binary writes
  no files** — so the calling skill keeps sole ownership of the report.
  [CONVENTIONS § Code style & formatting](/CONVENTIONS.md) enforces it structurally: the
  write half of the filesystem API is absent from the codebase and `.golangci.yml`'s
  `forbidigo` block rejects it.
- **Actual**: both halves of the enforcement hold — no write-API call appears in
  non-test source, and a full successful review leaves the reviewed repository
  byte-identical (`TestRun_SuccessfulReview_TouchesNoFile`). The dependency nonetheless
  writes on the binary's behalf, outside the repository, in `~/.pi/agent/`:
  `auth.json` is rewritten when a stored OAuth credential is expired
  (`models.GetAuth` → `resolveStoredOAuth` → `credentials.Modify` → `os.WriteFile`), and
  `auth.json.lock` is created whenever an existing store is read.
- **Because**: observed on the manual verification run recorded in PR #20, on a machine
  whose `openai-codex` credential is OAuth — the repository tree was identical before and
  after, while `~/.pi/agent/auth.json` moved from 2621 to 2305 bytes with its mtime
  updated. The path is unreachable from the test suite, which injects an offline registry
  over an in-memory credential store. No wording keeps the literal claim true while the
  binary still authenticates: the alternative is for this binary to read the credential
  file itself, which contradicts [SPECS § Data, configuration & auth](/SPECS.md) and is
  strictly worse for security.
- **Disposition**: accepted — [SCOPE](/SCOPE.md),
  [scope/09](/scope/09-report-output-contract.md) and [SPECS](/SPECS.md) amended on
  2026-08-09 to state what is actually guaranteed: the binary writes nothing inside the
  repository under review and produces no output file; credential storage belongs to
  `kern-link` and lives in `~/.pi/agent/`.
- **Revisit when**: Epic 2's `os.Root` confinement lands — it makes the in-repository
  half structural for reads as well as writes, at which point the amended sentence should
  be re-checked against what the code enforces.
- **Evidence**:
  [drift record 05](../epics/epic-1-walking-skeleton/drift/05-credential-store-writes.md),
  PR #20.

### 05 — the no-files check compares file mtimes, not directory mtimes

- **Decided**: issue 05's criterion
  [*"the tree's file set **and modification times** are unchanged"*](../epics/epic-1-walking-skeleton/issues/05-single-turn-model-call.md),
  the assertable half of [SCOPE](/SCOPE.md)'s and [SPECS § Interfaces](/SPECS.md)'s
  guarantee that the binary writes nothing inside the repository under review.
- **Actual**: `TestRun_SuccessfulReview_TouchesNoFile`
  (`internal/cli/roundtrip_test.go:276-299`) walks the tree twice and compares the file
  set and every **file** mtime. Directory mtimes are collected and deliberately not
  compared, which the test documents in place.
- **Because**: on Windows two consecutive walks of an untouched tree already report moved
  directory mtimes, so asserting on them makes the test flaky against the platform rather
  than against the binary — re-verified during PR #20. The narrowing costs nothing the
  criterion was protecting: a file created, modified or deleted inside the repository
  changes the file set or a file mtime, and a directory whose only change is its own mtime
  contains no such file.
- **Disposition**: accepted — issue 05's criterion amended on 2026-08-10 to state what is
  actually asserted, so the narrowing stops reading as an unmet criterion.
- **Revisit when**: Epic 2's `os.Root` confinement lands. It makes the in-repository
  guarantee structural for reads as well as writes, at which point what this test still
  needs to cover should be re-derived rather than inherited.
- **Evidence**: [implementation review report 2](../REPORT_2.md) § P3, which is where the
  narrowing was first noticed as undocumented; `internal/cli/roundtrip_test.go:276-299`;
  PR #20.
