#!/usr/bin/env node

import { glob } from 'glob';
import { run } from 'node:test';
import { spec as tapSpec } from 'node:test/reporters';

async function runTests() {
  try {
    // Find all test files using glob (cross-platform compatible)
    const testFiles = await glob('{app,src}/**/__tests__/**/*.test.js');
    
    if (testFiles.length === 0) {
      console.log('No test files found');
      process.exit(0);
    }
    
    console.log(`Found ${testFiles.length} test files`);
    
    // Use Node.js native test runner instead of spawning child processes
    // This avoids the slow process spawning that was causing performance issues
    const stream = run({
      files: testFiles,
      concurrency: 1, // Sequential execution to avoid test interference
    });
    
    // Use TAP reporter for clean output
    stream.compose(tapSpec).pipe(process.stdout);
    
    // Handle test completion
    stream.on('test:fail', () => {
      process.exitCode = 1;
    });
    
    stream.on('end', () => {
      process.exit(process.exitCode || 0);
    });
    
  } catch (error) {
    console.error('Error running tests:', error);
    process.exit(1);
  }
}

runTests();