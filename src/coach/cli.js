#!/usr/bin/env node

import { readFile } from 'fs/promises';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

async function launchCoachAgent() {
  try {
    // Get the coach agent prompt from spekk-cli's internal specs
    const promptPath = join(__dirname, '../../specs/coach-agent/coach-agent.prompt.md');
    const promptContent = await readFile(promptPath, 'utf8');
    
    console.log('🏃‍♀️ Launching Coach Agent...');
    console.log('Working directory:', process.cwd());
    console.log('\n' + '='.repeat(80));
    console.log('COACH AGENT PROMPT');
    console.log('='.repeat(80));
    console.log(promptContent);
    console.log('='.repeat(80));
    console.log('\nYou are now the Coach Agent. Follow the prompt above.');
    console.log('You can read local specs and work in the current directory.');
    
  } catch (error) {
    console.error('Error launching Coach Agent:', error.message);
    process.exit(1);
  }
}

// Run if called directly
if (import.meta.url === `file://${process.argv[1]}`) {
  launchCoachAgent();
}

export { launchCoachAgent };