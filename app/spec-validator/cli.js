#!/usr/bin/env node

import { runAllTests, formatOutput, getExitCode } from './run-sidecar-tests.js';

async function main() {
  try {
    const result = await runAllTests();
    const output = formatOutput(result);
    
    console.log(output);
    
    const exitCode = getExitCode(result);
    process.exit(exitCode);
  } catch (error) {
    console.error('Error running spec validation tests:', error.message);
    process.exit(1);
  }
}

// Only run if this file is executed directly
if (import.meta.url === `file://${process.argv[1]}`) {
  main();
}

export { main };