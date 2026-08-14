---
type: Reference
title: "Epic 3 — Success criteria verification"
description: "Four real reviews launched the way the swapped contract launches them — one document pass on gpt-5.5 and the same merged-diff pass on all three gpt-5.6 models — with every lead verified against the files and SCOPE's three v1 success criteria answered against measured numbers."
tags: [epic-3, verification]
timestamp: 2026-08-13T12:20:00Z
epic: 3
---

# Epic 3 — Success criteria verification

Epic 3's last deliverable is not code: it is the evidence that the three criteria
[SCOPE](../../planning/SCOPE.md) sets for v1 actually hold. Two of them can only be
answered by running the real thing, and the third is a judgment made from measured
numbers.

This records **four runs**. Run A is a document review — the shape `review-issues`
produces. Runs B, C and D are the *same* merged-diff review on each of the three
`gpt-5.6` models the user actually intends to use, so the three can be compared on one
task. Everything below is observation; where a number contradicts
[MEASUREMENTS](/epic-2-read-only-agentic-loop/MEASUREMENTS.md), the contradiction is
stated rather than smoothed.

## How to read the figures

- **`in=`** on a `turn` line is cumulative across the run and each figure includes cached
  prompt tokens. The **difference between two consecutive `in=` figures is that turn's
  prompt**, which is the number that matters for context. The *peak prompt* column below
  is the largest such delta, not the total.
- **`$`** is `kern-link`'s catalog price times tokens and is **notional**. The credential
  is subscription OAuth (`auth=OAuth`); nothing here was billed per token and no dollar
  figure below is money that was spent. The figures are the volume proxy that becomes
  real if a tier is ever assigned an API-key provider — read them as relative weight
  between models, nothing more.
- Wall clock is measured twice: by the `done` line from process start, and externally
  around the process. The two agree to under a second in all four runs; the external
  figure is in [the meta files](/epic-3-reviewer-selection-integration/verification/).

## How the runs were launched

Through the delegation contract in `_shared/review-interfaces.md` as rewritten by
[issue 10](/epic-3-reviewer-selection-integration/issues/10-review-interfaces-external-reviewer-swap.md):
the request object on stdin carrying `{"system", "task"}`, the contract's system prompt
**verbatim and unedited**, `--allow` naming only the subtrees the task needs, stderr kept
in full.

```
# Run A
external-reviewer review --allow docs/planning --allow docs/epics \
  --model openai-codex/gpt-5.5 <repo> < request.json

# Runs B, C, D — identical but for the model
external-reviewer review --allow . \
  --model openai-codex/gpt-5.6-sol|-terra|-luna <repo> < request-diff.json
```

**One departure from the contract, forced by the machine.** The contract says a skill
asks for a `--tier` and never names a vendor. This machine has **no tier assignment
file** — `external-reviewer tiers` names
`C:\Users\lgxju\AppData\Roaming\external-reviewer\config.toml` and reports all three
tiers `unassigned` — so every tier resolves to no reviewer and exits `1`. `--model` was
used instead, and no config file was written: that file is the user's own machine state,
not something a verification run creates. The consequence is recorded under criterion 1
below, because it is not a detail — it is the default state of the integration.

## The four runs

| Run | Model | Surface | `--allow` | Turns | Tool calls | in (cum) | out | Peak prompt | Notional | Wall clock | Stop | Exit |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| **A** | `gpt-5.5` | Epic 3's 11 issue files, `EPIC_3.md`, SCOPE/SPECS/CONVENTIONS/DRIFT and 18 decision docs | `docs/planning`, `docs/epics` | 6 | 11 | 376,794 | 5,725 | 104,351 | $1.6871 | **2m02s** | `stop` | 0 |
| **B** | `gpt-5.6-sol` | Epic 3's merged diff `24e1b24...729dee9` — 78 files, 7,485 added / 1,074 removed | `.` | **12** | **27** | 1,242,931 | 6,494 | 152,258 | $4.7299 | **2m54s** | `stop` | 0 |
| **C** | `gpt-5.6-terra` | the same diff | `.` | **5** | 16 | 454,619 | 4,384 | **205,590** | $0.8374 | **1m38s** | `stop` | 0 |
| **D** | `gpt-5.6-luna` | the same diff | `.` | 10 | 19 | 865,064 | 3,927 | 121,415 | **$0.1686** | **1m36s** | `stop` | 0 |

