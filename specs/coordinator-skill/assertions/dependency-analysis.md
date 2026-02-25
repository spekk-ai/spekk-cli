---
id: dependency-analysis
parent: coordinator-skill
created: 2026-02-25T00:00:00Z
priority: 1
status: done
---

# Coordinator Analyzes Dependencies Between Assertions

Coordinator uses LLM reasoning to infer single-parent dependency chains between draft/not_started assertions.

## Success Criteria

### Analysis Process

1. **Read all assertions** with `status: draft` or `status: not_started`
2. **For each assertion**, analyze:
   - What functionality/state must exist before this can be built?
   - Which other assertion (if any) provides that prerequisite?
   - Domain relationships (e.g., "form validation needs data model first")
3. **Identify single parent** for each assertion (or none if no dependencies)
4. **Build dependency tree** (forest of chains, not DAG)

### Dependency Inference Examples

**Good (clear single parent):**
```
chat-message-input depends-on: chat-session-model
# Reasoning: Can't send messages without a session context
```

**Needs sequencing:**
```
User spec says: "Chat needs authentication AND websocket connection"

Coordinator sequences:
1. authentication-system (no dep)
2. websocket-connection (depends-on: authentication-system)
3. chat-session-model (depends-on: websocket-connection)

# Forces explicit order instead of multi-parent
```

**Junction pattern:**
```
Multiple prerequisites that can be built in parallel:
1. database-schema (no dep)
2. api-endpoints (no dep)
3. frontend-service-layer (no dep)

Then:
4. integration-ready (manual junction - mark done when 1-3 complete)
5. feature-x (depends-on: integration-ready)
```

### Output Format

Coordinator presents dependency tree for user confirmation:

```
Dependency Analysis
===================

feature/chat-system:
  ├─ websocket-connection
  │   └─ chat-session-model
  │       └─ chat-message-input
  └─ user-presence-tracking

feature/clinical-trials:
  └─ trial-search-api
      └─ trial-eligibility-check

main (no dependencies):
  - update-button-styles
  - fix-header-typo

Does this look correct? [y/N]
```

## Implementation Notes

### LLM Prompt Strategy

```
You are analyzing software assertions to build a dependency tree.

For each assertion:
1. Identify WHAT prerequisite functionality must exist
2. Identify WHICH assertion provides that functionality
3. Choose the MOST DIRECT parent (closest prerequisite)

Rules:
- Each assertion has AT MOST one parent (single-parent tree)
- If multiple prerequisites, you must SEQUENCE them or create a junction
- No circular dependencies
- Prefer explicit over inferred (ask user to confirm ambiguous cases)

Assertions:
[JSON array of assertion metadata]

Output:
{
  "assertion-id": {
    "depends-on": "parent-assertion-id" | null,
    "reasoning": "Brief explanation"
  }
}
```

### Edge Cases

- **Circular dependencies**: Detect and reject (show error, ask user to fix specs)
- **Ambiguous dependencies**: Present options, ask user to choose
- **No clear parent**: Defaults to null (no dependency)

## Validation

- Coordinator reads draft/not_started assertions correctly
- LLM generates single-parent dependencies (no arrays)
- Dependency tree is presented clearly (visual tree or indented list)
- User can confirm or request changes
- Circular dependencies are detected and reported as errors

**Tests:** Unit tests with sample assertion sets, edge case scenarios
