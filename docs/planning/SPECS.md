---
type: Technical Specification
title: "External Reviewer — Technical Specs"
description: "A single Go binary that runs a bounded, read-only agent loop over an explicitly allowed slice of a repository, on a model outside the Anthropic family, and returns markdown on stdout."
tags: [planning, specs]
timestamp: 2026-08-09T07:50:00Z
status: final
---

# External Reviewer — Technical Specs

One Go binary, invoked by a skill over Bash, that drives a multi-turn tool loop against a
foreign model through `kern-link`, reads an explicitly granted slice of a repository
through four read-only tools, and writes a markdown report to stdout. No server, no
database, no daemon, no state between runs.

Two properties shape nearly every decision below and are worth stating before the detail.
**The binary is a mechanism, not a policy**: it supplies no system prompt, names no
provider outside one classification table, and imposes no task type — those come from the
caller. **Its guarantees are structural, not promised**: it cannot write to the repository
because the write half of the filesystem API is not in the codebase, and it cannot read
outside the granted paths because the kernel refuses.

## Stack

Single Go module `github.com/julienlegoux/external-reviewer` at **Go 1.26**, with
`package main` at the module root and everything else under `internal/`
([decision](/specs/01-language-module-toolchain.md)). The root layout is chosen by the
install story SCOPE commits to — `go install github.com/julienlegoux/external-reviewer@latest`
only yields a binary of that name if the root is `main`. `main.go` is a thin
`os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))`, which is what makes the whole
CLI testable in-process.

Two non-stdlib dependencies, and no others:

| Dependency | Version | Why |
|---|---|---|
| `github.com/julienlegoux/kern-link` | v0.1.1 | Provider coverage, credential resolution, streaming, the tool-call protocol as typed values, cost accounting ([decision](/specs/03-kern-link-dependency-policy.md)) |
| `github.com/BurntSushi/toml` | v1.6.0 | Decoding the tier assignment file; chosen for `MetaData.Undecoded()` ([decision](/specs/05-tier-assignment-schema.md)) |

No CLI framework — the command line is stdlib `flag` with one `FlagSet` per subcommand
([decision](/specs/02-cli-surface-and-argv.md)). No test framework, no logging library, no
config library. `git` is an external runtime dependency (below).

This table names the version that is actually installable: v0.2.0 was specified before it
was tagged, and the newest `kern-link` tag remains v0.1.1, which carries every symbol the
model-resolution sequence needs under the exact names used below
([drift](/DRIFT.md)). Bumping the pin once v0.2.0 is cut is an ordinary version-bump PR.

`go.mod` names a tagged `kern-link`; a `replace` directive is never committed, because a
committed `replace` breaks `go install` for anyone whose disk does not match the author's.
Iterating on both trees at once uses a gitignored `go.work`, which `go install` ignores —
and CI, having no workspace, is what catches a commit that only builds locally.

## Architecture

```text
main  →  run(argv, stdin, stdout, stderr) int
            │
            ├─ resolve   tier/flags/env → provider+model → family check → auth check
            ├─ confine   os.OpenRoot(repo) + --allow subtrees
            ├─ tools     list · read_file · search · git_read   (in-process, read-only)
            └─ loop      kern-link StreamSimple ⇄ tool dispatch, until the model stops
```

**Resolution runs before the loop and can end the run before any request is sent**
([decision](/specs/05-tier-assignment-schema.md)):

1. `--model provider/id`, else `EXTERNAL_REVIEWER_TIER_<TIER>`, else the tier's table in
   the config file ([decision](/specs/06-config-resolution-and-env-overrides.md)).
2. If `provider.CanRefreshModels()` — `openrouter`, `vercel-ai-gateway`, `nvidia`,
   `github-copilot` — `models.Refresh(ctx, provider)` **first**. These providers hold no
   models until refreshed, so skipping this makes a correctly configured tier resolve to
   "model not found" and fall back silently.
3. `models.GetModel(provider, id)` → nil ends the run at exit `1`.
4. Family classification → an excluded or unknown family ends the run at exit `1`.
5. `models.GetAuth(ctx, model)` → `(nil, nil)` ends the run at exit `1`; a typed
   `*ai.ModelsError` ends it at exit `2`.

