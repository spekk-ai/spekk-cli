#!/usr/bin/env node

import { run } from '../src/parser/cli.js';
import { launchCoachAgent } from '../src/coach/cli.js';
import { launchBuilderAgent } from '../src/builder/cli.js';
import { launchObserverAgent } from '../src/observer/cli.js';
import { runBuilderLoop, runCoachLoop } from '../src/loops/index.js';
import { showStatus } from '../src/status/cli.js';
import { showSpekk } from '../src/show/cli.js';
import { parseFlags } from '../src/cli/parse-flags.js';

// Parse command line arguments
const args = process.argv.slice(2);
const command = args[0];
const subcommand = args[1];

switch (command) {
  case 'loop':
    switch (subcommand) {
      case 'builder':
        await runBuilderLoop();
        break;
      case 'coach':
        await runCoachLoop();
        break;
      case '--help':
      case '-h':
      case 'help':
        console.log(`
spekk loop - Orchestration workflows for spec-driven development

USAGE:
  spekk loop [COMMAND]

COMMANDS:
  builder   Run the automated builder loop (gets next assertion, implements, commits, repeats)
  coach     Run the interactive coach loop (create specs, commit, repeat)
  help      Show this help message
`);
        break;
      default:
        console.log('Unknown loop command. Use "spekk loop help" for available commands.');
        process.exit(1);
    }
    break;

  case 'coach':
    await launchCoachAgent(args.slice(1));
    break;
  
  case 'builder':
    await launchBuilderAgent(args.slice(1));
    break;
  
  case 'observer':
    await launchObserverAgent(args.slice(1));
    break;

  case 'status':
    await showStatus();
    break;
  
  case 'show': {
    const showFlags = parseFlags(args.slice(1), {
      watch: { flags: ['--watch', '-w'], type: 'boolean' },
    });
    await showSpekk(showFlags);
    break;
  }
  
  case '--help':
  case '-h':
  case 'help':
    console.log(`
spekk - Spec-driven development CLI

USAGE:
  spekk [COMMAND]

COMMANDS:
  show      Generate and display spec explorer web interface (-w to watch)
  status    Show comprehensive overview of all specs and assertions
  coach     Launch the Coach Agent to create and refine specs
              Use "spekk coach meeting [file]" for meeting transcript processing
  builder   Launch the Builder Agent to implement specs
  observer  Launch the Observer Agent to monitor spec-code drift
  loop      Run orchestration workflows (builder/coach loops)
  help      Show this help message

DEFAULT:
  When no command is provided, spekk runs the spec parser to find the next assertion.
`);
    break;
  
  default:
    // Default behavior: run the parser with flags
    const options = {};
    for (let i = 0; i < args.length; i++) {
      const arg = args[i];
      switch (arg) {
        case '--all':
          options.all = true;
          break;
        case '--spec':
        case '-s':
          options.spec = args[++i];
          break;
        case '--assertion':
          options.assertion = args[++i];
          break;
      }
    }
    run(options);
}