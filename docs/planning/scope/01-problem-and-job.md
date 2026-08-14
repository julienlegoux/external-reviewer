---
type: Decision
title: "Problem & job-to-be-done"
description: "What pain does External Reviewer solve, and for what job?"
tags: [decision, scope]
timestamp: 2026-08-09T01:04:16Z
phase: scope
decision: 01
slug: problem-and-job
status: na
verdict: null
decided_via: na
depends_on: []
---

# Question

What pain does this project solve, and for what job?

# Options

Not re-opened. See below.

# Recommendation

Not applicable — settled upstream.

# Verdict

**N/A: already decided in [CONCEPT](/CONCEPT.md) § "The problem".** The lx pipeline
wants a reviewer that did not write the thing under review, and today buys that with a
whole second agent harness (`opencode`) that every user must install and authenticate
separately. The job is: *give a Claude Code session a second opinion from outside the
Anthropic family, without a second harness and without a proxy in front of the session.*

Re-deciding a validated concept spends the user's attention twice and risks
contradicting the document downstream skills read. Reclassify to `open` only if the
concept itself is being revisited.
