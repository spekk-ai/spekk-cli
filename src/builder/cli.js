#!/usr/bin/env node

import { readFile } from 'fs/promises';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

async function launchBuilderAgent() {
  try {
    // Get the builder agent prompt from spekk-cli's internal specs
    const promptPath = join(__dirname, '../../specs/builder-agent/builder-agent.prompt.md');
    const promptContent = await readFile(promptPath, 'utf8');
    
    console.log('🔧 Launching Builder Agent...');
    console.log('Working directory:', process.cwd());
    console.log('\n' + '='.repeat(80));
    console.log('BUILDER AGENT PROMPT');
    console.log('='.repeat(80));
    console.log(promptContent);
    console.log('='.repeat(80));
    console.log('\nYou are now the Builder Agent. Follow the prompt above.');
    console.log('You can read local specs and work in the current directory.');
    
  } catch (error) {
    console.error('Error launching Builder Agent:', error.message);
    process.exit(1);
  }
}

// Run if called directly
if (import.meta.url === `file://${process.argv[1]}`) {
  launchBuilderAgent();
}

export { launchBuilderAgent };