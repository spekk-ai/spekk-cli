#!/usr/bin/env node

import { spawn, execSync } from 'node:child_process';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';
import { createInterface } from 'readline';
import { launchAgentWithPrompt } from '../cli/prompt-resolver.js';
import { parseFlags as sharedParseFlags } from '../cli/parse-flags.js';

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

// Track the active Claude child process so SIGINT can interrupt it
let activeChildProcess = null;

function handleInterrupt(signal) {
  colorLog('yellow', `\n🛑 Received ${signal}. Exiting gracefully...`);
  process.exit(0);
}

/**
 * Builder-specific flag definitions using the shared parseFlags utility.
 */
const builderFlagDefs = {
  once:        { flags: ['--once'],              type: 'boolean' },
  dryRun:      { flags: ['--dry-run', '-d'],     type: 'boolean' },
  confirm:     { flags: ['--confirm', '-c'],     type: 'boolean' },
  interactive: { flags: ['--interactive', '-i'], type: 'boolean' },
  spec:        { flags: ['--spec', '-s'],        type: 'string'  },
  assertion:   { flags: ['--assertion'],         type: 'string'  },
  help:        { flags: ['--help', '-h'],        type: 'boolean' },
};

/**
 * Parse command line flags for the builder CLI.
 */
function parseFlags(args) {
  return sharedParseFlags(args, builderFlagDefs);
}

/**
 * Show help message
 */
function showHelp() {
  console.log(`
spekk builder - Build assertions from specs

USAGE:
  spekk builder [FLAGS]

FLAGS:
  (none)              Loop through all assertions continuously (default)
  --once              Build one assertion then exit
  --dry-run, -d       Preview what would be built, don't launch Claude
  --interactive, -i   Start builder prompt without auto-selecting an assertion
  --spec, -s <id>     Work only on assertions in this spec
  --assertion <id>    Work only on this specific assertion
  --confirm, -c       Ask y/n before each build
  --help, -h          Show this help message

EXAMPLES:
  spekk builder                    # Loop through all assertions (default)
  spekk builder --once             # Build next assertion and exit
  spekk builder --dry-run          # Preview next assertion
  spekk builder --spec auth        # Loop through assertions in auth spec
  spekk builder --spec auth --once # Build one assertion in auth spec then exit
  spekk builder --confirm          # Loop with confirmation prompts
  spekk builder --interactive      # Start builder in interactive mode
`);
}

/**
 * Get the spekk command to use
 * Always uses the local bin/spekk.js to ensure flag support matches
 */
function getSpekkCommand() {
  const spekkPath = join(__dirname, '../../bin/spekk.js');
  return `node "${spekkPath}"`;
}

/**
 * Get next assertion from the parser
 */
function getNextAssertion(flags) {
  const spekkCmd = getSpekkCommand();
  let command = `${spekkCmd} next`;

  // Add --spec filter if provided
  if (flags.spec) {
    command += ` --spec ${flags.spec}`;
  }

  // Add --assertion filter if provided
  if (flags.assertion) {
    command += ` --assertion ${flags.assertion}`;
  }

  try {
    const result = execSync(command, {
      encoding: 'utf8',
      stdio: ['pipe', 'pipe', 'pipe']
    });
    return JSON.parse(result);
  } catch (error) {
    // Check if the error output contains JSON
    if (error.stdout) {
      try {
        return JSON.parse(error.stdout);
      } catch {
        // Not JSON
      }
    }
    throw new Error(`Failed to get next assertion: ${error.message}`);
  }
}

/**
 * Display assertion details
 */
function displayAssertion(assertion) {
  colorLog('green', `📄 Assertion: ${assertion.id}`);
  colorLog('blue', `   Title: ${assertion.title}`);
  colorLog('blue', `   File: ${assertion.file}`);
  colorLog('blue', `   Priority: ${assertion.priority}`);
  colorLog('blue', `   Status: ${assertion.status}`);
  if (assertion.spec) {
    colorLog('blue', `   Spec: ${assertion.spec.id}`);
  }
}

/**
 * Ask for confirmation
 */
async function askConfirmation(message) {
  const rl = createInterface({
    input: process.stdin,
    output: process.stdout
  });

  return new Promise((resolve) => {
    rl.question(`${message} [y/n/q]: `, (answer) => {
      rl.close();
      const normalized = answer.toLowerCase().trim();
      if (normalized === 'q') {
        resolve('quit');
      } else if (normalized === 'n') {
        resolve('skip');
      } else {
        resolve('proceed');
      }
    });
  });
}

