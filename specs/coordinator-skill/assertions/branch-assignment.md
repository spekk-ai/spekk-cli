---
id: branch-assignment
parent: coordinator-skill
created: 2026-02-25T00:00:00Z
priority: 1
status: done
depends-on: dependency-analysis
---

# Coordinator Assigns Assertions to Feature Branches

Coordinator groups related assertions into semantic feature branches based on dependency clusters.

## What Must Be True

### Branch Assignment
- Related assertions (shared dependencies) are grouped into one cluster
- Each cluster is assigned one feature branch
- Assertions in a dependency chain (A → B → C) are in the same cluster
- Isolated assertions (no dependencies) are assigned to main or a "quick-wins" branch

### Branch Naming
- Branch names follow format: `feature/<name>` (lowercase, kebab-case)
- Names are semantic, derived from parent spec ID or cluster theme
- User can confirm or customize suggested names
- System warns if branch name already exists

### Clustering Rules
**Same cluster when:**
- Assertions share common dependency ancestor
- Assertions are in a dependency chain
- Assertions belong to same parent spec (unless logically independent)

**Different clusters when:**
- No shared dependencies
- Different parent specs with no domain overlap
- User explicitly requests separation

### Edge Cases Handled
- Single assertions get their own branch (or main if isolated)
- Large clusters (>10 assertions) prompt user about splitting
- Cross-spec dependencies are grouped into same branch
- Orphaned assertions (no parent spec) are handled explicitly

## Validation Checklist
- [ ] Assertions in dependency chains are in same branch
- [ ] Isolated assertions are grouped logically
- [ ] Branch names are semantic and follow convention
- [ ] User can customize branch names before finalizing
- [ ] Warnings shown for branch name conflicts
