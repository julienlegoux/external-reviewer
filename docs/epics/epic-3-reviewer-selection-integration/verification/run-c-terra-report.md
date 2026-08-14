## Leads

- **Top-level unknown TOML keys can panic instead of warning.**  
  **Location:** `internal/config/config.go`, new `LoadFile` hunk: `undecodedWarning` calls `joinDotted(key[:len(key)-1])`; `joinDotted` immediately indexes `parts[0]`. For an unrecognised root key such as `typo = "x"`, TOML metadata should supply a one-element key, making `section` empty and causing an index-out-of-range panic. This contradicts the loader’s stated “every unrecognised key” warning behavior and could bypass the normal diagnostic/done-line path. Confirm by loading a config containing a top-level unknown key.

- **A malformed explicit `--model` can be classified as a broken machine rather than usage.**  
  **Location:** `internal/cli/review.go`, `resolveAndReview` hunk calls `ensureModels` before `selectReviewer`; `selectReviewer` is where `Assigner.Assign` parses `req.Model`. Thus `review --model bad-value ...` can fail while constructing the real registry (for example, an unreadable credential-store location) before the malformed flag is detected. The later `malformedFlagAssignment` branch correctly labels flag-origin errors as `usage`, but it is never reached in this ordering. Confirm with a malformed `--model` and an environment that makes `reviewer.DefaultModels()` fail; the expected classification should remain `stop=usage`.

- **Environment precedence may not shield a review from an invalid lower-priority config file.**  
  **Location:** `internal/cli/review.go`, `selectReviewer` hunk loads and parses config unconditionally whenever `--model` is absent, before `Assigner.Assign` consults `EXTERNAL_REVIEWER_TIER_<TIER>`. A valid per-tier environment override therefore still fails on malformed TOML, despite `Assigner.Assign` documenting environment as higher precedence than config. This may be intended if configuration validity is global, but it is inconsistent with the stated layer ordering and with the special case that `--model` bypasses config parsing. Confirm the intended contract for a valid environment assignment plus malformed config; if overrides are meant to be decisive, config loading/parsing needs to be deferred or ignored in that case.

- **The config-location contract says the override is absolute, but accepts relative paths.**  
  **Location:** `internal/config/config.go`, `Locate` hunk. Its comment specifies `EXTERNAL_REVIEWER_CONFIG` as “an absolute path,” but it returns any non-empty value without an absolute-path check. This broadens the documented configuration rule and makes behavior depend on the caller’s working directory. Confirm whether the absolute-path requirement is normative; if so, reject relative values.

- **Stale test commentary says production ships no bounds after production began shipping them.**  
  **Location:** `internal/cli/loop_test.go`, `TestRun_UnsetBoundsAreWhatAnOrdinaryInvocationGets` hunk; compare `internal/cli/run.go`’s new `shippedBounds`. The test invokes the deliberately unbounded `RunWithModelsForTest` seam, so its assertion can be valid, but its name/comments say “what this epic actually ships” and “no values,” which is no longer true. This is a documentation/test-name inconsistency likely to mislead future changes.

## Checked and looked sound

- `git_read.paths` is schema-declared, resolved before git invocation, and refused for out-of-grant and sensitive paths; object/pathspec mixing is explicitly rejected.
- Family resolution is fail-closed at the resolver gate: unknown families produce `ReasonFamilyUnknown` and do not reach credential resolution.
- The assignment implementation itself gives explicit model, then per-tier environment, then config the expected precedence.
- The request-object decoder rejects unknown fields, trailing values, and empty system/task fields.