---
type: Decision
title: "Background work"
description: "Whether anything runs asynchronously — jobs, queues, schedulers, daemons."
tags: [decision, specs]
timestamp: 2026-08-09T01:30:00Z
phase: specs
decision: 19
slug: background-work
status: na
verdict: null
decided_via: na
depends_on: [performance-and-scale]
---

# Question

Whether v1 needs asynchronous work — a job queue, a scheduler, a background daemon, a
watcher.

# Options

Not applicable.

# Recommendation

**N/A — the binary is one synchronous run and then it exits.**

A process starts, resolves a model, loops until the model stops, writes a report to
stdout and returns an exit code. There is nothing to defer: the caller is blocked on the
result ([decision](/scope/09-report-output-contract.md)), so work moved off the critical
path would simply be work the caller waits for somewhere else.

Nothing outlives the process. No daemon, no scheduler, no watcher, no retry queue, no
warm cache, no state carried between runs — scope rules out shared context and memory
between runs at the product level ([decision](/scope/12-non-goals.md)), and this is the
technical form of the same decision.

The only concurrency in the program is the single pump goroutine each `kern-link` stream
owns, cancelled through the run's `context.Context`
([decision](/specs/15-performance-and-scale.md)).

# Verdict

N/A. Fully synchronous, single-process, nothing scheduled and nothing deferred.
