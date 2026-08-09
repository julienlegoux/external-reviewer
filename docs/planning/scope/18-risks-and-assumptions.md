---
type: Decision
title: "Risks & assumptions"
description: "What could sink this, and what is being assumed without proof?"
tags: [decision, scope]
timestamp: 2026-08-09T01:20:00Z
phase: scope
decision: 18
slug: risks-and-assumptions
status: decided
verdict: "Six risks, each tied to the milestone that retires it"
decided_via: triage
depends_on: [milestones, constraints]
---

# Question

What could sink the project, and what is currently assumed without evidence? Cheap now,
expensive in epic 3.

# Options

- **Record five, each with the milestone that retires it** — a risk with no retirement
  point is a worry, not a plan.
- **Record them as a flat list** — faster to write, and nothing ever tells you a risk
  stopped being one.

# Recommendation

**Record five, each tied to the milestone that retires it:**

- **`kern-link`'s multi-turn tool loop is unproven.** Its example covers one round trip;
  this project's core addition is the loop. Retired by Milestone 1–2. Mitigation: the
  author owns the upstream, so a gap is a fix rather than a blocker.
- **Foreign tool-calling fidelity varies by model.** A reviewer that cannot reliably
  call `read_file` is not a reviewer. Only Google is credentialed on this machine today
  (`GEMINI_API_KEY`), so v1 is effectively validated against one family. Retired by
  Milestone 2; mitigated by the tier config making the model swappable without a code
  change.
- **Context overflow on large surfaces.** An epic's merged diff is the heaviest read in
  the pipeline and nobody has measured it — the concept says so explicitly. Retired by
  the cost/time bound in [success criteria](/scope/14-success-criteria.md); mitigated by
  the coverage statement, which makes a truncated read visible instead of silent.
- **Unsupervised spend.** The binary runs inside a script, on the user's own credit,
  with no human watching. Retired by Milestone 2's caps; the residual risk is a cap set
  too high.
- **Assumed: leads-not-findings holds.** Everything downstream of a mediocre review is
  safe *only* because the calling skill verifies each lead against the files before it
  enters the report. If a future caller skips that step, a flawed reviewer starts
  publishing wrong findings. This is an assumption about the *caller*, not about this
  binary, and it is the one this project cannot enforce.

Assumed and not re-tested: that a same-family reviewer is worth avoiding at all, and
that the difference in family is what makes the second opinion valuable. Both come from
the concept.

# Verdict

**Accepted as recommended**, with two changes forced by the
[loop bounds](/scope/08-loop-bounds-and-termination.md) verdict. Six risks, each tied
to the milestone that retires it:

1. **`kern-link`'s multi-turn tool loop is unproven.** Retired by Milestones 1–2.
   Mitigated by the author owning the upstream.
2. **Foreign tool-calling fidelity varies by model.** Only Google is credentialed on
   this machine today, so v1 is validated against one family. Retired by Milestone 2;
   mitigated by the tier config making the model swappable without code changes.
3. **Context overflow on large surfaces.** An epic's merged diff is the heaviest read in
   the pipeline and nobody has measured it. *Now unmitigated in v1* — the coverage
   statement that would have made a truncated read visible was dropped with the caps.
   Milestone 2's measurements are what make it observable; the mitigation is deferred.
4. **Unsupervised spend.** *Retirement moved from Milestone 2 to Milestone 3.* Through
   Milestones 1–2 the author launches by hand and can interrupt, so the risk does not
   yet exist; it appears the moment the pipeline calls the binary, which is also when
   the numbers to size a cap exist.
5. **A runaway run during hand-run measurement.** New, and the direct cost of running
   unbounded: one bad Milestone-2 invocation can spend real money before it is noticed.
   Accepted deliberately — a cap guessed before any measurement would truncate
   legitimate reviews, which is the worse failure. Mitigated only by live stderr
   diagnostics and a human at the keyboard.
6. **Assumed: leads-not-findings holds.** Everything downstream of a mediocre review is
   safe *only* because the calling skill verifies each lead against the files. This is
   an assumption about the *caller*, and the one thing this project cannot enforce.

Assumed and not re-tested, both inherited from [CONCEPT](/CONCEPT.md): that a
same-family reviewer is worth avoiding, and that the difference in family is what makes
the second opinion valuable.
