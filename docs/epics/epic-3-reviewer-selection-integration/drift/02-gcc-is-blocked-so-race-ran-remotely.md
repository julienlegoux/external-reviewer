---
type: Drift record
title: "The native -race command did not run: gcc itself is now blocked, not the test binary"
description: "CONVENTIONS says `go test -race -ldflags=-s ./... -count=1` runs natively on the development machine; on 2026-08-13 every package failed in cgo, because gcc.exe now dies with STATUS_FAIL_FAST_EXCEPTION before compiling anything — so -race ran on the remote instead."
tags: [epic-3, drift]
timestamp: 2026-08-13T10:35:00Z
epic: 3
issue: 02
---

# The native -race command did not run: gcc itself is now blocked, not the test binary

## Decided

[CONVENTIONS § Testing](../../../planning/CONVENTIONS.md) states the command without
qualification — *"The whole command runs natively on the Windows development machine,
`-race` included, with one flag: `go test -race -ldflags=-s ./... -count=1`"* — and closes
the point explicitly: *"cgo works: a WinLibs MinGW-W64 UCRT toolchain compiles and its
unsigned output executes, so `-race` needs nothing remote."*
[DRIFT § 09, 10, 11](../../../planning/DRIFT.md) records the same as resolved on
2026-08-12, and this issue's instructions named that command for the red-green loop.

## Actual

The native command builds nothing at all on 2026-08-13. Every package in the repository,
touched or not:

```
runtime/cgo: C:\Program Files\Go\pkg\tool\windows_amd64\cgo.exe: exit status 2
FAIL	github.com/julienlegoux/external-reviewer/internal/cli [build failed]
… (every package)
```

So this issue's `-race` signal came from `scripts/test-remote.sh` with a distinct
`EXTERNAL_REVIEWER_TEST_DIR`, and the fast local loop ran `go test -ldflags=-s` without the
detector.

## Because

The block has moved one layer down: it is not the freshly linked test binary this time, it
is **`gcc.exe` itself**, and it dies before emitting a single diagnostic.

```
$ gcc scratch\t.c -o scratch\t.exe        # int main(void){return 0;}
gcc exit=-1058471934                       # 0xC0000602 STATUS_FAIL_FAST_EXCEPTION
```

`-1058471934` is `0xC0000602`, the fail-fast status Windows Application Control raises on a
refused image. `cgo.exe` inherits it as a silent `exit status 2`, which is why the failure
reads as a Go toolchain error rather than as a policy refusal. Two consequences follow, and
both were measured:

- The refusal is on the C toolchain, so **no re-roll of the Go build flags helps** —
  `-ldflags=-s` changes the *test* binary's hash, and the block is on a binary the Go build
  only invokes.
- Anything that does not need cgo is unaffected: `go build -race -o /dev/null
  ./internal/family` succeeded from the already-cached `runtime/cgo`, and
  `go test -ldflags=-s ./... -count=1` is green natively, whole suite. Only the path that
  has to *compile* `runtime/cgo` fails.

The hash rule from the Epic 0 entry still explains it — gcc is an unsigned WinLibs build
and a given image gets one permanent verdict — but the resolution recorded there
(*"cgo is not blocked either"*) was true of one gcc image and is not true of the one on the
machine today.

## Alternatives tried

- **Re-rolling with build flags**, the documented response: `-ldflags=-s` present and
  absent, both fail identically, because the block is upstream of the linked test binary.
- **Running `cgo.exe`'s exact command by hand** (recovered with `go test -race -x -work`):
  same silent exit 2, confirming the failure is in the child process and not in how the
  Go driver invokes it.
- **A trivial C file through gcc directly**, which is what isolated the fail-fast status
  and moved the diagnosis off the Go toolchain entirely.
- **`scripts/test-remote.sh`** — worked first try, whole suite green with `-race`, which is
  what this record exists to say was used.

## Revisit when

`gcc --version` runs at all on the development machine — the current image draws a
permanent refusal, so this changes when the toolchain is reinstalled or updated to an image
the policy has not judged, or when the machine's policy changes for unrelated reasons.
Re-verify with the one-line C file above before trusting `-race` locally again; a green
`go build -race` proves nothing while `runtime/cgo` can come from the build cache.

Worth carrying separately: § Testing's cgo sentence should be softened from a settled fact
to a check, since it has now been false twice for two different reasons, and each time the
symptom first read as a code failure.

## Evidence

- The gcc fail-fast run and the `cgo.exe` reproduction above, both from this issue's
  session, quoted verbatim with exit codes.
- The remote run that produced this PR's `-race` numbers:
  `EXTERNAL_REVIEWER_TEST_DIR=ci/external-reviewer-epic3-issue02 scripts/test-remote.sh
  ./... -count=1` — eight packages, all `ok`.
- [CONVENTIONS § Testing](../../../planning/CONVENTIONS.md),
  [DRIFT § 09, 10, 11](../../../planning/DRIFT.md), and Epic 2's
  [drift record 03](/epic-2-read-only-agentic-loop/drift/03-native-go-test-blocked-again.md),
  which this recurrence sits beside rather than replaces.
