import { spawn } from 'node:child_process';
import { parseFlags as sharedParseFlags } from '../cli/parse-flags.js';
import { PromptResolver } from '../cli/prompt-resolver.js';
import { loadGates } from './gate-loader.js';
import { evaluateGates } from './gate-engine.js';

const colors = {
  reset: '\x1b[0m',
  bright: '\x1b[1m',
  red: '\x1b[31m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  cyan: '\x1b[36m',
  dim: '\x1b[2m',
};

function colorLog(color, message) {
  console.log(`${colors[color]}${message}${colors.reset}`);
}

const reviewFlagDefs = {
  list:   { flags: ['--list'],            type: 'boolean' },
  dryRun: { flags: ['--dry-run'],         type: 'boolean' },
  gate:   { flags: ['--gate'],            type: 'string'  },
  force:  { flags: ['--force'],           type: 'string'  },
  skip:   { flags: ['--skip'],            type: 'string'  },
  noLlm:  { flags: ['--no-llm'],         type: 'boolean' },
  tags:   { flags: ['--tags'],            type: 'string'  },
  base:   { flags: ['--base'],            type: 'string'  },
  help:   { flags: ['--help', '-h'],      type: 'boolean' },
};

function showHelp() {
  console.log(`
spekk review - Evaluate quality gates on current branch

USAGE:
  spekk review [FLAGS]

FLAGS:
  --list              Show applicable/skipped gates with reasons (no LLM, no execution)
  --dry-run           Full evaluation including LLM judgment, but don't execute workflows
  --gate <id>         Evaluate and run a specific gate only
  --force <id>        Force-run a gate, skipping all preconditions and LLM judgment
  --skip <id>         Force-skip a gate
  --no-llm            Deterministic precondition checks only, no LLM judgment phase
  --tags <tag>        Filter gates by tag
  --base <branch>     Base branch for files-changed comparison (default: auto-detect)
  --help, -h          Show this help message

EXAMPLES:
  spekk review --list                # Show which gates apply to this branch
  spekk review --no-llm              # Run deterministic checks only
  spekk review --gate validate-testids  # Run a specific gate
  spekk review --force swagger-audit    # Force-run, skip all checks
  spekk review --tags frontend       # Only run frontend-tagged gates
`);
}

function displayListResults(results, gates) {
  const gateMap = new Map(gates.map(g => [g.id, g]));
  const applicable = results.filter(r => r.status === 'pass');
  const skipped = results.filter(r => r.status === 'skip');

  colorLog('bright', '\nApplicable:');
  if (applicable.length === 0) {
    colorLog('dim', '  (none)');
  } else {
    for (const r of applicable) {
      const gate = gateMap.get(r.id);
      const tags = gate?.tags?.length ? ` [${gate.tags.join(', ')}]` : '';
      colorLog('green', `  ${r.id}${tags}`);
    }
  }

  colorLog('bright', '\nSkipped:');
  if (skipped.length === 0) {
    colorLog('dim', '  (none)');
  } else {
    for (const r of skipped) {
      colorLog('yellow', `  ${r.id} — ${r.reason}`);
    }
  }

  console.log('');
}

export async function launchReview(args = []) {
  const flags = sharedParseFlags(args, reviewFlagDefs);

  if (flags.help) {
    showHelp();
    return;
  }

  let gates = loadGates();

  // Filter by tag
  if (flags.tags) {
    gates = gates.filter(g => g.tags.includes(flags.tags));
  }

  // Filter to specific gate
  if (flags.gate) {
    gates = gates.filter(g => g.id === flags.gate);
    if (gates.length === 0) {
      colorLog('red', `Gate not found: ${flags.gate}`);
      process.exit(1);
    }
  }

  // Force-skip
  if (flags.skip) {
    const gate = gates.find(g => g.id === flags.skip);
    if (!gate) {
      colorLog('red', `Gate not found: ${flags.skip}`);
      process.exit(1);
    }
    colorLog('yellow', `Force-skipped: ${flags.skip}`);
    return;
  }

  // Force-run: skip all preconditions
  if (flags.force) {
    const gate = gates.find(g => g.id === flags.force);
    if (!gate) {
      colorLog('red', `Gate not found: ${flags.force}`);
      process.exit(1);
    }
    colorLog('green', `Force-running: ${flags.force} (all preconditions bypassed)`);
    // In a full implementation, this would execute the gate workflow
    // For now, report that the gate would be force-run
    return;
  }

  // Evaluate gates
  const evalOptions = {};
  if (flags.base) {
    evalOptions.base = flags.base;
  }

  const results = evaluateGates(gates, evalOptions);

  // --list: just show applicable/skipped
  if (flags.list) {
    displayListResults(results, gates);
    return;
  }

  // --dry-run or --no-llm: show results but don't execute
  if (flags.dryRun || flags.noLlm) {
    displayListResults(results, gates);
    if (flags.dryRun) {
      colorLog('cyan', '(Dry run — no gate workflows executed)');
    }
    if (flags.noLlm) {
      colorLog('cyan', '(Deterministic checks only — no LLM judgment)');
    }
    return;
  }

  // Default: evaluate, then launch reviewer agent for applicable gates
  const applicable = results.filter(r => r.status === 'pass');

  if (applicable.length === 0) {
    displayListResults(results, gates);
    colorLog('cyan', 'No applicable gates — nothing to review.');
    return;
  }

  displayListResults(results, gates);

  // Build gate context for the reviewer agent
  const gateMap = new Map(gates.map(g => [g.id, g]));
  const gateContext = applicable.map(r => {
    const gate = gateMap.get(r.id);
    return `### Gate: ${gate.id}\n\n**File:** ${gate.file}\n**Tags:** ${gate.tags.join(', ') || 'none'}\n`;
  }).join('\n');

  const resolver = new PromptResolver();
  const activationMessage = resolver.createActivationMessage('reviewer');
  const fullMessage = `${activationMessage}\n\n---\n\n## Gates to Review\n\nThe following gates passed deterministic preconditions and need your LLM judgment + workflow execution:\n\n${gateContext}`;

  colorLog('cyan', 'Launching Reviewer Agent...');

  const claudeProcess = spawn('claude', ['--dangerously-skip-permissions', fullMessage], {
    stdio: 'inherit',
  });

  await new Promise((resolve) => {
    claudeProcess.on('error', (error) => {
      if (error.code === 'ENOENT') {
        colorLog('red', 'Claude Code CLI not found. Please install Claude Code first.');
      } else {
        colorLog('red', 'Error launching Claude Code: ' + error.message);
      }
      resolve();
    });

    claudeProcess.on('exit', (code) => {
      if (code === 0) {
        colorLog('green', 'Reviewer agent completed.');
      } else {
        colorLog('yellow', `Reviewer agent exited with code ${code}`);
      }
      resolve();
    });

    process.on('SIGINT', () => {
      colorLog('yellow', '\nStopping Reviewer Agent...');
      claudeProcess.kill('SIGINT');
      resolve();
    });
  });
}
