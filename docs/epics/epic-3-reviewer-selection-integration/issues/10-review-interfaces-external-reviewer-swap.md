---
type: Issue
title: "Swap review-interfaces.md from opencode to external-reviewer"
description: "In the lx skills repository: probe and call external-reviewer, remove the opencode branch, and leave the native fallback unchanged and unconditional."
tags: [epic-3]
timestamp: 2026-08-13T00:13:53Z
resource: https://github.com/julienlegoux/skills/issues/37
epic: 3
issue: 10
slug: review-interfaces-external-reviewer-swap
size: M
status: open
gh_issue: 37
depends_on: [5, 6, 8]
---

# Swap review-interfaces.md from opencode to external-reviewer

## Summary

> **⚠ This issue lands in a different repository.** The work is an edit to
> **`julienlegoux/skills`** (locally `D:/Project/skills`), not to `external-reviewer`.
> Per [conventions decision 04](../../../planning/conventions/04-cross-repo-conventions.md),
> **that repository's own authoring contract, branch model and review gate apply**. This
> project's CONVENTIONS.md does not travel — what travels is the acceptance criteria below.
> The GitHub issue for this work is filed on `julienlegoux/skills`; its `.md` lives here
> because it is part of Epic 3.

This is the swap the whole project exists for. `skills/_shared/review-interfaces.md`
§ *Delegating the analysis pass to another model* currently probes `command -v opencode`
and shells out to it; after this PR it probes and calls `external-reviewer`, and the
`opencode` branch is gone ([scope 13](../../../planning/scope/13-skills-integration-scope.md)).
A v1 that left `opencode` in place would have added a second tool beside the one it meant
to remove.

It is also where the product's one unenforceable assumption becomes a property of this
file: with the caller owning the system prompt
([specs 08](../../../planning/specs/08-system-prompt-and-task-assembly.md)), the
**leads-not-findings** instruction stops being a guarantee the binary makes and becomes
text written here.

## Scope

In `julienlegoux/skills`:

- Rewrite § *Delegating the analysis pass to another model* in
  `skills/_shared/review-interfaces.md` around `external-reviewer`:
  - **Capability probe**: `command -v external-reviewer`. Absent → run natively, say
    nothing, record `external review: not available` in the report's Scope. Unchanged in
    spirit from today's rule — a skill that advertises tooling the user never asked for is
    noise.
  - **Invocation**, as the binary actually accepts it after issues 05 and 08:
    `external-reviewer review --allow <path> [--allow <path>…] --tier <light|standard|heavy> <repo>`
    with `{"system": …, "task": …}` on stdin.
  - **`--allow` is required and is the reach of the run** — the reviewing skill declares
    which subtrees the external model may read, and the exposure of any run is visible in
    the command line that produced it.
  - **Tier, not model**: the skill asks for a weight and never names a vendor, which is
    what retires "pick a model from a family other than this session's" as a judgment the
    skill has to make each time. The existing guidance about matching the model to the
    weight of the review becomes guidance about choosing the tier.
  - **The system prompt**, written here in full: who the reviewer is, that its four
    read-only tools are the only way it sees the repository and nothing was pre-summarised
    on its behalf, **leads not findings** (report what looks wrong and where to check it;
    do not assert a defect that has not been read in the file; a review that confirms is a
    legitimate outcome), and the markdown report as the final message.
  - **Exit codes, so the caller can act on them**: `1` — no reviewer was reachable; fall
    back natively and say nothing, which is the silent-fallback case by design. `2` —
    reached and failed; fall back natively and record it in the report's Scope, because a
    reviewer that broke is a fact the reader needs. Both leave stdout empty except a
    partially written report, which announces itself on stderr.
  - **Setup pointer**: `external-reviewer tiers` is what a user runs to see which tiers
    resolve on their machine; the skill never writes config and never prompts for one.
  - **What comes back is still leads, not findings** — verify each against the files before
    it enters the report. That rule is unchanged and stays prominent.
  - Name the tier and the resolved `provider/model` in the report's Scope, so a later reader
    knows who looked.
