---
id: coordinator-skill-invocation
parent: coordinator-skill
created: 2026-02-25T00:00:00Z
priority: 1
status: done
---

# Coordinator Skill Can Be Invoked

Coach agent has coordinator skill that can be invoked via `spekk coach coordinator` and auto-detected in conversations.

## Success Criteria

### Command-line Invocation
```bash
spekk coach coordinator
```

Launches coach with coordinator skill active.

### Auto-detection Trigger

Coach auto-suggests coordinator skill when user mentions:
- "plan the work"
- "organize branches"
- "dependency graph"
- "what can we build in parallel"
- "coordinate development"
- "scope the work"

**Example interaction:**
```
User: "We have a bunch of draft specs. Can you help me plan out how to build these?"

Coach: "I can analyze your specs and create a dependency-aware work plan with branch assignments. This will let the builder work autonomously on parallel feature branches. Want me to do that?"

User: "Yes"

Coach: [activates coordinator skill workflow]
```

## Implementation

### CLI Integration

Update `src/coach/cli.js`:
```javascript
if (args[0] === 'coordinator') {
  // Load coordinator skill prompt
  // Pass to coach agent
  // Launch interactive session
}
```

### Coach Prompt Updates

Add to `specs/coach-agent/coach-agent.prompt.md` skill detection section:
```markdown
**Work Coordination Triggers:**
Check if the user mentions:
- "plan the work"
- "organize branches"
- "dependency graph"
- "what can we build in parallel"

**If work coordination is detected:**
Suggest: "I can analyze your specs and create a dependency-aware work plan with branch assignments. Want me to coordinate the work?"
```

## Validation

- `spekk coach coordinator` launches successfully
- Coach displays coordinator skill intro/help
- Auto-detection triggers in conversation when keywords mentioned
- User can accept or decline coordinator skill suggestion

**Tests:** Manual testing with sample spec directories
