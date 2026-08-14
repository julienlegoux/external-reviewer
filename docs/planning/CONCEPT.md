---
type: Product Concept
title: "External Reviewer — Concept"
description: "A CLI that runs a read-only review of a repository on a model from outside the Anthropic family and returns a report to the Claude Code session that called it — without a second agent harness and without a proxy in front of the session."
tags: [planning, concept]
timestamp: 2026-08-09T00:18:52Z
status: draft
---

# External Reviewer

## The problem

The lx skills pipeline already wants a reviewer that did not write the thing under
review, and already says why: `review-epics` and `review-issues` hand their analysis
pass to a model from a different family *"since that difference is the entire point"*.
A self-review is weakest exactly where it re-reads its own reasoning.

Today that capability arrives through `opencode` on `PATH` — a second agent harness,
separately installed, separately authenticated, run read-only against the repo. The
capability is right. The dependency is what it costs: every user who wants a second
opinion installs a whole other agent.

## What it is

A single CLI binary. A skill running inside Claude Code invokes it over Bash, hands it
a task and a repository, and receives a report as text. The binary runs its own agent
loop against a model outside the Anthropic family, reads what it needs from the repo,
and stops.

It is not part of the calling session. It shares no context, holds no conversation,
opens no pull request, and writes nothing into the repository. The calling skill keeps
sole ownership of the report file — which is already the pipeline's rule: *the one file
a review writes is its own report*.

## Why this is not a gateway

The obvious alternative is to interpose: point `ANTHROPIC_BASE_URL` at a router that
forwards Claude models to Anthropic and sends everything else to its provider, so that
a subagent declared with a foreign model simply works. That shape is deliberately not
this product.

Interposing puts a locally maintained proxy in the path of every request of every
session, including the sessions used to repair it, and Anthropic does not support
routing Claude Code to non-Claude models through a gateway — it can break on any
release. But the decisive argument is narrower: a gateway exists to relay a tool loop
between Claude Code and a foreign model, translating tool calls in both directions
mid-stream. **A reviewer does not need that loop relayed, because the loop can live
inside the binary.** Nothing pretends to be Anthropic, so there is no protocol to
translate.

The gateway becomes necessary under exactly one condition: foreign models used as
genuine subagents inside a session — implementers writing code rather than reviewers
reading it. Until that is wanted, interposing on every request buys nothing this
product needs.

## "External" means the family, not the model

The requirement is not a better model or a cheaper one. It is a model that did not
write the thing, and that is a property of the *pairing*, not of any model in
isolation.

Family is not the same as provider, and the difference is a trap rather than a
subtlety: `amazon-bedrock` serves both Amazon's own models and Anthropic's. Selecting a
reviewer by "provider is not Anthropic" therefore sends Claude's work to Claude for
review. Whatever chooses the reviewer chooses on family.

## What it reads, and what it never writes

The reviewer reads the repository itself, through read-only tools, and that is the
reason the loop lives inside the binary rather than being replaced by pre-assembled
evidence.

The alternative — the calling session gathers the diff, the issue files and the
standards, and pastes them into a single prompt — fails on its own premise: the party
under review decides what the reviewer is allowed to see, so its blind spots are copied
into the selection. It also truncates silently when the evidence outgrows the context,
and a report that saw sixty percent of the material reads exactly as confident as one
that saw all of it.

Reading is also strictly bounded. The reviewer never edits, never commits, never touches
GitHub state. A reviewer that repairs what it reviews destroys the evidence that
anything was wrong.

## Where the judgment about models lives

Which models exist, what they cost, and what they can technically do are facts, and an
SDK can supply them. Which model suits which job is a judgment, and nobody's SDK has
it. Somebody has to write it down.

That judgment is written **by the user, on the user's machine, against the models their
own credentials actually reach** — not shipped inside the skills. A distributed artifact
that names vendors and rates them starts rotting the day it is written, recommends
models the reader has no account for, and makes its claims with full confidence long
after they stopped being true.

So the split is: the skills ship the **procedure** for building that catalogue and the
**vocabulary** it is expressed in; the machine holds the **assignment** of models to
that vocabulary. A skill asks for a kind of reviewer and never names a vendor; an entry
with nothing assigned to it simply has no reviewer, and the review runs natively.

The vocabulary does not have to be invented. `implement-epic` already grades work into
tiers and picks horsepower accordingly; the same grading describes the weight of a
review — a handful of issue files against one epic is not the same read as an entire
epic's merged diff.

## Built on kern-link

`kern-link` already carries the parts that are expensive and uninteresting to rebuild:
provider coverage, credential resolution across API keys and OAuth, streaming, the
tool-call protocol as first-class types, per-call cost accounting and token estimation.

What this product adds on top is small and specific: the multi-turn loop (kern-link's
own example covers a single round trip), the read-only tools, the assembly of the task
prompt, and the CLI surface.

It also decides where model-catalogue staleness lives. kern-link's catalogue is
versioned, tracks an upstream, and has a sync procedure — a far better home for a fact
that expires every few months than a file in each user's repository.

## How it fails

Every failure mode ends in the same place: the review runs natively and says so.

- The binary is not installed, or no foreign credentials resolve → capability absent,
  review runs natively, and the report records that no external pass happened. This is
  already the pipeline's rule, and it exists so that a skill never advertises tooling
  the user did not ask for.
- The run errors, hangs, or returns something unusable → same fallback. A review is
  never blocked on it.
- The run succeeds but reasons badly → what comes back is treated as **leads, not
  findings**, verified against the files before anything enters the report.

That last rule is what makes this safe to build imperfectly. The worst outcome of a
flawed reviewer is a mediocre set of leads that were going to be checked anyway — not a
wrong finding published, and never a modified repository.

## Open questions

Left deliberately unresolved, and inherited rather than rediscovered:

- **Signalling a stale catalogue.** When the assignments a user made no longer match
  what is reachable or sensible, something should tell them to revisit. Parked by
  explicit decision; no mechanism chosen.
- **Reviewing code, not just planning artifacts.** `review-implementation` does not
  delegate externally today. An agentic reader removes the objection in principle, but
  an entire epic's merged diff is the heaviest surface in the pipeline and nobody has
  measured whether it is comfortable.
- **What "seen the whole thing" means.** A reader that chooses its own path can also
  stop early. How a report states its own coverage — so a gap is visible to the reader
  — is unsettled.
- **The installation path.** Getting the binary, the credentials and the catalogue in
  place is a first-run experience nobody has designed.
- **The gateway's trigger.** Named above as a non-goal with one condition attached.
  Whether that condition is ever wanted is open.
