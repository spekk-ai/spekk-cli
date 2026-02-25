---
id: extensible-skills-framework
parent: coach-skills-system
created: 2026-01-23T22:14:00Z
priority: 3
status: in_progress
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
- Coach receives spekk installation path in activation message
- Coach prompt lists available skills in `specs/coach-skills-system/`
- Coach detects triggers in user input
- Coach reads skill markdown file from installation path
- Coach executes workflow steps using its intelligence
- Coach validates results (e.g., with parser)
- **No JavaScript loader/registry needed** - Coach reads files directly
- **No skill classes needed** - Skills are instructions for the AI
- **No API calls** - Coach IS the AI

### Path Resolution
Since `spekk` can be run from any directory:
- Working directory = user's project (`process.cwd()`)
- Spekk installation = where spekk is installed (calculated from `__dirname`)
- Skills location = `{spekk-installation}/specs/coach-skills-system/`

Coach needs to know spekk installation path to find skills.

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
- Coach prompt lists available skills
- Coach receives spekk installation path in activation message
- Skills follow consistent format (triggers, workflow, validation)

Example activation message addition:
```
Spekk installation: /usr/local/lib/node_modules/@spekk/cli
Skills directory: /usr/local/lib/node_modules/@spekk/cli/specs/coach-skills-system/
```

### What Must NOT Exist
- ❌ No `markdown-skill-loader.js` or similar (coach reads files directly)
- ❌ No JavaScript skill classes (deleted, including deprecated ones)
- ❌ No skill registry infrastructure (coach knows where skills are)
- ❌ No `IMPLEMENTATION_SUMMARY.md` (lives in PR/commits)

### Testing
- Coach receives installation path in activation message
- Coach can read skills from installation path (not cwd)
- Coach successfully detects skill triggers
- Coach reads and executes workflow steps
- Works when spekk is run from any directory
- New skills added by just creating markdown file
- No code changes needed for new skills
