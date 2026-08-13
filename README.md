# external-reviewer

A CLI that hands a repository to a second model for review. The model reads through four
read-only tools — `list`, `read_file`, `search`, `git_read` — against an explicitly
allowed slice of the repository, and returns markdown on stdout: leads for a reviewing
skill to verify against the files, not a verdict.

## Validation boundary

The mechanism is fully generic: no provider is named anywhere in the binary except the
family classifier's data table, so all ~35 providers `kern-link` reaches stay usable. But
**v1 is validated against `openai-codex` over OAuth alone** — every other provider is
wired and reachable, not tested. Treat that as a boundary to act on: a provider outside
this one is unverified, whatever the classifier allows.

## What it does not do

- **Read-only, enforced structurally.** The write half of the filesystem API is not in
  the codebase, and CI rejects it if it reappears. The binary writes nothing inside the
  repository under review and produces no output file. The one exception lives outside
  the repository: `kern-link`'s own credential store, rewritten in `~/.pi/agent/` when a
  stored OAuth token is expired.
- No server, no daemon, no state between runs — one round trip per invocation.

## Where the plan lives

- [`docs/planning/SCOPE.md`](docs/planning/SCOPE.md) — what v1 ships.
- [`docs/planning/SPECS.md`](docs/planning/SPECS.md) — the stack and the CLI grammar.
- [`docs/planning/CONVENTIONS.md`](docs/planning/CONVENTIONS.md) — the repo's decided
  standards.
- [`docs/epics/`](docs/epics/) — where implementation currently stands.
