#!/usr/bin/env node

import { spawn } from 'node:child_process';
import { existsSync, readFileSync } from 'node:fs';
import { resolve, join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { launchAgentWithPrompt } from '../cli/prompt-resolver.js';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const projectRoot = join(__dirname, '../..');

// Mapping from subcommand name to skill filename in specs/coach-skills-system/
const SKILL_MAP = {
  meeting: 'meeting-notes-to-specs-skill.md',
};

/**
 * Resolve and read a skill markdown file for a given subcommand.
 * Returns the full content of the skill file.
 * Throws if the subcommand has no mapped skill or the file is missing.
 */
function resolveSkillContent(subcommand) {
  const skillFilename = SKILL_MAP[subcommand];
  if (!skillFilename) {
    return null;
  }
  const skillPath = join(projectRoot, 'specs', 'coach-skills-system', skillFilename);
  if (!existsSync(skillPath)) {
    throw new Error(`Skill file not found: ${skillPath}`);
  }
  return readFileSync(skillPath, 'utf8');
}

/**
 * Build the activation message for a skill subcommand.
 * Exported for testing.
 */
function buildSkillActivationMessage(baseMessage, subcommand, args) {
  const skillContent = resolveSkillContent(subcommand);
  if (!skillContent) {
    return baseMessage;
  }

  let message = baseMessage;
  message += '\n\n---\n\n**Skill Activation: `spekk coach ' + subcommand + '`**\n\n';
  message += 'The user has launched you with a skill active via `spekk coach ' + subcommand + '`.\n';
  message += 'Follow the inlined skill workflow below immediately — do not wait for trigger detection.\n';
  message += '\n<skill-content>\n' + skillContent + '\n</skill-content>\n';

  // Subcommand-specific argument handling
  if (subcommand === 'meeting') {
    const transcriptFile = args[1];
    if (transcriptFile) {
      const resolvedPath = resolve(process.cwd(), transcriptFile);
      if (!existsSync(resolvedPath)) {
        throw new Error(`Transcript file not found: ${resolvedPath}`);
      }
      const content = readFileSync(resolvedPath, 'utf8');
      message += `\nThe user provided a transcript file: ${transcriptFile}\n`;
      message += '\n<transcript>\n' + content + '\n</transcript>\n';
      message += '\nProcess this transcript now.\n';
    } else {
      message += '\nNo transcript file was provided. Ask the user to paste or provide their meeting transcript.\n';
    }
  }

  return message;
}

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
    let message;

    // Handle skill subcommands
    if (subcommand && SKILL_MAP[subcommand]) {
      try {
        message = buildSkillActivationMessage(activationMessage, subcommand, args);
      } catch (error) {
        console.error(`Error: ${error.message}`);
        process.exit(1);
      }
    } else {
      message = activationMessage;
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

OPTIONS:
  --help, -h       Show this help message

EXAMPLES:
  spekk coach                          # Launch interactive coach
  spekk coach meeting                  # Launch coach in meeting mode (prompts for transcript)
  spekk coach meeting notes.txt        # Process a transcript file
`);
}

// Run if called directly
if (import.meta.url === `file://${process.argv[1]}`) {
  launchCoachAgent();
}

export { launchCoachAgent, buildSkillActivationMessage, resolveSkillContent, SKILL_MAP };
