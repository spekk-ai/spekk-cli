---
id: npm-scripts-launch-agents
parent: spekk-cli
created: 2026-01-21T19:30:00Z
priority: 1
status: not_started
---

# NPM Scripts Launch Agents

## Requirement

`npm run coach` and `npm run builder` commands launch actual Claude Code agent sessions with the appropriate prompts, providing the looping behavior described in the orchestration specs.

## Success Criteria

### Coach Script (`npm run coach`):
- Launches Claude Code with coach agent prompt from `specs/coach-agent/coach-agent.prompt.md`
- Provides interactive session for spec creation
- Loops back to coach after each spec is created and committed
- Continues until user exits (Ctrl+C)
- Operates on current working directory's specs

### Builder Script (`npm run builder`):
- Launches Claude Code with builder agent prompt from `specs/builder-agent/builder-agent.prompt.md`
- Gets next priority assertion via spec parser
- Launches builder agent with assertion context
- Automatically commits changes after completion
- Loops to next assertion until none remain
- Handles interrupts gracefully

### Current Issue

Both scripts currently only display their prompts instead of launching Claude Code sessions:

```javascript
// Current behavior: just prints prompt
console.log(promptContent);
console.log('You are now the Builder Agent...');
```

### Required Behavior

Scripts should:
1. Launch Claude Code with the appropriate prompt file
2. Handle the session lifecycle (start, monitor, restart on completion)
3. Provide the looping behavior that matches coach-loop.sh and builder-loop.sh patterns
4. Integrate with git for automatic commits
5. Show status and progress indicators

**Tests:** app/cli/__tests__/npm-scripts.test.js