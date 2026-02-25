---
id: dependency-analysis
parent: coordinator-skill
created: 2026-02-25T00:00:00Z
priority: 1
status: in_progress
---

# Coordinator Analyzes Dependencies Using Claude

Coordinator asks Claude to analyze assertions and propose single-parent dependency chains.

## What Must Be True

### Reading Assertions
- [ ] Uses existing parser to read draft/not_started assertions
- [ ] Sends assertion content (title, description, parent spec) to Claude

### LLM Analysis
- [ ] Coordinator provides Claude with structured prompt
- [ ] Claude analyzes relationships between assertions
- [ ] Claude returns JSON: `{assertionId: {dependsOn: parentId | null, reasoning: string}}`
- [ ] Single-parent enforced (prompt instructs: max one parent)
- [ ] Claude detects circular dependencies

### User Confirmation
- [ ] Dependency tree displayed clearly
- [ ] User can approve or request changes
- [ ] No updates happen without confirmation

### No Manual Algorithms
- [ ] No graph traversal code
- [ ] No heuristics or keyword matching
- [ ] Claude does the intelligent analysis
- [ ] Code just orchestrates (read → prompt → show → confirm)

## Validation

- Run coordinator on test project
- Verify Claude returns valid structure
- Verify circular dependencies caught
- Verify user can confirm/reject
