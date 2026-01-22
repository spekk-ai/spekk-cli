---
id: cli-command-launches-observer-loop
parent: observer-agent
created: 2026-01-22T17:00:00Z
priority: 2
status: in_progress
---

# CLI Command Launches Observer in Continuous Loop

A CLI command starts the observer agent in continuous monitoring mode.

## Success Criteria

- [ ] Command `npm run observer` exists in package.json
- [ ] Command launches observer agent with continuous loop behavior (NOT just JSON output)
- [ ] Observer runs indefinitely until manually stopped (Ctrl+C)
- [ ] Observer scans codebase and specs at regular intervals (every 30 seconds by default)
- [ ] Observer outputs progress/status messages to console during scanning
- [ ] Command accepts optional parameters (scan interval, quiet mode)
- [ ] Observer gracefully handles interruption signals
- [ ] Observer creates observations in observations/ directory when drift is detected

**Tests:** src/observer/__tests__/observer.test.js, src/observer/__tests__/cli.test.js