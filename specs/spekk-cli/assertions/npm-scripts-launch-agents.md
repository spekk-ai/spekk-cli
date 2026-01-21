---
id: npm-scripts-launch-agents
parent: spekk-cli
created: 2026-01-21T19:30:00Z
priority: 1
status: done
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

Simple bash loops that:

**Coach Script (`npm run coach`):**
```bash
#!/bin/bash
while true; do
  claude --dangerously-skip-permissions << 'EOF'
You are the Coach Agent - read the prompt and follow the instructions exactly.
EOF
  # Loop continues until user Ctrl+C
done
```

**Builder Script (`npm run builder`):**
```bash
#!/bin/bash
while true; do
  # Get next assertion
  NEXT=$(npm run next --silent 2>/dev/null)
  
  # Check if all done
  if [[ $NEXT == *'"type":"complete"'* ]]; then
    echo "🎉 All assertions completed!"
    break
  fi
  
  # Launch builder agent
  claude --dangerously-skip-permissions << 'EOF'
You are the Builder Agent - read the prompt and follow the instructions exactly.
EOF
done
```

Keep it simple - just bash loops around Claude Code with hard-coded prompts, like coach-loop.sh and builder-loop.sh patterns.

**Tests:** src/coach/__tests__/coach-cli.test.js, src/builder/__tests__/builder-cli.test.js