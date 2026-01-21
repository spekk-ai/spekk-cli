import { test, describe, before, after } from 'node:test';
import assert from 'node:assert';
import fs from 'node:fs/promises';
import path from 'node:path';
import { exec } from 'node:child_process';
import { promisify } from 'node:util';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const execAsync = promisify(exec);
const runSidecarTests = await import('../run-sidecar-tests.js');

describe('Sidecar Test Runner', () => {
  let testCounter = 0;
  
  const getTestDir = () => {
    return path.join(__dirname, `temp-test-specs-${Date.now()}-${++testCounter}`);
  };
  
  after(async () => {
    // Clean up any remaining temporary test directories
    try {
      const files = await fs.readdir(__dirname);
      for (const file of files) {
        if (file.startsWith('temp-test-specs-')) {
          await fs.rm(path.join(__dirname, file), { recursive: true, force: true });
        }
      }
    } catch (error) {
      // Ignore cleanup errors
    }
  });

  test('discovers test files correctly', async () => {
    const testDir = getTestDir();
    
    // Create test files
    const test1 = path.join(testDir, 'feature1', 'assertions', 'test1.test.sh');
    const test2 = path.join(testDir, 'feature2', 'assertions', 'test2.test.sh');
    
    await fs.mkdir(path.dirname(test1), { recursive: true });
    await fs.mkdir(path.dirname(test2), { recursive: true });
    await fs.writeFile(test1, '#!/usr/bin/env bash\nexit 0', { mode: 0o755 });
    await fs.writeFile(test2, '#!/usr/bin/env bash\nexit 0', { mode: 0o755 });
    
    const testFiles = await runSidecarTests.discoverTestFiles(testDir);
    
    assert.strictEqual(testFiles.length, 2);
    assert.ok(testFiles.some(f => f.includes('test1.test.sh')));
    assert.ok(testFiles.some(f => f.includes('test2.test.sh')));
    
    // Cleanup
    await fs.rm(testDir, { recursive: true, force: true });
  });

  test('handles no test files found', async () => {
    const testDir = getTestDir();
    const emptyDir = path.join(testDir, 'empty');
    await fs.mkdir(emptyDir, { recursive: true });
    
    const testFiles = await runSidecarTests.discoverTestFiles(emptyDir);
    
    assert.strictEqual(testFiles.length, 0);
    
    // Cleanup
    await fs.rm(testDir, { recursive: true, force: true });
  });

  test('executes passing bash scripts', async () => {
    const testDir = getTestDir();
    const passingTest = path.join(testDir, 'feature1', 'assertions', 'passing.test.sh');
    await fs.mkdir(path.dirname(passingTest), { recursive: true });
    await fs.writeFile(passingTest, '#!/usr/bin/env bash\nexit 0', { mode: 0o755 });
    
    const result = await runSidecarTests.executeTest(passingTest);
    
    assert.strictEqual(result.exitCode, 0);
    assert.strictEqual(result.passed, true);
    assert.strictEqual(result.stdout, '');
    assert.strictEqual(result.stderr, '');
    
    // Cleanup
    await fs.rm(testDir, { recursive: true, force: true });
  });

  test('captures failing bash scripts with output', async () => {
    const testDir = getTestDir();
    const failingTest = path.join(testDir, 'feature1', 'assertions', 'failing.test.sh');
    await fs.mkdir(path.dirname(failingTest), { recursive: true });
    await fs.writeFile(failingTest, 
      '#!/usr/bin/env bash\necho "Something went wrong"\nexit 1', 
      { mode: 0o755 });
    
    const result = await runSidecarTests.executeTest(failingTest);
    
    assert.strictEqual(result.exitCode, 1);
    assert.strictEqual(result.passed, false);
    assert.ok(result.stdout.includes('Something went wrong'));
    
    // Cleanup
    await fs.rm(testDir, { recursive: true, force: true });
  });

  test('runs multiple tests and collects results', async () => {
    const testDir = getTestDir();
    
    // Create multiple test files with different outcomes
    const passingTest = path.join(testDir, 'feature1', 'assertions', 'pass.test.sh');
    const failingTest = path.join(testDir, 'feature2', 'assertions', 'fail.test.sh');
    
    await fs.mkdir(path.dirname(passingTest), { recursive: true });
    await fs.mkdir(path.dirname(failingTest), { recursive: true });
    await fs.writeFile(passingTest, '#!/usr/bin/env bash\nexit 0', { mode: 0o755 });
    await fs.writeFile(failingTest, 
      '#!/usr/bin/env bash\necho "Test failed"\nexit 1', 
      { mode: 0o755 });
    
    const result = await runSidecarTests.runAllTests(testDir);
    
    assert.strictEqual(result.totalTests, 2);
    assert.strictEqual(result.failedTests, 1);
    assert.strictEqual(result.results.length, 2);
    
    const passingResult = result.results.find(r => r.file.includes('pass.test.sh'));
    const failingResult = result.results.find(r => r.file.includes('fail.test.sh'));
    
    assert.ok(passingResult.passed);
    assert.ok(!failingResult.passed);
    
    // Cleanup
    await fs.rm(testDir, { recursive: true, force: true });
  });

  test('formats output correctly for all tests passing', async () => {
    const testDir = getTestDir();
    const passingTest = path.join(testDir, 'feature1', 'assertions', 'only-pass.test.sh');
    await fs.mkdir(path.dirname(passingTest), { recursive: true });
    await fs.writeFile(passingTest, '#!/usr/bin/env bash\nexit 0', { mode: 0o755 });
    
    const result = await runSidecarTests.runAllTests(testDir);
    const output = runSidecarTests.formatOutput(result);
    
    assert.ok(output.includes('✅ All spec validation tests passed'));
    assert.ok(output.includes('1 tests'));
    
    // Cleanup
    await fs.rm(testDir, { recursive: true, force: true });
  });

  test('formats output correctly for some tests failing', async () => {
    const testDir = getTestDir();
    const passingTest = path.join(testDir, 'feature1', 'assertions', 'pass2.test.sh');
    const failingTest = path.join(testDir, 'feature2', 'assertions', 'fail2.test.sh');
    
    await fs.mkdir(path.dirname(passingTest), { recursive: true });
    await fs.mkdir(path.dirname(failingTest), { recursive: true });
    await fs.writeFile(passingTest, '#!/usr/bin/env bash\nexit 0', { mode: 0o755 });
    await fs.writeFile(failingTest, 
      '#!/usr/bin/env bash\necho "Error message"\nexit 1', 
      { mode: 0o755 });
    
    const result = await runSidecarTests.runAllTests(testDir);
    const output = runSidecarTests.formatOutput(result);
    
    assert.ok(output.includes('❌ Failed:'));
    assert.ok(output.includes('fail2.test.sh'));
    assert.ok(output.includes('Error message'));
    assert.ok(output.includes('Failed: 1 of 2 tests'));
    
    // Cleanup
    await fs.rm(testDir, { recursive: true, force: true });
  });

  test('returns correct exit code', async () => {
    // Test with all passing
    const testDir1 = getTestDir();
    const passingTest = path.join(testDir1, 'feature1', 'assertions', 'exit-pass.test.sh');
    await fs.mkdir(path.dirname(passingTest), { recursive: true });
    await fs.writeFile(passingTest, '#!/usr/bin/env bash\nexit 0', { mode: 0o755 });
    
    let result = await runSidecarTests.runAllTests(testDir1);
    assert.strictEqual(runSidecarTests.getExitCode(result), 0);
    
    // Test with some failing
    const testDir2 = getTestDir();
    const passingTest2 = path.join(testDir2, 'feature1', 'assertions', 'exit-pass.test.sh');
    const failingTest = path.join(testDir2, 'feature2', 'assertions', 'exit-fail.test.sh');
    await fs.mkdir(path.dirname(passingTest2), { recursive: true });
    await fs.mkdir(path.dirname(failingTest), { recursive: true });
    await fs.writeFile(passingTest2, '#!/usr/bin/env bash\nexit 0', { mode: 0o755 });
    await fs.writeFile(failingTest, '#!/usr/bin/env bash\nexit 1', { mode: 0o755 });
    
    result = await runSidecarTests.runAllTests(testDir2);
    assert.strictEqual(runSidecarTests.getExitCode(result), 1);
    
    // Cleanup
    await fs.rm(testDir1, { recursive: true, force: true });
    await fs.rm(testDir2, { recursive: true, force: true });
  });

  test('handles no tests found gracefully', async () => {
    const testDir = getTestDir();
    const emptyDir = path.join(testDir, 'no-tests');
    await fs.mkdir(emptyDir, { recursive: true });
    
    const result = await runSidecarTests.runAllTests(emptyDir);
    
    assert.strictEqual(result.totalTests, 0);
    assert.strictEqual(result.failedTests, 0);
    assert.strictEqual(runSidecarTests.getExitCode(result), 0);
    
    const output = runSidecarTests.formatOutput(result);
    assert.ok(output.includes('✅ All spec validation tests passed (0 tests)'));
    
    // Cleanup
    await fs.rm(testDir, { recursive: true, force: true });
  });
});