import { run as parserRun, getSpekkInstallationDirectory } from './index.js';
import { parseFlags } from '../cli/parse-flags.js';
import { spawnSync } from 'child_process';
import { existsSync } from 'fs';
import path from 'path';

// Parser-specific flag definitions
const parserFlagDefs = {
  all:          { flags: ['--all'],              type: 'boolean' },
  allBranches:  { flags: ['--all-branches'],     type: 'boolean' },
  spec:         { flags: ['--spec', '-s'],       type: 'string'  },
  assertion:    { flags: ['--assertion'],        type: 'string'  },
};

/**
 * Find the Go binary at known locations relative to the spekk installation.
 * Returns the path if found and executable, null otherwise.
 */
function findGoBinary() {
  const installDir = getSpekkInstallationDirectory();
  const candidates = [
    path.join(installDir, 'bin', 'spekk-go'),
    path.join(installDir, 'spekk-go'),
  ];

  for (const candidate of candidates) {
    if (existsSync(candidate)) {
      return candidate;
    }
  }
  return null;
}

/**
 * Build CLI args from options object for delegation to Go binary.
 */
function buildGoArgs(options) {
  const args = ['next'];
  if (options.all) args.push('--all');
  if (options.allBranches) args.push('--all-branches');
  if (options.spec) args.push('--spec', options.spec);
  if (options.assertion) args.push('--assertion', options.assertion);
  return args;
}

/**
 * Attempt to delegate to the Go binary. Returns true if delegation
 * succeeded, false if the binary was not found (caller should fall back).
 */
function delegateToGo(options) {
  const goBinary = findGoBinary();
  if (!goBinary) return false;

  const goArgs = buildGoArgs(options);
  const result = spawnSync(goBinary, goArgs, {
    stdio: ['inherit', 'pipe', 'inherit'],
    encoding: 'utf8',
  });

  // If the binary failed to execute (e.g. permission denied, not a valid binary),
  // fall back to Node parser.
  if (result.error) return false;

  if (result.stdout) process.stdout.write(result.stdout);
  process.exit(result.status ?? 0);
}

/**
 * Run the parser — delegates to Go binary if available, otherwise uses Node parser.
 */
export function run(options = {}) {
  if (delegateToGo(options)) return;
  parserRun(options);
}

// Direct invocation support
if (import.meta.url === `file://${process.argv[1]}`) {
  const args = process.argv.slice(2);
  const options = parseFlags(args, parserFlagDefs);
  run(options);
}
