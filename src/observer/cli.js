#!/usr/bin/env node

import { spawn } from 'node:child_process';

async function launchObserverAgent(cliArgs = null) {
  try {
    // Parse command line arguments  
    // When called from main CLI, use provided args; otherwise use process.argv
    const args = cliArgs || process.argv.slice(2);
    
    // Handle help first
    if (args.includes('--help') || args.includes('-h')) {
      showHelp();
      return;
    }
    
    // Check for programmatic mode (legacy support)
    if (args.includes('--programmatic')) {
      console.log('🔍 Launching programmatic observer (legacy mode)...');
      const { startObserver } = await import('./programmatic.js');
      const options = parseOptions(args);
      await startObserver(options);
      return;
    }

    console.log('🔍 Launching Observer Agent with Claude Code...');
    console.log('Working directory:', process.cwd());
    
    // Parse options to pass context to Claude agent
    const options = parseOptions(args);
    let contextMessage = 'You are the Observer Agent - read the prompt and follow the instructions exactly.';
    
    if (Object.keys(options).length > 0) {
      contextMessage += '\n\nCLI Options provided:\n';
      if (options.scanInterval) {
        contextMessage += `- Scan interval: ${options.scanInterval} seconds\n`;
      }
      if (options.quiet) {
        contextMessage += '- Quiet mode: enabled\n';
      }
      contextMessage += '\nYou can use these preferences in your monitoring approach. The programmatic observer tool is available in src/observer/programmatic.js if you want to use it.';
    }
    
    console.log('Press Ctrl+C to exit the observation session.');
    console.log(''); // Empty line for better readability
    
    // Launch Claude Code with the observer agent message
    const claudeProcess = spawn('claude', ['--dangerously-skip-permissions'], {
      stdio: ['pipe', 'inherit', 'inherit']
    });
    
    // Send the agent activation message with context
    claudeProcess.stdin.write(contextMessage + '\n');
    claudeProcess.stdin.end();
    
    // Handle process events
    claudeProcess.on('error', (error) => {
      if (error.code === 'ENOENT') {
        console.error('❌ Error: Claude Code CLI not found. Please install Claude Code first.');
        console.error('Visit: https://claude.ai/code for installation instructions.');
      } else {
        console.error('❌ Error launching Claude Code:', error.message);
      }
      process.exit(1);
    });
    
    claudeProcess.on('exit', (code) => {
      if (code !== 0) {
        console.error(`❌ Claude Code exited with code ${code}`);
        process.exit(code);
      }
    });
    
    // Handle interrupts gracefully
    process.on('SIGINT', () => {
      console.log('\n🛑 Stopping Observer Agent...');
      claudeProcess.kill('SIGINT');
      process.exit(0);
    });
    
  } catch (error) {
    console.error('❌ Error launching Observer Agent:', error.message);
    process.exit(1);
  }
}

function parseOptions(args) {
  const options = {};
  
  for (let i = 0; i < args.length; i++) {
    const arg = args[i];
    
    if (arg === '--interval' && i + 1 < args.length) {
      const interval = parseInt(args[i + 1]);
      if (!isNaN(interval) && interval > 0) {
        options.scanInterval = interval;
      } else {
        console.error('❌ Error: --interval must be a positive number');
        process.exit(1);
      }
      i++; // Skip next arg since it's the interval value
    } else if (arg === '--quiet') {
      options.quiet = true;
    } else if (arg === '--programmatic') {
      // Already handled above, skip
      continue;
    } else if (arg !== '--help' && arg !== '-h') {
      console.error(`❌ Unknown option: ${arg}`);
      showHelp();
      process.exit(1);
    }
  }
  
  return options;
}

function showHelp() {
  console.log(`
🔍 Observer Agent - Intelligent spec-code drift monitoring with Claude

Usage: npm run observer [options]

Options:
  --interval <seconds>   Preferred scan interval (Claude agent can adjust)
  --quiet               Preference for minimal output (Claude agent decides)
  --programmatic        Use legacy programmatic observer (Node.js script)
  --help, -h            Show this help message

Examples:
  npm run observer                      # Launch Claude agent observer
  npm run observer -- --interval 60    # Claude agent with 60s interval preference  
  npm run observer -- --quiet          # Claude agent with quiet preference
  npm run observer -- --programmatic   # Use legacy programmatic observer
  
The Observer Agent is now a Claude agent that:
- Uses intelligent reasoning to detect spec-code drift
- Understands context and semantics, not just pattern matching
- Makes thoughtful observations based on understanding
- Can analyze complex relationships between specs and implementation
- Has access to all Claude tools (Read, Grep, Bash, Edit, etc.)
- Can optionally use the programmatic observer as a tool

The observer detects four types of drift:
- Code-spec misalignment (missing commands, files, behavior mismatch)
- Outdated specifications (specs no longer reflect current needs)
- Spec compression opportunities (multiple specs that could merge)
- Spec conflicts (contradictory requirements between specs)

Observations are written to observations/ directory for human review.
`);
}

// Run if called directly
if (import.meta.url === `file://${process.argv[1]}`) {
  launchObserverAgent();
}

export { launchObserverAgent };