**Family classification is the one piece of provider-aware code in the binary, and it
exists because `kern-link` has none** ([decision](/specs/04-model-family-classification.md)).
`ai.Model` carries provider, id, pricing and limits — nothing about who made the model.
Since `amazon-bedrock` and `openrouter` both serve Anthropic's models, a provider-based
filter would send Claude's work back to Claude. The classifier is a data table over
`(Provider, ID)`: a vendor provider yields its own family; a reseller or aggregator yields
the id's vendor segment after stripping a regional qualifier (`us.`, `global.`, …); and
anything else is **unknown, which is treated as excluded**. The asymmetry is deliberate —
a false exclusion costs a native review, a false inclusion is undetectable after the fact.
A table-driven test runs the whole embedded catalog through it.

**The loop** is `models.StreamSimple` + `Stream.Result(ctx)` per turn over an accumulating
`[]ai.Message`, exactly the shape `kern-link` documents
([decision](/specs/07-agent-loop-mechanics.md)). Streaming rather than `CompleteSimple`
because the run must be observable while it happens. Termination is checked in order:
a bound exceeded, then context cancellation (`SIGINT`), then
`StopReason != StopReasonToolUse`.

`bounds.check(state)` sits at the top of every turn, watching turns, accumulated cost and
elapsed wall clock, and is present **from the first commit with its values unset**. SCOPE
defers real caps until measurements exist ([decision](/scope/08-loop-bounds-and-termination.md));
having the seam means setting them later changes data and flags, never the loop's
structure. An exceeded bound stops the loop and returns what the run has, rather than
failing.

Two failure levels, and the distinction is load-bearing: **a failing tool is a turn** —
returned as `ToolResultMessage{IsError: true}` so the reviewer can try another path — while
**a failing turn ends the run**.

## Reading the repository

The reviewer chooses what to read; that is why the loop lives in the binary rather than
being replaced by evidence the reviewed party pre-selected ([CONCEPT](/CONCEPT.md)). What
the caller controls is the run's *reach*, not its *path*.

**Access is an allow-list passed per invocation** ([decision](/specs/10-repository-confinement.md)).
`review` requires at least one `--allow <relpath>`; omitting it is a usage error, not an
implicit run over the whole tree. `--allow .` grants the entire repository, but has to be
said.

Enforcement is `os.OpenRoot(repoPath)` at startup, and **every** read goes through that
`*os.Root`. The standard library then carries the whole requirement: names that escape the
root are refused, symlinks may not point outside it and may not be absolute, and on Windows
reserved device names are rejected — the list a hand-rolled `filepath.Clean` + prefix check
would have to reproduce, and where it would fail first on the platform SCOPE names as the
development machine.

Above that root, and never in place of it, sits a check on the wire-path *vocabulary*:
`internal/confine.Scope.Resolve` refuses a spelling that is not a repo-relative,
`/`-separated name before `os.Root` is asked anything. The allow-list test needs a
canonical relative path to compare against; the two matrix OSes must agree about an
absolute path that is a legal single-segment filename on one of them; and
`git show <rev>:<path>` must be validated for paths that exist only in history, where
there is no file for `os.Root` to open. Every kernel-visible escape — symlinks, device
names, a directory swapped for a link mid-resolution — is still the root's own answer,
reported under the same rule ([drift](/DRIFT.md)).

`Scope`, meaning that root plus the allow-list and the sensitive-file floor, is the one
handle the file tools hold: `Scope.ReadDir` and `Scope.Stat` back `list` and `search`, so
one confinement mechanism serves all three file tools. `Root.FS()` is deliberately **not**
exported — an `fs.FS` over the root reads on the root's authority alone, with neither the
allow-list nor the floor in its way ([drift](/DRIFT.md)). Only the read half of the API
appears anywhere in the codebase.

**Four tools** ([decision](/specs/09-read-only-tool-contract.md)), all scoped by the
allow-list, all returning plain text with repo-relative `/`-separated paths in both
directions on every platform:

