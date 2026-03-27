# Spekk CLI 1.2.0 — Coordinator Skill & Skills Architecture

## Coordinator Skill (PR #25) 🎯

The coach can now analyze draft assertions and create a **dependency-aware work plan with branch assignments**.

### What it does

The coordinator skill helps you organize complex work before you start building:

1. **Analyzes dependencies** — identifies which assertions need to be built before others
2. **Groups related work** — clusters assertions into logical feature branches
3. **Shows dependency tree** — visualize what can be built in parallel vs. serially
4. **Updates YAML frontmatter** — adds `depends-on` and `branch` fields to assertions
5. **Validates with parser** — catches circular dependencies and invalid references

### Usage

```bash
# Launch coordinator
spekk coach coordinate

# Or trigger in regular coach session
spekk coach
> "Plan the work for these draft assertions"
```

### Example output

```
Dependency tree:

feature/chat-system (4 assertions):
  websocket-connection (no dependencies)
    ↓
  chat-session-model (depends-on: websocket-connection)
    ↓
  chat-message-input (depends-on: chat-session-model)
  user-presence-tracking (no dependencies)

feature/authentication (2 assertions):
  password-hashing (no dependencies)
  session-tokens (no dependencies)

main (isolated work):
  update-button-styles (no dependencies)
  fix-header-typo (no dependencies)

Proceed with these updates? [y/N]
```

### New YAML fields

**`depends-on`** — Single assertion ID that must be completed first (optional)

```yaml
depends-on: websocket-connection
```

**`branch`** — Git branch where this assertion lives (defaults to `main`)

```yaml
branch: feature/chat-system
```

### Why single-parent dependencies?

Each assertion can depend on **at most one** other assertion. This forces clearer thinking:

- If you have multiple prerequisites → sequence them (A → B → C)
- If you really need convergence → create a "prerequisites-ready" junction assertion

### Builder respects branches

The `spekk next` command and builder now filter to your current git branch by default:

```bash
# Get next assertion on current branch
spekk next

# See assertions across all branches
spekk next --all-branches
```

This prevents accidentally building feature-branch work on `main`.

---

## Skills Architecture Overhaul (PR #27)

Coach skills are now **markdown files** instead of classes. This makes skills:
- **Easier to create** — no code, just markdown
- **Easier to read** — workflow steps are plain instructions
- **Easier to extend** — anyone can add a skill

### What changed

**Before (1.1.0):** Skills were JavaScript classes with methods

**Now (1.2.0):** Skills are markdown files with sections

### Skill format

```markdown
---
id: my-skill
created: 2024-01-15T00:00:00Z
---

# Skill Name

Brief description.

## Triggers

- "keyword 1"
- "keyword 2"

## Workflow

1. Step one
2. Step two
3. Step three

## Validation

- Success criterion 1
- Success criterion 2
```

### Available skills

All existing skills converted to markdown:

1. **Business Model Validator** — `business-model-validator-skill.md`
2. **Meeting Notes to Specs** — `meeting-notes-to-specs-skill.md`
3. **Coordinator** — `coordinator-skill.md` (new!)

See `docs/COACH-SKILLS.md` for detailed documentation.

---

## CLI Subcommands Fix (PR #28)

Fixed a bug where CLI subcommands weren't properly inlining skill markdown content. The coach now correctly loads and uses skill instructions when you run:

```bash
spekk coach meeting
spekk coach coordinate
```

---

## Migration Notes

### If you have custom coach skills

Convert them from JS classes to markdown files:

1. Create `specs/coach-skills-system/my-skill.md`
2. Add YAML frontmatter with `id` and `created`
3. Add sections: Triggers, Workflow, Validation
4. Delete the old `.js` file

The coach will automatically detect and use markdown skills.

### If you use the coordinator

The coordinator adds new YAML fields to assertions. After running `spekk coach coordinate`:

1. Review the proposed changes
2. Approve the updates
3. Create feature branches: `git checkout -b feature/my-feature`
4. Build on the correct branch: `git checkout feature/my-feature && spekk builder`

---

## Quick Test Checklist

```bash
# Tests pass
npm test

# Coordinator launches
spekk coach coordinate

# Skills documentation is current
cat docs/COACH-SKILLS.md

# Parser respects branch filtering
spekk next
spekk next --all-branches

# Skills are markdown files
ls specs/coach-skills-system/
```

---

## Breaking Changes

None. All changes are additive. Existing specs work without modification.

New YAML fields (`depends-on`, `branch`) are optional — omit them and behavior is unchanged.

---

## What's Next

Possible future improvements:

- Visual dependency graph viewer
- Auto-detect assertion dependencies via code analysis
- Branch creation automation
- Parallel builder instances (one per branch)

---

## Contributors

Thanks to everyone who tested and provided feedback on the coordinator skill workflow!
