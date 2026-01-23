#!/usr/bin/env node

import { startObserver } from './index.js';

async function launchObserverAgent(cliArgs = null) {
  try {
    // Parse command line arguments
    // When called from main CLI, use provided args; otherwise use process.argv
    const args = cliArgs || process.argv.slice(2);
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
      } else if (arg === '--help' || arg === '-h') {
        showHelp();
        return;
      } else {
        console.error(`❌ Unknown option: ${arg}`);
        showHelp();
        process.exit(1);
      }
    }

    console.log('🔍 Launching Observer Agent...');
    console.log('Working directory:', process.cwd());
    if (options.scanInterval) {
      console.log(`Scan interval: ${options.scanInterval}s`);
    }
    if (options.quiet) {
      console.log('Quiet mode: enabled');
    }
    console.log('Press Ctrl+C to stop monitoring.');
    console.log(''); // Empty line for better readability
    
    await startObserver(options);
    
  } catch (error) {
    console.error('❌ Error launching Observer Agent:', error.message);
    process.exit(1);
  }
}

function showHelp() {
  console.log(`
🔍 Observer Agent - Continuous spec-code drift monitoring

Usage: npm run observer [options]

Options:
  --interval <seconds>   Scan interval in seconds (default: 30)
  --quiet               Suppress progress messages
  --help, -h            Show this help message

Examples:
  npm run observer                    # Start with default settings
  npm run observer -- --interval 60  # Scan every 60 seconds
  npm run observer -- --quiet        # Run silently
  
The observer continuously monitors for drift between specs and implementation:
- Code-spec misalignment (missing commands, files, etc.)
- Outdated specifications 
- Spec compression opportunities
- Conflicting requirements

Observations are written to observations/ directory for human review.
`);
}

// Run if called directly
if (import.meta.url === `file://${process.argv[1]}`) {
  launchObserverAgent();
}

export { launchObserverAgent };