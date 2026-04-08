---
id: builder-uses-lock-store
parent: external-lock-backend
created: 2026-04-08T16:00:00Z
priority: 1
status: not_started
depends-on: wire-backend-into-next
branch: feature/external-lock-backend
---

# Builder Uses LockStore

## What Must Be True

The builder agent prompt and any builder-side helpers use the configured `LockStore` for acquiring and releasing locks instead of hand-editing the `locked-by` frontmatter field. When the configured backend is `local`, the builder still writes `locked-by` to frontmatter for backwards compatibility.

## Success Criteria

- ✅ Builder prompt (`specs/builder-agent/builder-agent.prompt.md`) updated section 5 to describe the new flow:
  - If `spekk.config.yaml` specifies a non-local backend → acquire via `spekk lock <id>` (or equivalent helper), do NOT edit `locked-by` frontmatter
  - If no config or `type: local` → continue current behavior (edit `locked-by` in frontmatter)
- ✅ New helper CLI commands (or flags on existing commands) for builders to claim and release locks:
  - `spekk lock <assertion-id>` — calls `LockStore.Acquire`, prints lock ID on success, exits non-zero on `ErrLockHeld` with holder info
  - `spekk unlock <assertion-id> --lock-id <id>` — calls `LockStore.Release`
- ✅ Builder prompt updated to document: "on `ErrLockHeld`, pick a different assertion" — no merge-conflict recovery path is needed with non-local backends
- ✅ When using a non-local backend, the builder's commits no longer contain `locked-by` churn — only real content changes land in git
- ✅ When using `local` backend, all current locking tests (`TestIsLockStale_*`, `FindNextAssertion` lock-related tests) continue to pass unchanged
- ✅ Integration test: run builder against `file` backend, verify it acquires via `spekk lock`, does not write `locked-by` to frontmatter, releases via `spekk unlock` on completion
- ✅ Builder prompt updated to reference `spekk locks` and `spekk force-unlock` as the recovery tools when crashed builds leave stale locks

## Backwards Compatibility

- **Solo users with no config**: zero behavior change. `locked-by` still gets written, stale lock detection still works, no new commands required.
- **Teams adopting backends**: explicit opt-in via `spekk.config.yaml`. Once adopted, all builders in the team must use the same config (otherwise some will write to frontmatter and others to the backend and they won't see each other's locks).

## Builder Prompt Changes (rough sketch)

Replace section 5 ("Update Status and Release Lock") with:

```markdown
### 5. Claim and Release Locks

**Check your lock backend:**
```bash
spekk locks --info     # prints current backend: local | file | redis
```

**Claim the assertion:**
```bash
spekk lock <assertion-id>
# prints: Acquired. Lock ID: builder-...
```

If `spekk lock` fails with `ErrLockHeld`, run `spekk next` again to pick a
different assertion. Do not attempt to force-unlock — the original holder
may still be working.

**On completion:**
```bash
spekk unlock <assertion-id> --lock-id <id>
```

Then update the assertion's frontmatter `status` field to `done` or `failed`
and commit. With a non-local backend, you do NOT touch the `locked-by`
field — it's not used.
```

## Out of Scope

- Migrating existing in-flight `locked-by` frontmatter values — just wait for them to drain or expire
- Automatically detecting backend drift between builders (two builders on different configs)
- `spekk lock --wait` blocking acquisition (for now, acquire-or-fail is sufficient)

## Notes

This assertion closes the loop. After it lands, the full end-to-end story works: team configures a backend, builders claim via the backend, visibility via `spekk locks`, recovery via `spekk force-unlock`, solo users see zero change.
