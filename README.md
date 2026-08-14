# external-reviewer

A CLI that hands a repository to a second model for review. The model reads through four
read-only tools — `list`, `read_file`, `search`, `git_read` — against an explicitly
allowed slice of the repository, and returns markdown on stdout: leads for a reviewing
skill to verify against the files, not a verdict.

## Install

Go 1.26 or newer, and `git` on `PATH` for the `git_read` tool:

```
go install github.com/julienlegoux/external-reviewer@v0.1.0-beta.1
```

`@latest` resolves to the newest tag, which for now is the beta above — there is no
stable release yet.

## Quickstart

**1. Give `kern-link` a credential.** This binary reads none, stores none and has no
`login` command: provider credentials are resolved wholly by `kern-link`, from env vars
or its store at `~/.pi/agent/auth.json`, populated by its own `pi-ai login`. Use
`openai-codex` unless you have read the validation boundary below.

**2. Assign the tiers.** One hand-written, never-committed TOML at
`%AppData%\external-reviewer\config.toml` on Windows,
`$XDG_CONFIG_HOME/external-reviewer/config.toml` (or `~/.config/…`) elsewhere;
`EXTERNAL_REVIEWER_CONFIG` overrides the location with an absolute path.

```toml
[tiers.standard]
provider = "openai-codex"
model = "gpt-5.5"
```

**3. Check what this machine can actually reach**, before spending anything on a run:

```
external-reviewer tiers
external-reviewer models
```

**4. Review.** The system prompt and the task come from the caller, as a JSON request
object on stdin; `--allow` grants the subtrees the reviewer may read, and nothing outside
them is readable:

```
echo '{"system":"You are reviewing Go.","task":"Find correctness bugs in the tool layer."}' \
  | external-reviewer review --allow internal/tools --allow docs/planning .
```

`external-reviewer help` prints the whole grammar. The report is on stdout; turn-by-turn
cost and diagnostics are on stderr.

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
