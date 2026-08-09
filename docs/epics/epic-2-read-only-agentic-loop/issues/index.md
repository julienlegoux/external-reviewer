# Issues — Epic 2: Read-only agentic loop

Cut in dependency order: the boundary first, then the loop, then the tools that plug into
it, then the measurements the whole epic exists to produce.

* [Confine the run with --allow and os.OpenRoot](./01-allow-list-and-root-confinement.md) - M, open, [#9](https://github.com/julienlegoux/external-reviewer/issues/9)
* [Enumerate the repository through git ls-files and add the list tool](./02-enumeration-and-list-tool.md) - M, open, [#10](https://github.com/julienlegoux/external-reviewer/issues/10)
* [Drive the multi-turn loop with tool dispatch and the bounds seam](./03-multi-turn-loop-and-dispatch.md) - M, open, [#11](https://github.com/julienlegoux/external-reviewer/issues/11)
* [Add the batched read_file tool](./04-read-file-tool.md) - M, open, [#12](https://github.com/julienlegoux/external-reviewer/issues/12)
* [Add the search tool with surrounding context lines](./05-search-tool.md) - M, open, [#13](https://github.com/julienlegoux/external-reviewer/issues/13)
* [Add the confined git_read tool](./06-git-read-tool.md) - M, open, [#14](https://github.com/julienlegoux/external-reviewer/issues/14)
* [Run a real review by hand and record the measurements](./07-hand-run-measurements.md) - S, open, [#15](https://github.com/julienlegoux/external-reviewer/issues/15)
