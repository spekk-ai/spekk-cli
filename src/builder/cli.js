#!/usr/bin/env node

import { spawn, execSync } from 'node:child_process';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';
import { launchAgentWithPrompt } from '../cli/prompt-resolver.js';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

// Colors for console output
const colors = {
  reset: '\x1b[0m',
  bright: '\x1b[1m',
  red: '\x1b[31m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  magenta: '\x1b[35m',
  cyan: '\x1b[36m'
};

function colorLog(color, message) {
  console.log(`${colors[color]}${message}${colors.reset}`);
}

function handleInterrupt(signal) {
  colorLog('yellow', `\n🛑 Received ${signal}. Exiting gracefully...`);
  process.exit(0);
}

async function launchBuilderAgent() {
  colorLog('cyan', '🔧 Starting Builder Agent Loop...');
  colorLog('blue', 'This will continuously get next assertions and launch Claude Code to implement them.');
  colorLog('yellow', 'Press Ctrl+C to exit gracefully.');
  
  // Handle interrupts gracefully
  process.on('SIGINT', () => handleInterrupt('SIGINT'));
  process.on('SIGTERM', () => handleInterrupt('SIGTERM'));
  
  let iterationCount = 0;
  
  try {
    while (true) {
      iterationCount++;
      colorLog('bright', `\n--- Builder Loop Iteration ${iterationCount} ---`);
      
      // Step 1: Get next priority assertion
      colorLog('blue', '📋 Getting next priority assertion...');
      let nextResult;
      try {
        const parserPath = join(__dirname, '../parser/cli.js');
        nextResult = execSync(`node "${parserPath}"`, { 
          encoding: 'utf8',
          stdio: ['pipe', 'pipe', 'pipe']
        });
      } catch (error) {
        colorLog('red', '❌ Failed to get next assertion:');
        console.error(error.message);
        process.exit(1);
      }
      
      let parsedResult;
      try {
        parsedResult = JSON.parse(nextResult);
      } catch (error) {
        colorLog('red', '❌ Invalid JSON from parser:');
        console.log(nextResult);
        process.exit(1);
      }
      
      // Check if we have any assertions to work on
      if (parsedResult.type === 'complete') {
        colorLog('green', '✨ All assertions completed. Waiting for new work...');
        // Wait a bit before checking again
        await new Promise(resolve => setTimeout(resolve, 5000));
        continue;
      }
      
      if (parsedResult.type !== 'assertion') {
        colorLog('red', '❌ Unexpected result from parser:');
        console.log(JSON.stringify(parsedResult, null, 2));
        process.exit(1);
      }
      
      const assertion = parsedResult;
      colorLog('green', `📄 Working on: ${assertion.id} (${assertion.title})`);
      colorLog('blue', `   File: ${assertion.file}`);
      colorLog('blue', `   Status: ${assertion.status}`);
      colorLog('blue', `   Priority: ${assertion.priority}`);
      
      // Step 2: Launch Claude Code with builder agent
      colorLog('magenta', '🤖 Launching Claude Code Builder Agent...');
      
      try {
        const { activationMessage, resolver } = launchAgentWithPrompt('builder-agent');
        
        // Launch Claude Code with the builder agent message and prompt
        const claudeProcess = spawn('claude', ['--dangerously-skip-permissions'], {
          stdio: ['pipe', 'inherit', 'inherit']
        });
        
        // Send the agent activation message with full prompt content
        claudeProcess.stdin.write(activationMessage + '\n');
        claudeProcess.stdin.end();
        
        // Wait for Claude Code to complete
        await new Promise((resolve, reject) => {
          claudeProcess.on('error', (error) => {
            if (error.code === 'ENOENT') {
              colorLog('red', '❌ Error: Claude Code CLI not found. Please install Claude Code first.');
              colorLog('blue', 'Visit: https://claude.ai/code for installation instructions.');
            } else {
              colorLog('red', '❌ Error launching Claude Code: ' + error.message);
            }
            reject(error);
          });
          
          claudeProcess.on('exit', (code) => {
            resolver.cleanupCopiedFiles();
            if (code === 0) {
              colorLog('green', '   ✅ Builder agent completed work');
              resolve();
            } else {
              colorLog('red', `❌ Claude Code exited with code ${code}`);
              reject(new Error(`Claude Code exited with code ${code}`));
            }
          });
        });
        
      } catch (error) {
        colorLog('red', '❌ Builder agent failed:');
        console.error(error.message);
        process.exit(1);
      }
      
      // Step 3: Check if there are more assertions
      colorLog('blue', '🔄 Checking for more work...');
      
      // Brief pause before next iteration
      await new Promise(resolve => setTimeout(resolve, 1000));
    }
    
  } catch (error) {
    colorLog('red', '❌ Builder loop encountered an error:');
    console.error(error.message);
    process.exit(1);
  }
  
  colorLog('green', '🏁 Builder loop completed successfully!');
}

// Run if called directly
if (import.meta.url === `file://${process.argv[1]}`) {
  launchBuilderAgent();
}

export { launchBuilderAgent };