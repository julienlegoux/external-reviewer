# Implementation Review Report 2

## Scope

- **Epic reviewed**: [`docs/epics/epic-1-walking-skeleton/EPIC_1.md`](/epics/epic-1-walking-skeleton/EPIC_1.md) — "Walking skeleton".
- **Diff reviewed**: `9bb1745^1..84f3d21` — 5 PRs, 31 files, +2817/-15 lines (~2200 of it
  Go source and tests). Every issue's `gh_pr` resolved to a real MERGED PR against
  `develop`, and the range is contiguous: `git log --merges 9bb1745^1..84f3d21` returns
  exactly the epic's five merges and no foreign work, so the range *is* the surface.

  | Issue | PR | Merge commit | State |
  |---|---|---|---|
  | 01 scaffold-module-and-ci | #16 | `9bb1745` | MERGED |
  | 02 review-invocation-parsing | #17 | `d5b40f1` | MERGED |
  | 03 diagnostics-and-exit-codes | #18 | `a3f3911` | MERGED |
  | 04 model-resolution-and-auth | #19 | `4445291` | MERGED |
  | 05 single-turn-model-call | #20 | `84f3d21` | MERGED |

  Merges are unsquashed, so per-issue attribution survived `close-epic`'s branch deletion
  and every finding below names the PR it came from.
- **Yardsticks**: the epic's 6 acceptance criteria and the five issues' own
  "Acceptance criteria / Definition of done"; [`CONVENTIONS.md`](/planning/CONVENTIONS.md);
  [`SPECS.md`](/planning/SPECS.md); [`DRIFT.md`](/planning/DRIFT.md) — **2 accepted
  entries excluded** (the `kern-link` v0.1.1 pin and its consequent `openai-codex/gpt-5.5`
  model choice; the credential-store carve-out from "the binary writes no files").
  Neither is re-reported here.
- **Method**: five lenses, each a separate reviewer over the same diff — acceptance
  honesty, test integrity, seams, convention erosion, correctness & risk. Findings merged
  and deduped across lenses; three findings below were reached independently by two or
  three lenses, and that is noted where it happened.
- **Test integrity**: 46 acceptance criteria read against their tests; **24 mutations
  probed**, one at a time, in a disposable worktree (removed afterwards, `git worktree
  list` clean); **6 mutants survived**.
- **Not verified**:
  - **`go test ./... -race` was never run.** The review machine has no C toolchain
    (`-race requires cgo`; `gcc` not on PATH), so the baseline was established with
    `go test ./... -count=1` — green, 3 packages. The epic's criterion and issues 01–05's
    repetition of it are **CI-verified only** (`.github/workflows/ci.yml:23`, ubuntu-latest
    and windows-latest). No finding below is race-dependent.
  - `golangci-lint run` was not executed as a gate; `.golangci.yml` was read, and its
    behaviour probed empirically for the two findings that turn on it (P1-3, P2-1).
  - `reviewer.DefaultModels` and `reviewModels`' real-registry branch — unreachable
    offline by construction; no mutation of them would mean anything.
  - The live-model round trip and issue 05's manual risk-retirement run — skip-gated and
    human-only by their own criteria. PR #20's body was read and does record the manual
    run (exit 0, real token/cost/elapsed figures).
  - Credential-leak assertions (`preflight_test.go:272`, `resolve_test.go:201`) were read
    but not mutated: they are wide-net `strings.Contains` checks, and a mutation that
    leaked a credential would have to introduce a new print rather than alter existing
    code.

## Findings

### P0 — The task prompt is never asserted to reach the model

- **Location**: `internal/reviewer/run.go:45` (`Content: ai.UserText(task)`); issue 05, PR #20.
- **Violates**: EPIC_1 § Acceptance criteria — *"Invoked with a repository path **and a
  prompt**, the binary returns the model's markdown on stdout and exits `0`."*
- **Problem**: the epic's headline criterion — that the user's prompt is what the reviewer
  answers — has no test. Every test scripts `faux` with a fixed response and asserts only
  that the response came back, so the request side of the round trip is unobserved.
- **Evidence**: mutating `ai.UserText(task)` → `ai.UserText("")` leaves the **entire suite
  green** (all 3 packages, 9 test files). A binary that silently discarded the prompt and
  reviewed nothing would ship. `faux.StepFunc` already receives the `ai.Context` — it is
  used that way at `roundtrip_test.go:118` — so one assertion inside a scripted step
  closes this.
- **Disposition**: fix-now — recorded here only; no issue filed (see *Dispositions*).

### P0 — `FinalText` is only ever exercised with a single text block

- **Location**: `internal/reviewer/run.go:103-113`; issue 05, PR #20.
- **Violates**: EPIC_1 § Scope — *"the report as markdown on **stdout**, verbatim and with
  no envelope"*; issue 05 — *"puts the model's final text on stdout **byte-for-byte
  unchanged**"*.
- **Problem**: `FinalText` concatenates the message's text blocks in order and filters out
  thinking blocks and tool calls. Neither behaviour is asserted: every scripted response
  in the epic is one `faux.TextMessage`, so the type switch at `run.go:109` only ever sees
  all-text, single-block messages.
- **Evidence**: adding `break` after `out.WriteString(text.Text)` — making the report the
  *first* text block only — leaves the whole suite green. The doc comment's promise that
  "thinking blocks and tool calls … never reach stdout" is likewise unverified. Epic 2's
  tool loop is built directly on both properties.
- **Disposition**: fix-now — recorded here only.

### P1 — SIGINT is trapped but unobservable before the model call: `review <repo>` becomes uninterruptible

