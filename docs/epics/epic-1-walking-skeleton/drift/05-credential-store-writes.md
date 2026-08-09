---
type: Drift record
title: "The binary writes no files, except the ones kern-link's credential store writes"
description: "SPECS states that the binary writes no files; a real run modifies ~/.pi/agent/auth.json when kern-link rotates an OAuth token, and a read of an existing store creates an auth.json.lock sidecar beside it."
tags: [epic-1, drift]
timestamp: 2026-08-09T13:10:00Z
epic: 1
issue: 05
---

# The binary writes no files, except the ones kern-link's credential store writes

## Decided

[SCOPE § Report output contract](../../../planning/scope/09-report-output-contract.md) and
[SPECS § Interfaces](../../../planning/SPECS.md) state the guarantee without qualification:
**the binary writes no files**, so the calling skill keeps sole ownership of the report.
[CONVENTIONS § Code style & formatting](../../../planning/CONVENTIONS.md) enforces it
structurally — the write half of the filesystem API is not in the codebase, and
`.golangci.yml`'s `forbidigo` block rejects it. Issue 05's acceptance criteria restate it
as behaviour: *"the binary creates, modifies or deletes no file anywhere"*.

## Actual

Both halves of the enforcement hold, and the guarantee is still narrower than stated. No
write-API call appears in this repository's non-test source, and a full successful review
leaves the reviewed repository byte-identical (asserted by
`TestRun_SuccessfulReview_TouchesNoFile`). But the dependency writes on the binary's
behalf, outside the repository, in `~/.pi/agent/`:

- **`auth.json` is rewritten** when the stored OAuth credential is expired. Resolution
  calls `models.GetAuth`, which reaches `ai.ResolveProviderAuth` →
  `resolveStoredOAuth` → `credentials.Modify(...)` → `FileCredentialStore.save`, an
  `os.WriteFile` plus an `os.Chmod` on the store. The rotated token has to be persisted or
  every process would refresh separately, so this is the dependency working correctly.
- **`auth.json.lock` is created** whenever the store is read and the store file exists.
  `FileCredentialStore.Read` skips locking only when `auth.json` is absent
  (`ai/auth/filestore.go:123`); otherwise it takes `flock.New(s.path + ".lock")`, and
  `flock` creates the lock file if it is not there. `withLock` also calls
  `os.MkdirAll(filepath.Dir(s.path), 0o700)` first.

Neither is reachable from the test suite, which injects an offline registry over an
in-memory credential store — which is why this only surfaced in the hand run.

## Because

Observed on the manual verification run recorded in PR #20, on a machine whose
`openai-codex` credential is OAuth:

```
$ external-reviewer review --prompt "Reply with a single markdown heading line saying OK and nothing else." <tiny repo>
# OK
model   openai-codex/gpt-5.5  auth=OAuth
turn 1  tools=0  in=29 out=32  $0.0011  4.0s
done    turns=1  in=29 out=32  $0.0011  4.7s  stop=ok
exit 0

repository tree before/after:  identical
~/.pi/agent before/after:      auth.json 2621 bytes -> 2305 bytes (mtime moved)
```

`auth.json.lock` was already present on this machine from another kern-link tool, so its
creation was not observed directly; the code path above is unconditional for a store that
exists.

There is no version of this that keeps the literal wording true while the binary still
authenticates. The alternative would be for this binary to read the credential file itself
and never hand kern-link a store — which contradicts
[SPECS § Data, configuration & auth](../../../planning/SPECS.md) ("this binary reads no
credential, stores none and refreshes none") and would be strictly worse for security.

## Alternatives tried

- **A read-only credential store wrapper** whose `Modify` is a no-op — rejected: it does
  not stop the write so much as break the refresh, so an expired token would be re-refreshed
  on every invocation and, on a provider that rotates refresh tokens, invalidated.
- **Asserting the no-files property over `$HOME` as well as the repository** — rejected as
  a test that would fail for a correct binary. The assertion that carries the product
  guarantee is the one about the *reviewed repository*, and that one passes.

## Revisit when

The wording is triaged at epic close. The likely disposition is `accepted` with SCOPE and
SPECS amended to say what is actually guaranteed — **the binary writes nothing inside the
repository under review, and produces no output file; credential storage belongs to
kern-link and lives in `~/.pi/agent/`**. Epic 2's `os.Root` confinement makes the first
half structural for reads as well as writes, at which point the amended sentence is exactly
what the code enforces.

## Evidence

- The hand run above, PR #20, branch `issue-8-single-turn-model-call`.
- `ai/auth/filestore.go` (`Read`, `withLock`, `save`) and `ai/resolve.go`
  (`resolveStoredOAuth`) at `github.com/julienlegoux/kern-link v0.1.1`.
- [Issue 05](../issues/05-single-turn-model-call.md).
