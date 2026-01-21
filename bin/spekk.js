#!/usr/bin/env node

import { run } from '../src/parser/cli.js';
import { launchCoachAgent } from '../src/coach/cli.js';
import { launchBuilderAgent } from '../src/builder/cli.js';
import { runBuilderLoop, runCoachLoop } from '../src/loops/index.js';
import { showStatus } from '../src/status/cli.js';

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
    await launchCoachAgent();
    break;
  
  case 'builder':
    await launchBuilderAgent();
    break;
  
  case 'status':
    await showStatus();
    break;
  
  case '--help':
  case '-h':
  case 'help':
    console.log(`
spekk - Spec-driven development CLI

USAGE:
  spekk [COMMAND]

COMMANDS:
  status    Show comprehensive overview of all specs and assertions
  coach     Launch the Coach Agent to create and refine specs
  builder   Launch the Builder Agent to implement specs
  loop      Run orchestration workflows (builder/coach loops)
  help      Show this help message

DEFAULT:
  When no command is provided, spekk runs the spec parser to find the next assertion.
`);
    break;
  
  default:
    // Default behavior: run the parser
    const allFlag = args.includes('--all');
    if (allFlag) {
      run({ all: true });
    } else {
      run();
    }
}