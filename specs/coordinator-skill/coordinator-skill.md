---
id: coordinator-skill
created: 2026-02-25T00:00:00Z
priority: 1
---

# Coordinator Skill

Extends the coach agent with work planning and dependency analysis capabilities that enable autonomous, branch-scoped development.

## Problem

Current state:
- Priority is the only ordering mechanism (global, not dependency-aware)
- No way to scope work to branches (builder could work on wrong things)
- No dependency tracking (builder might start work blocked by unfinished prerequisites)
- Manual coordination required to prevent conflicts

## What Must Be True

### Metadata Structure
- Assertions support `depends-on` field (single parent ID or null/omitted)
- Assertions support `branch` field (git branch name)
- Omitted `depends-on` means no dependency (backward compatible)
- Parser validates these fields without breaking existing specs

### Coordinator Capability
- Coach can be invoked with coordination mode
- Coordinator analyzes draft/not_started assertions
- Dependency relationships are inferred from domain knowledge
- Related assertions are grouped into logical feature clusters
- Each cluster is assigned a semantic branch name
- YAML frontmatter is updated across affected files
- Changes are committed with clear summary
- User can review plan before applying

### Branch-Aware Operation
- `spekk next` defaults to filtering by current git branch
- Only assertions with satisfied dependencies are returned
- Priority breaks ties among ready assertions
- Assertions on different branches are independent

### Dependency Model
- Dependencies form single-parent chains (trees, not DAGs)
- Circular dependencies are prevented
- Cross-branch dependencies are disallowed

## Validation

The system must be verifiable by:
- Running coordinator on a test set of assertions
- Confirming correct dependency inference
- Checking branch assignment makes semantic sense
- Verifying `spekk next` respects branch and dependencies
- Confirming priority still works as tiebreaker
- Testing backward compatibility with existing specs (no `depends-on`/`branch` fields)

## Non-Goals

- Multi-parent dependencies (use single parent or create junction assertions)
- Cross-branch dependencies (assertions on different branches should be independent)
- Automatic branch creation (coordinator updates metadata, user creates branches manually)
- Automatic merging or branch orchestration

## Design Rationale

### Single Parent Dependencies
- Simpler mental model than DAG
- Forces clearer thinking about sequence vs parallelism
- Junction assertions handle multiple prerequisites explicitly
- Easier to visualize and reason about

### Branch Clustering
- Dependent assertions belong on same branch
- Isolated assertions can be grouped thematically or kept on main
- Branch names should communicate intent clearly
- Clusters enable parallel development across branches

### Sparse YAML
- Only add `depends-on` when dependency exists
- Cleaner, more readable specs
- Backward compatible by default

## Context

This addresses a core scalability challenge: as projects grow, manual coordination becomes a bottleneck. The coordinator skill enables automated, parallelized development while maintaining coherence through explicit dependencies and branch isolation.

## Related Work

- **Coach Skills System** - Framework for extending coach capabilities
- **Meeting Notes to Specs** - Existing skill that updates multiple files and commits
- **Parser** - Needs updates for new field validation
- **Spec Format Docs** - Defines YAML schema conventions