All four resolved on the line `model   openai-codex/<id>  auth=OAuth`, ended on the
model's own stop reason, wrote a real markdown report to stdout (4,686 / 7,352 / 3,806 /
2,050 bytes) and exited `0`. **No run failed, and no run was retried** — against
MEASUREMENTS' observed one-in-four transient provider failure, this is four for four.

### Which tools each run actually used

| Run | `git_read` | `read_file` | `list` | `search` |
|---|---|---|---|---|
| A | **0** | 8 | 3 | 0 |
| B | **21** | 0 | 2 | 4 |
| C | **16** | 0 | 0 | 0 |
| D | **16** | 1 | 1 | 1 |

Run A called `git_read` zero times: a document review pulls no diff, which is why it
cannot speak to the `paths` parameter at all and why runs B–D were made diff-shaped.
`git_read`'s `log` subcommand was reached for the first time by a real model (run B,
turn 1). `status` still has not been.

## Criterion 1 — the pipeline calls this binary, and `opencode` is gone

> `_shared/review-interfaces.md` no longer mentions `opencode`, and the external pass
> runs through `external-reviewer`.

**Pass, with one caveat that is not cosmetic.**

- `grep -rin opencode` over the skills repository's `skills/` tree on branch
  `feat/external-reviewer-delegation` returns **nothing**. The only surviving mentions are
  in `docs/log.md`, which is a changelog recording the removal.
- The external pass demonstrably ran through this binary four times. The evidence is the
  first line of each transcript — `model   openai-codex/gpt-5.6-sol  auth=OAuth` and its
  three siblings — which no other tool writes.
- **The caveat**: the rewritten contract lives in an **unmerged** pull request
  (`julienlegoux/skills#42`), and the copy installed at `~/.claude/skills` still probes
  `command -v opencode`. The criterion holds for the artifact; it does not yet hold for
  the machine, and will not until that PR merges.
- **The second caveat, and the more consequential one**: with no tier assignment file, a
  skill following the contract exactly asks for `--tier standard`, gets exit `1`, and
  takes the contract's *silent* native fallback. The contract only points a user at
  `external-reviewer tiers` "when the user asks how to set it up". So on a machine in its
  default state the delegation is off, and nothing says so. Every run recorded here had to
  bypass tiers to happen at all.

## Criterion 2 — leads that survive verification

> A real `review-epics` or `review-issues` run on this project's own artifacts produced
> leads, at least some of which survived verification against the files.

**Pass. 18 leads across four runs; 16 survived verification; 2 did not.** Every lead was
checked against the file it named, and the ones that could be settled by running something
were run rather than reasoned about — the transcripts are in
[the experiments file](/epic-3-reviewer-selection-integration/verification/lead-verification-experiments.md).

| Run | Leads | Held | Did not hold |
|---|---|---|---|
| A | 4 | 3 | 1 |
| B | 6 | 5 | 1 |
| C | 5 | **5** | 0 |
| D | 3 | 3 | 0 |
| **Total** | **18** | **16** | **2** |

### The lead that mattered most — a reproducible crash

Run C, lead 1: `internal/config/config.go`'s `undecodedWarning` hands
`key[:len(key)-1]` to `joinDotted`, which indexes `parts[0]`. A **top-level** unrecognised
key in the tier file makes that slice empty. Reproduced exactly as the lead said it would
be:

```
$ EXTERNAL_REVIEWER_CONFIG=toplevel-key.toml external-reviewer tiers
done    turns=0  in=0 out=0  $0.0000  0.0s  stop=
panic: runtime error: index out of range [0] with length 0
```

