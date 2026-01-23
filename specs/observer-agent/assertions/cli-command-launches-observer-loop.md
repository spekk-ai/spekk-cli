---
id: cli-command-launches-observer-loop
parent: observer-agent
created: 2026-01-22T17:00:00Z
priority: 2
status: done
---

# CLI Command Launches Observer in Continuous Loop

A CLI command starts the observer agent in continuous monitoring mode.

## Success Criteria

- [ ] Command `npm run observer` exists in package.json
- [ ] Command `spekk observer` is available in main CLI (bin/spekk.js)
- [ ] Both commands launch observer agent with continuous loop behavior (NOT JSON output)
- [ ] Observer runs indefinitely until manually stopped (Ctrl+C)
- [ ] Observer scans codebase and specs at regular intervals (every 30 seconds by default)
- [ ] Observer outputs progress/status messages to console during scanning
- [ ] Commands accept optional parameters (scan interval, quiet mode)
- [ ] Observer gracefully handles interruption signals
- [ ] Observer creates observations in observations/ directory when drift is detected

**Tests:** src/observer/__tests__/observer.test.js, src/observer/__tests__/cli.test.js