- Remove every remaining `opencode` reference outside historical log entries:
  `docs/pipeline.md` (the paragraph describing the external pass) is the other live one.
- Append an entry to the skills repo's own `docs/log.md` describing the change and which
  skills the `_shared` edit reaches (`review-epics`, `review-issues`) — editing a shared
  contract changes behaviour for every skill pointing at it, and that repo's convention is
  to say so out loud.

## Out of scope

- Keeping an `opencode` branch for compatibility. v1 has exactly one user and that user
  will have the new binary; two delegation contracts would have to stay correct forever for
  compatibility with nobody.
- Changing the **native fallback**, which stays unchanged and unconditional — it is the
  branch that actually protects users without the capability.
- `review-implementation`, which grades shipped code against yardsticks of its own and is
  explicitly not one of the two skills this contract governs.
- Any change to `external-reviewer` itself. If the invocation this file needs does not
  work, that is a defect in the issue that shipped it, not a patch here.
- Installing or packaging the binary, and any install documentation — named non-goals.

## Acceptance criteria / Definition of done

- [ ] `grep -rn opencode` across `julienlegoux/skills` matches nothing outside historical
      `docs/log.md` entries.
- [ ] `skills/_shared/review-interfaces.md` names the exact command line, including
      `--allow`, `--tier`, the positional repository path, and the JSON request object on
      stdin — copy-pasteable, with no placeholder that isn't obviously one.
- [ ] The system prompt written into the contract carries the leads-not-findings
      instruction and the "a review that confirms is a legitimate outcome" clause; the
      verify-each-lead rule survives verbatim in substance.
- [ ] The probe is `command -v external-reviewer`; the absent case is silent and records
      `external review: not available`.
- [ ] Exit codes `1` and `2` are documented with what the caller does about each.
- [ ] The report template's Scope line names the tier and the resolved `provider/model`.
- [ ] `docs/pipeline.md` no longer describes the external pass in `opencode` terms.
- [ ] `docs/log.md` gains an entry naming the two skills the `_shared` edit reaches.
- [ ] `claude plugin validate .` passes, and the two skills are still discovered
      (`claude plugin details lx@skills-dir`) — that repo's own validation step.
- [ ] The PR follows **the skills repository's** branch model, commit conventions and
      review gate, not this project's.

## Relevant files / areas

In `julienlegoux/skills` (verified against the clone at `D:/Project/skills`):

- `skills/_shared/review-interfaces.md` — §§ *Delegating the analysis pass to another
  model* (lines ~68–102), *The report* (the Scope block)
- `docs/pipeline.md` — the external-pass paragraph (~line 111)
- `docs/log.md` — append an entry
- `skills/review-epics/`, `skills/review-issues/` — the two consumers; read to confirm
  neither restates the delegation rules locally

Governing decisions (in *this* repository):
[scope 13 — Skills-side integration](../../../planning/scope/13-skills-integration-scope.md),
[specs 08 — System prompt & task assembly](../../../planning/specs/08-system-prompt-and-task-assembly.md),
[specs 02 — CLI surface & argv grammar](../../../planning/specs/02-cli-surface-and-argv.md),
[conventions 04 — Cross-repository conventions](../../../planning/conventions/04-cross-repo-conventions.md).

## Dependencies

- Blocked by: [05 — CLI tier flags](/epic-3-reviewer-selection-integration/issues/05-review-tier-flags-and-exit-codes.md)
  (the invocation), [06 — tiers command](/epic-3-reviewer-selection-integration/issues/06-tiers-command.md)
  (the setup pointer), [08 — request object and version](/epic-3-reviewer-selection-integration/issues/08-request-object-and-version.md)
  (the system prompt channel).
- Blocks: [11 — success-criteria verification run](/epic-3-reviewer-selection-integration/issues/11-success-criteria-verification-run.md),
  which runs a real review through this contract.

## PR size note

Markdown only, and well under the ~500-line target. If it grows past ~1000, the extra is
prompt text that belongs in the contract's own reference rather than inline.