| Tool | Parameters | Notes |
|---|---|---|
| `list` | `pattern` | Glob orientation |
| `read_file` | `paths[]`, `offset`, `limit` | **Batched** — eight issue files in one turn; a bad path in a batch is reported inline, the rest still return |
| `search` | `pattern`, `path`, `glob`, `max_results`, `context` | Regex with surrounding lines, so a match rarely needs a follow-up read |
| `git_read` | `command` ∈ {`log`,`diff`,`show`,`status`}, `args[]` | `args` is an array, never a string — there is no shell to parse one |

**Truncation is always announced** (`[truncated: showing 200 of 431 matches]`). The
concept's objection to pre-assembled evidence is that it truncates silently and *"a report
that saw sixty percent of the material reads exactly as confident as one that saw all of
it"*; a tool that truncated silently would rebuild that flaw one call at a time. Caps are
generous — 5000 lines per file, 200 matches — because the binding limit is the model's
context window, not spend.

**Implementation** ([decision](/specs/11-tool-implementation-strategy.md)): `git ls-files -z`
(plus `--others --exclude-standard`) enumerates, in-process `regexp` matches, `os.Root`
confines. Ripgrep was reconsidered and rejected — not on speed, but because a subprocess
reading the filesystem on its own authority sits outside the root handle, downgrading
kernel-enforced confinement to path validation after the fact. Using `git ls-files` recovers
the one thing `rg` would have contributed for free, exact gitignore semantics, from the
implementation that defines them. Outside a git repository, enumeration falls back to
`fs.WalkDir` — degraded, not broken.

`git_read` is `exec.CommandContext(ctx, "git", "-C", repo, sub, args...)` with `sub`
allowlisted *before* the command is built, never a shell, with `GIT_CONFIG_GLOBAL` and
`GIT_CONFIG_SYSTEM` neutralised. It is confined too: `log` and `diff` take the allowed
subtrees as pathspecs, and `show`'s `<rev>:<path>` form has its path validated — without
which `git show HEAD:.env` reads anything in history and every file-level control is
decorative.

Within allowed subtrees a small non-configurable floor still refuses `.env*`, `*.pem`,
`*.key`, `id_rsa*`, `*.p12`, `*.pfx`, `.npmrc`, `.netrc`, `credentials*` and `.git/`
([decision](/specs/14-security-and-data-exposure.md)). It is a floor, not the boundary —
the allow-list is the boundary. Refusals name which rule refused.

## Interfaces

**The command line is a contract with a script**, since `review-epics` and `review-issues`
build it from a template in another repository ([decision](/specs/02-cli-surface-and-argv.md)):

```
external-reviewer review --allow <relpath> [--allow <relpath>…]
                         [--tier light|standard|heavy | --model provider/id]
                         [--exclude-family anthropic[,…]]
                         [--system <text> --prompt <text>] <repo-path>
                         # …or a JSON request object on stdin
external-reviewer models  [--provider <id>] [--refresh] [--all]
external-reviewer tiers   [--exclude-family anthropic[,…]]
external-reviewer help | --help | -h | version
```

Long-form flags only; the repository path is positional; unknown flags and subcommands
exit `2`.

**The caller owns the system prompt** ([decision](/specs/08-system-prompt-and-task-assembly.md)).
The binary supplies none and never substitutes a default — an adjustment to how the
reviewer is instructed must be a change to a *skill*, not a new build of the binary. It
arrives as a JSON request object on stdin, decoded with `DisallowUnknownFields`:

```json
{ "system": "…who the reviewer is and how to report…", "task": "…what to review…" }
```

A file path was considered and rejected: a prompt generated in the calling turn has no
file, and writing a temp file per invocation is machinery standing in for an argument.
`--system` / `--prompt` remain as by-hand shorthand. What the binary *does* keep is the
tool declarations — `ai.Tool` names, descriptions and JSON schemas live in Go beside the
implementations, because a description that drifts from its implementation tells a
reviewer it may pass an argument that does not exist.

**Output** ([decision](/specs/12-output-and-diagnostics-format.md)): stdout is the model's
final assistant text, **verbatim, with no envelope** — the calling skill keeps sole
ownership of the report file, so the binary is not in the business of formatting someone
else's document.

