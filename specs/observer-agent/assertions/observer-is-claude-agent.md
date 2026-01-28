---
id: observer-is-claude-agent
parent: observer-agent
created: 2026-01-22T17:25:00Z
priority: 1
status: done
---

# Observer Is Claude Agent

The Observer must be implemented as a Claude agent (like Coach and Builder) that uses the observer-agent.prompt.md, not just a programmatic script.

## Success Criteria

- [ ] Observer runs as Claude agent instance with observer-agent.prompt.md
- [ ] Observer has access to standard Claude tools (Read, Grep, Bash, Edit, etc.)
- [ ] Observer can reason intelligently about code-spec alignment, not just pattern matching
- [ ] Observer understands context and semantics when detecting drift
- [ ] Observer makes reasoned decisions about severity and impact
- [ ] Observer can analyze complex relationships between specs and implementation
- [ ] Observer creates thoughtful observations based on understanding, not just file existence checks
- [ ] Current programmatic drift detection becomes a tool the Claude agent can optionally use

## Context

The current implementation in `src/observer/index.js` is just a Node.js script doing regex patterns and file checks. This should be replaced with (or supplemented by) an actual Claude agent that can think critically about drift and misalignment.