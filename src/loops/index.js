import { execSync, spawn } from 'node:child_process';
import { readFile } from 'fs/promises';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';

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

export async function runBuilderLoop() {
  colorLog('cyan', '🔧 Starting Builder Loop...');
  colorLog('blue', 'This will continuously get next assertions and implement them.');
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
        nextResult = execSync('node src/parser/cli.js', { 
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
        colorLog('green', '🎉 All assertions completed! No more work to do.');
        break;
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
      
      // Step 2: Launch builder agent with assertion context
      colorLog('magenta', '🤖 Launching Builder Agent...');
      
      try {
        colorLog('cyan', '   Agent Context: Working on assertion ' + assertion.id);
        
        // Launch Claude Code with the builder agent message
        const claudeProcess = spawn('claude', ['--dangerously-skip-permissions'], {
          stdio: ['pipe', 'inherit', 'inherit']
        });
        
        // Send the agent activation message
        claudeProcess.stdin.write('You are the Builder Agent - read the prompt and follow the instructions exactly.\n');
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
      
      // Step 3: Commit changes with proper commit message
      colorLog('blue', '📝 Committing changes...');
      
      try {
        // Check git status
        const gitStatus = execSync('git status --porcelain', { 
          encoding: 'utf8',
          stdio: ['pipe', 'pipe', 'pipe']
        });
        
        if (gitStatus.trim()) {
          // Stage changes
          execSync('git add .', { stdio: ['pipe', 'pipe', 'pipe'] });
          
          // Create commit message
          const commitMessage = `Complete ${assertion.id}\n\n${assertion.title}`;
          
          // Commit
          execSync(`git commit -m "${commitMessage}"`, { 
            stdio: ['pipe', 'pipe', 'pipe']
          });
          
          colorLog('green', '   ✅ Changes committed successfully');
        } else {
          colorLog('yellow', '   ⚠️  No changes to commit');
        }
        
      } catch (error) {
        colorLog('red', '❌ Git operations failed:');
        console.error(error.message);
        // Don't exit - continue with next assertion
      }
      
      // Step 4: Brief pause before next iteration
      colorLog('blue', '⏳ Preparing for next iteration...');
      await new Promise(resolve => setTimeout(resolve, 500));
    }
    
  } catch (error) {
    colorLog('red', '❌ Builder loop encountered an error:');
    console.error(error.message);
    process.exit(1);
  }
  
  colorLog('green', '🏁 Builder loop completed successfully!');
}

export async function runCoachLoop() {
  colorLog('cyan', '🏃‍♀️ Starting Coach Loop...');
  colorLog('blue', 'This will launch the coach agent for interactive spec creation.');
  colorLog('yellow', 'Press Ctrl+C to exit gracefully.');
  
  // Handle interrupts gracefully
  process.on('SIGINT', () => handleInterrupt('SIGINT'));
  process.on('SIGTERM', () => handleInterrupt('SIGTERM'));
  
  let sessionCount = 0;
  
  try {
    while (true) {
      sessionCount++;
      colorLog('bright', `\n--- Coach Loop Session ${sessionCount} ---`);
      
      // Step 1: Launch coach agent in interactive mode
      colorLog('magenta', '🤖 Launching Coach Agent...');
      
      try {
        colorLog('blue', '   Interactive mode starting...');
        
        // Launch Claude Code with the coach agent message
        const claudeProcess = spawn('claude', ['--dangerously-skip-permissions'], {
          stdio: ['pipe', 'inherit', 'inherit']
        });
        
        // Send the agent activation message
        claudeProcess.stdin.write('You are the Coach Agent - read the prompt and follow the instructions exactly.\n');
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
            if (code === 0) {
              colorLog('green', '   ✅ Coach session completed');
              resolve();
            } else {
              colorLog('red', `❌ Claude Code exited with code ${code}`);
              reject(new Error(`Claude Code exited with code ${code}`));
            }
          });
        });
        
      } catch (error) {
        colorLog('red', '❌ Coach agent failed:');
        console.error(error.message);
        process.exit(1);
      }
      
      // Step 2: Check for new specs and commit them
      colorLog('blue', '📝 Checking for new specs to commit...');
      
      try {
        // Check git status for new spec files
        const gitStatus = execSync('git status --porcelain', { 
          encoding: 'utf8',
          stdio: ['pipe', 'pipe', 'pipe']
        });
        
        if (gitStatus.trim()) {
          // Check if there are spec files
          const hasSpecs = gitStatus.includes('specs/') || gitStatus.includes('.md');
          
          if (hasSpecs) {
            // Stage spec changes
            execSync('git add specs/', { stdio: ['pipe', 'pipe', 'pipe'] });
            
            // Create commit message for specs
            const commitMessage = `Add new specs from coach session ${sessionCount}`;
            
            // Commit
            execSync(`git commit -m "${commitMessage}"`, { 
              stdio: ['pipe', 'pipe', 'pipe']
            });
            
            colorLog('green', '   ✅ New specs committed successfully');
          } else {
            colorLog('yellow', '   ⚠️  No new specs to commit');
          }
        } else {
          colorLog('yellow', '   ⚠️  No changes detected');
        }
        
      } catch (error) {
        colorLog('red', '❌ Git operations failed:');
        console.error(error.message);
        // Don't exit - continue with next session
      }
      
      // Step 3: Ask user if they want to continue (simulate for now)
      colorLog('blue', '❓ Coach would ask: Continue with another session? (y/n)');
      
      // For simulation, we'll exit after first iteration
      // In real implementation, this would be interactive
      colorLog('yellow', '   📝 Simulating user choosing to exit...');
      break;
    }
    
  } catch (error) {
    colorLog('red', '❌ Coach loop encountered an error:');
    console.error(error.message);
    process.exit(1);
  }
  
  colorLog('green', '🏁 Coach loop completed successfully!');
}