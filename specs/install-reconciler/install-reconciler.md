---
id: install-reconciler
created: 2026-07-27T00:00:00Z
priority: 2
---

# Install Reconciler — Deterministic, Idempotent Harness Install

## Overview

`spekk install` writes managed files into a host coding assistant (the agent
shims and the `spekk-dev-loop` skill). Today the command is additive: it writes
files but never removes files that a newer layout no longer needs. When a
managed file moves (for example, when a role moves from an agent to a skill), the
old file stays behind and the host tool still uses it.

This spec makes `spekk install` a reconciler. The command drives the managed
files to a desired final state. It writes new files, updates changed files, and
removes managed files that are no longer part of the desired set.

## Principle

Do not keep state that you cannot make again from version-controlled content.
The reconciler uses no external manifest. It uses two things only:

1. A desired set that the command calculates from the binary content, the
   selected target, and the scope. The calculation is deterministic.
2. An in-file stamp on every managed file. The stamp marks the file as
   spekk-managed and holds a hash of the file body. A scan of the known install
   locations finds the stamped files. This is the owned set.

## Reconcile

- Write or update each file in the desired set. Add the stamp to each file.
- Remove each owned file that is not in the desired set.
- Before you write over or remove a file, check the body hash against the stamp.
  If the hash does not agree, the user changed the file. Make a `.bak` backup and
  give a warning. Do not change the file.

## Update

`spekk update` runs the scan and the desired-set calculation only. It changes no
file. If an owned file is not in the desired set, an old layout is present.
`spekk update` gives a warning that shows the `spekk install` command.

## Non-goals

- This spec does not move any role from an agent to a skill. It builds the
  machinery only. The content move is separate work.
- This spec does not add a manifest or any file under an XDG state directory.
