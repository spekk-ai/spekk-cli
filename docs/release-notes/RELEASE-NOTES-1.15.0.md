# Spekk CLI 1.15.0 — One-Session Dev Loop and a Self-Migrating Install

This release reworks the dev loop. A pilot showed that, for a feature that fits in one session, one session is far cheaper and faster than a pipeline of sub-agents, at the same quality (issue #154). So the dev loop now runs in one session, and the install changes to match.

## The dev loop is one session

The `spekk-dev-loop` skill now drives one continuous session that plays the roles in turn: scope (coach), build, and verify. It fetches each role with `spekk prompt <role>`, so the per-role overrides still apply. It does not dispatch a sub-agent per role.

For a feature that fits in one session, this uses about 5 times fewer tokens, costs about 3 times less, and finishes about 3 times faster than the old coach → builders → review pipeline. The build and verify phases reuse the context from the scope phase, instead of a new agent that reads the code again. For a feature too large for one session, the skill names the escalation to parallel sub-agents.

## The coach and builder are skills

`spekk install` now writes the observer as an agent, and the coach, the builder, and the dev-loop as skills. Each role file is thin: it fetches its full instructions with `spekk prompt <role>`, so a user can still run one role on its own and a per-role override still applies. A host with no skill path keeps the coach and builder as agent shims, so every host keeps the roles.

## Install is a reconciler

`spekk install` now drives the managed files to their final state. It writes the desired files, and it removes a managed file that a new layout no longer needs. It is idempotent: a second install changes nothing. It never writes over a file you changed by hand — it makes a `.bak` backup and leaves the file.

## Automatic migration

When you update and re-run `spekk install`, the old coach and builder agent shims migrate to skills. The install backs up and removes an old shim, and writes the new skill. `spekk update` also checks your install locations and warns you when an old layout is present.

## Upgrade

```bash
spekk update
```

After you update, re-run `spekk install --target <tool>` to migrate to the new layout.
