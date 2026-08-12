# Drift — Epic 2: Read-only agentic loop

Standards this epic's implementation could not follow as decided, with the evidence.
`close-epic` promotes these into [DRIFT](../../../planning/DRIFT.md).

* [Wire-path spellings are validated before os.Root sees them, not only by it](/epic-2-read-only-agentic-loop/drift/01-wire-path-validation-before-os-root.md) - issue 01, open, accepted-pending-triage
* [Enumeration walks through the Scope's own ReadDir, not Root.FS()](/epic-2-read-only-agentic-loop/drift/02-enumeration-does-not-use-root-fs.md) - issue 02, open, accepted-pending-triage
* [The native half of the test command stopped working again](/epic-2-read-only-agentic-loop/drift/03-native-go-test-blocked-again.md) - issue 05, resolved 2026-08-12, no disposition needed
* [git_read scopes status too, and validates &lt;rev&gt;:&lt;path&gt; in every subcommand](/epic-2-read-only-agentic-loop/drift/04-git-read-scopes-status-and-every-object-argument.md) - issue 06, open, accepted-pending-triage
