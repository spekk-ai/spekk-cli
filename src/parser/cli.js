import { run as parserRun } from './index.js';

// Export the run function for programmatic use
export const run = parserRun;

// Direct invocation support
if (import.meta.url === `file://${process.argv[1]}`) {
  const args = process.argv.slice(2);
  const allFlag = args.includes('--all');
  
  if (allFlag) {
    run({ all: true });
  } else {
    run();
  }
}