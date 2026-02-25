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
- Coach prompt lists available skills (or tells coach to scan directory)
- Coach detects triggers in user input
- Coach reads skill markdown file directly
- Coach executes workflow steps using its intelligence
- Coach validates results (e.g., with parser)
- **No JavaScript loader/registry needed** - Coach reads files directly
- **No skill classes needed** - Skills are instructions for the AI
- **No API calls** - Coach IS the AI

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

### What Must Exist
- Skill markdown files in `specs/coach-skills-system/`
- Coach prompt updated to list skills or tell coach where to find them
- Skills follow consistent format (triggers, workflow, validation)

### What Must NOT Exist
- ❌ No `markdown-skill-loader.js` or similar (coach reads files directly)
- ❌ No JavaScript skill classes (deleted, including deprecated ones)
- ❌ No skill registry infrastructure (coach knows where skills are)
- ❌ No `IMPLEMENTATION_SUMMARY.md` (lives in PR/commits)

### Testing
- Coach successfully detects skill triggers
- Coach reads and executes workflow steps
- New skills added by just creating markdown file
- No code changes needed for new skills