**Nothing is written to stdout until the reviewer's final message is complete**, and it is
then written exactly once, never as the text deltas arrive. Every failure the run can reach
before that write therefore leaves stdout empty, including the ones that stream a paragraph
of prose before falling over. The write itself is the one failure that survives past that
point, and bytes already handed to the caller cannot be unwritten — buffering the report to
a file the binary would have to own is precisely what this section forbids. So the guarantee
is stated as it actually holds: **stdout is empty on every failure but a failed write of the
report, and a failed write is announced on stderr as an `error:` line naming the report
incomplete, with the bytes written and the report's full length.** A caller reading the
transcript can always tell a truncated report from a whole one. `external-reviewer review …
| head -20` is the ordinary case — the process handles `SIGPIPE` rather than dying by it, so
it exits `2` through the usual seam with a `done` line, never `141`.

The binary **writes nothing inside the repository
under review and produces no output file**; the one thing written anywhere on its behalf
is `kern-link`'s credential store under `~/.pi/agent/`, which the dependency owns
([drift](/DRIFT.md)). stderr carries prefixed human-readable lines:

```
model   openai-codex/gpt-5.5  auth=OAuth
turn 3  tools=2  in=48210 out=1104  $0.0231  42.8s
tool    list pattern="**/*.go"
warn    config: unrecognised key "models" in [tiers.standard]
done    turns=7  in=182430 out=9106  $0.0918  2m14s  stop=end_turn
```

The `model` line is written once, by pre-flight, as soon as a reviewer is resolved:
`provider/id` verbatim from `kern-link`'s own catalog, and `auth=<source>` — the only
credential-adjacent value that ever reaches stderr (`AuthResult.Source`, e.g. `OAuth` or
`ANTHROPIC_API_KEY`; never the credential itself).

The `tool` line is written once per dispatch and names the call **generically** — the
tool's name, then its arguments as `name=value` in a stable (sorted) order — because the
loop renders it and cannot know which of any one tool's fields are the interesting ones.
String values are converted to wire form before they are quoted, so a `/`-separated path
is what reaches the transcript on either platform.

The `done` line is emitted on **every** termination path, including failure and
interruption. Its load-bearing fields are turns, tokens and elapsed time — the token
counts are accumulated over the run and printed on the line itself, so a transcript reader
never has to sum the `turn` lines; on a path that never reached a model they are `0`, not
absent. The `$` figure is
notional on the current credentials and becomes real only if a tier names an API-key
provider. No colour, no spinner — the caller is a script.

**The `stop=` field is a closed set that carries two vocabularies**, because it answers two
different questions and a usage error can never carry a model's own reason:

- **On the success path**, it is the model's own `StopReason` from `kern-link`'s
  `ai.AssistantMessage`, spelled exactly as `kern-link` spells it — no translation table
  between the two. `unspecified` is the named fallback for the rare successful message
  whose `StopReason` is itself empty, so `stop=` can never render with nothing after it.
- **On every termination the model never reaches**, it is one of five fixed CLI words:
  `usage` (a malformed invocation — bad flags, a missing or non-directory repository path,
  an empty prompt), `help` (`help`/`--help`/`-h`), `no_reviewer` (exit `1`: no reviewer was
  ever reachable), `interrupted` (`SIGINT` or a cancelled context), `failed` (reached and
  then unusable for any other reason — a failed turn, a broken credential, an empty final
  message, a report that could not be written to stdout), and `bounds` (the run reached its
  own turn, cost or elapsed ceiling — exit `0` with the report the run had, since a bounded
  run neither failed nor was interrupted; a `warn` line names which of the three bit).

This is a union that only grows: a later epic adds a value no code can produce yet rather
than repurposing one of the above. `bounds` was added by Epic 2's loop
([issue 03](../epics/epic-2-read-only-agentic-loop/issues/03-multi-turn-loop-and-dispatch.md)),
which ships the ceiling as a seam with **no values set** — so no invocation can produce
the word yet, and Epic 3 sets the numbers that make it reachable.

**Exit codes turn on whether a reviewer was ever reachable**
([decision](/specs/13-error-handling-and-failure-classification.md)). Not reached → `1`,
a silent native fallback nobody needs told about: no config, no tier entry, model absent
from the catalog, family excluded, provider unconfigured. Reached and then failed → `2`,
worth a line in the report: a failed turn, a `StopReason` of `error`/`aborted`, context
overflow, rate limits, an empty final message, `SIGINT`, and usage errors. The sharp edge
is credentials — an *absent* one is `1`, a *broken* one (`*ai.ModelsError` with code
`oauth` or `auth`) is `2`, because something is wrong that a human must fix. Nothing panics
across the CLI boundary.

