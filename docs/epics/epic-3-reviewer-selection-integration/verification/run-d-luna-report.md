## Leads

- **`internal/tools/git_read.go`, `gitReadPathspecs` hunk (~lines 294–310): sensitive-file floor can be bypassed with Git pathspec magic.** `paths` entries are checked lexically by `Scope.Resolve`, then passed directly to Git. With `--allow .`, an entry such as `:(literal).env` is not recognized as `.env` by `deniedByFloor`, but Git interprets it as the sensitive file path. Confirm with a repository containing `.env` and a `git_read` diff/show using `paths: [":(literal).env"]`; the expected result is a floor refusal before Git runs.

- **`internal/cli/review.go`, `runReviewCommand` and `selectReviewer` comments:** the implementation ships nonzero bounds (`MaxTurns: 30`, `MaxElapsed: 20m` in `internal/cli/run.go`), while several comments still say bounds arrive unset and that Epic 3 “sets numbers on the way in.” This is a code/comment inconsistency; confirm whether the stale comments are intended to describe an older contract or whether the shipped caps should differ.

- **`internal/cli/review.go`, `errPromptTooLarge` declaration:** the error remains named `"task prompt exceeds the maximum size"` even though stdin now carries a JSON request object containing both `system` and `task`. This can misclassify or mislead users about what exceeded the limit; confirm the intended diagnostic wording for an oversized request object.

## Checked and looking right

- `paths` is optional and normally replaces, rather than widens, the allowed pathspecs; object/path combinations are refused.
- Family classification is fail-closed for unknown providers, bare reseller IDs, and unknown vendor segments; the exclusion gate runs before credential lookup.
- Explicit `--model` bypasses tier environment/config resolution; tier environment overrides config; config is used last.
- Malformed assignments, absent reviewers, unconfigured providers, and broken credentials have distinct typed paths and tests.
- Request-object decoding rejects unknown fields, trailing values, missing/empty fields, and preserves prompt text verbatim.