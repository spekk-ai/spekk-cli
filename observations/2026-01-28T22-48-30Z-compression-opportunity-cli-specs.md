---
id: compression-opportunity-cli-specs
created: 2026-01-28T22:48:30Z
type: compression_opportunity
severity: high
affected_specs:
  - cli-prompt-resolution
  - fix-cli-context-bug
  - spekk-cli
---

# Spec Compression Opportunity: CLI Context Resolution

## Issue Description
Three specifications have overlapping functionality around CLI working directory and context resolution that could be consolidated into a single comprehensive spec. This overlap is causing conflicts (as documented in observation `spec-conflict-parser-directory`) and implementation confusion.

## Evidence
### Overlapping Requirements

**cli-prompt-resolution:**
- Agent prompts accessible from any directory
- Spec parser reads from installation directory
- Claude Code runs in user directory

**fix-cli-context-bug:**
- CLI reads specs from current working directory
- Contradicts cli-prompt-resolution's parser behavior

**spekk-cli:**
- Defines "context awareness" as core principle
- General CLI design and commands

### Current Issues
- Direct conflict between specs about where parser reads specs from
- Redundant assertions across multiple specs
- Implementation confusion leading to test failures
- Developers must consult 3 different specs for CLI directory behavior

## Impact
- High confusion for implementers trying to understand correct behavior
- Conflicting requirements cannot both be satisfied
- Maintenance overhead of keeping 3 specs in sync
- Risk of introducing new conflicts as specs evolve independently

## Recommendation
Consolidate these three specs into a single "CLI Directory Context Resolution" spec that:

1. **Clearly defines working directory behavior:**
   - When CLI uses current working directory
   - When CLI uses installation directory
   - How to resolve conflicts between the two

2. **Unified assertions covering:**
   - Agent prompt accessibility
   - Spec file discovery
   - Context-aware operations
   - Consistent behavior across all commands

3. **Single source of truth for:**
   - Directory resolution logic
   - Path handling requirements
   - Context switching behavior

This consolidation would eliminate the existing conflicts and provide clear guidance for implementation.