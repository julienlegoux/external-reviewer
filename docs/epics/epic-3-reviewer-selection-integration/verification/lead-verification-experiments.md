# Lead verification — the experiments, verbatim

Raw evidence for the lead-by-lead verdicts in
[VERIFICATION](/epic-3-reviewer-selection-integration/VERIFICATION.md). Every lead that
could be settled by running something was, rather than by reading the code alone. This
file is the transcript of those runs; it is evidence, not a document to maintain.

## 1. The top-level unrecognised config key panics (run C, lead 1) — reproduced

`config/config.go`'s `undecodedWarning` passes `key[:len(key)-1]` to `joinDotted`, which
indexes `parts[0]`. A root-level unrecognised key makes that slice empty.

```
$ cat toplevel-key.toml
typo = "x"
[tiers.standard]
provider = "openai-codex"
model = "gpt-5.6-terra"

$ EXTERNAL_REVIEWER_CONFIG=toplevel-key.toml external-reviewer tiers
done    turns=0  in=0 out=0  $0.0000  0.0s  stop=
panic: runtime error: index out of range [0] with length 0

goroutine 1 [running]:
github.com/julienlegoux/external-reviewer/internal/config.joinDotted(...)
        internal/config/config.go:189
github.com/julienlegoux/external-reviewer/internal/config.undecodedWarning({...})
        internal/config/config.go:185 +0x14b
github.com/julienlegoux/external-reviewer/internal/config.LoadFile({...})
        internal/config/config.go:131 +0x38f
github.com/julienlegoux/external-reviewer/internal/config.Load()
        internal/config/config.go:98 +0x2b
github.com/julienlegoux/external-reviewer/internal/cli.runTiersCommand(...)
        internal/cli/tiers.go:74 +0x465
github.com/julienlegoux/external-reviewer/internal/cli.run(...)
        internal/cli/run.go:180 +0x23a
github.com/julienlegoux/external-reviewer/internal/cli.runProcess(...)
        internal/cli/run.go:100 +0x1ad
main.main()
        main.go:13 +0xa6
exit=2
```

The `done` line is written by the deferred writer with an empty `stop=` before the panic
unwinds, so the failure is not even classified. `review` reaches `config.Load` by the same
path (`selectReviewer`), so it panics identically whenever `--model` is absent.

## 2. The pathspec experiments (run B lead 1, run D lead 1)

A throwaway repository with two commits, a `docs/.env`, a `docs/notes.md` and later a
root `.env`. Each experiment is paired with its **control** — the same read with no
`paths` at all, which is what `gitReadPathspecs` emits when the model names none.

```
A. git diff HEAD~1..HEAD -- 'docs/*'          -> returns the full patch of docs/.env
B. git diff --name-only HEAD~1..HEAD -- docs  -> docs/.env, docs/notes.md   [CONTROL]

D. git diff HEAD~1..HEAD -- ':(literal)docs/.env' -> full patch of docs/.env
E. git show HEAD --stat -- ':(literal)docs/.env'  -> docs/.env | 2 +-
F. git show HEAD --name-only -- '.'               -> docs/.env, docs/notes.md  [CONTROL]

G. git diff HEAD~1..HEAD -- ':(literal).env'  -> full patch of the root .env
H. git diff --name-only HEAD~1..HEAD -- '.'   -> .env                        [CONTROL]
```

What the controls settle:

- **The `paths` parameter widens nothing.** Every file a magic or wildcard pathspec
  reached, the plain granted-subtree pathspec reached too. `gitReadPathspecs` is a strict
  narrowing, exactly as its comment claims.
- **The floor's matcher does not decode pathspec magic.** `deniedByFloor` splits on `/`
  and matches each segment, so `docs/.env` and `:(literal)docs/.env` are both refused —
  the second because its trailing segment is still literally `.env` — while
  `:(literal).env` is one segment that matches no floor pattern and passes.
- **But the floor never guarded diff output in the first place.** Controls B, F and H
  return the same files with no `paths` and no magic. The floor is a check on names the
  model *spells*; the contents git returns for a subtree pathspec were never filtered by
  it, on any subcommand, since Epic 2.

## 3. Line-number checks on the leads that cited one

Read directly, at the revisions the runs saw:

| Cited | Claimed | Actual |
|---|---|---|
| `internal/config/config.go` `joinDotted` | ~189 | 188–194, `parts[0]` at 189 — correct |
| `internal/cli/models.go` `providerCredentialed` | 167–174 | 165–172 — correct within two lines |
| `internal/cli/tiers.go` `resolveTierRow` | 126–143 | 126–148, no `Warn` field set — correct |
| `internal/config/config.go` `Locate` | 82–92 | 80–89 — correct within two lines |
| `internal/cli/review.go` `errPromptTooLarge` | — | 377, `"task prompt exceeds the maximum size"` — correct |
| `internal/tools/git_read.go` `gitReadPathspecs` | ~294–310 (run D) | **200–229** — wrong range; 304–309 is `pathsRefusal` |
| `internal/cli/loop_test.go` `TestRun_UnsetBoundsAreWhatAnOrdinaryInvocationGets` | — | `TestRun_UnsetBounds_AreWhatAnOrdinaryInvocationGets`, 402–423 — **one underscore wrong**, quoted comment text exact |
| `internal/cli/review.go` `runReviewCommand` / `selectReviewer` comments (run D) | those two functions | the stale comment is in **`resolveAndReview`**, 139–143 — wrong function, right file and right claim |

## 4. Precedence and ordering, read rather than run

- `selectReviewer` (`internal/cli/review.go:178–191`) calls `config.Load()` and returns
  its error before `Assigner.Assign` ever reads `EXTERNAL_REVIEWER_TIER_<TIER>`. A valid
  environment assignment therefore does not survive a malformed config file.
- `resolveAndReview` (`internal/cli/review.go:118–125`) calls `ensureModels` before
  `selectReviewer`, so a registry construction failure is classified before a malformed
  `--model` is ever parsed.
- `config.EnvVar`'s comment (`internal/config/config.go:21–24`) and
  [specs 06](../../../planning/specs/06-config-resolution-and-env-overrides.md) line 41
  both say **absolute path**; `Locate` returns any non-empty value unchanged.
