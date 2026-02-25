---
id: branch-assignment
parent: coordinator-skill
created: 2026-02-25T00:00:00Z
priority: 1
status: not_started
depends-on: dependency-analysis
---

# Coordinator Assigns Assertions to Feature Branches

Coordinator groups related assertions into semantic feature branches based on dependency clusters.

## Success Criteria

### Branch Assignment Algorithm

1. **Identify connected components** in dependency tree
   - Assertions that share dependencies = one cluster
   - Walk up/down dependency chains to find related work

2. **Assign branch per cluster**
   - Each cluster → one feature branch
   - Isolated assertions (no deps) → main or "quick-wins" branch

3. **Generate semantic names**
   - Use parent spec ID: `feature/<spec-id>`
   - Or derive from cluster theme: `feature/chat-system`, `feature/clinical-trials`
   - Ask user to confirm/customize names

### Example Output

```
Branch Assignments
==================

feature/chat-system (4 assertions):
  - websocket-connection
  - chat-session-model
  - chat-message-input
  - user-presence-tracking

feature/clinical-trials (2 assertions):
  - trial-search-api
  - trial-eligibility-check

main (2 assertions):
  - update-button-styles (isolated, no dependencies)
  - fix-header-typo (isolated, no dependencies)

Branch names OK? [y] Edit? [e]
```

### Branch Naming Rules

- **Format**: `feature/<name>` (lowercase, kebab-case)
- **Semantic**: Derived from parent spec or cluster purpose
- **No conflicts**: Check if branch name already exists (warn user)
- **User override**: Allow custom names if suggested name isn't right

### Clustering Logic

**Same cluster if:**
- Assertions share a common dependency ancestor
- Assertions are in dependency chain (A → B → C all in same cluster)
- Assertions belong to same parent spec (unless logically independent)

**Different clusters if:**
- No shared dependencies
- Different parent specs with no domain overlap
- User explicitly requests separation

## Implementation Notes

### Pseudo-code

```javascript
function assignBranches(dependencyTree) {
  const clusters = [];
  const visited = new Set();
  
  // Find all root nodes (no dependencies)
  const roots = dependencyTree.filter(a => !a.dependsOn);
  
  for (const root of roots) {
    const cluster = walkDependencyChain(root, dependencyTree);
    clusters.push(cluster);
  }
  
  // Assign branch names
  for (const cluster of clusters) {
    const branchName = generateBranchName(cluster);
    cluster.branch = branchName;
  }
  
  return clusters;
}

function generateBranchName(cluster) {
  // Try parent spec ID first
  const parentSpecs = [...new Set(cluster.map(a => a.parent))];
  if (parentSpecs.length === 1) {
    return `feature/${parentSpecs[0]}`;
  }
  
  // Multiple specs - derive semantic name
  // (Use LLM to suggest based on assertion titles)
  return suggestThematicName(cluster);
}
```

### Edge Cases

- **Single assertion**: Gets its own branch (or main if truly isolated)
- **Large clusters** (>10 assertions): Ask user if should split
- **Cross-spec dependencies**: Rare, but group into same branch
- **Orphaned assertions**: No parent spec → assign to "misc" or ask user

## Validation

- Assertions in dependency chains assigned to same branch
- Isolated assertions assigned to main or grouped logically
- Branch names are semantic and follow convention
- User can customize branch names before finalizing
- No branch conflicts with existing git branches (warning shown)

**Tests:** Unit tests with various dependency tree structures
