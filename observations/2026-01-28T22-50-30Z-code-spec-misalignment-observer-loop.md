---
id: code-spec-misalignment-observer-loop
created: 2026-01-28T22:50:30Z
type: code_spec_misalignment
severity: high
affected_specs:
  - observer-agent
affected_files:
  - src/observer/cli.js
  - specs/observer-agent/assertions/cli-command-launches-observer-loop.md
---

# Observer CLI Doesn't Implement Continuous Monitoring Loop

## Issue Description
The observer CLI command is marked as "done" but doesn't implement the continuous monitoring loop as specified. Instead, it launches Claude Code agent, requiring manual `--programmatic` flag for the actual monitoring functionality.

## Evidence
From the spec `/specs/observer-agent/assertions/cli-command-launches-observer-loop.md`:
```
The CLI command launches the observer in an infinite monitoring loop that:
- Runs every 30 seconds (configurable)
- Scans the entire codebase
- Creates observations when drift is detected
- Outputs progress to console
```

Actual implementation in `src/observer/cli.js`:
```javascript
if (process.argv.includes('--programmatic')) {
  // Only runs continuous loop with special flag
  runProgrammaticObserver(interval);
} else {
  // Default: Launches Claude Code agent, not continuous loop
  launchClaudeWithPrompt(observerPromptPath, { scanInterval: `${interval} seconds` });
}
```

## Impact
- Users running `npm run observer` get Claude Code agent instead of automated monitoring
- The continuous monitoring loop is hidden behind an undocumented flag
- Spec marked as "done" but primary functionality not accessible by default
- Contradicts the specification's intent for automated drift detection

## Recommendation
Either:
1. Make continuous monitoring the default behavior for `npm run observer`
2. Update the spec to clarify that observer launches Claude Code by default
3. Add clear documentation about the `--programmatic` flag
4. Update assertion status to "in_progress" until resolved