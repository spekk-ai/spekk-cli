import { test, describe } from 'node:test';
import assert from 'node:assert';
import { mockExecSync } from '../../__tests__/helpers/process-mocks.js';
import fs from 'fs';
import path from 'path';
import { parseAllSpecs } from '../index.js';

describe('Malformed File Handling', () => {
  describe('Parser handles malformed spec files gracefully', () => {
    test('logs warnings for malformed spec files to stderr and continues processing', () => {
      const tempDir = path.join(process.cwd(), 'temp-malformed-spec-test');
      const validSpecDir = path.join(tempDir, 'valid-spec');
      const malformedSpecDir = path.join(tempDir, 'malformed-spec');
      const validAssertionsDir = path.join(validSpecDir, 'assertions');
      const malformedAssertionsDir = path.join(malformedSpecDir, 'assertions');
      
      try {
        fs.mkdirSync(tempDir, { recursive: true });
        fs.mkdirSync(validSpecDir, { recursive: true });
        fs.mkdirSync(malformedSpecDir, { recursive: true });
        fs.mkdirSync(validAssertionsDir, { recursive: true });
        fs.mkdirSync(malformedAssertionsDir, { recursive: true });
        
        // Create valid spec
        const validSpecContent = `---
id: valid-spec
created: 2026-01-28T21:35:00Z
priority: 1
---
# Valid Spec

This spec should be parsed successfully.`;
        
        const validAssertionContent = `---
id: valid-assertion
parent: valid-spec
created: 2026-01-28T21:35:00Z
priority: 1
status: not_started
---
# Valid Assertion

This assertion should be parsed successfully.`;
        
        // Create malformed spec (invalid YAML)
        const malformedSpecContent = `---
id: malformed-spec
created: invalid-date-format
priority: "not-a-number"
---
# Malformed Spec

This spec has invalid YAML frontmatter.`;
        
        fs.writeFileSync(path.join(validSpecDir, 'valid-spec.md'), validSpecContent);
        fs.writeFileSync(path.join(validAssertionsDir, 'valid-assertion.md'), validAssertionContent);
        fs.writeFileSync(path.join(malformedSpecDir, 'malformed-spec.md'), malformedSpecContent);
        
        const originalValidSpecPath = path.join(process.cwd(), 'specs', 'valid-spec');
        const originalMalformedSpecPath = path.join(process.cwd(), 'specs', 'malformed-spec');
        fs.symlinkSync(validSpecDir, originalValidSpecPath);
        fs.symlinkSync(malformedSpecDir, originalMalformedSpecPath);
        
        try {
          // Capture stderr to check for warnings
          let stderrOutput = '';
          const originalStderrWrite = process.stderr.write;
          process.stderr.write = function(chunk) {
            stderrOutput += chunk;
            return originalStderrWrite.call(this, chunk);
          };
          
          const { specs, assertions } = parseAllSpecs();
          
          // Restore stderr
          process.stderr.write = originalStderrWrite;
          
          // Should find valid spec but skip malformed one
          const validSpec = specs.find(s => s.id === 'valid-spec');
          const malformedSpec = specs.find(s => s.id === 'malformed-spec');
          const validAssertion = assertions.find(a => a.id === 'valid-assertion');
          
          assert.ok(validSpec, 'Should parse valid spec successfully');
          assert.ok(validAssertion, 'Should parse valid assertion successfully');
          assert.equal(malformedSpec, undefined, 'Should skip malformed spec');
          
          // Should log warning to stderr
          assert.ok(stderrOutput.includes('Warning: Skipping malformed spec file'), 
            'Should log warning for malformed spec file');
          
        } finally {
          fs.unlinkSync(originalValidSpecPath);
          fs.unlinkSync(originalMalformedSpecPath);
        }
        
      } finally {
        if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true });
      }
    });

    test('logs warnings for malformed assertion files and continues processing', () => {
      const tempDir = path.join(process.cwd(), 'temp-malformed-assertion-test');
      const assertionsDir = path.join(tempDir, 'assertions');
      
      try {
        fs.mkdirSync(tempDir, { recursive: true });
        fs.mkdirSync(assertionsDir, { recursive: true });
        
        // Create valid spec
        const specContent = `---
id: test-malformed-assertion
created: 2026-01-28T21:35:00Z
priority: 1
---
# Test Spec

Parent spec for testing malformed assertions.`;
        
        // Create valid assertion
        const validAssertionContent = `---
id: valid-assertion
parent: test-malformed-assertion
created: 2026-01-28T21:35:00Z
priority: 1
status: not_started
---
# Valid Assertion

This assertion should be parsed successfully.`;
        
        // Create malformed assertion (missing required parent field)
        const malformedAssertionContent = `---
id: malformed-assertion
created: 2026-01-28T21:35:00Z
priority: 1
status: not_started
---
# Malformed Assertion

This assertion is missing the required parent field.`;
        
        fs.writeFileSync(path.join(tempDir, 'test-malformed-assertion.md'), specContent);
        fs.writeFileSync(path.join(assertionsDir, 'valid-assertion.md'), validAssertionContent);
        fs.writeFileSync(path.join(assertionsDir, 'malformed-assertion.md'), malformedAssertionContent);
        
        const originalSpecsPath = path.join(process.cwd(), 'specs', 'test-malformed-assertion');
        fs.symlinkSync(tempDir, originalSpecsPath);
        
        try {
          // Capture stderr to check for warnings
          let stderrOutput = '';
          const originalStderrWrite = process.stderr.write;
          process.stderr.write = function(chunk) {
            stderrOutput += chunk;
            return originalStderrWrite.call(this, chunk);
          };
          
          const { specs, assertions } = parseAllSpecs();
          
          // Restore stderr
          process.stderr.write = originalStderrWrite;
          
          // Should find spec and valid assertion but skip malformed assertion
          const spec = specs.find(s => s.id === 'test-malformed-assertion');
          const validAssertion = assertions.find(a => a.id === 'valid-assertion');
          const malformedAssertion = assertions.find(a => a.id === 'malformed-assertion');
          
          assert.ok(spec, 'Should parse spec successfully');
          assert.ok(validAssertion, 'Should parse valid assertion successfully');
          assert.equal(malformedAssertion, undefined, 'Should skip malformed assertion');
          
          // Should log warning to stderr
          assert.ok(stderrOutput.includes('Warning: Skipping malformed assertion file'), 
            'Should log warning for malformed assertion file');
          
        } finally {
          fs.unlinkSync(originalSpecsPath);
        }
        
      } finally {
        if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true });
      }
    });
  });

  describe('CLI returns valid JSON even with malformed files', () => {
    test('spekk next returns valid JSON when malformed observation files exist', () => {
      const tempObservationsDir = path.join(process.cwd(), 'temp-observations');
      
      try {
        fs.mkdirSync(tempObservationsDir, { recursive: true });
        
        // Create valid observation
        const validObservationContent = `---
id: valid-observation
created: 2026-01-28T21:35:00.123Z
type: bug
severity: medium
affected_specs: []
affected_files: []
---
# Valid Observation

This observation should be parsed successfully.`;
        
        // Create malformed observation (affected_specs not an array)
        const malformedObservationContent = `---
id: malformed-observation
created: 2026-01-28T21:35:00.123Z
type: bug
severity: medium
affected_specs: "not-an-array"
affected_files: []
---
# Malformed Observation

This observation has malformed frontmatter.`;
        
        fs.writeFileSync(path.join(tempObservationsDir, 'valid-observation.md'), validObservationContent);
        fs.writeFileSync(path.join(tempObservationsDir, 'malformed-observation.md'), malformedObservationContent);
        
        // Move current observations if they exist
        const originalObservationsDir = path.join(process.cwd(), 'observations');
        const backupObservationsDir = path.join(process.cwd(), 'observations-backup');
        
        if (fs.existsSync(originalObservationsDir)) {
          fs.renameSync(originalObservationsDir, backupObservationsDir);
        }
        fs.symlinkSync(tempObservationsDir, originalObservationsDir);
        
        try {
          // Run the CLI and verify it returns valid JSON
          const result = mockExecSync('node src/parser/cli.js', { encoding: 'utf8' });
          
          let parsed;
          assert.doesNotThrow(() => {
            parsed = JSON.parse(result);
          }, 'CLI should return valid JSON even with malformed observation files');
          
          assert.equal(typeof parsed, 'object', 'Output should be an object');
          assert.ok(!Array.isArray(parsed), 'Output should not be an array');
          
        } finally {
          fs.unlinkSync(originalObservationsDir);
          if (fs.existsSync(backupObservationsDir)) {
            fs.renameSync(backupObservationsDir, originalObservationsDir);
          }
        }
        
      } finally {
        if (fs.existsSync(tempObservationsDir)) fs.rmSync(tempObservationsDir, { recursive: true });
      }
    });

    test('spekk next works even with malformed observation files', () => {
      const tempSpecDir = path.join(process.cwd(), 'temp-robust-test');
      const tempObservationsDir = path.join(process.cwd(), 'temp-observations-robust');
      const assertionsDir = path.join(tempSpecDir, 'assertions');
      
      try {
        fs.mkdirSync(tempSpecDir, { recursive: true });
        fs.mkdirSync(assertionsDir, { recursive: true });
        fs.mkdirSync(tempObservationsDir, { recursive: true });
        
        // Create a valid spec and assertion so there's something to work on
        const specContent = `---
id: robust-test
created: 2026-01-28T21:35:00Z
priority: 1
---
# Robust Test Spec

Test spec to ensure parser works with malformed observations.`;
        
        const assertionContent = `---
id: robust-test-assertion
parent: robust-test
created: 2026-01-28T21:35:00Z
priority: 1
status: not_started
---
# Robust Test Assertion

Test assertion that should be returned even with malformed observations.`;
        
        // Create malformed observation
        const malformedObservationContent = `---
id: malformed-observation-robust
created: 2026-01-28T21:35:00.123Z
type: bug
severity: medium
affected_specs: "not-an-array"
affected_files: "also-not-an-array"
---
# Malformed Observation

This observation has malformed frontmatter.`;
        
        fs.writeFileSync(path.join(tempSpecDir, 'robust-test.md'), specContent);
        fs.writeFileSync(path.join(assertionsDir, 'robust-test-assertion.md'), assertionContent);
        fs.writeFileSync(path.join(tempObservationsDir, 'malformed-observation.md'), malformedObservationContent);
        
        // Setup temporary directories
        const originalSpecsPath = path.join(process.cwd(), 'specs', 'robust-test');
        const originalObservationsDir = path.join(process.cwd(), 'observations');
        const backupObservationsDir = path.join(process.cwd(), 'observations-backup');
        
        fs.symlinkSync(tempSpecDir, originalSpecsPath);
        
        if (fs.existsSync(originalObservationsDir)) {
          fs.renameSync(originalObservationsDir, backupObservationsDir);
        }
        fs.symlinkSync(tempObservationsDir, originalObservationsDir);
        
        try {
          // Run spekk next - should work despite malformed observations
          const result = mockExecSync('node src/parser/cli.js', { encoding: 'utf8' });
          const parsed = JSON.parse(result);
          
          // Should return complete status since all specs are done
          assert.equal(parsed.type, 'complete', 'Should return complete type');
          assert.equal(parsed.status, 'complete', 'Should return complete status');
          assert.equal(parsed.message, 'All specifications are complete', 'Should have correct message');
          
        } finally {
          fs.unlinkSync(originalSpecsPath);
          fs.unlinkSync(originalObservationsDir);
          if (fs.existsSync(backupObservationsDir)) {
            fs.renameSync(backupObservationsDir, originalObservationsDir);
          }
        }
        
      } finally {
        if (fs.existsSync(tempSpecDir)) fs.rmSync(tempSpecDir, { recursive: true });
        if (fs.existsSync(tempObservationsDir)) fs.rmSync(tempObservationsDir, { recursive: true });
      }
    });
  });

  describe('Builder can continue to operate despite file parsing issues', () => {
    test('parseAllSpecs continues processing when some files are malformed', () => {
      const tempDir = path.join(process.cwd(), 'temp-builder-robust-test');
      const validSpecDir = path.join(tempDir, 'valid-builder-spec');
      const malformedSpecDir = path.join(tempDir, 'malformed-builder-spec');
      const validAssertionsDir = path.join(validSpecDir, 'assertions');
      const malformedAssertionsDir = path.join(malformedSpecDir, 'assertions');
      
      try {
        fs.mkdirSync(tempDir, { recursive: true });
        fs.mkdirSync(validSpecDir, { recursive: true });
        fs.mkdirSync(malformedSpecDir, { recursive: true });
        fs.mkdirSync(validAssertionsDir, { recursive: true });
        fs.mkdirSync(malformedAssertionsDir, { recursive: true });
        
        // Create valid spec and assertion
        const validSpecContent = `---
id: valid-builder-spec
created: 2026-01-28T21:35:00Z
priority: 1
---
# Valid Builder Spec

This spec should be parsed successfully.`;
        
        const validAssertionContent = `---
id: valid-builder-assertion
parent: valid-builder-spec
created: 2026-01-28T21:35:00Z
priority: 1
status: not_started
---
# Valid Builder Assertion

This assertion should be parsed successfully.`;
        
        // Create malformed spec file (corrupted YAML)
        const malformedSpecContent = `---
id: malformed-builder-spec
created: 2026-01-28T21:35:00Z
priority: not-a-number
invalid-yaml: [unclosed array
---
# Malformed Builder Spec

This spec has corrupted YAML frontmatter.`;
        
        fs.writeFileSync(path.join(validSpecDir, 'valid-builder-spec.md'), validSpecContent);
        fs.writeFileSync(path.join(validAssertionsDir, 'valid-builder-assertion.md'), validAssertionContent);
        fs.writeFileSync(path.join(malformedSpecDir, 'malformed-builder-spec.md'), malformedSpecContent);
        
        const originalValidSpecPath = path.join(process.cwd(), 'specs', 'valid-builder-spec');
        const originalMalformedSpecPath = path.join(process.cwd(), 'specs', 'malformed-builder-spec');
        fs.symlinkSync(validSpecDir, originalValidSpecPath);
        fs.symlinkSync(malformedSpecDir, originalMalformedSpecPath);
        
        try {
          // Parser should continue processing despite malformed files
          const { specs, assertions, observations } = parseAllSpecs();
          
          // Should have parsed valid files
          const validSpec = specs.find(s => s.id === 'valid-builder-spec');
          const validAssertion = assertions.find(a => a.id === 'valid-builder-assertion');
          const malformedSpec = specs.find(s => s.id === 'malformed-builder-spec');
          
          assert.ok(validSpec, 'Should successfully parse valid spec');
          assert.ok(validAssertion, 'Should successfully parse valid assertion');
          assert.equal(malformedSpec, undefined, 'Should skip malformed spec');
          
          // Should return arrays (not throw errors)
          assert.ok(Array.isArray(specs), 'specs should be an array');
          assert.ok(Array.isArray(assertions), 'assertions should be an array');
          assert.ok(Array.isArray(observations), 'observations should be an array');
          
        } finally {
          fs.unlinkSync(originalValidSpecPath);
          fs.unlinkSync(originalMalformedSpecPath);
        }
        
      } finally {
        if (fs.existsSync(tempDir)) fs.rmSync(tempDir, { recursive: true });
      }
    });
  });
});