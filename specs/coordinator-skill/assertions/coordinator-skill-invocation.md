---
id: coordinator-skill-invocation
parent: coordinator-skill
created: 2026-02-25T00:00:00Z
priority: 1
status: done
---

# Coordinator Skill Can Be Invoked

Coach agent has coordinator skill that can be invoked via CLI and auto-detected in conversations.

## What Must Be True

### Command-Line Invocation
- `spekk coach coordinator` launches coach with coordinator skill active
- Coach displays coordinator skill intro/help on launch

### Auto-Detection
Coach suggests coordinator skill when user mentions:
- "plan the work"
- "organize branches"
- "dependency graph"
- "what can we build in parallel"
- "coordinate development"
- "scope the work"

### User Control
- User can accept or decline coordinator skill suggestion
- Skill activates only after user confirmation

## Validation Checklist
- [ ] `spekk coach coordinator` launches successfully
- [ ] Coach displays coordinator intro when invoked
- [ ] Auto-detection triggers on coordination keywords
- [ ] User can accept/decline skill suggestion
