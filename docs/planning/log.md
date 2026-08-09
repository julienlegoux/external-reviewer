# Log

## 2026-08-09

* **Update**: ran `define-scope` and wrote [SCOPE](/SCOPE.md) from an 18-item
  [decision ledger](/scope/index.md) — 16 decided, 2 marked N/A because
  [CONCEPT](/CONCEPT.md) already settled them (the problem, and the CLI delivery form).
  Premise confirmed before enumerating: a Go CLI on `kern-link`, author-first,
  replacing the pipeline's `opencode` dependency.

* **v1 runs unbounded, and measures instead of capping.** The recommendation was three
  hard caps (turns, cost, wall clock) plus a coverage statement, on the argument that
  the binary spends the user's own credit unsupervised. The user rejected it on the
  ground that nobody knows yet how many turns or how long a review takes, so any cap
  chosen now is a guess — and a guess set too low silently truncates legitimate
  reviews. The counter-argument turned out not to apply yet: through Milestones 1–2 the
  author launches by hand and can interrupt; the unsupervised case only arrives at
  Milestone 3, by which point the numbers exist. So Milestone 2 ships instrumentation
  (turns, cost, elapsed time on stderr) and Milestone 3 inherits caps sized from it.
  Rippled into six decisions; the honest cost is recorded as two risks — context
  overflow is now unmitigated in v1, and a runaway hand-run is accepted deliberately.

* **The skills-side edit is inside v1.** Milestone 3 removes the `opencode` branch from
  `_shared/review-interfaces.md` rather than adding a second delegation path beside it:
  keeping both would maintain two contracts forever for compatibility with nobody, since
  v1 has exactly one user and that user will have the new binary. The native fallback
  stays unchanged. Consequence flagged for `split-epics`: Milestone 3's issues land in
  the **lx skills repository**, not this one.

* **Weight tiers, not roles.** The concept fixed the split (skills ship the vocabulary,
  the machine holds the assignment) but not the vocabulary itself. Chose `light` /
  `standard` / `heavy`, reusing the grading `implement-epic` already applies — a role
  taxonomy like "security-reviewer" would push a vendor-shaped judgment back into the
  shipped artifact, which is exactly what the concept moved onto the user's machine.

* **The binary enforces the family exclusion, not the caller.** The concept settled the
  principle; the open question was the mechanism. A claim that depends on every caller
  remembering to check is not a claim, and enforcing centrally codes the
  `amazon-bedrock` trap once, next to the catalogue where family metadata lives.

* **Creation**: established this bundle and wrote [CONCEPT](/CONCEPT.md) from a
  working session that started out designing an LLM gateway and ended somewhere else.

* **The product is a called binary, not a gateway.** The starting memo priced three
  topologies and recommended the cheapest; the session set out to build the most
  expensive one anyway, "as simply as possible". Simplifying it far enough inverted
  the answer. A gateway's real work is relaying a tool loop between Claude Code and a
  foreign model, translating tool calls in both directions mid-stream — and a reviewer
  does not need that relayed, because the loop can run inside the binary and speak no
  Anthropic protocol at all. The project was renamed accordingly (see below). The
  gateway survives as a named non-goal with one trigger: foreign models used as real
  subagents in a session, implementers rather than reviewers.

* **The reviewer reads the repository itself.** The cheaper design has the calling
  session assemble the evidence and paste it into one prompt. Rejected: the party under
  review would decide what the reviewer is allowed to see, copying its own blind spots
  into the selection, and would truncate silently once the evidence outgrew the
  context. A read-only loop inside the binary costs little because `kern-link` already
  carries the tool-call protocol.

* **"External" is about model family, not provider.** Verified against `kern-link`'s
  own catalogue: the `amazon-bedrock` provider serves both Amazon and Anthropic models,
  so selecting on "provider is not Anthropic" would send Claude's work to Claude. The
  distinction is dormant while only one foreign family is configured, but the concept
  holds it explicitly so a later configuration cannot lose it.

* **Model judgment is written by the user, on the user's machine.** Considered and
  rejected: shipping a maintained list of models and their strengths inside the skills.
  It rots on a two-month cycle, recommends models the reader may have no account for,
  and keeps its confident tone after it stops being true. What ships is the procedure
  and the vocabulary; what stays local is the assignment. The vocabulary is not new —
  `implement-epic` already grades work into tiers and picks horsepower from that
  grading.

* **Catalogue staleness lands on `kern-link`.** It is versioned, tracks an upstream and
  has a sync procedure, which makes it a better home for facts that expire than a file
  in each user's repository. Its embedded catalogue was behind the models the user
  intends to use at the time of writing; the fix is upstream, not a local override.

* **Named `external-reviewer`.** The working name was `claude-gateway`, kept until a
  check found three problems: `claude gateway` is a first-party Claude Code subcommand
  (the enterprise auth/telemetry gateway), the repository name is already taken twice
  on GitHub, and every comparable project in that crowded category routes Claude Code
  to other providers — precisely what this one decided not to do. The name would have
  advertised the rejected design.

* **Parked, deliberately**: how a stale catalogue signals that it needs revisiting;
  whether `review-implementation` joins and whether an epic's merged diff is a
  comfortable surface; how a report states its own coverage; the installation path.
  Recorded in the concept's open questions rather than answered.