/**
 * Build the spekk next command with flags
 */
function buildSpekkNextCommand(flags) {
  let cmd = `${getSpekkCommand()} next`;
  if (flags.spec) {
    cmd += ` --spec ${flags.spec}`;
  }
  if (flags.assertion) {
    cmd += ` --assertion ${flags.assertion}`;
  }
  return cmd;
}

/**
 * Build a single assertion by launching Claude Code
 */
async function buildAssertion(assertion, flags = {}) {
  colorLog('magenta', '🤖 Launching Claude Code Builder Agent...');

  const { activationMessage } = launchAgentWithPrompt('builder-agent');

  // Build the command with flags so Claude works on the correct assertion
  const spekkCommand = buildSpekkNextCommand(flags);
  let fullMessage = activationMessage;

  // If flags were provided, override the default spekk next command
  if (flags.spec || flags.assertion) {
    const commandOverride = `**IMPORTANT: Use this command instead of the default \`spekk next\`:**

\`\`\`bash
${spekkCommand}
\`\`\`

This ensures you work on the correct assertion based on the user's filter.

---

`;
    fullMessage = commandOverride + activationMessage;
  }

  const claudeProcess = spawn('claude', ['--dangerously-skip-permissions'], {
    stdio: ['pipe', 'inherit', 'inherit']
  });

  activeChildProcess = claudeProcess;

  // Handle stdin errors
  claudeProcess.stdin.on('error', (error) => {
    if (error.code !== 'EPIPE') {
      colorLog('red', '❌ Error writing to Claude stdin: ' + error.message);
    }
  });

  // Send the agent activation message (with flag overrides if any)
  try {
    claudeProcess.stdin.write(fullMessage + '\n');
    claudeProcess.stdin.end();
  } catch (error) {
    if (error.code !== 'EPIPE') {
      throw error;
    }
  }

  // Wait for Claude Code to complete
  return new Promise((resolve, reject) => {
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
      activeChildProcess = null;
      if (code === 0) {
        colorLog('green', '✅ Builder agent completed work');
        resolve(true);
      } else {
        colorLog('yellow', `⚠️ Claude Code exited with code ${code}`);
        resolve(false);
      }
    });
  });
}

/**
 * Build the Claude spawn args for a given mode.
 * Interactive mode: --system-prompt loads context, user drives the session
 * Headless mode: stdin piped with activation message (autonomous)
 */
function buildClaudeSpawnConfig(interactive, activationMessage) {
  const args = ['--dangerously-skip-permissions'];

  if (interactive) {
    // Interactive: load builder context via --system-prompt so Claude waits for user input
    args.push('--system-prompt', activationMessage);
    return { args, options: { stdio: 'inherit' } };
  }

  // Headless: pipe stdin to send activation message
  return { args, options: { stdio: ['pipe', 'inherit', 'inherit'] } };
}

/**
 * Launch builder in interactive mode (just the prompt, no auto-selected assertion)
 */
async function launchInteractiveBuilder(flags) {
  colorLog('cyan', '🔧 Starting Builder Agent (interactive mode)...');
  colorLog('yellow', 'Ctrl+C interrupts the current action. Use /exit to end the session.');

  const { activationMessage } = launchAgentWithPrompt('builder-agent');

  // Build command hint with any filters provided
  const spekkCommand = buildSpekkNextCommand(flags);
  let fullMessage = activationMessage;

  // If filters were provided, add them as a hint
  if (flags.spec || flags.assertion) {
    const commandHint = `**Note: When you run the spekk next command, use these filters:**

\`\`\`bash
${spekkCommand}
\`\`\`

---

`;
    fullMessage = commandHint + activationMessage;
  }

  // Launch Claude in interactive mode: context loaded via --system-prompt,
  // Claude waits for user input instead of auto-executing
  const { args, options } = buildClaudeSpawnConfig(true, fullMessage);
  const claudeProcess = spawn('claude', args, options);

  return new Promise((resolve, reject) => {
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
        colorLog('green', '✅ Builder agent session ended');
        resolve(true);
      } else {
        colorLog('yellow', `⚠️ Claude Code exited with code ${code}`);
        resolve(false);
      }
    });
  });
}

