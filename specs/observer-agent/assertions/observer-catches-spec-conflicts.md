---
id: observer-catches-spec-conflicts
parent: observer-agent
created: 2026-01-22T17:00:00Z
priority: 2
status: done
---

# Observer Catches Spec Conflicts

The observer detects contradictory requirements between different specifications that need resolution.

## Success Criteria

- [ ] Observer identifies specs with mutually exclusive requirements
- [ ] Observer detects conflicting file/directory structure requirements
- [ ] Observer finds contradictory behavior specifications for same features
- [ ] Observer identifies priority conflicts (high-priority specs blocking each other)
- [ ] Observer creates observations highlighting specific conflict details
- [ ] Observer references all conflicting specs in each conflict observation
- [ ] Observer suggests which conflicts are blocking vs informational