- **Location**: `internal/cli/run.go:35` × `internal/cli/review.go:194`; issues 01 and 02
  (PRs #16, #17) meeting. Found independently by the acceptance-honesty and
  correctness lenses.
- **Violates**: EPIC_1 § Acceptance criteria — *"The `done` line appears on all three
  paths"*; issue 03 § Scope — *"Interruption terminates the run at exit `2` with a reason
  on stderr and an empty stdout."*
- **Problem**: `signal.NotifyContext(context.Background(), os.Interrupt)` **replaces Go's
  default terminate-on-SIGINT** for the whole lifetime of `Run`. Cancellation is then only
  observed where a `ctx` is actually consulted — `run`'s entry check, `Resolve`, and the
  stream. The prompt read at `review.go:194` is `io.ReadAll(stdin)`, which takes no
  context, and is the *documented* invocation form (`usage.go:16-17`: "the task prompt
  comes from `--prompt`, or from stdin when `--prompt` is not given").
- **Evidence**: run `external-reviewer review /path/to/repo` with stdin on a terminal.
  `io.ReadAll` blocks for EOF. Ctrl-C is delivered to the notify goroutine, cancels a
  context nobody reads, and the `read(2)` resumes. Every subsequent Ctrl-C does the same —
  `stop()` is deferred to the end of `Run`, so the signal stays trapped. The process hangs
  indefinitely, **emits no `done` line and returns no exit code**; only Ctrl-D or `kill`
  escapes. Before this wiring existed, the default handler would have terminated it. No
  test covers interruption outside a pre-cancelled context (`run_test.go:37`) and
  mid-stream cancellation (`roundtrip_test.go:173`), both of which enter through code that
  does consult `ctx`.
- **Disposition**: fix-now — recorded here only.

### P1 — A failing stdout write leaves a partial report on stdout with exit 2, or kills the process before the `done` line

- **Location**: `internal/cli/review.go:77` × `internal/cli/run.go:49`; issues 05 and 03
  (PRs #20, #18) meeting.
- **Violates**: EPIC_1 § Scope — *"the report as markdown on **stdout** … **empty on any
  failure**"*; CONVENTIONS § Error handling L142-145 — *"stdout is the model's final text
  verbatim and is empty on any failure"*.
- **Problem**: `io.WriteString(stdout, report)` is treated as all-or-nothing. It is not.
- **Evidence**: two manifestations of one hole.
  - *Error mid-write*: redirect stdout to a filesystem that fills. `poll.FD.Write` loops,
    so a silent short write is impossible, but an error after N bytes is not: the first N
    bytes are already on stdout, `WriteString` returns `ENOSPC`, the run exits `2` — and
    the calling skill's report file now holds a truncated report beside a failure code.
    The invariant fails exactly where it is load-bearing.
  - *Broken pipe*: `external-reviewer review /repo --prompt "…" | head -20`. `head` exits
    and closes the read end; the multi-KB write gets `EPIPE` on fd 1. The program
    registers `os/signal` only for `os.Interrupt`, so the Go runtime's default rule
    applies — **SIGPIPE on fd 1 kills the process by signal**. `defer diag.WriteDone`
    never runs; the shell sees 141, not 0/1/2. Piping into a pager and quitting does the
    same.
- **Disposition**: fix-now — recorded here only.

### P1 — `gofmt` is enforced by nothing

- **Location**: `.golangci.yml` (no `formatters:` section anywhere in the file);
  `.github/workflows/ci.yml:21-23,34-36`; issue 01, PR #16.
- **Violates**: CONVENTIONS § Code style & formatting L25 — *"`gofmt` is law: **it runs in
  CI**, and style is never debated in review."*
- **Problem**: golangci-lint **v2 moved formatters out of `linters`**. `gofmt` runs only
  when listed under a top-level `formatters:` block, which this config does not have. The
  CI `test` job runs `go build` and `go test` and nothing else. So no job in the repository
  checks formatting.
- **Evidence**: a deliberately malformed file (`func   BadlyFormatted( ) int {` with mixed
  indentation) run against the repo's own config → **0 issues**; `golangci-lint fmt
  --diff` reports it. Enabled linters confirmed as `errcheck forbidigo gosec govet
  ineffassign staticcheck unused` — no formatter among them. The source is currently
  gofmt-clean (verified against the committed blobs, not the CRLF working tree), so this
  costs nothing today and everything the first time an agent implementer's editor differs.
  Related: the repository has no `.gitattributes` while the dev machine has
  `core.autocrlf=true`, so adding gofmt enforcement without `* text=auto eol=lf` would
  produce immediate false CI failures.
- **Disposition**: fix-now — recorded here only.

### P1 — The OS-path/wire-path split was never established, and issue 02's criterion was ticked without being met

- **Location**: `internal/cli/review.go:171,173,175,178,180`; issue 02, PR #17. Reached by
  the convention-erosion lens and corroborated by acceptance honesty.
- **Violates**: CONVENTIONS § Paths and platforms L72-76 — *"**OS paths** use
  `path/filepath` … **Wire paths** — tool input and output, `--allow` arguments, **stderr
  diagnostics** — are `path`-package, `/`-separated and repo-relative on every platform.
  Conversion happens at the boundary, with `filepath.ToSlash` / `filepath.FromSlash`."*
  And issue 02's own criterion: *"Repository paths are handled with `path/filepath` as OS
  paths, per the OS-path/wire-path split in CONVENTIONS."*
- **Problem**: `path/filepath` appears in **zero** product files. The positional repository
  argument goes raw into `os.Stat` and raw back out to stderr through `%q`.
- **Evidence**: on Windows,
  `error: repository path "C:\\no\\such\\repo": GetFileAttributesEx C:\no\such\repo: The
  system cannot find the path specified.` — three breaches in one line: backslashes on a
  stderr diagnostic, `%q` double-escaping them, and the OS's capitalised, full-stopped
  sentence spliced into a message CONVENTIONS L134-135 requires lowercase and unpunctuated
  (it lands in the wrapped error at `review.go:175` too). The same text differs on Linux.
  PR #17's body ticks the criterion as "Repository paths are handled with
  path/filepath/os.Stat as OS paths"; `grep filepath` over non-test source returns
  nothing. The two-OS matrix did not catch it because no test asserts on a path — exactly
  the case CONVENTIONS L76 legislates for (*"A test that asserts on a path asserts the wire
  form"*). Epic 2's `--allow` arguments and `os.Root` confinement are *the* wire-path
  boundary and arrive with no precedent to copy and a stderr formatter demonstrating the
  wrong habit.
- **Disposition**: fix-now — recorded here only.

### P1 — The `error:` reason line has four shapes, and the empty-argv path emits none

- **Location**: `internal/cli/run.go:53,59,75-76`; `internal/cli/review.go:160,165,173,178,196,203,224,227`;
  issues 01, 02 and 03 (PRs #16, #17, #18). Found by the seams lens, corroborated by
  acceptance honesty.
- **Violates**: issue 02 — *"Every exit-`2` case writes a reason to stderr naming what was
  wrong"*; EPIC_1 § Acceptance criteria — *"…or a usage error — stdout is empty and the
  exit code is `2`, **with the reason on stderr**"*.
- **Problem**: one concept, four renderings, and one path with nothing.

  | Path | What stderr gets | From |
  |---|---|---|
  | `run.go:58-63` empty argv | usage text only — **no reason line at all** | issue 01 |
  | `run.go:74-80` unknown command | `error: unknown command "x"` **+** usage text | issue 02 |
  | `review.go:152-156` bad flag | `flag`'s own message + `Usage of review:` — no `error:` prefix, and the *subcommand's* usage | issue 02 |
  | `review.go:160-206` other usage errors | `error: …`, **no** usage text | issue 02 |

- **Evidence**: `git show 9bb1745:internal/cli/run.go` shows `run.go:58-63` is issue 01's
  original `Run` body (`Fprint(stderr, usage); return 2`), which issues 02 and 03 wrapped
  in a switch and re-labelled `stop=usage` without ever retrofitting the `error:` line
  their sibling branch six lines below now writes. `errors.New("no command given")` is
  created, classified and never printed. The only test on it (`run_test.go:23-25`) asserts
  `stderr.Len() != 0`, which the leftover usage text satisfies. Compounding: `internal/diag`
  claims to implement "the run's stderr diagnostics" and issue 03 scoped the vocabulary as
  `turn`/`tool`/`warn`/`done`, yet the most important line a failing run emits is
  hand-rolled `fmt.Fprint*` at **eight sites across two files**; there is no
  `diag.WriteError`, and `error: ` is not padded to the 8-column prefix width every
  `diag.Write*` uses. Epic 2 adds refusal reasons, tool failures and bound-exceeded
  terminations to this same surface. *No path double-prints* — all traced; the defect is
  omission and divergence.
- **Disposition**: fix-now — recorded here only.

### P1 — The exit-1 path's "needs no line" property is unasserted

- **Location**: `internal/cli/review.go:220-221`; issue 04, PR #19.
- **Violates**: EPIC_1 § Acceptance criteria — *"stdout is empty and the exit code is `1`,
  **the silent-fallback case the caller needs no line about**"*, restated at
  `review.go:143-146` as the reason the not-reached error is deliberately not written to
  stderr.
- **Problem**: the silence is the criterion, and nothing tests it.
- **Evidence**: adding `_, _ = fmt.Fprintln(stderr, "error:", err)` to the
  `case errors.Is(err, ErrNoReviewer):` branch leaves the whole suite green.
  `preflight_test.go:184` and `exit_test.go:44` assert exit code, empty stdout and
  `stop=no_reviewer` — never the absence of an error line. `TestRun_BrokenCredential_ExitsTwo`
  *does* assert the presence of `"error:"` on the exit-2 path, so the pair that makes the
  1-versus-2 distinction observable is half-written.
- **Disposition**: fix-now — recorded here only.

### P1 — The cancellation-precedence guard is dead to the suite, and the race it guards is a coin flip upstream

- **Location**: `internal/reviewer/run.go:91-93`; issue 05, PR #20.
- **Violates**: issue 05 — *"Cancelling the context mid-stream → exit `2`, stdout empty,
  **interruption reason** on stderr."*
- **Problem**: the guard exists precisely for the case where the provider's aborted message
  wins the race against `Result` observing cancellation. No test constructs that case, so
  the branch is never entered.
- **Evidence**: deleting the three lines verbatim —
  ```go
  if err := ctx.Err(); err != nil {
      return turn, fmt.Errorf("the reviewer's turn was interrupted: %w", err)
  }
  ```
  — leaves the whole suite green. `TestRun_CancelledMidStream_ExitsTwo`
  (`roundtrip_test.go:173`) always wins the race the guard exists to lose. A scripted
  `StopReasonAborted` message *plus* a cancelled context would enter it deterministically.
  Compounding, from kern-link v0.1.1: `Stream.Result` (`ai/stream.go:155-158`) selects over
  `<-s.resultCh` and `<-ctx.Done()`, so when the stream has already completed and the user
  then interrupts, **both cases are ready and Go picks one at random** — a coin flip on
  whether a fully delivered, already-billed answer is returned or replaced by `ctx.Err()`.
- **Disposition**: fix-now — recorded here only.

### P2 — The write-API guard is under-inclusive against its own stated purpose

- **Location**: `.golangci.yml:15-59`; issue 01, PR #16.
- **Violates**: CONVENTIONS § Code style & formatting L30, L38-40 — *"**The write half of
  the filesystem API is forbidden, and CI enforces it** … a structural guarantee that
  nothing checks degrades into a promise by the third contributor, human or agent."*
- **Problem**: the config matches CONVENTIONS' enumeration exactly, and adds `os.Root.Link`
  on top — 20/20 patterns verified firing, `_test.go` exemption verified. **The
  enumeration itself is the gap.** Eight write operations outside it pass the repo's own
  config with zero findings: `os.Chmod`, `os.Chtimes`, `os.Truncate`, `os.Link`,
  `os.Root.Chmod`, and — the material ones — `(*os.File).Write`, `(*os.File).WriteString`,
  `(*os.File).Truncate`.
- **Evidence**: `os.Create`/`os.OpenFile` are blocked, but Epic 2 opens files for reading
  through `os.Root.Open`, and nothing then stops a `.Write` on the returned handle. "The
  write half of the filesystem API is not in the codebase" is currently true by author
  discipline, not by the guard. Fixing it means amending CONVENTIONS' list and
  `.golangci.yml` together, before Epic 2 opens file handles.
- **Disposition**: fix-now — recorded here only.

### P2 — No timeout anywhere: a silent provider hangs the CLI forever with no `done` line

- **Location**: `internal/reviewer/run.go:65,73`; issue 05, PR #20.
- **Violates**: correctness (unbounded blocking); EPIC_1 § Scope — the `done` line "emitted
  on **every** termination path".
- **Problem**: `StreamSimple(ctx, model, chat, &ai.SimpleStreamOptions{})` leaves
  `StreamOptions.Timeout` at zero, and kern-link arms no default.
- **Evidence**: verified in v0.1.1 — SSE path `ai/apis/internal/httpretry/httpretry.go:96`
  builds `&http.Client{}` with **no `Timeout`**; the WebSocket path (the default for codex,
  `Transport: auto`) sets `idleTimeout = opts.Timeout` = 0 at `ai/apis/codex/websocket.go:297-299`
  and arms a read deadline only `if idleTimeout > 0` (`websocket.go:355`); the 15 s
  `defaultWebSocketConnectTimeout` covers the handshake only. Scenario: the backend accepts
  the WebSocket then stops sending frames (silent middlebox, dropped route).
  `for range stream.Events(ctx) {}` blocks forever, nothing cancels `ctx`, and — per P1
  above — SIGINT is the only escape and is itself fragile. No output, no `done` line, no
  exit code. The same shape applies to `resolve.go:93` → `flock.TryLockContext(ctx, 20ms)`,
  which retries indefinitely if another process holds `~/.pi/agent/auth.json.lock`.
- **Disposition**: fix-now — recorded here only.

### P2 — Provider-controlled text is written unescaped into a line-oriented protocol, and the redaction the comment relies on does not exist

- **Location**: `internal/reviewer/run.go:95`; `internal/cli/review.go:107-119`;
  `internal/diag/diag.go:62,74`; issues 03 and 05 (PRs #18, #20) meeting.
- **Violates**: correctness (log/transcript injection); SPECS § Security, as quoted by the
  comment at `review.go:107-110`.
- **Problem**: two defects in one place.
  - *Injection*: `message.ErrorMessage` comes straight from the provider's JSON error body
    and `diagnostic.Error.Message` is `err.Error()` verbatim. Both land in
    `fmt.Fprintf(stderr, "error: %v\n", err)` and `warn    %s\n` with no escaping, while
    neighbouring sites in the same file correctly use `%q`. A response of
    `{"error":{"message":"rate limited\ndone    turns=1  in=0 out=0  $0.0000  0.0s  stop=ok"}}`
    puts a **forged `done … stop=ok` line** on stderr ahead of the real one. SPECS makes the
    transcript a parseable contract, so a caller reading the `done` line concludes success
    while the real outcome is exit 2 with empty stdout. ANSI/OSC sequences in the same field
    reach the terminal raw.
  - *False premise*: the comment states *"kern-link redacts these upstream … which is what
    keeps a credential from being reintroduced by a diagnostic."* **There is no redaction
    anywhere in kern-link v0.1.1.** `ai/diagnostics.go:41-51` `ExtractDiagnosticError` does
    `info := &DiagnosticErrorInfo{Message: e.Error()}` — it drops the stack and nothing
    else; `grep -rni redact` over the module returns only Anthropic's *thinking*-block
    redaction, unrelated; `ai/sanitize.go` strips unpaired surrogates, not secrets.
- **Evidence**: independently re-verified against the module cache during this review. No
  live leak exists today — v0.1.1's only diagnostic producer is
  `ai/apis/codex/codex.go:223` (`provider_transport_failure`, a dial error carrying no
  header value), and no adapter puts an API key in a URL. So the SPECS § Security property
  holds **by coincidence, not by construction**, and the comment tells the next reviewer not
  to look.
- **Disposition**: fix-now — recorded here only.

### P2 — `ai.CalculateCost` clobbers the adapter's service-tier-adjusted cost

- **Location**: `internal/reviewer/run.go:82-83`; issue 05, PR #20.
- **Violates**: correctness (incorrect accounting); issue 05 § Scope — *"Real
  turn/token/cost/elapsed values fed into the run state issue 03 renders, so the `turn`
  line and the `done` line carry actual numbers."*
- **Problem**: the adapter has **already** filled `message.Usage.Cost` and then scaled it:
  `ai/apis/openairesponses/stream.go:319` calls `ai.CalculateCost`, and line 334 calls
  `applyServiceTierPricing`, which multiplies every field and recomputes `Total`. Codex
  routes through this path. `run.go:83` copies the usage and calls `ai.CalculateCost`
  again, which unconditionally overwrites `Cost.Input/Output/CacheRead/CacheWrite/Total`
  from the base price sheet. The call can only discard information the adapter computed.
- **Evidence**: re-verified in the module cache during this review. `ai/cost.go:16-20`
  assigns all five fields unconditionally; `stream.go:376-386` scales them and
  `serviceTierCostMultiplier` (`stream.go:389-401`) returns **2.5** for `priority` on
  `gpt-5.5` and **0.5** for `flex`. So a `priority` response is reported at **40 %** of its
  real cost, and a `flex` response at **2×**.
- **Disposition**: fix-now — recorded here only.

### P2 — A completed, paid-for report is discarded when cancellation lands a moment after it arrives

- **Location**: `internal/reviewer/run.go:91-93`; issue 05, PR #20.
- **Violates**: correctness (lost work). Same lines as the P1 above, different failure.
- **Problem**: `stream.Result(ctx)` returns the complete message; `ctx.Err()` is checked
  *after*. The ordering is defensible for *reporting* (both outcomes are exit 2) but it
  throws away a good result.
- **Evidence**: a 90-second Codex turn finishes; the user, who stopped watching, presses
  Ctrl-C at t=90.001 s. The report is on the wire and already billed; the user gets an
  empty stdout, `stop=interrupted` and exit 2. The second window is upstream and worse: with
  `resultCh` already closed, `Stream.Result`'s select has both cases ready and chooses at
  random. Preferring a complete, non-error `message` over a late `ctx.Err()` costs nothing.
- **Disposition**: fix-now — recorded here only.

### P2 — A whitespace-only prompt passes validation and buys a real API call

- **Location**: `internal/cli/review.go:202`; issue 02, PR #17.
- **Violates**: issue 02 — *"an empty prompt (no `--prompt` and empty stdin) … exit `2`,
  stdout empty, reason on stderr"*; correctness (insufficient validation).
- **Problem**: `if task == ""` is the only guard.
- **Evidence**: `echo | external-reviewer review /repo` — stdin yields `"\n"`, which is
  non-empty. The run resolves credentials and sends a one-newline task to
  `openai-codex/gpt-5.5`: real money, real latency, a report generated from no instruction.
  Same for `--prompt " "`. The neighbouring test row (`review_test.go:87`) covers only the
  exact-empty case. `strings.TrimSpace(task) == ""` is the fix.
- **Disposition**: fix-now — recorded here only.

### P2 — `io.ReadAll(stdin)` has no size bound

- **Location**: `internal/cli/review.go:194`; issue 02, PR #17.
- **Violates**: correctness (unbounded resource consumption).
- **Problem/evidence**: `cat 8GB.bin | external-reviewer review /repo` buffers the whole
  stream into a Go string before any validation; the process is OOM-killed — again with no
  `done` line and no exit code. Short of OOM, the entire blob is transmitted verbatim to a
  third party, which for this product is the security surface CONVENTIONS § Dependencies
  L155-156 calls out. `io.ReadAll(io.LimitReader(stdin, N))` with an explicit over-limit
  error is the shape this wants, and the bound is the natural place to reject a non-text
  prompt.
- **Disposition**: fix-now — recorded here only.

### P2 — `stop=ok` is Epic 1's invention; SPECS and Epic 2 both want the model's own stop reason

- **Location**: `internal/cli/review.go:219`; `internal/reviewer/run.go:84,94`; issues 03
  and 05 (PRs #18, #20).
- **Violates**: `SPECS.md` § Interfaces — `done    turns=7  in=182430 out=9106  $0.0918
  2m14s  stop=end_turn`, quoted verbatim in issue 03 § Scope; and
  `docs/epics/epic-2-read-only-agentic-loop/EPIC_2.md` § Acceptance criteria 1 — *"the run
  ending on the model's own stop reason"*.
- **Problem**: two `StopReason` concepts now compete. `ai.AssistantMessage.StopReason` is
  read at `reviewer/run.go:94` purely to detect `error`/`aborted`, then discarded —
  `reviewer.Turn` does not expose it, though it carries `Message`. `diag.State.StopReason`
  gets a hand-written CLI vocabulary instead: `ok`, `failed`, `usage`, `help`,
  `no_reviewer`, `interrupted`.
- **Evidence**: on the success path the binary prints `stop=ok` where SPECS' worked example
  prints `stop=end_turn`. The two packages' tests encode the disagreement:
  `diag_test.go:44,73` asserts on `"end_turn"` while every `cli` test asserts on `"ok"`.
  Epic 2 must reconcile this on day one, and its AC 1 currently contradicts the shipped
  behaviour whichever way it is resolved.
- **Disposition**: **undecided — see Open questions.** This is the one finding where the
  code may be right and the standard wrong; it was raised for triage and not resolved.

### P2 — Accounting lives on the wrong side of the seam Epic 2's loop needs

- **Location**: `internal/cli/review.go:88-105` (`recordTurn`); `internal/reviewer/run.go:61-98`
  (`Next`); issues 03 and 05.
- **Violates**: `EPIC_2.md` § Scope — *"The `bounds.check(state)` seam at the top of every
  turn … watching turns, accumulated cost and elapsed wall clock"*; EPIC_1 § Goal — Epics 2
  and 3 *"add capability inside"* Epic 1's shapes "rather than reshaping them".
- **Problem**: Epic 1 put the per-turn round trip in `internal/reviewer` and the
  accumulation of its numbers in `internal/cli`. Epic 2 puts the multi-turn loop in
  `internal/reviewer` and needs the accumulated turn count, cost and wall clock *at the top
  of each turn*. Those totals exist only in `cli`, one call frame above, and only after
  `Next` has already returned.
- **Evidence**: Epic 2 must either move `recordTurn` into `reviewer` or thread
  `*diag.State` + `stderr` down into the loop — reshaping the seam, which is what EPIC_1 §
  Goal says should not be necessary. Same for `diag.WriteTool`: Epic 2 reads tool calls off
  the wire inside `Conversation.Next` (the empty `for range stream.Events(ctx)` at
  `run.go:73-74` is explicitly reserved for it), but the stderr writer is reachable only
  from `cli`.
- **Disposition**: recorded for Epic 2's planning — not a defect in Epic 1's own terms.

### P2 — Two package-level mutable test seams in product code, one of them superseded

- **Location**: `internal/cli/review.go:31` (`var performReview = resolveAndReview`),
  `internal/cli/review.go:38` (`var models ai.Models`), `internal/cli/export_test.go:21-38`;
  issues 03 and 04.
- **Violates**: CONVENTIONS § Testing L119-121 — *"Prefer integration-level tests where the
  real thing is cheap; mock only true externals. The model is the one true external, and
  `kern-link`'s `faux` provider is how it is mocked."*
- **Problem**: issue 03 needed to reach the exit-1 and exit-2 terminations before a real
  reviewer existed, so it put a mutable function pointer in production code plus an adapter
  that flattens `reviewRequest` back into `(repoPath, task string)`. Issue 04 then arrived
  with the *sanctioned* mechanism — `SetModelsForTest` over `faux`.
- **Evidence**: issues 04/05 prove the sanctioned seam drives every one of those
  terminations for real — `preflight_test.go:184` (`stop=no_reviewer`),
  `preflight_test.go:225` and `roundtrip_test.go:110` (`stop=failed`) cover exactly what
  `exit_test.go:44` and `exit_test.go:73` stub out. Only one assertion survives uniquely in
  the stubbed pair: the two-level wrap at `exit_test.go:45`, which is issue 03's explicit
  criterion and can be asserted against `classify` directly. Retiring `performReview` would
  delete a mutable global from shipped code. Separately: no test uses `t.Parallel()` and
  every override uses `defer restore()`, so there is **no data race today** — but `cli.Run`
  is exported, and the first `t.Parallel()` or concurrent caller makes it one.
- **Disposition**: fix-now — recorded here only.

### P2 — The "reusable `faux` harness" is a per-package copy, duplicated twice

- **Location**: `internal/reviewer/resolve_test.go:25-121` vs `internal/cli/preflight_test.go:17-143`;
  `roundtrip_test.go:50-55`; issue 04 wrote both copies inside PR #19. Found by three lenses.
- **Violates**: issue 05 § Scope — *"A reusable `faux`-provider test harness the later
  epics build their loop tests on."*
- **Problem**: ~75 lines are byte-identical modulo one constant rename — `authProvider`,
  `credentialedAuth`, `unconfiguredAuth`, `brokenAPIKeyAuth`, `expiredOAuthAuth`, and the
  credential literal itself (`secret` / `preflightSecret`, same value). Neither copy is
  importable: both are test-only files in `package cli_test` / `package reviewer_test`.
- **Evidence**: the registry builders have *diverged* rather than converged — `fauxRegistry`
  (`resolve_test.go:31`, takes `providerID`, no price sheet), `registryWith`/`registry`
  (`preflight_test.go:43,67`, hard-codes `DefaultProviderID`, adds `ModelCost`), `scripted`
  (`roundtrip_test.go:50`). Epic 2's loop tests live in `internal/reviewer` and cannot build
  on the `internal/cli` copy — they will make a third, and Epic 2 adds four tools each
  needing the same offline registry. CONVENTIONS bans assertion *frameworks*, not a shared
  fixture package. Credit where due: issue 05 did **not** re-invent the harness — it
  extended `registry` into `registryWith` in place and reused `credentialedAuth`; the
  cross-package copy is entirely inside issue 04's own PR. PR #20's body names the
  duplication and defers it deliberately.
- **Disposition**: fix-now — recorded here only.

### P2 — `internal/reviewer/run.go` has no test file of its own

- **Location**: no `internal/reviewer/run_test.go` exists; issue 05 § Relevant files names
  both it and `internal/reviewer/faux_test.go`.
- **Violates**: nothing literally — issue 05's criteria say "driven through `Run`", and they
  are met. It is reported because of what it *cost*.
- **Evidence**: **three of the four criterion-bearing surviving mutants live in that file**
  (the prompt seeding, `FinalText`'s concatenation, the `ctx.Err()` guard), plus a fourth
  non-criterion survivor (`c.messages = append(...)` can be deleted with the suite green —
  it only matters once Epic 2's loop lands on that accumulator). Coverage is misleading
  here: the package profile reads 0.0 % because no test file lives in the package; with
  `-coverpkg=./...` it is `Next 93.8 % / FinalText 85.7 % / toolCalls 80.0 %`. High
  coverage, four survivors — which is precisely why coverage is used only for navigation in
  this review.
- **Disposition**: fix-now (subsumes the two P0s) — recorded here only.

### P2 — No red phase is visible in any of the five PRs

- **Location**: all five branches. `git log --oneline --name-status 9bb1745^1..84f3d21`.
- **Violates**: CONVENTIONS § Testing L99-103 — *"**Strict red-green test-first** — failing
  test, minimal pass, refactor — is mandatory wherever correctness is an assertable
  behavior"*; and the "written failing first" line restated in issues 01, 02, 03 and 05.
- **Evidence**: in every PR the tests and the implementation they exercise landed in the
  *same* commit — `94c740d` (run.go + run_test.go), `bde3a9f` (review.go + review_test.go),
  `2813a2f` (exit.go + diag.go + both tests), `f5a9132` (resolve.go + resolve_test.go),
  `7f9676b` (reviewer/run.go + roundtrip_test.go). Merges are unsquashed, so this is the
  real branch history, not an artefact. It does not *prove* the red phase was skipped —
  an implementer may have iterated locally and committed once — but the history cannot
  corroborate the convention for a single test in the epic, and the two P0 mutants above
  are what a missing red phase produces.
- **Disposition**: won't-fix — unverifiable retroactively, and no code change would
  discharge it. Recorded as feedback for the pipeline: if this convention is to be
  enforceable, `implement-issue` has to commit the red phase separately.

### P2 — "Was this interrupted?" is decided in three independent places

- **Location**: `internal/cli/run.go:51`; `internal/cli/review.go:222`;
  `internal/reviewer/run.go:91`; issues 03 and 05.
- **Violates**: issue 03 § Scope — *"Run state … carried in one struct that the `done` line
  is rendered from, so no path can terminate without one."*
- **Problem**: the entry check inspects `ctx.Err()`, the middle inspects the *returned
  error* with `errors.Is`, and the innermost inspects `ctx.Err()` again. The gap between
  them is reachable: `Resolver.Resolve` makes network calls (catalog refresh, OAuth token
  refresh) with `ctx`, and kern-link wraps auth failures as `*ai.ModelsError`. A `ModelsError`
  that did not wrap `context.Canceled` would fall to `runReview`'s `default` branch and
  print `stop=failed` for a run a human interrupted — the exact distinction
  `review.go:211-214`'s own comment says the transcript must record. (In v0.1.1 the cause
  *is* preserved, so this does not fire today — see *What holds up*.) Checking `ctx.Err()`
  in `runReview` would collapse all three sites to one. Related tidiness: the literal
  `"usage"` appears at eleven sites with no named constant, and `state.StopReason` is
  assigned at 15 sites in total.
- **Evidence**: **no path reaches `WriteDone` with an empty stop reason** — all 15 branches
  assign before returning, verified by reading every return in `run` and `runReviewCommand`.
  The defect is the scattered decision, not a missing one. Exit code is 2 either way, so the
  damage is confined to the `done` line.
- **Disposition**: fix-now (fold into the `error:`-line consolidation above) — recorded here
  only.

### P2 — The cache-token half of the `in=` figure is unasserted

- **Location**: `internal/cli/review.go:97`; issue 05, PR #20.
- **Violates**: issue 03 — the `done` line "contains the turn count, the accumulated input
  and output token counts…".
- **Evidence**: dropping `+ turn.Usage.CacheRead + turn.Usage.CacheWrite` leaves the whole
  suite green — `faux` reports zero cache tokens and no test scripts otherwise. The
  comment's claim ("what makes the `in=` figure comparable between a cold run and a warm
  one") is not backed by a test. The arithmetic itself is correct — see *What holds up*.
- **Disposition**: fix-now — recorded here only.

### P3 — Recorded, low stakes

Each names what it violates; none is worth its own section.

| Finding | Location | Violates |
|---|---|---|
| The `model   <provider>/<id>  auth=<source>` line ships outside SPECS' documented stderr vocabulary; `grep` finds no `auth=` anywhere in `docs/planning/`. Issue 03 set the precedent that a change to the transcript shape is amended into SPECS *by the same PR*; issue 04 did not. | `internal/diag/diag.go:54-56` | SPECS § Interfaces |
| The no-files test compares file mtimes but **not directory mtimes**. Defensible (two consecutive walks on Windows already report moved directory mtimes, re-verified in PR #20) and documented in the test — but it is a narrowing of a literal criterion, and unlike the credential-store narrowing beside it, it is not in DRIFT.md. | `internal/cli/roundtrip_test.go:276-299` | issue 05, "the tree's file set **and modification times** are unchanged" |
| `flag`'s own output and `printUsage(stderr)` emit four unprefixed lines (one tab-indented, one reading `Usage of review:`) ahead of the single prefixed `done` line. | `internal/cli/review.go:148-153`, `run.go:59,76` | CONVENTIONS L143-145, "stderr as prefixed lines" |
| The `lint` job runs on `ubuntu-latest` only, while `test` runs the two-OS matrix. Harmless for Epic 1's platform-independent rules; not harmless once Epic 2 ships `os.Root` code whose `gosec`/`govet` findings are GOOS-conditional, or the first `//go:build` file appears. | `.github/workflows/ci.yml:25-26` | CONVENTIONS L93-94, "lint + tests, **both OSes**" |
| `diag_test.go` is eight flat functions, four of them the same shape (call a writer, compare an exact string). Every other test file in the epic uses tables. | `internal/diag/diag_test.go:102-148` | CONVENTIONS L111, "table-driven is the default shape" |
| `review --help` exits `2`. `flag.ContinueOnError` returns `flag.ErrHelp`, which the handler classifies as `stop=usage`. Top-level `help`/`--help`/`-h` correctly exit 0. | `internal/cli/review.go:148,152` | issue 02, "help is not a failure path" (by analogy) |
| `formatElapsed` has no hour unit (a 95-minute run renders `95m0s`), and its `d < time.Minute` boundary survives mutation to `<=` — tests use 42.8 s and 2m14s only. | `internal/diag/diag.go:81-88` | SPECS § Interfaces, elapsed format |
| `diag.WriteTool` has exactly one caller in the repository: its own test. `reviewRequest.RepoPath` is set and read by nothing in product code — its sole consumer is `export_test.go:24`. Both are scoped-dead by issue 03 and issue 02 respectively, with callers due in Epic 2. | `internal/diag/diag.go:40-43`, `internal/cli/review.go:23,208` | dead code (deliberate) |
| Two near-identical `done`-line regexes with different anchoring, and two `turn`-line encodings (exact string vs re-derived regex). A format change touches four places. `runReviewOver` is `runReview` plus a parameter, and shadows the name of the product function `runReview`. | `exit_test.go:19-36` vs `diag_test.go:14-24`; `preflight_test.go:147-154` vs `roundtrip_test.go:36-43` | duplication |
| `os.Stat` follows symlinks (a symlink to a directory passes `IsDir()`) and the check is a classic TOCTOU gap. Inert in Epic 1 because `req.RepoPath` is never read again — but Epic 2's `os.Root` work must not build on this `Stat`. | `internal/cli/review.go:171` | correctness (latent) |
| Collapsing the `fs.Visit` "was the flag present" check to `promptSet := *prompt != ""` survives mutation, so `--prompt ""` silently changing from "usage error" to "read stdin" is undetected. Not in issue 02's table. | `internal/cli/review.go:183-188` | test coverage |
| No `.gitattributes` while the dev machine has `core.autocrlf=true`: the working tree is CRLF, the index LF. Harmless today; it makes a naive `gofmt -l .` report all 16 files as a false positive, and would produce real CI failures the moment gofmt enforcement lands without `* text=auto eol=lf`. | repository root | operational |

## What holds up

A report of only defects misrepresents the work, and this epic got the hard parts right.

- **The `run(ctx, …)` inner seam with its single `defer diag.WriteDone` is the strongest
  thing the epic built.** It makes "no path terminates without a `done` line" structurally
  true rather than per-branch discipline, and it is mutation-confirmed: gating the defer on
  success kills tests. Fifteen assignment sites all set a stop reason before returning; no
  in-process path reaches `WriteDone` with an empty one. Both P1 escapes above are
  *non-returns* (a hang, a signal death), not holes in the seam.
- **Exit-code classification is sound in both directions.** `ErrNoReviewer` is one
  `errors.New` value, produced at exactly two sites, inspected only with `errors.Is`. No
  `*ai.ModelsError` can reach exit 1 — `ModelsError.Unwrap` returns its `Cause`, never the
  sentinel — and conversely an `oauth`/`auth` error cannot reach exit 1. Six separate
  mutations of this logic all died, including swapping which failure returns which
  sentinel. Issue 04 moved the sentinel to `internal/reviewer` and left a one-value
  re-export in `cli`; no competing error taxonomy exists anywhere in the epic, which is the
  usual cost of five blind implementers and was avoided here.
- **A SIGINT during OAuth refresh is classified correctly today.** kern-link's
  `NewModelsError` keeps the cause, so `errors.Is(err, context.Canceled)` fires and the
  transcript reads `stop=interrupted`. (The P2 above is that this holds by upstream
  behaviour rather than by our own check.)
- **The resolution order is genuinely pinned.** Refresh-before-`GetModel`, and only when
  `CanRefreshModels()`, is asserted by a test that records call order *and* a counter-test
  with a static provider — inverting the order, or dropping the `CanRefreshModels` guard,
  both go red.
- **Token accounting is correct and not double-counted.** Verified in kern-link:
  `ai/apis/openairesponses/stream.go:300-302` computes `input := InputTokens - cachedTokens`
  and puts the cached count in `CacheRead`, so `Input` *excludes* cache tokens and
  `ai/cost.go` prices them as disjoint terms. Summing the three is right.
- **No credential value can reach the streams, by type rather than by habit.**
  `reviewer.Resolution` carries only `AuthSource`; the `AuthResult` never leaves `Resolve`.
  The tests plant a credential-shaped secret in *every* `AuthResult` field and assert it
  appears on neither stream.
- **The no-files test has real teeth** — inserting an `os.WriteFile` into the review path
  turns it red.
- **The `Events` drain terminates and leaks nothing that matters**, verified against
  kern-link's `Stream`: `next` returns `ok=false` once drained or as soon as
  `ctx.Err() != nil`, and `Events` closes both channels, so the cancel watcher always
  retires. `run.go:73` ranges to completion and never returns early, honouring the
  contract that an early-stopping consumer must cancel `ctx`.
- **Bookkeeping and repository hygiene are clean.** 21/21 commits carry conventional
  prefixes in imperative mood with no trailing period; branches are `issue-<n>-<slug>`;
  each PR opens with exactly one `Closes #<n>`; one direct dependency with everything else
  `// indirect`, no committed `replace`, `go.work` gitignored; the top level is exactly
  what CONVENTIONS allows; all three `internal/` packages carry exactly one package comment;
  8/8 test files are `package foo_test` (`export_test.go` is `package cli` by Go's own
  rules and contains zero `Test*` functions — the canonical export seam, not an evasion of
  the black-box rule); and all ~25 exported doc comments open with their identifier's name.
- **18 of 24 mutations were killed**, including every one aimed at the grammar table, the
  done-line shape, the resolution sequence and the exit-code switch. The suite is not
  weak — it has four specific, nameable holes, all on the request side of the round trip.

## Dispositions

At the user's direction, **this report is the only artefact of the review**: no follow-up
GitHub issues were created, no `docs/planning/DRIFT.md` entries were written, and no
product code was touched. Every finding marked `fix-now` above is a recommendation
recorded here, not scheduled work — a `fix-now` with no issue behind it is a wish, and
this file is where the wish is written down so the next reviewer finds it instead of
re-deriving it.

`lx:triage-reports` is the skill that turns this report series into a remediation epic when
that work is wanted.

## Open questions

1. **`stop=ok` versus `stop=end_turn`** — raised for triage and left undecided. Either the
   code is wrong (and should surface `ai.AssistantMessage.StopReason` through
   `reviewer.Turn`), or SPECS § Interfaces and `EPIC_2.md`'s AC 1 are wrong (and should be
   amended, with a DRIFT entry recording it). Epic 2 cannot avoid the question: its first
   acceptance criterion names "the model's own stop reason", and `diag_test.go` and the
   `cli` tests currently assert two different vocabularies for the same field.
2. **Is `stop=usage`/`stop=help` part of the transcript contract?** SPECS documents the
   `done` line's shape but not its stop-reason vocabulary. Epic 1 invented six values. A
   calling skill parsing the transcript needs the closed set, and Epic 2 adds at least
   `bounds` to it.
3. **Should the write-API guard's list live in one place?** It is currently enumerated in
   both CONVENTIONS L30-35 and `.golangci.yml`, and the P2 above requires amending both in
   step. Whether the document or the config is the source of truth is undecided.