/**
 * Main builder function
 */
async function launchBuilderAgent(args = []) {
  const flags = parseFlags(args);

  // Handle help flag
  if (flags.help) {
    showHelp();
    return;
  }

  // Interactive mode: only when explicitly requested via --interactive flag.
  // Parent suppresses SIGINT so Claude Code handles Ctrl+C natively (Esc-style interrupt).
  // The session ends when Claude exits on its own (e.g. /exit, Ctrl+D).
  if (flags.interactive) {
    process.on('SIGINT', () => {});
    process.on('SIGTERM', () => handleInterrupt('SIGTERM'));
    await launchInteractiveBuilder(flags);
    return;
  }

  // Non-interactive modes: Ctrl+C interrupts the current build (Esc-style)
  // without killing the parent process. Between builds, Ctrl+C exits.
  process.on('SIGINT', () => {
    if (activeChildProcess) {
      colorLog('yellow', '\n⚡ Interrupting current build...');
      try { activeChildProcess.kill('SIGINT'); } catch {}
    } else {
      handleInterrupt('SIGINT');
    }
  });
  process.on('SIGTERM', () => handleInterrupt('SIGTERM'));

  // Determine mode
  const once = flags.once;
  const dryRun = flags.dryRun;
  const needsConfirm = flags.confirm;

  if (dryRun) {
    colorLog('cyan', '🔍 Dry run - showing what would be built...');
  } else if (once) {
    colorLog('cyan', '🔧 Starting Builder Agent (single build)...');
  } else {
    colorLog('cyan', '🔧 Starting Builder Agent (continuous mode)...');
    colorLog('yellow', 'Press Ctrl+C to exit gracefully.');
  }

  let iterationCount = 0;

  while (true) {
    iterationCount++;

    if (!once) {
      colorLog('bright', `\n--- Iteration ${iterationCount} ---`);
    }

    // Get next assertion
    colorLog('blue', '📋 Getting next assertion...');

    let result;
    try {
      result = getNextAssertion(flags);
    } catch (error) {
      colorLog('red', '❌ ' + error.message);
      process.exit(1);
    }

    // Check result type
    if (result.type === 'complete') {
      if (!once) {
        colorLog('green', '✨ All assertions completed. Waiting for new work...');
        await new Promise(resolve => setTimeout(resolve, 5000));
        continue;
      } else {
        colorLog('green', '✨ No assertions to work on.');
        process.exit(0);
      }
    }

    if (result.type === 'error') {
      colorLog('red', `❌ ${result.message}`);
      process.exit(1);
    }

    if (result.type !== 'assertion') {
      colorLog('red', '❌ Unexpected result from parser:');
      console.log(JSON.stringify(result, null, 2));
      process.exit(1);
    }

    const assertion = result;
    displayAssertion(assertion);

    // Dry run - just display and exit
    if (dryRun) {
      colorLog('cyan', '\n(Dry run - no build performed)');
      process.exit(0);
    }

    // Confirm before building if flag set
    if (needsConfirm) {
      const answer = await askConfirmation('Build this assertion?');
      if (answer === 'quit') {
        colorLog('yellow', '👋 Exiting builder.');
        process.exit(0);
      } else if (answer === 'skip') {
        colorLog('blue', '⏭️ Skipping assertion...');
        if (once) {
          process.exit(0);
        }
        continue;
      }
    }

    // Build the assertion
    try {
      const success = await buildAssertion(assertion, flags);

      if (!success) {
        colorLog('yellow', '⚠️ Build did not succeed.');
      }

      // In --once mode, exit after one build
      if (once) {
        process.exit(success ? 0 : 1);
      }

      // Default continuous mode: continue regardless of success/failure

    } catch (error) {
      colorLog('red', '❌ Build failed: ' + error.message);
      if (once) {
        process.exit(1);
      }
      // Continuous mode: log and continue to next assertion
      colorLog('yellow', '⚠️ Continuing to next assertion...');
    }

    // Brief pause before next iteration
    colorLog('blue', '🔄 Checking for more work...');
    await new Promise(resolve => setTimeout(resolve, 1000));
  }
}

// Run if called directly
if (import.meta.url === `file://${process.argv[1]}`) {
  const args = process.argv.slice(2);
  launchBuilderAgent(args);
}

export { launchBuilderAgent, parseFlags, buildSpekkNextCommand, buildClaudeSpawnConfig };