**Integrations** are exactly two, as SCOPE fixed ([decision](/scope/16-systems-of-record.md)):
`kern-link` as a Go dependency, and the repository under review through read-only tools.
No GitHub API, no telemetry, no network egress but the model provider's.

## Data, configuration & auth

**No database, and no persistent state the program owns.** A run holds one conversation in
memory as `[]ai.Message` and exits.

**The tier assignment** is one hand-written user-level TOML, never committed
([decision](/specs/05-tier-assignment-schema.md)):

```toml
[tiers.standard]
provider = "openai-codex"
model    = "gpt-5.1-codex"
# The judgment about which model suits this weight of review.
```

`provider` and `model` are `kern-link`'s own identifiers verbatim — no aliasing, no
translation layer. There is no `family` key: a rescue override was proposed and dropped as
the one mechanism able to weaken the fail-closed family rule from a file where being wrong
is silent. Unrecognised keys are **warned, not ignored**, since a typo would otherwise
present as a tier that mysteriously has no reviewer.

Resolution is `--model` → `EXTERNAL_REVIEWER_TIER_<TIER>` → `EXTERNAL_REVIEWER_CONFIG` →
`os.UserConfigDir()/external-reviewer/config.toml`
([decision](/specs/06-config-resolution-and-env-overrides.md)). `os.UserConfigDir`
implements the `%APPDATA%` / `$XDG_CONFIG_HOME` split in one stdlib call, which is where a
POSIX assumption would otherwise first appear. **The binary never writes config** — no
`init`, no wizard, no prompts. Setup is `tiers` printing what resolved, and the human
editing the file.

**There is no authentication or authorization in this system**
([decision](/specs/18-auth-and-authorization.md)). No server, no identity, no second
principal; the only permission question — what may be read — is answered by confinement
rather than identity. Provider credentials are resolved **wholly** by `kern-link`: env
vars, its cross-process-locked store at `~/.pi/agent/auth.json`, and OAuth flows driven by
its own `pi-ai login`. This binary reads no credential, stores none, refreshes none, and
has no `login` command. The only credential-adjacent value that ever reaches stderr is
`AuthResult.Source` — a label like `OAuth`, never a secret.

**Discovery is a query, not a document.** `models` lists the intersection of the catalog,
what this machine's credentials reach, and what the family rule allows, with price and
context window per row — built from `GetModels`, `Refresh` and `GetAuth`. A shipped list of
recommended models would start rotting the day it was written.

**No background work and no migrations** ([decisions](/specs/19-background-work.md),
[20](/specs/20-data-migrations.md)). One synchronous run, nothing scheduled, nothing
deferred, nothing outliving the process. A program that writes no file of its own has no
file to migrate; a renamed config key surfaces through the unrecognised-key warning, and
`kern-link`'s credential store migrates with `kern-link`, not with this binary.

## Testing

Standard library `testing` only — no `testify`, matching `kern-link`, where it appears
zero times across 102 test files ([decision](/specs/16-testing-infrastructure.md)).

The hard problem is testing a multi-turn agent loop against a foreign model without a
network or a bill, and the answer comes with the dependency: **`kern-link`'s
`ai/providers/faux`** is an in-process provider that scripts responses and replays the full
streaming event protocol — its own docs call it "the executable specification of the event
contract". Registering it in a `MutableModels` and scripting a tool-call sequence exercises
the real loop, real tool dispatch, real cost accounting and real stop conditions, offline
and free. That retires most of SCOPE's first risk at no cost.

Around it: table-driven tests for the family classifier over the entire embedded catalog
plus fixtures for dynamic-provider id shapes; `t.TempDir()` repositories for confinement,
allow-list and traversal refusals, including symlink escapes where the platform supports
them; throwaway `git init` repositories for `git_read` and `git ls-files`, skipped when
`git` is absent; and full CLI coverage in-process, since `run()` takes its streams as
parameters and returns an exit code. Live model tests are gated at runtime by
`os.Getenv` + `t.Skip` naming the missing variable rather than by build tags, so
`go test ./...` is green offline on a clean checkout. No golden files for model output —
it is not deterministic and asserting on it produces a suite that fails for the wrong
reasons.

