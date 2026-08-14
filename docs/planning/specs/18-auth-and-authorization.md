---
type: Decision
title: "Auth & authorization"
description: "Whether the binary has any notion of identity or permission of its own."
tags: [decision, specs]
timestamp: 2026-08-09T01:30:00Z
phase: specs
decision: 18
slug: auth-and-authorization
status: na
verdict: null
decided_via: na
depends_on: [config-resolution-and-env-overrides]
---

# Question

Whether v1 has authentication or authorization of its own — accounts, sessions, roles,
permissions.

# Options

Not applicable.

# Recommendation

**N/A — there is no identity in this system, and provider credentials are entirely
`kern-link`'s.**

The binary is a local process invoked by the user (or by a skill running as the user) on
their own machine. There is no server, no multi-tenancy, no second principal, and nothing
to authorize: the only actor is the person who ran the command, and the only permission
question — what the reviewer may read — is answered by confinement rather than by
identity ([decision](/specs/10-repository-confinement.md)).

Provider credentials are resolved wholly by `kern-link`: per-provider env vars, its
cross-process-locked credential store at `~/.pi/agent/auth.json`, and OAuth logins driven
by its own `pi-ai login` CLI. Scope was explicit that Milestone 1 uses that resolution
with "no credential handling of its own" ([SCOPE](/SCOPE.md)), and nothing in later
milestones changes it. This binary reads no credential, stores none, refreshes none, and
has no `login` command — if a provider is not configured, the fix is `pi-ai login` or an
env var, and the answer here is exit `1`
([decision](/specs/13-error-handling-and-failure-classification.md)).

The one credential-adjacent surface is read-only reporting: `models.GetAuth` yields an
`AuthResult.Source` display label (`GEMINI_API_KEY`, `OAuth`, `stored credential`) that
the `tiers` command shows, so a human can see how a tier authenticates without ever
seeing a secret ([decision](/specs/14-security-and-data-exposure.md)).

# Verdict

N/A. No identity, no authorization; credentials delegated to `kern-link` in full.
