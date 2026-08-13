# Issues — Epic 3: Reviewer selection and pipeline integration

Cut in dependency order: first the measured defect that distorts the numbers this epic
sizes its caps from, then the two independent halves of selection (family, config), then
the resolution chain and the surfaces over it, then the caps, and finally the cross-repo
swap and the run that proves v1's success criteria. Issue 10 lands in the **lx skills
repository**, not this one.

* [Make a path-scoped diff reachable through git_read](/epic-3-reviewer-selection-integration/issues/01-git-read-path-scoped-diff.md) - M, done, [#60](https://github.com/julienlegoux/external-reviewer/issues/60), [PR #81](https://github.com/julienlegoux/external-reviewer/pull/81)
* [Add the fail-closed model family classifier](/epic-3-reviewer-selection-integration/issues/02-model-family-classifier.md) - L, done, [#61](https://github.com/julienlegoux/external-reviewer/issues/61), [PR #82](https://github.com/julienlegoux/external-reviewer/pull/82)
* [Read the machine-local tier assignment TOML](/epic-3-reviewer-selection-integration/issues/03-tier-assignment-config-file.md) - M, done, [#62](https://github.com/julienlegoux/external-reviewer/issues/62), [PR #80](https://github.com/julienlegoux/external-reviewer/pull/80)
* [Resolve a tier to a reachable, allowed model](/epic-3-reviewer-selection-integration/issues/04-tier-resolution-chain.md) - L, done, [#63](https://github.com/julienlegoux/external-reviewer/issues/63), [PR #85](https://github.com/julienlegoux/external-reviewer/pull/85)
* [Select the reviewer from the command line with --tier, --model and --exclude-family](/epic-3-reviewer-selection-integration/issues/05-review-tier-flags-and-exit-codes.md) - L, done, [#64](https://github.com/julienlegoux/external-reviewer/issues/64), [PR #86](https://github.com/julienlegoux/external-reviewer/pull/86)
* [Add the tiers command](/epic-3-reviewer-selection-integration/issues/06-tiers-command.md) - M, done, [#65](https://github.com/julienlegoux/external-reviewer/issues/65), [PR #87](https://github.com/julienlegoux/external-reviewer/pull/87)
* [Add the models command](/epic-3-reviewer-selection-integration/issues/07-models-command.md) - M, done, [#66](https://github.com/julienlegoux/external-reviewer/issues/66), [PR #89](https://github.com/julienlegoux/external-reviewer/pull/89)
* [Accept the JSON request object on stdin, add --system and version](/epic-3-reviewer-selection-integration/issues/08-request-object-and-version.md) - M, done, [#67](https://github.com/julienlegoux/external-reviewer/issues/67), [PR #88](https://github.com/julienlegoux/external-reviewer/pull/88)
* [Set the loop caps from Epic 2's measurements](/epic-3-reviewer-selection-integration/issues/09-loop-caps-from-measurements.md) - S, done, [#68](https://github.com/julienlegoux/external-reviewer/issues/68), [PR #83](https://github.com/julienlegoux/external-reviewer/pull/83)
* [Swap review-interfaces.md from opencode to external-reviewer](/epic-3-reviewer-selection-integration/issues/10-review-interfaces-external-reviewer-swap.md) - M, pr-open, [julienlegoux/skills#37](https://github.com/julienlegoux/skills/issues/37), [PR julienlegoux/skills#42](https://github.com/julienlegoux/skills/pull/42) — lands in the lx skills repository
* [Run a real review through the pipeline and record the success criteria](/epic-3-reviewer-selection-integration/issues/11-success-criteria-verification-run.md) - S, in-progress, [#69](https://github.com/julienlegoux/external-reviewer/issues/69)
