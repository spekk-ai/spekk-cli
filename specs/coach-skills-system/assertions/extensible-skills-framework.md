---
id: extensible-skills-framework
parent: coach-skills-system
created: 2026-01-23T22:14:00Z
priority: 3
status: done
---

# Extensible Skills Framework

Skills system supports adding new specialized coaching capabilities as markdown workflow files.

## What Must Be True

### Skill Format
- Skills are markdown files in `specs/coach-skills-system/`
- Naming: `skillname-skill.md`
- Required sections:
  - `# Skill Name`
  - `## Triggers` - List of trigger keywords
  - `## Workflow` - Numbered steps
  - `## Validation` - Success criteria
- Optional: Context, Examples

### Coach Execution
- Coach scans for skill markdown files
- Detects triggers in user input
- Reads workflow steps from markdown
- Executes steps using its intelligence
- Validates results (e.g., with parser)
- No JavaScript classes needed
- No API calls (coach IS the AI)

### Example Skill Structure
```markdown
# Coordinator Skill

## Triggers
- "plan the work"
- "dependency graph"

## Workflow
1. Read draft assertions
2. Analyze dependencies
3. Show tree to user
4. Update YAML
5. Validate with parser
6. Commit
```

## Validation

- Skills can be created by writing markdown
- Coach successfully executes workflow steps
- New skills added without code changes
- All skills follow consistent pattern
