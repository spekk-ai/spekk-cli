import fs from 'node:fs/promises';
import path from 'node:path';
import { exec } from 'node:child_process';
import { promisify } from 'node:util';

const execAsync = promisify(exec);

/**
 * Discovers all *.test.sh files in the specs tree
 * @param {string} baseDir - Base directory to search (defaults to 'specs')
 * @returns {Promise<string[]>} Array of test file paths
 */
async function discoverTestFiles(baseDir = 'specs') {
  const testFiles = [];
  
  async function scanDirectory(dir) {
    try {
      const entries = await fs.readdir(dir, { withFileTypes: true });
      
      for (const entry of entries) {
        const fullPath = path.join(dir, entry.name);
        
        if (entry.isDirectory()) {
          await scanDirectory(fullPath);
        } else if (entry.isFile() && entry.name.endsWith('.test.sh')) {
          // Only include files in assertions directories
          if (fullPath.includes('/assertions/')) {
            testFiles.push(fullPath);
          }
        }
      }
    } catch (error) {
      // Silently ignore directories that can't be read
    }
  }
  
  await scanDirectory(baseDir);
  return testFiles.sort();
}

/**
 * Executes a single bash test file
 * @param {string} testFile - Path to the test file
 * @returns {Promise<Object>} Test result with exitCode, stdout, stderr, passed
 */
async function executeTest(testFile) {
  try {
    const { stdout, stderr } = await execAsync(`bash "${testFile}"`);
    return {
      file: testFile,
      exitCode: 0,
      stdout: stdout.trim(),
      stderr: stderr.trim(),
      passed: true
    };
  } catch (error) {
    return {
      file: testFile,
      exitCode: error.code || 1,
      stdout: (error.stdout || '').trim(),
      stderr: (error.stderr || '').trim(),
      passed: false
    };
  }
}

/**
 * Runs all sidecar tests in the specified directory
 * @param {string} baseDir - Base directory to search for tests
 * @returns {Promise<Object>} Results summary with totalTests, failedTests, results
 */
async function runAllTests(baseDir = 'specs') {
  const testFiles = await discoverTestFiles(baseDir);
  const results = [];
  
  for (const testFile of testFiles) {
    const result = await executeTest(testFile);
    results.push(result);
  }
  
  const failedTests = results.filter(r => !r.passed).length;
  
  return {
    totalTests: results.length,
    failedTests,
    results
  };
}

/**
 * Formats the output according to specification
 * @param {Object} result - Test results from runAllTests
 * @returns {string} Formatted output string
 */
function formatOutput(result) {
  const { totalTests, failedTests, results } = result;
  
  if (failedTests === 0) {
    return `✅ All spec validation tests passed (${totalTests} tests)`;
  }
  
  const failedResults = results.filter(r => !r.passed);
  let output = '';
  
  for (const failed of failedResults) {
    output += `❌ Failed: ${failed.file}\n`;
    
    // Show the error message (combine stdout and stderr)
    const errorMessage = failed.stdout || failed.stderr || 'Test failed';
    if (errorMessage) {
      // Indent error message
      const indentedMessage = errorMessage.split('\n')
        .map(line => `   ${line}`)
        .join('\n');
      output += `${indentedMessage}\n\n`;
    }
  }
  
  output += `Failed: ${failedTests} of ${totalTests} tests`;
  
  return output;
}

/**
 * Determines the exit code based on test results
 * @param {Object} result - Test results from runAllTests
 * @returns {number} Exit code (0 for success, 1 for failure)
 */
function getExitCode(result) {
  return result.failedTests > 0 ? 1 : 0;
}

export {
  discoverTestFiles,
  executeTest,
  runAllTests,
  formatOutput,
  getExitCode
};