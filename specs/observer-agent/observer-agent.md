---
id: observer-agent
created: 2026-01-22T17:00:00Z
priority: 2
status: not_started
---

# Observer Agent

The observer agent continuously monitors the system for drift and misalignment between specifications and implementation. It acts as a quality assurance layer that identifies issues for human review and coaching.

## Purpose

The observer detects four key types of drift:

1. **Code-spec misalignment** - Implementation doesn't match what specs declare
2. **Outdated specs** - Specifications that no longer reflect current needs
3. **Spec compression opportunities** - Multiple specs that could be consolidated
4. **Spec conflicts** - Contradictory requirements between different specs

## Behavior

The observer runs in a continuous loop, scanning the codebase and specs directory to identify drift. It records findings in the `observations/` directory as markdown files with YAML frontmatter for human review.

Unlike coach and builder agents, the observer does not make changes - it only observes and reports. Humans review observations with the coach agent to determine appropriate spec updates.

## Integration

- Observations are parsed by the same spec parser that handles specs and assertions
- CLI provides commands to launch observer in continuous monitoring mode
- Observer findings feed back into the spec-driven development cycle through human review