#!/usr/bin/env node

import { spawn } from 'node:child_process';

async function launchCoachAgent() {
  try {
    console.log('🏃‍♀️ Launching Coach Agent with Claude Code...');
    console.log('Working directory:', process.cwd());
    console.log('Press Ctrl+C to exit the coaching session.');
    console.log(''); // Empty line for better readability
    
    // Launch Claude Code with the coach agent message
    const claudeProcess = spawn('claude', ['--dangerously-skip-permissions'], {
      stdio: ['pipe', 'inherit', 'inherit']
    });
    
    // Send the agent activation message
    claudeProcess.stdin.write('You are the Coach Agent - read the prompt and follow the instructions exactly.\n');
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
      console.log('\n🛑 Stopping Coach Agent...');
      claudeProcess.kill('SIGINT');
      process.exit(0);
    });
    
  } catch (error) {
    console.error('❌ Error launching Coach Agent:', error.message);
    process.exit(1);
  }
}

// Run if called directly
if (import.meta.url === `file://${process.argv[1]}`) {
  launchCoachAgent();
}

export { launchCoachAgent };