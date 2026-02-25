---
id: coordinator-skill
created: 2026-02-25T00:00:00Z
priority: 1
---

# Coordinator Skill

Extends the coach agent with work planning and dependency analysis capabilities that enable autonomous, branch-scoped development.

## Overview

The coordinator skill analyzes draft and not_started specs to:
- Build dependency graphs (single-parent chains)
- Identify parallelizable vs serial work
- Assign assertions to feature branches
- Update YAML frontmatter with dependency and branch metadata
- Enable branch-aware builder automation

This allows the builder to work autonomously within a branch, confident it's following a coherent plan that respects dependencies.

## Problem

Current state:
- Priority is the only ordering mechanism (global, not dependency-aware)
- No way to scope work to branches (builder could work on wrong things)
- No dependency tracking (builder might start work blocked by unfinished prerequisites)
- Manual coordination required to prevent conflicts

## Solution

**New YAML fields:**
```yaml
depends-on: assertion-id  # Single parent dependency (or null/omitted)
branch: feature/name      # Git branch where this assertion lives
```

**Coordinator workflow:**
1. User runs `spekk coach coordinator`
2. Coach analyzes draft/not_started assertions
3. Uses LLM to infer dependencies based on domain relationships
4. Groups related assertions into dependency chains
5. Assigns branches to chains (feature clusters)
6. Presents plan for user confirmation
7. Updates YAML frontmatter across affected files
8. Commits changes with clear summary

**Parser becomes branch-aware:**
- `spekk next` defaults to current git branch
- Returns next assertion where dependencies are satisfied
- Priority breaks ties among ready assertions

## Success Criteria

- Coach can be invoked with `spekk coach coordinator`
- Coordinator analyzes assertions and builds dependency tree (single-parent chains)
- Coordinator assigns semantic branch names to assertion clusters
- Coordinator updates YAML frontmatter with `depends-on` and `branch` fields
- Coordinator commits changes with clear summary
- Parser validates new fields (`depends-on`, `branch`)
- `spekk next` filters to current branch by default
- `spekk next` only returns assertions where dependencies are done
- Priority still used to break ties among ready assertions

## Non-Goals

- Multi-parent dependencies (use single parent or create junction assertions)
- Cross-branch dependencies (assertions on different branches should be independent)
- Automatic branch creation (coordinator updates metadata, user creates branches manually)

## Design Decisions

### Single Parent Dependency Model

**Why not array?**
- Forces clearer thinking (sequence vs junction)
- Simpler tree structure (vs DAG)
- Easier to visualize and reason about
- Simpler builder logic (single check instead of "all done?")

**How to handle multiple prerequisites?**
1. **Sequence them**: `A → B → C`
2. **Create junction**: Manually complete a "prerequisites-ready" assertion after verifying dependencies

### Branch Assignment Strategy

**Algorithm:**
1. Build dependency graph (tree of single-parent chains)
2. Identify connected components (clusters sharing dependencies)
3. Each cluster → one feature branch
4. Isolated assertions → remain on main or group into "quick-wins"
5. Name branches semantically: `feature/<spec-id>` or `feature/<theme>`

**Example:**
```
feature/chat-system:
  websocket-connection (no dep)
    ↓
  chat-session-model (depends-on: websocket-connection)
    ↓
  chat-message-input (depends-on: chat-session-model)

feature/clinical-trials:
  trial-search-api (no dep)
    ↓
  trial-eligibility (depends-on: trial-search-api)

main:
  update-button-styles (no dep, isolated)
  fix-header-typo (no dep, isolated)
```

### Sparse YAML Updates

Only add `depends-on` field when there IS a dependency. Omitted field means no dependency (same as `depends-on: null`).

**Why?**
- Cleaner YAML (no clutter)
- Backward compatible (existing specs work as-is)
- Explicit about relationships (presence of field signals intent)

## Related Work

- **Coach Skills System** (`specs/coach-skills-system/`) - Framework for extending coach
- **Meeting Notes to Specs** - Existing skill that updates multiple files and commits
- **Parser** (`src/parser/index.js`) - Needs updates for new fields
- **Spec Format Docs** (`specs/nested-spec-organization/`) - Defines YAML schema

## Context

This addresses a core scalability challenge: as projects grow, manual coordination becomes a bottleneck. The coordinator skill enables automated, parallelized development while maintaining coherence through explicit dependencies and branch isolation.
