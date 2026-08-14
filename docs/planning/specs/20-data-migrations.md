---
type: Decision
title: "Data migrations"
description: "How stored state evolves once real data exists — of which there is one file, and it is hand-written."
tags: [decision, specs]
timestamp: 2026-08-09T01:30:00Z
phase: specs
decision: 20
slug: data-migrations
status: na
verdict: null
decided_via: na
depends_on: [tier-assignment-schema]
---

# Question

How schema and data changes roll out once real data exists.

# Options

Not applicable.

# Recommendation

**N/A — there is no database and no persistent state this program owns.**

There is no schema to migrate. The only file the project has an opinion about is the tier
assignment TOML ([decision](/specs/05-tier-assignment-schema.md)), and it is not data the
program produced: a human wrote it, by hand, with comments, and the binary only ever
reads it ([decision](/specs/06-config-resolution-and-env-overrides.md)). A program that
never writes a file cannot migrate one.

The one thing that *could* be called schema evolution — renaming a config key after
machines already have the file — is handled without machinery, and deliberately so:

- Unrecognised keys are reported on stderr rather than ignored
  ([decision](/specs/05-tier-assignment-schema.md)), so an outdated file announces itself
  the next time `tiers` or `review` runs instead of degrading into a silently unassigned
  tier.
- The `tiers` command shows what actually resolved on this machine right now, which is
  the whole diagnostic surface a twelve-line hand-edited file needs.
- No `version` key, no upgrade path, no rewrite-on-read. The population is one machine
  ([scope](/scope/02-target-users.md)); a rename is a sentence in a commit message and
  one line edited by hand.

The credential store is `kern-link`'s and evolves with `kern-link`
([decision](/specs/18-auth-and-authorization.md)).

# Verdict

N/A. No owned persistent state; the one config file is hand-written, read-only to this
program, and self-diagnosing via unrecognised-key warnings.
