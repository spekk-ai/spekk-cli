/**
 * CLI entry point for the parser — delegates directly to the Go binary.
 * No Node.js fallback; the Go binary is required.
 */
import { getSpekkInstallationDirectory } from './index.js';
import { parseFlags } from '../cli/parse-flags.js';
import { spawnSync } from 'child_process';
import { existsSync } from 'fs';
import path from 'path';

const parserFlagDefs = {
  all:          { flags: ['--all'],              type: 'boolean' },
  allBranches:  { flags: ['--all-branches'],     type: 'boolean' },
  spec:         { flags: ['--spec', '-s'],       type: 'string'  },
  assertion:    { flags: ['--assertion'],        type: 'string'  },
};

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

function buildGoArgs(options) {
  const args = ['next'];
  if (options.all) args.push('--all');
  if (options.allBranches) args.push('--all-branches');
  if (options.spec) args.push('--spec', options.spec);
  if (options.assertion) args.push('--assertion', options.assertion);
  return args;
}

export function run(options = {}) {
  const goBinary = findGoBinary();
  if (!goBinary) {
    console.error('Error: Go binary not found. Run "go build -o bin/spekk-go ./cmd/spekk/" to build it.');
    process.exit(1);
  }

  const goArgs = buildGoArgs(options);
  const result = spawnSync(goBinary, goArgs, {
    stdio: ['inherit', 'pipe', 'inherit'],
    encoding: 'utf8',
  });

  if (result.error) {
    console.error(`Error: Failed to execute Go binary: ${result.error.message}`);
    process.exit(1);
  }

  if (result.stdout) process.stdout.write(result.stdout);
  process.exit(result.status ?? 0);
}

// Direct invocation support
if (import.meta.url === `file://${process.argv[1]}`) {
  const args = process.argv.slice(2);
  const options = parseFlags(args, parserFlagDefs);
  run(options);
}
