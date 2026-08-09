---
type: Decision
title: "Performance & scale targets"
description: "The honest expected load, so nothing gets architected for a scale that will never arrive."
tags: [decision, specs]
timestamp: 2026-08-09T01:30:00Z
phase: specs
decision: 15
slug: performance-and-scale
status: decided
verdict: "Single run, sequential tools, no caching, no concurrency; tokens are the only budgeted resource"
decided_via: triage
depends_on: [agent-loop-mechanics]
---

# Question

Worth an explicit verdict precisely because the honest answer is "almost none", and an
unstated "almost none" is how a single-user CLI acquires a worker pool, a cache layer and
a concurrency knob nobody needed.

The real numbers: one machine ([scope](/scope/02-target-users.md)), one review at a time,
invoked by a skill a handful of times a day at most. The wall clock of a run is model
latency and nothing else — a `search` over a repository is microseconds against a
round trip measured in seconds.

# Options

- **Explicitly single-run, sequential, no caching.** Matches reality; leaves a real
  bottleneck unaddressed if one appears.
- **Parallel tool execution within a turn.** A model can emit several tool calls per
  turn, and executing them concurrently would shave a fraction of a second off a run
  measured in minutes.
- **Cache file reads across turns.** Saves re-reading a file the model asks for twice, and
  introduces staleness in a tool whose entire value is reporting what the file actually
  says.

# Recommendation

**Single run, sequential tools, no caching, no concurrency beyond the one goroutine
`kern-link`'s stream already owns.**

Stated targets, so that "is this fast enough?" has an answer:

- **Throughput: one review per process, one process at a time.** No daemon, no server, no
  queue, no reuse between runs.
- **Latency: unbudgeted.** A review takes as long as the model takes. Nothing in this
  binary has a performance target, because nothing in it is measurably on the critical
  path.
- **The only resource that matters is tokens** — cost and context window, not CPU or
  memory. Every optimisation worth making in v1 is a token optimisation: the result caps
  and truncation markers of the tool contract
  ([decision](/specs/09-read-only-tool-contract.md)), and the binary-file and size skips
  in `search` ([decision](/specs/11-tool-implementation-strategy.md)).
- **Memory: the whole conversation is held in RAM as `[]ai.Message`** for the run's
  duration, and no attempt is made to compact, summarise or trim it. On a large surface
  that is the mechanism by which context overflow happens — which scope names as risk 3
  and leaves **unmitigated in v1**, observable through Milestone 2's measurements
  ([decision](/scope/18-risks-and-assumptions.md)). Recording it here as a deliberate
  non-optimisation rather than an oversight.

Tool results are streamed to their caps rather than being read whole and then trimmed
(a `read_file` on a 400 MB file must not allocate 400 MB), which is the one performance
property that is a correctness property.

# Verdict

Accepted at triage as recommended, including the explicit record that context overflow (scope risk 3) is left unmitigated in v1 as a deliberate non-optimisation, and that streaming tool results to their caps is a correctness property rather than a performance one.

**Refreshed after [decision 09](/specs/09-read-only-tool-contract.md).** One sentence
above is now wrong: *"the only resource that matters is tokens — cost and context
window"*. The credentialed provider is a **subscription** (`openai-codex` OAuth), so
tokens are not billed by the token and per-run cost is flat.

What remains binding, in order:

1. **The context window.** A hard limit no billing model changes, and the mechanism
   behind [scope risk 3](/scope/18-risks-and-assumptions.md), which v1 leaves
   unmitigated. This is now the *only* reason result caps exist.
2. **Quota and rate limits.** The subscription's real scarce resource. It is also
   shared with whatever else the account does, so a runaway review degrades more than
   itself.
3. **Wall-clock time.** What success criterion 3 actually turns on now
   ([scope](/scope/14-success-criteria.md)).

Cost is still computed and reported per turn — it is the cleanest available proxy for
volume, and it becomes real money the day a tier is assigned an API-key provider. It is
simply not a budget any more.