`go test ./... -race` is the command.

## Distribution & operations

`go install github.com/julienlegoux/external-reviewer@latest` is the entire distribution
story ([decision](/specs/17-distribution-and-ci.md)). No releases, no tags, no changelog,
no Dockerfile, no signing — `@latest` on an untagged module resolves to the default
branch's newest commit, which is what a single-user tool wants. `version` reports
`runtime/debug.ReadBuildInfo()`, so a report traces to a build without release machinery.

One workflow on push and pull request, `go-version-file: go.mod`: a **`test`** job
(`go build ./...`, `go test ./... -race`) on **ubuntu-latest and windows-latest**, and a
**`lint`** job (`golangci-lint`, pinned, with `gosec`/`errcheck`/`govet` — the safety
properties here are path handling and process execution). The two-OS matrix is the
constraint check, not thoroughness: Windows is the development machine, Linux is where CI
and every implementer agent run, and `os.Root` behaviour and path separators are exactly
where that splits.

**Observability is stderr and nothing else.** No logging library, no metrics, no tracing,
no error reporting. `kern-link`'s non-fatal `AssistantMessageDiagnostic` values are printed
as `warn` lines rather than discarded — no version of `kern-link` redacts them, so the
`warn`/`error:` writers escape the message's control characters themselves before printing,
which stops a diagnostic or a provider's error body from forging a line or reaching the
terminal raw without claiming to redact anything it carries.

**Performance targets, stated so nothing gets gold-plated**
([decision](/specs/15-performance-and-scale.md)): one review per process, one process at a
time, sequential tool execution, no caching, no concurrency beyond the single pump
goroutine each `kern-link` stream owns. Latency is unbudgeted — a review takes as long as
the model takes, and a `search` is microseconds against a round trip measured in seconds.
The binding resources are the **context window** (a hard limit that result caps address,
and the mechanism behind SCOPE's unmitigated risk 3), then **quota and rate limits**, then
wall clock. Tool results stream to their caps rather than being read whole and trimmed,
which is the one performance property that is a correctness property.

**What leaves the machine** is the contents of the allowed subtrees, sent to a third-party
model. That is the function, not a leak — and because reach is declared per invocation, the
exposure of any run is visible in the command line that produced it.

## Departures from SCOPE

Recorded because SCOPE drifting from SPECS costs a user nothing and costs a developer an
afternoon. The first three are amended inline in SCOPE with pointers to the decision that
caused them; the fourth is deliberately not.

1. **The credentialed family is OpenAI, not Google** — subscription OAuth via
   `openai-codex`, which has no API-key path at all and speaks WebSocket + zstd rather than
   SSE. SCOPE's risk 2 named Google ([decision](/specs/05-tier-assignment-schema.md)).
2. **Cost is flat, so it is not the quantity that matters.** Success criterion 3 now turns
   on wall-clock time and quota headroom; "unsupervised spend" is quota and rate-limit
   exhaustion shared with everything else on the account, not a bill. Cost figures are kept
   as a volume proxy that becomes real if a tier ever names an API-key provider
   ([decision](/specs/09-read-only-tool-contract.md)).
3. **SCOPE's constraint that `kern-link`'s example covers a single round trip was already
   stale** when written — `docs/usage.md` § Tool calls documents the full multi-turn loop.
   The loop is still this project's code, so the constraint is weaker rather than void
   ([decision](/specs/03-kern-link-dependency-policy.md)).
4. **The mechanism accepts tasks other than reviewing**, because the caller owns the system
   prompt, while SCOPE's non-goals still exclude other task types. That is not a
   contradiction to repair: the non-goal governs what v1 ships, documents, supports and
   tests — not what the mechanism permits. It is deliberately unadvertised, and **no
   task-type guard should be added** ([decision](/specs/08-system-prompt-and-task-assembly.md)).
   The real containment is unaffected: read-only tools, an allow-list, no writes, no GitHub,
   no shared session context, one synchronous run.
