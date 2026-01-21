#!/usr/bin/env node

import { glob } from 'glob';
import { spawn } from 'child_process';

async function runTests() {
  try {
    // Find all test files using glob for cross-platform compatibility
    const testFiles = await glob('src/**/__tests__/**/*.test.js');
    
    if (testFiles.length === 0) {
      console.log('No test files found');
      process.exit(0);
    }
    
    console.log(`Found ${testFiles.length} test files:`);
    testFiles.forEach(file => console.log(`  - ${file}`));
    console.log();
    
    // Run Node.js test runner with discovered files
    const nodeTest = spawn('node', ['--test', ...testFiles], {
      stdio: 'inherit'
    });
    
    nodeTest.on('close', (code) => {
      console.log(`\nTest runner finished with exit code: ${code}`);
      process.exit(code);
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