This is a defect against one of this epic's own acceptance criteria — *"an unrecognised
config key produces a `warn` line rather than a silently missing tier"* — for the one
shape of unrecognised key nobody tested. `review` reaches the same `config.Load` whenever
`--model` is absent, so it panics identically. It is out of scope to fix here: a
verification PR that also changes behaviour has verified something that no longer exists.

### A lead that held

Run B, lead 3, on `internal/cli/models.go:165–172`:

> `providerCredentialed` returns `err == nil && result != nil`. Every error is therefore
> treated exactly like `(nil, nil)` … That can misreport a broken OAuth credential,
> credential-store failure, or canceled context as an absent capability.

Read at the file: the two lines are quoted correctly, and the contrast the lead draws is
real — the review path preserves `*ai.ModelsError` as exit 2 and `tiers` renders
`credential broken` as its own status, while `models` collapses all of it into a silent
absence.

### A lead that did not hold

Run B, lead 1: *"`git_read.paths` appears to let Git expand a pathspec across the
sensitive-file floor"*, with `--allow docs` and `paths: ["docs/*"]` reaching `docs/.env`.

The first half is true and the conclusion is false. `git diff … -- 'docs/*'` does return
`docs/.env`'s full patch — **and so does the control the lead did not propose**,
`git diff … -- docs`, which is exactly what `gitReadPathspecs` emits when the model names
no paths at all. The parameter therefore widens nothing; it is the strict narrowing its
comment claims. The lead named a regression that is not one.

Run A's discarded lead was weaker still — a claimed `provider/model` versus `provider/id`
wording drift where both spellings name the same string and no artifact would record a
wrong value.

### What the verification found that no run did

Chasing run B's lead 1 to its control turned up the real property underneath, which all
three diff runs circled and none stated: **the floor is a check on names the model
spells, not a filter on what git returns.** `deniedByFloor` refuses `.env` and
`:(literal)docs/.env` (whose trailing segment is still `.env`), and lets
`:(literal).env` through as one segment matching no pattern — but under `--allow .` every
one of those files is already in the diff that the plain granted-subtree pathspec
produces. The floor's stated purpose is that *"`--allow .` still cannot read a
credential"*; through `git_read diff`, `show` and `log` it can, and could since Epic 2.
Run D's lead 1 found the `:(literal)` spelling; the pre-existing reach is this document's
own finding.

## Criterion 3 — wall clock and quota headroom

> A standard-tier review's measured wall-clock time and quota headroom are recorded, and
> the author judges them low enough to reach for the tool again.

**Pass on wall clock. The quota half is answered by absence, and that is itself the
finding.**

- **Wall clock: 1m36s to 2m54s.** The whole four-run exercise, including the heaviest
  model on the heaviest surface, took under three minutes per run.
- **Quota headroom: the provider exposes none through this path.** No rate-limit
  allowance, window or remaining-quota figure reaches stderr — `kern-link`'s
  `openai-codex` path surfaces the credential's *source* and nothing about its budget. No
  `429`, no throttling and no `stop=failed` occurred across four consecutive runs
  totalling ~2.94M cumulative prompt tokens in about eight minutes, and nothing else on
  the account was observed to fail. That is the strongest statement the available
  instrumentation supports: **the ceiling was not reached, and the distance to it is not
  observable.** Criterion 3's second half cannot be answered with a number today.
- **Notional cost — the volume proxy, not spend**: $1.69 (A), $4.73 (B), $0.84 (C),
  $0.17 (D). The spread between sol and luna on the identical task is **28×**, which is
  the number to look at if a tier is ever assigned an API-key provider.
- **The judgment, stated plainly: yes.** A three-minute wall clock on a whole-epic diff
  is low enough to reach for the tool again without thinking about it, and it is fast
  enough that the honest constraint on using it is remembering to, not waiting for it.
  Recorded by this run's implementer from the numbers above; the author can overrule it,
  and a "no" would have been a real answer.

## The caps: nothing bit, and how close it came

The shipped ceiling is `MaxTurns: 30`, `MaxElapsed: 20m`, `MaxCost` unset
([issue 09](/epic-3-reviewer-selection-integration/issues/09-loop-caps-from-measurements.md)).

| Bound | Value | Closest run | Used | Margin |
|---|---|---|---|---|
| `MaxTurns` | 30 | B (sol), 12 turns | 40% | 18 turns spare |
| `MaxElapsed` | 20m | B (sol), 2m54s | 14.5% | 17m06s spare |
| `MaxCost` | unset | — | never checked | — |

**No run stopped on a bound**; all four carry `stop=stop`, the model's own reason. The
tested margin is therefore 2.5× on turns and 6.9× on wall clock — comfortable, and
comfortable for the same reason MEASUREMENTS predicted: the ceiling is a runaway guard,
not a tuning knob.

The one bound nobody has exercised remains the **context window**. Every model here has a
272,000-token window, and run C's largest single prompt was **205,590 — 75.6% of it**.
That is 2.07× the largest prompt MEASUREMENTS ever observed (99,163), reached on a
comparable surface, and it moves "context is not the binding constraint at this project's
scale" from comfortably true to nearly false. It is now the closest of the four ceilings
to being hit, and it is the one with no bound and no warning attached.

## Exit codes the caller actually took

All four runs exited **`0`** on the exit-0 branch of the contract: report on stdout,
diagnostics on stderr, the resolved `provider/model` read off the `model` line. Neither
the exit-1 nor the exit-2 branch was exercised by a run.

Exit `2` was nonetheless observed once, while verifying run C's lead 1 — and observed
taking the *wrong* path: the panic above exits 2 through Go's runtime, having already
written a `done` line with an empty `stop=`, rather than through
`internal/cli`'s classification with an `error:` line. A caller following the contract
would read that as `stop=` nothing and find no `error:` line to record.

## Against MEASUREMENTS

- **The `git_read` prediction, tested at last, and only half right.** MEASUREMENTS held
  that Epic 2's 12-turn run C was inflated by a path-scoped diff being unreachable, and
  that closing the gap would make the same review 6–8 turns. On the same shape of task the
  three models took **5 (terra), 10 (luna) and 12 (sol)** turns. The gap is genuinely
  closed — `paths` was used on 21 of sol's 27 tool calls, 16 of 16 for terra, 16 of 19 for
  luna, and no run repeated Epic 2's five wasted turns fumbling for a scoped diff. But the
  turn count did not collapse to 6–8: it turns out to be a **property of the model**, not
  of the surface. The prediction should be retired rather than restated.
- **Peak context more than doubled.** 99,163 → 205,590. Same repository, comparable diff.
- **Reliability improved.** MEASUREMENTS: one hand run in four died on a transient
  provider error. Here: zero in four.
- **Cost tracks turns and context, not surface size** — confirmed again. Runs B, C and D
  read the identical surface for $4.73, $0.84 and $0.17 notional.

## The three models on one task

Runs B, C and D differ in exactly one argument, so the comparison is clean.

| | **sol** | **terra** | **luna** |
|---|---|---|---|
| Turns / tool calls | 12 / 27 | **5 / 16** | 10 / 19 |
| Wall clock | 2m54s | 1m38s | **1m36s** |
| Peak prompt | 152,258 | **205,590** | 121,415 |
| Report size | **7,352 B** | 3,806 B | 2,050 B |
| Leads (held / total) | 5 / 6 | **5 / 5** | 3 / 3 |
| Notional | $4.7299 | $0.8374 | **$0.1686** |

- **sol** reads the most and reports the most: 21 `git_read` calls plus 4 `search`, the
  only run to search the test files for the absence of a case, and the widest coverage —
  it was alone in finding the `models` credential collapse, the dropped `tiers` refresh
  warning, and the oversized-stdin wording. It is also the only run that produced a lead
  that did not hold, and the only one that **invented a filename mid-run**: it asked twice
  for `docs/planning/specs/06-tier-resolution-and-fallback.md`, which does not exist,
  before recovering with a `list` and citing the correct
  `06-config-resolution-and-env-overrides.md` in its report. The error never reached the
  deliverable.
- **terra** is the one to reach for. Fewest turns, fastest to a conclusion, every lead
  held, and it found the crash — reading in far bigger gulps (205,590-token peak) rather
  than more often. Best leads-per-token of the three by a wide margin.
- **luna** is remarkable value: 3 leads for $0.17 notional, all three substantively
  correct, including the `:(literal)` pathspec spelling neither larger model found. Its
  weakness is citation hygiene — two of its three leads pointed at the wrong line range or
  the wrong function while describing the right thing, which is precisely the failure mode
  a verifying caller absorbs cheaply and an unverifying one propagates.

Against the user's mapping — sol ≈ Opus, terra ≈ Sonnet, luna ≈ Haiku — the ordering by
review quality on this task was **terra > sol > luna**, not the price ordering. On this
evidence a `standard` tier assigned `terra` is the right default, `heavy` on `sol` earns
its cost only when breadth matters more than precision, and `light` on `luna` is usable
provided the caller does what the contract already requires and verifies every citation.

## Environment notes worth carrying forward

- **`version` misreports in a git worktree.** The binary built here printed
  `revision=526e69b… dirty=true` while the tree it was built from was `0f9ba1f` and clean.
  Go's VCS stamping walks past a linked worktree's `.git` *file* to the shared checkout.
  Whatever [specs 17](../../planning/specs/17-distribution-and-ci.md) intends by "a report
  can be traced back to a build", a worktree build does not deliver it.
- **No tier config was created.** All measurements above were taken with
  `%APPDATA%\external-reviewer\config.toml` absent, which is the machine's real state.

## What this leaves open

Recorded here, fixed elsewhere — this document changes no `.go` file.

1. **The top-level-key panic.** A crash on a hand-written config file, on the path both
   `tiers` and `review` take. The highest-priority follow-up of the four runs.
2. **The floor does not filter diff output.** `--allow .` plus `git_read diff` returns
   credential file contents. Pre-existing since Epic 2, and a decision for the author:
   either the floor's promise narrows in `floor.go` and specs 10/14, or `git_read` filters
   what git returns.
3. **The environment tier override does not survive a malformed config file**, and the
   documented precedence says it should. Two runs found this independently.
4. **`EXTERNAL_REVIEWER_CONFIG` is documented absolute and accepts relative.**
5. **Stale commentary about unset bounds** in `internal/cli/review.go:139–143` and
   `internal/cli/loop_test.go:402–404`, both saying this epic ships no cap values, which
   stopped being true when issue 09 merged.
6. **`models` collapses every auth error into "uncredentialed"**, and `tiers` discards
   catalog-refresh warnings.
7. **`errPromptTooLarge` still says "task prompt"** when stdin now carries the request
   object.
8. **Delegation is silently off on a machine with no tier file** — the integration gap
   under criterion 1, and the only one on this list that is about the *contract* rather
   than this repository.

## Evidence

All under
[`verification/`](/epic-3-reviewer-selection-integration/verification/), one set per run —
the full stderr transcript, the model's report as returned, the externally measured timing
and exit code, and the two request objects exactly as piped in:

```
verification/run-a-stderr.txt        run-a-report.md        run-a-meta.json
verification/run-b-sol-stderr.txt    run-b-sol-report.md    run-b-sol-meta.json
verification/run-c-terra-stderr.txt  run-c-terra-report.md  run-c-terra-meta.json
verification/run-d-luna-stderr.txt   run-d-luna-report.md   run-d-luna-meta.json
verification/run-a-request.json      verification/runs-bcd-request.json
verification/lead-verification-experiments.md
```

The reports are kept as **evidence for the lead verdicts above**, not published as review
reports: no `docs/reviews/` document was written from them, because verifying leads is
this document's job and grading the external model's prose is out of scope.
