import { run as parserRun } from './index.js';
import { parseFlags } from '../cli/parse-flags.js';

// Export the run function for programmatic use
export const run = parserRun;

// Parser-specific flag definitions
const parserFlagDefs = {
  all:       { flags: ['--all'],              type: 'boolean' },
  spec:      { flags: ['--spec', '-s'],       type: 'string'  },
  assertion: { flags: ['--assertion'],        type: 'string'  },
};

// Direct invocation support
if (import.meta.url === `file://${process.argv[1]}`) {
  const args = process.argv.slice(2);
  const options = parseFlags(args, parserFlagDefs);
  run(options);
}