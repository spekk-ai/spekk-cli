# Spekk CLI 1.15.1 — Migrate the Old Dev-Loop Skill

This is a patch for the 1.15.0 install migration.

1.15.0 recognized only the role shims as old spekk files. So an unstamped dev-loop skill from an earlier version was treated as a hand-edited file: `spekk install` backed it up and left the old skill in place. An existing user did not get the new single-session dev-loop skill — the main point of 1.15.0.

This release fixes the migration. `spekk install` now recognizes the dev-loop skill by its heading and updates it in place, with a `.bak` backup.

## Upgrade

```bash
spekk update
```

After you update, re-run `spekk install --target <tool>` to migrate the dev-loop skill.
