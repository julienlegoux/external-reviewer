# Issues — Epic 0: Skeleton hardening

Plan and standard repairs first — each one changes the contract the code work is judged
against — then the CI gate that protects every PR after it, then the code.

* [Settle the done line's stop-reason vocabulary and document it in SPECS](./01-stop-reason-vocabulary.md) - S, done, [#23](https://github.com/julienlegoux/external-reviewer/issues/23), [PR #37](https://github.com/julienlegoux/external-reviewer/pull/37)
* [Widen the write-API guard to every write operation CONVENTIONS names](./02-write-api-guard-coverage.md) - S, done, [#24](https://github.com/julienlegoux/external-reviewer/issues/24), [PR #36](https://github.com/julienlegoux/external-reviewer/pull/36)
* [Carry Epic 1's structural warnings into Epic 2's issues](./03-epic-2-plan-amendments.md) - S, done, [#25](https://github.com/julienlegoux/external-reviewer/issues/25), [PR #35](https://github.com/julienlegoux/external-reviewer/pull/35)
* [Enforce gofmt in CI, normalise line endings, and lint on both OSes](./04-ci-formatting-gate.md) - S, done, [#26](https://github.com/julienlegoux/external-reviewer/issues/26), [PR #38](https://github.com/julienlegoux/external-reviewer/pull/38)
* [Extract one importable faux harness and retire the mutable package-level seams](./05-shared-faux-harness.md) - M, done, [#27](https://github.com/julienlegoux/external-reviewer/issues/27), [PR #40](https://github.com/julienlegoux/external-reviewer/pull/40)
* [Kill the four surviving mutants on the request side of the round trip](./06-request-side-mutants.md) - M, open, [#28](https://github.com/julienlegoux/external-reviewer/issues/28)
* [Write one prefixed error: line from diag on every exit-2 path](./07-single-error-line.md) - M, done, [#29](https://github.com/julienlegoux/external-reviewer/issues/29), [PR #39](https://github.com/julienlegoux/external-reviewer/pull/39)
* [Handle the repository path as an OS path and emit wire paths on stderr](./08-os-and-wire-paths.md) - M, done, [#30](https://github.com/julienlegoux/external-reviewer/issues/30), [PR #41](https://github.com/julienlegoux/external-reviewer/pull/41)
* [Read the task prompt under a context, a byte bound and real validation](./09-bounded-validated-prompt.md) - M, open, [#31](https://github.com/julienlegoux/external-reviewer/issues/31)
* [Survive a failing or closed stdout without escaping the done-line seam](./10-stdout-write-failures.md) - M, open, [#32](https://github.com/julienlegoux/external-reviewer/issues/32)
* [Bound the turn and fix its cost and cancellation precedence](./11-turn-hardening.md) - M, pr-open, [#33](https://github.com/julienlegoux/external-reviewer/issues/33), [PR #45](https://github.com/julienlegoux/external-reviewer/pull/45)
* [Escape provider-controlled text on stderr and finish diag's polish](./12-diag-escaping-and-polish.md) - M, open, [#34](https://github.com/julienlegoux/external-reviewer/issues/34)
