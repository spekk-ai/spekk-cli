#!/usr/bin/env node

import { glob } from 'glob';
import { spawn } from 'child_process';

async function runTests() {
  try {
    // Find all test files using glob (cross-platform compatible)
    const testFiles = await glob('{app,src}/**/__tests__/**/*.test.js');
    
    if (testFiles.length === 0) {
      console.log('No test files found');
      process.exit(0);
    }
    
    console.log(`Found ${testFiles.length} test files`);
    
    // Run Node.js test runner with all test files (parallel execution enabled)
    const nodeTest = spawn('node', ['--test', ...testFiles], {
      stdio: 'inherit',
      shell: false // Avoid shell-specific behavior for cross-platform compatibility
    });
    
    // Exit with the same code as the test runner
    nodeTest.on('close', (code) => {
      process.exit(code || 0);
    });
    
    nodeTest.on('error', (error) => {
      console.error('Failed to start test runner:', error);
      process.exit(1);
    });
    
  } catch (error) {
    console.error('Error finding test files:', error);
    process.exit(1);
  }
}

runTests();