#!/usr/bin/env node

import { spawn } from 'node:child_process';
import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { launchAgentWithPrompt } from '../cli/prompt-resolver.js';

async function launchCoachAgent(cliArgs = null) {
  try {
    const args = cliArgs || process.argv.slice(2);
    const subcommand = args[0];

    // Handle help
    if (subcommand === '--help' || subcommand === '-h' || subcommand === 'help') {
      showHelp();
      return;
    }

    const { activationMessage } = launchAgentWithPrompt('coach-agent');
    let message = activationMessage;

    // Handle meeting subcommand
    if (subcommand === 'meeting') {
      const transcriptFile = args[1];
      message += '\n\n---\n\n**Skill Activation: Meeting Notes to Specs**\n\n';
      message += 'The user has launched you with the meeting-processing skill active via `spekk coach meeting`.\n';
      message += 'Activate your meeting-notes-to-specs skill immediately — do not wait for trigger detection.\n';

      if (transcriptFile) {
        const resolvedPath = resolve(process.cwd(), transcriptFile);
        if (existsSync(resolvedPath)) {
          const content = readFileSync(resolvedPath, 'utf8');
          message += `\nThe user provided a transcript file: ${transcriptFile}\n`;
          message += '\n<transcript>\n' + content + '\n</transcript>\n';
          message += '\nProcess this transcript now.\n';
        } else {
          console.error(`Error: Transcript file not found: ${resolvedPath}`);
          process.exit(1);
        }
      } else {
        message += '\nNo transcript file was provided. Ask the user to paste or provide their meeting transcript.\n';
      }
    }

    // Handle coordinate subcommand
    if (subcommand === 'coordinate') {
      message += '\n\n---\n\n**Skill Activation: Work Coordination & Dependency Analysis**\n\n';
      message += 'The user has launched you with the coordinator skill active via `spekk coach coordinate`.\n';
      message += 'Activate your coordinator skill immediately — do not wait for trigger detection.\n\n';
      message += '**Your Coordinator Workflow:**\n\n';
      message += '1. Analyze all draft and not_started assertions across specs/\n';
      message += '2. Build dependency graph (single-parent chains)\n';
      message += '3. Identify parallelizable vs serial work\n';
      message += '4. Group related assertions into feature branch clusters\n';
      message += '5. Assign semantic branch names (e.g., feature/chat-system)\n';
      message += '6. Present the plan for user confirmation\n';
      message += '7. Update YAML frontmatter with `depends-on` and `branch` fields\n';
      message += '8. Commit changes with clear summary\n\n';
      message += '**Key Rules:**\n';
      message += '- Use single-parent dependencies only (A depends-on B, not [B, C])\n';
      message += '- Omit `depends-on` field when no dependency (sparse YAML)\n';
      message += '- Group dependent assertions into same branch\n';
      message += '- Isolated assertions can stay on main or group into "quick-wins"\n';
      message += '- Present plan BEFORE updating files\n';
      message += '- Show dependency tree visually (ASCII art or markdown)\n\n';
      message += 'Start by scanning specs/ for draft and not_started assertions.\n';
    }

    // Launch Claude Code with the coach agent message and prompt
    const claudeProcess = spawn('claude', ['--dangerously-skip-permissions'], {
      stdio: ['pipe', 'inherit', 'inherit']
    });

    // Handle stdin errors (like EPIPE when Claude exits quickly)
    claudeProcess.stdin.on('error', (error) => {
      // Ignore EPIPE errors when Claude exits quickly
      if (error.code !== 'EPIPE') {
        console.error('Error writing to Claude stdin:', error.message);
      }
    });

    // Send the agent activation message with full prompt content
    try {
      claudeProcess.stdin.write(message + '\n');
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
        console.error('Error: Claude Code CLI not found. Please install Claude Code first.');
        console.error('Visit: https://claude.ai/code for installation instructions.');
      } else {
        console.error('Error launching Claude Code:', error.message);
      }
      process.exit(1);
    });

    await new Promise((resolve, reject) => {
      claudeProcess.on('exit', (code) => {
        if (code !== 0) {
          console.error(`Claude Code exited with code ${code}`);
          reject(new Error(`Claude Code exited with code ${code}`));
        } else {
          resolve();
        }
      });

      // Handle interrupts gracefully
      process.on('SIGINT', () => {
        console.log('\nStopping Coach Agent...');
        claudeProcess.kill('SIGINT');
        resolve();
      });
    });

  } catch (error) {
    console.error('Error launching Coach Agent:', error.message);
    process.exit(1);
  }
}

function showHelp() {
  console.log(`
spekk coach - Launch the Coach Agent

USAGE:
  spekk coach [SUBCOMMAND] [OPTIONS]

SUBCOMMANDS:
  meeting [file]   Launch coach with meeting-processing skill active
                   If a transcript file is provided, it will be processed immediately.
                   Without a file, the coach will prompt for a transcript.

  coordinate       Launch coach with work coordination skill active
                   Analyzes specs, builds dependency graphs, assigns branches.

OPTIONS:
  --help, -h       Show this help message

EXAMPLES:
  spekk coach                          # Launch interactive coach
  spekk coach meeting                  # Launch coach in meeting mode (prompts for transcript)
  spekk coach meeting notes.txt        # Process a transcript file
  spekk coach coordinate               # Launch coach in coordinator mode
`);
}

// Run if called directly
if (import.meta.url === `file://${process.argv[1]}`) {
  launchCoachAgent();
}

export { launchCoachAgent };
