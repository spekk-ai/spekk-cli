import { test, describe } from 'node:test';
import assert from 'node:assert';
import { execSync } from 'node:child_process';
import fs from 'fs';
import path from 'path';
import { parseAllSpecs } from '../index.js';

describe('Assertions Have Tests When Possible', () => {
  describe('Test Framework Setup', () => {
    test('JavaScript test framework is available and working', () => {
      // Test that Node.js test runner works
      const testCode = `
import { test } from 'node:test';
import assert from 'node:assert';

test('basic test framework validation', () => {
  assert.equal(1 + 1, 2, 'Math should work');
});
      `.trim();

      // Create .tmp directory if it doesn't exist
      const tmpBase = path.join(process.cwd(), '.tmp');
      if (!fs.existsSync(tmpBase)) {
        fs.mkdirSync(tmpBase, { recursive: true });
      }
      
      const tempTestFile = path.join(tmpBase, 'temp-test-framework-check.js');
      
      try {
        fs.writeFileSync(tempTestFile, testCode);
        
        // Run the test - should not throw
        assert.doesNotThrow(() => {
          execSync(`node --test ${tempTestFile}`, { encoding: 'utf8', stdio: 'pipe' });
        }, 'JavaScript test framework should be working');
        
      } finally {
        if (fs.existsSync(tempTestFile)) {
          fs.unlinkSync(tempTestFile);
        }
      }
    });

    test('Bash test framework is available and working', () => {
      const testScript = `#!/usr/bin/env bash
# Test basic bash functionality
if [ "hello" = "hello" ]; then
  exit 0  # Success
else
  echo "❌ Basic bash test failed"
  exit 1  # Failure
fi
      `.trim();

      // Create .tmp directory if it doesn't exist
      const tmpBase = path.join(process.cwd(), '.tmp');
      if (!fs.existsSync(tmpBase)) {
        fs.mkdirSync(tmpBase, { recursive: true });
      }
      
      const tempTestFile = path.join(tmpBase, 'temp-bash-test-check.sh');
      
      try {
        fs.writeFileSync(tempTestFile, testScript);
        fs.chmodSync(tempTestFile, 0o755); // Make executable
        
        // Run the test - should not throw
        assert.doesNotThrow(() => {
          execSync(`bash ${tempTestFile}`, { encoding: 'utf8', stdio: 'pipe' });
        }, 'Bash test framework should be working');
        
      } finally {
        if (fs.existsSync(tempTestFile)) {
          fs.unlinkSync(tempTestFile);
        }
      }
    });
  });

  describe('Test Directory Structure', () => {
    test('implementation tests directory exists', () => {
      const implementationTestsDir = path.join(process.cwd(), 'src', 'parser', '__tests__');
      assert.ok(fs.existsSync(implementationTestsDir), 'Implementation tests directory should exist at src/parser/__tests__');
      
      const stats = fs.statSync(implementationTestsDir);
      assert.ok(stats.isDirectory(), 'Implementation tests path should be a directory');
    });

    test('implementation tests can be found and run', () => {
      // Check that we can find existing test files
      const testFiles = fs.readdirSync(path.join(process.cwd(), 'src', 'parser', '__tests__'))
        .filter(file => file.endsWith('.test.js'));
      
      assert.ok(testFiles.length > 0, 'Should find JavaScript test files in implementation tests directory');
      
      // Try to run one of the existing tests to verify framework works
      if (testFiles.length > 0) {
        const testFile = path.join(process.cwd(), 'src', 'parser', '__tests__', testFiles[0]);
        assert.doesNotThrow(() => {
          execSync(`node --test ${testFile}`, { encoding: 'utf8', stdio: 'pipe' });
        }, 'Should be able to run existing implementation tests');
      }
    });
  });

  describe('Sidecar Test Support', () => {
    test('bash sidecar tests can be executed', () => {
      // Create .tmp directory if it doesn't exist
      const tmpBase = path.join(process.cwd(), '.tmp');
      if (!fs.existsSync(tmpBase)) {
        fs.mkdirSync(tmpBase, { recursive: true });
      }
      
      const tempDir = path.join(tmpBase, 'temp-sidecar-test');
      const assertionsDir = path.join(tempDir, 'assertions');
      
      try {
        fs.mkdirSync(tempDir, { recursive: true });
        fs.mkdirSync(assertionsDir, { recursive: true });
        
        // Create a sample sidecar test
        const sidecarTest = `#!/usr/bin/env bash
# Test that this assertion file exists
ASSERTION_FILE="specs/temp-sidecar-test/assertions/test-assertion.md"

[ -f "\$ASSERTION_FILE" ] || {
  echo "❌ Assertion file missing at \$ASSERTION_FILE"
  exit 1
}

exit 0  # Silent on success
        `.trim();

        const assertionContent = `---
id: test-assertion
parent: temp-sidecar-test
created: 2026-01-20T16:00:00Z
priority: 1
status: not_started
---

# Test Assertion

This is a test assertion for sidecar test validation.`;

        const specContent = `---
id: temp-sidecar-test
created: 2026-01-20T16:00:00Z
priority: 1
---

# Temp Sidecar Test Spec

Test spec for sidecar test validation.`;

        fs.writeFileSync(path.join(tempDir, 'temp-sidecar-test.md'), specContent);
        fs.writeFileSync(path.join(assertionsDir, 'test-assertion.md'), assertionContent);
        fs.writeFileSync(path.join(assertionsDir, 'test-assertion.test.sh'), sidecarTest);
        fs.chmodSync(path.join(assertionsDir, 'test-assertion.test.sh'), 0o755);
        
        const originalSpecsPath = path.join(process.cwd(), 'specs', 'temp-sidecar-test');
        fs.symlinkSync(tempDir, originalSpecsPath);
        
        try {
          // Run the sidecar test
          const testFile = path.join(assertionsDir, 'test-assertion.test.sh');
          assert.doesNotThrow(() => {
            const result = execSync(`bash ${testFile}`, { 
              encoding: 'utf8', 
              stdio: 'pipe',
              cwd: process.cwd()
            });
          }, 'Sidecar test should execute successfully');
          
        } finally {
          fs.unlinkSync(originalSpecsPath);
        }
        
      } finally {
        if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true });
      }
    });

    test('sidecar tests can detect missing files', () => {
      // Create .tmp directory if it doesn't exist
      const tmpBase = path.join(process.cwd(), '.tmp');
      if (!fs.existsSync(tmpBase)) {
        fs.mkdirSync(tmpBase, { recursive: true });
      }
      
      const tempDir = path.join(tmpBase, 'temp-sidecar-fail-test');
      const assertionsDir = path.join(tempDir, 'assertions');
      
      try {
        fs.mkdirSync(tempDir, { recursive: true });
        fs.mkdirSync(assertionsDir, { recursive: true });
        
        // Create a sidecar test that checks for a missing file
        const sidecarTest = `#!/usr/bin/env bash
# Test that should fail - checking for nonexistent file
MISSING_FILE="specs/temp-sidecar-fail-test/nonexistent-file.md"

[ -f "\$MISSING_FILE" ] || {
  echo "❌ File missing at \$MISSING_FILE"
  exit 1
}

exit 0
        `.trim();

        const specContent = `---
id: temp-sidecar-fail-test
created: 2026-01-20T16:00:00Z
priority: 1
---

# Temp Sidecar Fail Test Spec

Test spec for sidecar test failure validation.`;

        fs.writeFileSync(path.join(tempDir, 'temp-sidecar-fail-test.md'), specContent);
        fs.writeFileSync(path.join(assertionsDir, 'fail-test.test.sh'), sidecarTest);
        fs.chmodSync(path.join(assertionsDir, 'fail-test.test.sh'), 0o755);
        
        const originalSpecsPath = path.join(process.cwd(), 'specs', 'temp-sidecar-fail-test');
        fs.symlinkSync(tempDir, originalSpecsPath);
        
        try {
          // Run the sidecar test - should fail
          const testFile = path.join(assertionsDir, 'fail-test.test.sh');
          
          let testFailed = false;
          try {
            execSync(`bash ${testFile}`, { 
              encoding: 'utf8', 
              stdio: 'pipe',
              cwd: process.cwd()
            });
          } catch (error) {
            testFailed = true;
            assert.equal(error.status, 1, 'Sidecar test should exit with code 1 on failure');
          }
          
          assert.ok(testFailed, 'Sidecar test should fail when checking for missing file');
          
        } finally {
          fs.unlinkSync(originalSpecsPath);
        }
        
      } finally {
        if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true });
      }
    });
  });

  describe('Test Discovery and Linking', () => {
    test('can identify assertions with linked tests', () => {
      // Create .tmp directory if it doesn't exist
      const tmpBase = path.join(process.cwd(), '.tmp');
      if (!fs.existsSync(tmpBase)) {
        fs.mkdirSync(tmpBase, { recursive: true });
      }
      
      const tempDir = path.join(tmpBase, 'temp-test-linking');
      const assertionsDir = path.join(tempDir, 'assertions');
      
      try {
        fs.mkdirSync(tempDir, { recursive: true });
        fs.mkdirSync(assertionsDir, { recursive: true });
        
        const specContent = `---
id: temp-test-linking
created: 2026-01-20T16:00:00Z
priority: 1
---

# Test Linking Spec

Test spec for test linking validation.`;

        const assertionWithTest = `---
id: assertion-with-test
parent: temp-test-linking
created: 2026-01-20T16:00:00Z
priority: 1
status: not_started
---

# Assertion With Test

This assertion has a linked test.

**Tests:** \`src/parser/__tests__/test-linking.test.js\``;

        const assertionWithoutTest = `---
id: assertion-without-test
parent: temp-test-linking
created: 2026-01-20T16:01:00Z
priority: 1
status: not_started
---

# Assertion Without Test

This assertion has no linked test.`;

        fs.writeFileSync(path.join(tempDir, 'temp-test-linking.md'), specContent);
        fs.writeFileSync(path.join(assertionsDir, 'with-test.md'), assertionWithTest);
        fs.writeFileSync(path.join(assertionsDir, 'without-test.md'), assertionWithoutTest);
        
        const originalSpecsPath = path.join(process.cwd(), 'specs', 'temp-test-linking');
        fs.symlinkSync(tempDir, originalSpecsPath);
        
        try {
          const { assertions } = parseAllSpecs();
          const testAssertions = assertions.filter(a => a.parent === 'temp-test-linking');
          
          const withTest = testAssertions.find(a => a.id === 'assertion-with-test');
          const withoutTest = testAssertions.find(a => a.id === 'assertion-without-test');
          
          assert.ok(withTest, 'Should find assertion with test');
          assert.ok(withoutTest, 'Should find assertion without test');
          
          // Check that we can detect test links in content
          assert.ok(withTest.content.includes('**Tests:**'), 'Assertion with test should have Tests: marker');
          assert.ok(!withoutTest.content.includes('**Tests:**'), 'Assertion without test should not have Tests: marker');
          
        } finally {
          fs.unlinkSync(originalSpecsPath);
        }
        
      } finally {
        if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true });
      }
    });

    test('can validate test file paths exist', () => {
      // Create a real test file to validate against
      const testDir = path.join(process.cwd(), 'src', 'parser', '__tests__');
      const testFile = path.join(testDir, 'temp-validation-test.test.js');
      
      const testContent = `import { test } from 'node:test';
import assert from 'node:assert';

test('temporary validation test', () => {
  assert.ok(true, 'This test should always pass');
});`;

      try {
        fs.writeFileSync(testFile, testContent);
        
        // Test that the file exists and can be detected
        assert.ok(fs.existsSync(testFile), 'Test file should exist after creation');
        
        // Test that it can be run
        assert.doesNotThrow(() => {
          execSync(`node --test ${testFile}`, { encoding: 'utf8', stdio: 'pipe' });
        }, 'Created test file should be executable');
        
      } finally {
        if (fs.existsSync(testFile)) {
          fs.unlinkSync(testFile);
        }
      }
    });
  });

  describe('Test Validation Requirements', () => {
    test('validates that testable assertions should have tests', () => {
      // This is more of a documentation test - showing the principle
      const testableCategories = [
        'Code behavior',
        'File existence', 
        'Spec structure',
        'Data validation',
        'Algorithm correctness',
        'Output format validation'
      ];
      
      const nonTestableCategories = [
        'Subjective quality',
        'Business judgment calls',
        'Manual processes'
      ];
      
      // Test that we understand the categories correctly
      assert.ok(testableCategories.length > 0, 'Should have testable categories defined');
      assert.ok(nonTestableCategories.length > 0, 'Should have non-testable categories defined');
      
      // Test behavior categories should be different
      const overlap = testableCategories.filter(cat => nonTestableCategories.includes(cat));
      assert.equal(overlap.length, 0, 'Testable and non-testable categories should not overlap');
    });

    test('implementation tests use correct structure', () => {
      // Validate that existing implementation tests follow the right pattern
      const implementationTestsDir = path.join(process.cwd(), 'src', 'parser', '__tests__');
      const testFiles = fs.readdirSync(implementationTestsDir)
        .filter(file => file.endsWith('.test.js'));
      
      // Check at least one test file for structure
      if (testFiles.length > 0) {
        const testFile = path.join(implementationTestsDir, testFiles[0]);
        const content = fs.readFileSync(testFile, 'utf8');
        
        // Should use Node.js test framework
        assert.ok(content.includes('from \'node:test\''), 'Implementation tests should use Node.js test framework');
        assert.ok(content.includes('from \'node:assert\''), 'Implementation tests should use Node.js assert module');
        
        // Should have proper test structure
        assert.ok(content.includes('test(') || content.includes('describe('), 'Implementation tests should have test functions');
      }
    });
  });

  describe('Test Running Integration', () => {
    test('can run all implementation tests', () => {
      const implementationTestsDir = path.join(process.cwd(), 'src', 'parser', '__tests__');
      
      // Should be able to run tests in the directory
      assert.doesNotThrow(() => {
        execSync(`node --test ${implementationTestsDir}`, { 
          encoding: 'utf8', 
          stdio: 'pipe',
          timeout: 30000  // 30 second timeout for all tests
        });
      }, 'Should be able to run all implementation tests');
    });

    test('npm test command works if available', () => {
      // Check if npm test is configured
      const packageJsonPath = path.join(process.cwd(), 'package.json');
      
      if (fs.existsSync(packageJsonPath)) {
        const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, 'utf8'));
        
        if (packageJson.scripts && packageJson.scripts.test) {
          // If test script exists, it should work
          assert.doesNotThrow(() => {
            execSync('npm test', { 
              encoding: 'utf8', 
              stdio: 'pipe',
              timeout: 60000  // 60 second timeout for full test suite
            });
          }, 'npm test should work if configured');
        }
      }
    });
  });

  describe('Test Framework Requirements', () => {
    test('supports both implementation and sidecar test types', () => {
      // Validate that both test frameworks are available
      
      // JavaScript/Node.js for implementation tests
      assert.doesNotThrow(() => {
        const result = execSync('node --version', { encoding: 'utf8' });
        assert.ok(result.includes('v'), 'Node.js should be available for implementation tests');
      }, 'Node.js should be available');
      
      // Bash for sidecar tests
      assert.doesNotThrow(() => {
        const result = execSync('bash --version', { encoding: 'utf8' });
        assert.ok(result.includes('bash'), 'Bash should be available for sidecar tests');
      }, 'Bash should be available');
    });

    test('test frameworks produce proper exit codes', () => {
      // Test JavaScript test success
      const jsTestCode = `
import { test } from 'node:test';
import assert from 'node:assert';

test('success test', () => {
  assert.equal(2 + 2, 4);
});
      `.trim();

      const tempJsTest = path.join(process.cwd(), 'temp-success-test.js');
      
      try {
        fs.writeFileSync(tempJsTest, jsTestCode);
        
        const result = execSync(`node --test ${tempJsTest}`, { encoding: 'utf8' });
        // Should not throw - exit code 0 expected
        
      } finally {
        if (fs.existsSync(tempJsTest)) {
          fs.unlinkSync(tempJsTest);
        }
      }

      // Test bash test success
      const bashTestCode = `#!/usr/bin/env bash
exit 0  # Success
      `.trim();

      const tempBashTest = path.join(process.cwd(), 'temp-success-test.sh');
      
      try {
        fs.writeFileSync(tempBashTest, bashTestCode);
        fs.chmodSync(tempBashTest, 0o755);
        
        execSync(`bash ${tempBashTest}`, { encoding: 'utf8' });
        // Should not throw - exit code 0 expected
        
      } finally {
        if (fs.existsSync(tempBashTest)) {
          fs.unlinkSync(tempBashTest);
        }
      }
    });
  });
});