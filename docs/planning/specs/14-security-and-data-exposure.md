---
type: Decision
title: "Security & data exposure posture"
description: "What leaves the machine when a review runs, and what is refused."
tags: [decision, specs]
timestamp: 2026-08-09T01:30:00Z
phase: specs
decision: 14
slug: security-and-data-exposure
status: decided
verdict: "The allow-list is the boundary; the deny-list is a floor; subscription OAuth carries no ToS concern outside Anthropic, which is excluded by construction"
decided_via: discussion
depends_on: [repository-confinement]
---

# Question

Scope records no compliance regime — "the tool reads local repositories the user already
has open" ([decision](/scope/15-constraints.md)) — and that is true about *ownership*
while being incomplete about *movement*. The binary's whole function is to send the
contents of a local repository to a third-party model over the network. A private repo
that has never left the machine leaves it the first time this runs.

That is the intended behaviour and not a problem. What deserves an explicit verdict is
where the edge of it sits, because two categories are not what the user meant to send: a
secret checked into a working tree (a `.env`, a private key, a `.netrc`) and the
credentials the tool itself uses.

# Options

- **No filtering: the reviewer reads whatever is in the tree.** Honest — the user's own
  credentials, the user's own repo — and it means one `.env` in a working tree is
  transmitted to a third party the first time a reviewer greps for `API_KEY`.
- **Deny-list by filename, refused with an explanatory error.** Catches the common cases
  cheaply, and it is a heuristic that will miss a secret in a file named something else.
- **Content scanning for secret-shaped strings.** Catches more, costs a scanner, a false
  positive rate, and a maintenance burden — for a single-user tool.

# Recommendation

**Deny-list by filename, fail-closed, with the limitation stated rather than papered
over.**

- `.env*`, `*.pem`, `*.key`, `id_rsa*`, `*.p12`, `*.pfx`, `.npmrc`, `.netrc`,
  `credentials*`, and `.git/` are refused by `read_file`, and filtered out of `list` and
  `search` results ([decision](/specs/10-repository-confinement.md)). A refusal says what
  was refused and why, so the model does not retry.
- The deny-list is a **speed bump, not a boundary**. It does not catch a secret in
  `config.local.yaml`, and it is not claimed to. The honest control is that the user
  points this at repositories they choose, on their own machine, against their own
  account — the deny-list only removes the cases where a reviewer would have exfiltrated
  a credential *while doing its job correctly*.
- Everything else in the tree is fair game and is expected to be transmitted. This is
  worth stating in the README so it is a decision the user made, not one they discover.

**Credentials.** The binary never reads, stores, logs or prints a credential. Resolution
is entirely `kern-link`'s ([decision](/specs/18-auth-and-authorization.md)), and the only
credential-adjacent thing that ever reaches stderr is `AuthResult.Source` — a display
label like `GEMINI_API_KEY` or `OAuth`, never a value. No error message is re-formatted
in a way that could re-expose one.

**Terms of service.** `kern-link` documents a real split: API keys carry no ToS risk,
while subscription OAuth (Claude Pro/Max, ChatGPT Plus/Pro, Copilot) has the library
present itself as a first-party client, which providers can revoke an account over. For
this project the exposure is mostly moot — the reviewer is by construction never
Anthropic — but the general point holds for any subscription-OAuth provider assigned to a
tier, and Milestone 3 runs the binary unsupervised inside a script. **API-key
credentials are the recommended posture for any tier assignment**, and the `tiers`
command shows `AuthResult.Source` so the human can see which kind is actually in play.

**No telemetry, no analytics, no crash reporting, no network egress other than the model
provider's API** ([scope](/scope/16-systems-of-record.md)).

# Verdict

**The deny-list is not the posture; the allow-list is.** Superseded in shape by
[decision 10](/specs/10-repository-confinement.md): a required, repeatable `--allow`
names the subtrees a run may reach, and everything else is refused — including through
`git_read`, whose `show <rev>:<path>` form previously read anything in history and is now
validated the same way. The filename deny-list stays as a floor within allowed subtrees,
which is the honest description of what it always was: a speed bump that catches the case
where a reviewer would exfiltrate a credential *while doing its job correctly*, not a
control that would stop anyone trying.

**What leaves the machine, stated plainly for the README:** the contents of the allowed
subtrees, sent to a third-party model. That is the function, not a leak. What is new is
that the caller now declares the reach per invocation instead of inheriting the whole
repository by default, so the exposure of any given run is visible in the command line
that produced it.

**Terms of service — the user's ruling, recorded as such.** `kern-link`'s documentation
frames subscription-OAuth risk broadly, across Claude Pro/Max, ChatGPT Plus/Pro and
Copilot. The user's judgment is that the concern is Anthropic-specific and the others are
fine in practice. That is their account and their call, and it is also close to moot here:
Anthropic is excluded by construction ([decision 04](/specs/04-model-family-classification.md)),
so the one provider the broad framing most clearly covers is the one this tool can never
select. The earlier recommendation to prefer API-key credentials for tier assignments is
withdrawn: `openai-codex` has no API-key path at all, so following it would have meant not
using the provider the project is validated against.

**One real consequence of the subscription, and it is not money.** An unsupervised or
runaway run burns **quota and rate limits** shared with everything else on that account
([scope risks 4 and 5](/scope/18-risks-and-assumptions.md), amended). The failure mode is
other work starting to fail, not a bill.

**Credentials.** Unchanged and absolute: the binary never reads, stores, refreshes, logs
or prints one. Resolution is wholly `kern-link`'s
([decision 18](/specs/18-auth-and-authorization.md)); the only credential-adjacent value
that reaches stderr is `AuthResult.Source`, a display label such as `OAuth`, never a
secret.

**No telemetry, no analytics, no crash reporting**, and no network egress other than the
model provider's API ([scope](/scope/16-systems-of-record.md)).
