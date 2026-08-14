---
type: Decision
title: "Delivery form"
description: "What does the user receive — CLI, service, library, MCP server?"
tags: [decision, scope]
timestamp: 2026-08-09T01:04:16Z
phase: scope
decision: 03
slug: delivery-form
status: na
verdict: null
decided_via: na
depends_on: [problem-and-job]
---

# Question

What form does the product take?

# Options

Not re-opened. See below.

# Recommendation

Not applicable — settled upstream.

# Verdict

**N/A: already decided in [CONCEPT](/CONCEPT.md) § "What it is" and § "Why this is not
a gateway".** A single CLI binary, invoked over Bash by a skill inside Claude Code,
returning a report as text. Explicitly not a gateway, not a proxy, not a long-running
service, not an MCP server, not a second agent harness.

The *shape* of that CLI — arguments, flags, streams, exit codes — is still open and is
decided separately in [the invocation contract](/scope/09-report-output-contract.md)
and [the reviewer selection vocabulary](/scope/04-reviewer-selection-vocabulary.md).
Only the form itself is settled here.
