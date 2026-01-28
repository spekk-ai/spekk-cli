import { test, describe, beforeEach, afterEach } from 'node:test';
import assert from 'node:assert';
import fs from 'fs';
import path from 'path';
import os from 'os';
import { fileURLToPath } from 'url';
import { parseObservations, validateObservationFields, parseAllSpecs } from '../index.js';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

describe('Observation Parsing', () => {
  let tempDir;

  beforeEach(() => {
    // Create temporary directory for test files
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'spekk-test-'));
  });

  afterEach(() => {
    // Clean up temporary directory
    fs.rmSync(tempDir, { recursive: true, force: true });
  });

  describe('validateObservationFields', () => {
    test('should validate required fields', () => {
      const validObservation = {
        id: 'test-observation',
        created: '2026-01-22T10:00:00.000Z',
        type: 'code_spec_misalignment',
        severity: 'high',
        affected_specs: ['spec-one'],
        affected_files: ['src/file.js']
      };

      // Should not throw
      assert.doesNotThrow(() => validateObservationFields(validObservation, 'test.md'));
    });

    test('should throw on missing required fields', () => {
      const invalidObservations = [
        { created: '2026-01-22T10:00:00Z', type: 'test', severity: 'high', affected_specs: [], affected_files: [] },
        { id: 'test', type: 'test', severity: 'high', affected_specs: [], affected_files: [] },
        { id: 'test', created: '2026-01-22T10:00:00Z', severity: 'high', affected_specs: [], affected_files: [] },
        { id: 'test', created: '2026-01-22T10:00:00Z', type: 'test', affected_specs: [], affected_files: [] },
        { id: 'test', created: '2026-01-22T10:00:00Z', type: 'test', severity: 'high', affected_files: [] },
        { id: 'test', created: '2026-01-22T10:00:00Z', type: 'test', severity: 'high', affected_specs: [] }
      ];

      invalidObservations.forEach(obs => {
        assert.throws(() => validateObservationFields(obs, 'test.md'), /Missing required field/);
      });
    });

    test('should validate severity values', () => {
      const observation = {
        id: 'test',
        created: '2026-01-22T10:00:00Z',
        type: 'test',
        severity: 'invalid',
        affected_specs: [],
        affected_files: []
      };

      assert.throws(() => validateObservationFields(observation, 'test.md'), /Invalid severity value/);
    });

    test('should validate array fields', () => {
      const obsWithNonArraySpecs = {
        id: 'test',
        created: '2026-01-22T10:00:00Z',
        type: 'test',
        severity: 'high',
        affected_specs: 'not-an-array',
        affected_files: []
      };

      const obsWithNonArrayFiles = {
        id: 'test',
        created: '2026-01-22T10:00:00Z',
        type: 'test',
        severity: 'high',
        affected_specs: [],
        affected_files: 'not-an-array'
      };

      assert.throws(() => validateObservationFields(obsWithNonArraySpecs, 'test.md'), /must be an array/);
      assert.throws(() => validateObservationFields(obsWithNonArrayFiles, 'test.md'), /must be an array/);
    });

    test('should validate timestamp formats', () => {
      const validTimestamps = [
        '2026-01-22T10:00:00.000Z',
        '2026-01-22T10:00:00Z'
      ];

      const invalidTimestamps = [
        '2026-01-22',
        '2026-01-22T10:00:00',
        '2026-01-22 10:00:00Z',
        'invalid-date'
      ];

      validTimestamps.forEach(timestamp => {
        const obs = {
          id: 'test',
          created: timestamp,
          type: 'test',
          severity: 'high',
          affected_specs: [],
          affected_files: []
        };
        assert.doesNotThrow(() => validateObservationFields(obs, 'test.md'));
      });

      invalidTimestamps.forEach(timestamp => {
        const obs = {
          id: 'test',
          created: timestamp,
          type: 'test',
          severity: 'high',
          affected_specs: [],
          affected_files: []
        };
        assert.throws(() => validateObservationFields(obs, 'test.md'), /Invalid ISO 8601 timestamp/);
      });
    });
  });

  describe('parseObservations', () => {
    test('should parse observation files from observations directory', () => {
      const observationsDir = path.join(tempDir, 'observations');
      fs.mkdirSync(observationsDir);

      // Create test observation file
      const observationContent = `---
id: test-observation-1
created: 2026-01-22T10:00:00.000Z
type: code_spec_misalignment
severity: high
affected_specs:
  - spec-parser
affected_files:
  - src/parser/index.js
---

# Test Observation

This is a test observation.`;

      fs.writeFileSync(path.join(observationsDir, 'test-observation-1.md'), observationContent);

      const observations = parseObservations(tempDir);

      assert.strictEqual(observations.length, 1);
      assert.deepStrictEqual(observations[0], {
        id: 'test-observation-1',
        created: '2026-01-22T10:00:00.000Z',
        type: 'code_spec_misalignment',
        severity: 'high',
        affected_specs: ['spec-parser'],
        affected_files: ['src/parser/index.js'],
        file: 'observations/test-observation-1.md',
        title: 'Test Observation',
        content: 'This is a test observation.'
      });
    });

    test('should skip files without frontmatter', () => {
      const observationsDir = path.join(tempDir, 'observations');
      fs.mkdirSync(observationsDir);

      // Create file without frontmatter
      fs.writeFileSync(path.join(observationsDir, 'no-frontmatter.md'), '# Just a regular markdown file');

      // Create valid observation
      const observationContent = `---
id: valid-observation
created: 2026-01-22T10:00:00Z
type: test
severity: low
affected_specs: []
affected_files: []
---

# Valid Observation`;

      fs.writeFileSync(path.join(observationsDir, 'valid-observation.md'), observationContent);

      const observations = parseObservations(tempDir);

      assert.strictEqual(observations.length, 1);
      assert.strictEqual(observations[0].id, 'valid-observation');
    });

    test('should return empty array if observations directory does not exist', () => {
      const observations = parseObservations(tempDir);
      assert.deepStrictEqual(observations, []);
    });
  });

  describe('parseAllSpecs with observations', () => {
    test('should include observations in output', () => {
      // Create specs directory with a spec
      const specsDir = path.join(tempDir, 'specs');
      const specDir = path.join(specsDir, 'test-spec');
      const assertionsDir = path.join(specDir, 'assertions');
      fs.mkdirSync(assertionsDir, { recursive: true });

      const specContent = `---
id: test-spec
created: 2026-01-22T10:00:00Z
priority: 1
---

# Test Spec`;

      fs.writeFileSync(path.join(specDir, 'test-spec.md'), specContent);

      // Create observations directory with observation
      const observationsDir = path.join(tempDir, 'observations');
      fs.mkdirSync(observationsDir);

      const observationContent = `---
id: test-observation
created: 2026-01-22T10:00:00Z
type: code_spec_misalignment
severity: medium
affected_specs:
  - test-spec
affected_files:
  - src/test.js
---

# Test Observation`;

      fs.writeFileSync(path.join(observationsDir, 'test-observation.md'), observationContent);

      const result = parseAllSpecs(tempDir);

      assert.ok(result.hasOwnProperty('specs'));
      assert.ok(result.hasOwnProperty('assertions'));
      assert.ok(result.hasOwnProperty('observations'));
      assert.strictEqual(result.observations.length, 1);
      assert.strictEqual(result.observations[0].id, 'test-observation');
    });

    test('should validate observation spec references', () => {
      // Create specs directory with a spec
      const specsDir = path.join(tempDir, 'specs');
      const specDir = path.join(specsDir, 'test-spec');
      const assertionsDir = path.join(specDir, 'assertions');
      fs.mkdirSync(assertionsDir, { recursive: true });

      const specContent = `---
id: test-spec
created: 2026-01-22T10:00:00Z
priority: 1
---

# Test Spec`;

      fs.writeFileSync(path.join(specDir, 'test-spec.md'), specContent);

      // Create observation referencing non-existent spec
      const observationsDir = path.join(tempDir, 'observations');
      fs.mkdirSync(observationsDir);

      const observationContent = `---
id: test-observation
created: 2026-01-22T10:00:00Z
type: code_spec_misalignment
severity: high
affected_specs:
  - non-existent-spec
affected_files:
  - src/test.js
---

# Test Observation`;

      fs.writeFileSync(path.join(observationsDir, 'test-observation.md'), observationContent);

      assert.throws(() => parseAllSpecs(tempDir), /references non-existent spec/);
    });

    test('should validate observation file paths are relative', () => {
      // Create specs directory with a spec
      const specsDir = path.join(tempDir, 'specs');
      const specDir = path.join(specsDir, 'test-spec');
      const assertionsDir = path.join(specDir, 'assertions');
      fs.mkdirSync(assertionsDir, { recursive: true });

      const specContent = `---
id: test-spec
created: 2026-01-22T10:00:00Z
priority: 1
---

# Test Spec`;

      fs.writeFileSync(path.join(specDir, 'test-spec.md'), specContent);

      // Create observation with absolute file path
      const observationsDir = path.join(tempDir, 'observations');
      fs.mkdirSync(observationsDir);

      const observationContent = `---
id: test-observation
created: 2026-01-22T10:00:00Z
type: code_spec_misalignment
severity: high
affected_specs:
  - test-spec
affected_files:
  - /absolute/path/to/file.js
---

# Test Observation`;

      fs.writeFileSync(path.join(observationsDir, 'test-observation.md'), observationContent);

      assert.throws(() => parseAllSpecs(tempDir), /contains absolute path/);
    });
  });
});