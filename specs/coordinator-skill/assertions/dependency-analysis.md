---
id: dependency-analysis
parent: coordinator-skill
created: 2026-02-25T00:00:00Z
priority: 1
status: done
---

# Coordinator Analyzes Dependencies Between Assertions

Coordinator uses LLM reasoning to infer single-parent dependency chains between draft/not_started assertions.

## What Must Be True

### Reading Assertions
- [ ] Coordinator reads all assertions with `status: draft` or `status: not_started`
- [ ] Each assertion's metadata is available for analysis

### Dependency Inference
- [ ] For each assertion, the coordinator identifies which other assertion (if any) is its prerequisite
- [ ] Each assertion has at most one parent (single-parent tree structure)
- [ ] When multiple prerequisites exist, they are sequenced or a junction is created
- [ ] Dependencies are based on functional relationships (what must exist first)

### Dependency Tree Output
- [ ] Dependencies are presented in a visual tree structure (indented or ASCII tree)
- [ ] Assertions are grouped by feature branch
- [ ] Assertions without dependencies are clearly shown
- [ ] User can review and confirm the analysis

### Validation
- [ ] Circular dependencies are detected and reported as errors
- [ ] Ambiguous dependencies prompt user for clarification
- [ ] Dependency relationships are logical and defensible
- [ ] Tree structure is correct (no multi-parent nodes)
