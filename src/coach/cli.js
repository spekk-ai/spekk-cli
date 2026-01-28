#!/usr/bin/env node

import { spawn } from 'node:child_process';
import { launchAgentWithPrompt } from '../cli/prompt-resolver.js';

async function launchCoachAgent() {
  try {
    const { activationMessage } = launchAgentWithPrompt('coach-agent');
    
    // Launch Claude Code with the coach agent message and prompt
    const claudeProcess = spawn('claude', ['--dangerously-skip-permissions'], {
      stdio: ['pipe', 'inherit', 'inherit']
    });
    
    // Handle stdin errors (like EPIPE when Claude exits quickly)
    claudeProcess.stdin.on('error', (error) => {
      // Ignore EPIPE errors when Claude exits quickly
      if (error.code !== 'EPIPE') {
        console.error('❌ Error writing to Claude stdin:', error.message);
      }
    });
    
    // Send the agent activation message with full prompt content
    try {
      claudeProcess.stdin.write(activationMessage + '\n');
      claudeProcess.stdin.end();
    } catch (error) {
      // Ignore EPIPE errors when Claude exits quickly
      if (error.code !== 'EPIPE') {
        throw error;
      }
    }
  
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
    
    await new Promise((resolve, reject) => {
      claudeProcess.on('exit', (code) => {
        if (code !== 0) {
          console.error(`❌ Claude Code exited with code ${code}`);
          reject(new Error(`Claude Code exited with code ${code}`));
        } else {
          resolve();
        }
      });
      
      // Handle interrupts gracefully
      process.on('SIGINT', () => {
        console.log('\n🛑 Stopping Coach Agent...');
        claudeProcess.kill('SIGINT');
        resolve();
      });
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