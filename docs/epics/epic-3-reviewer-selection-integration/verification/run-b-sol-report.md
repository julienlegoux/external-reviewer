# Epic 3 merged-diff review

## Leads

### 1. `git_read.paths` appears to let Git expand a pathspec across the sensitive-file floor

**Location:** `internal/tools/git_read.go`, added `gitReadPathspecs` hunk around `@@ -175,8 +196,39 @@`

Each `paths` entry is passed through `Scope.Resolve`, but the resulting string is then handed to Git as a pathspec. `Resolve` validates the literal string; it does not expand Git wildcards or pathspec magic. For example, with `--allow docs`, `paths: ["docs/*"]` passes because the literal lies under `docs` and contains no denied segment. Git can subsequently expand it to `docs/.env`, `docs/credentials.json`, or `docs/private.key` and include their historical contents in a diff.

The new comment claims wildcards and pathspec magic are safe because they pass through `Scope.Resolve`; that does not appear sufficient once Git interprets them afterward. The tests only exercise literal `.env` and `.git/config` entries.

**What would confirm it:** Create a commit containing `docs/.env`, grant `docs`, and call `git_read diff` or `show` with `paths: ["docs/*"]`. If the denied file’s patch or name is returned, the new parameter widens past the floor. Literal-only pathspecs, or explicit rejection/escaping of Git pathspec syntax, would settle the intended rule.

---

### 2. A higher-precedence tier environment override can still be defeated by a malformed lower-precedence config file

**Location:** `internal/cli/review.go`, `selectReviewer` hunk added after `resolveAndReview`

When `--model` is absent, `selectReviewer` calls `config.Load()` before `Assigner.Assign`. Thus a malformed TOML file aborts the review even when `EXTERNAL_REVIEWER_TIER_STANDARD` contains a complete valid assignment that should win according to the documented order:

`--model` → per-tier environment variable → config-path environment variable → config file.

The explicit `--model` path correctly avoids loading the file, but the per-tier environment layer does not. The same ordering risk exists in `internal/cli/tiers.go`, where the entire config is loaded before any environment overrides are resolved.

**What would confirm it:** Set a valid `EXTERNAL_REVIEWER_TIER_STANDARD`, point `EXTERNAL_REVIEWER_CONFIG` at malformed TOML, and request `standard`. If the intended meaning of precedence is that a winning layer bypasses invalid lower layers, the review should resolve from the environment. A test specifying whether malformed lower-precedence configuration remains globally fatal would settle the ambiguity.

---

### 3. `models` collapses broken credentials, cancellation, and other auth failures into “uncredentialed”

**Location:** `internal/cli/models.go`, new `providerCredentialed` function around lines 167–174

`providerCredentialed` returns:

```go
result, err := registry.GetAuth(ctx, probe)
return err == nil && result != nil
```

Every error is therefore treated exactly like `(nil, nil)`. In the default view the provider silently disappears; under `--all` its models are labelled `uncredentialed`. That can misreport a broken OAuth credential, credential-store failure, or canceled context as an absent capability.

This differs from the review path, which preserves `*ai.ModelsError` as exit 2, and from `tiers`, which reports `credential broken` distinctly.

**What would confirm it:** Run `models` against the existing `BrokenAPIKeyAuth`/expired-OAuth fixtures and against an auth resolver returning `context.Canceled`. If these should be reported as broken-machine/query failures—or at least labelled “credential broken”—`providerCredentialed` needs to return an error or richer status instead of a boolean.

---

### 4. `tiers` silently drops catalog-refresh warnings and can mislabel the resulting failure

**Location:** `internal/cli/tiers.go`, new `resolveTierRow` function around lines 126–143

The `reviewer.Resolver` created here has no `Warn` callback. A refresh failure is therefore discarded. If the last-known catalog is empty, the row becomes `not in the catalog`, even though the immediate cause may be a failed refresh. This contrasts with `review`, which wires `Warn`, and `models --refresh`, which emits the failed-refresh warning.

**What would confirm it:** Use a refreshable provider whose refresh fails and whose existing model list is empty. If `tiers` prints only `not in the catalog` and no warning, it is concealing information needed to distinguish a bad assignment from a transient catalog failure. Wiring the command’s existing `warn` callback through resolution would confirm the intended behavior.

---

### 5. The documented “absolute” config-path contract is not enforced

**Location:** `internal/config/config.go`, new `Locate` function around lines 82–92

Both the comment and `docs/planning/specs/06-config-resolution-and-env-overrides.md` describe `EXTERNAL_REVIEWER_CONFIG` as an absolute path, but `Locate` returns any non-empty value unchanged. A relative value is resolved against the process working directory, making the selected machine configuration depend on where the command was launched.

**What would confirm it:** Decide whether relative paths are supported. If not, add a relative-path test and classify its rejection as malformed machine configuration. If they are supported, the comments and decision document should stop promising an absolute path.

---

### 6. The oversized-stdin diagnostic still describes only a task prompt

**Location:** `internal/cli/review.go`, `errPromptTooLarge` hunk near `maxPromptBytes`

The stdin contract changed from raw task text to a JSON object containing both system prompt and task, but the sentinel remains:

```go
errors.New("task prompt exceeds the maximum size")
```

The adjacent comments now correctly describe the request object. The emitted error therefore disagrees with the code and current interface.

**What would confirm it:** Exercise the existing over-limit stdin test and inspect its `error:` line. If compatibility does not require the old wording, rename it to identify the request object or stdin body.

## What looked right

- The family gate is fail-closed end to end: unknown families are refused in `Exclusion.AllowsFamily`, and `Resolver.Resolve` performs the family check before credential resolution.
- Anthropic models served through OpenRouter and Bedrock are classified and excluded; the catalog-wide tests specifically guard these reseller shapes.
- The Azure OpenAI rule departure is explicit and supported by the documented bare-ID catalog evidence rather than being silently hidden.
- `--model` correctly bypasses tier assignment and the config file, while the per-tier environment layer overrides the corresponding config assignment.
- Review-path error classification preserves absent reviewer as exit 1 and broken credentials as exit 2; malformed command-line model assignments are kept distinct from machine configuration failures.
- The JSON request decoder rejects unknown fields, trailing values, missing/empty prompts, and preserves accepted text verbatim.
- The shipped turn and elapsed caps are wired into real invocations, and a bounded run with no prose is no longer reclassified as failed.
- Local GPT-5.6 registration preserves the upstream provider implementation and gives upstream catalog definitions precedence on ID collisions.