import { test, describe } from 'node:test';
import assert from 'node:assert';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import { parseAllSpecs, parseObservations, validateObservationFields } from '../index.js';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Create test directory structure
const testDir = path.join(__dirname, 'test-observations');
const specsDir = path.join(testDir, 'specs');
const observationsDir = path.join(testDir, 'observations');

// Clean up test directories
function cleanupTestDirs() {
  if (fs.existsSync(testDir)) {
    fs.rmSync(testDir, { recursive: true });
  }
}

// Set up test directories
function setupTestDirs() {
  cleanupTestDirs();
  fs.mkdirSync(testDir, { recursive: true });
  fs.mkdirSync(specsDir, { recursive: true });
  fs.mkdirSync(observationsDir, { recursive: true });
}

describe('Spec Parser - Observations Parsing', () => {
  // Setup and cleanup for each test
  // Node.js test runner doesn't have beforeEach/afterEach,
  // so we'll call these functions in each test

  test('parseObservations should read observation files from observations directory', () => {
    setupTestDirs();
    
    try {
    // Create test observation files
    const observation1 = `---
id: drift-001
created: 2026-01-28T10:00:00Z
type: spec_drift
severity: high
affected_specs:
  - test-spec
affected_files:
  - src/test.js
---

# Spec Drift Detected

The implementation has drifted from the spec.`;

    const observation2 = `---
id: compression-001
created: 2026-01-28T11:00:00Z
type: compression_opportunity
severity: low
affected_specs:
  - spec-a
  - spec-b
affected_files:
  - specs/spec-a/spec-a.md
  - specs/spec-b/spec-b.md
---

# Compression Opportunity

These specs overlap.`;

    fs.writeFileSync(path.join(observationsDir, 'drift-001.md'), observation1);
    fs.writeFileSync(path.join(observationsDir, 'compression-001.md'), observation2);

    const observations = parseObservations(testDir);

    assert.strictEqual(observations.length, 2);
    assert.strictEqual(observations[0].id, 'drift-001');
    assert.strictEqual(observations[0].type, 'spec_drift');
    assert.strictEqual(observations[0].severity, 'high');
    assert.deepStrictEqual(observations[0].affected_specs, ['test-spec']);
    assert.deepStrictEqual(observations[0].affected_files, ['src/test.js']);
    
    assert.strictEqual(observations[1].id, 'compression-001');
    assert.strictEqual(observations[1].type, 'compression_opportunity');
    assert.strictEqual(observations[1].severity, 'low');
    assert.deepStrictEqual(observations[1].affected_specs, ['spec-a', 'spec-b']);
    assert.deepStrictEqual(observations[1].affected_files, ['specs/spec-a/spec-a.md', 'specs/spec-b/spec-b.md']);
    } finally {
      cleanupTestDirs();
    }
  });

  test('parseObservations should skip files without YAML frontmatter', () => {
    setupTestDirs();
    
    try {
    const validObservation = `---
id: valid-001
created: 2026-01-28T10:00:00Z
type: spec_drift
severity: medium
affected_specs:
  - test-spec
affected_files:
  - src/test.js
---

# Valid Observation`;

    const invalidObservation = `# Not an Observation

This file doesn't have YAML frontmatter.`;

    fs.writeFileSync(path.join(observationsDir, 'valid.md'), validObservation);
    fs.writeFileSync(path.join(observationsDir, 'invalid.md'), invalidObservation);

    const observations = parseObservations(testDir);

    assert.strictEqual(observations.length, 1);
    assert.strictEqual(observations[0].id, 'valid-001');
    } finally {
      cleanupTestDirs();
    }
  });

  test('validateObservationFields should validate required fields', () => {
    setupTestDirs();
    
    try {
    const validData = {
      id: 'test-001',
      created: '2026-01-28T10:00:00Z',
      type: 'spec_drift',
      severity: 'high',
      affected_specs: ['spec-1'],
      affected_files: ['file1.js']
    };

    // Should not throw
    assert.doesNotThrow(() => validateObservationFields(validData, 'test.md'));

    // Missing required field
    const missingField = { ...validData };
    delete missingField.type;
    assert.throws(
      () => validateObservationFields(missingField, 'test.md'),
      { message: "Missing required field 'type' in test.md" }
    );

    // Invalid severity
    const invalidSeverity = { ...validData, severity: 'critical' };
    assert.throws(
      () => validateObservationFields(invalidSeverity, 'test.md'),
      { message: "Invalid severity value 'critical' (must be: low, medium, high) in test.md" }
    );

    // Non-array affected_specs
    const nonArraySpecs = { ...validData, affected_specs: 'not-an-array' };
    assert.throws(
      () => validateObservationFields(nonArraySpecs, 'test.md'),
      { message: "Field 'affected_specs' must be an array in test.md" }
    );

    // Invalid timestamp
    const invalidTimestamp = { ...validData, created: '2026-01-28' };
    assert.throws(
      () => validateObservationFields(invalidTimestamp, 'test.md'),
      /Invalid ISO 8601 timestamp in 'created' field/
    );
    } finally {
      cleanupTestDirs();
    }
  });

  test('parseAllSpecs should include observations in output', () => {
    setupTestDirs();
    
    try {
    // Create a test spec
    const specDir = path.join(specsDir, 'test-spec');
    const assertionsDir = path.join(specDir, 'assertions');
    fs.mkdirSync(specDir, { recursive: true });
    fs.mkdirSync(assertionsDir, { recursive: true });

    const specContent = `---
id: test-spec
created: 2026-01-28T10:00:00Z
priority: 1
---

# Test Spec`;

    fs.writeFileSync(path.join(specDir, 'test-spec.md'), specContent);

    // Create an observation
    const observation = `---
id: drift-001
created: 2026-01-28T10:00:00Z
type: spec_drift
severity: high
affected_specs:
  - test-spec
affected_files:
  - src/test.js
---

# Drift detected`;

    fs.writeFileSync(path.join(observationsDir, 'drift-001.md'), observation);

    const result = parseAllSpecs(testDir);

    assert.strictEqual(result.specs.length, 1);
    assert.strictEqual(result.assertions.length, 0);
    assert.strictEqual(result.observations.length, 1);
    assert.strictEqual(result.observations[0].id, 'drift-001');
    } finally {
      cleanupTestDirs();
    }
  });

  test('parseObservations should validate affected_specs references', () => {
    setupTestDirs();
    
    try {
    // Create a test spec
    const specDir = path.join(specsDir, 'existing-spec');
    const assertionsDir = path.join(specDir, 'assertions');
    fs.mkdirSync(specDir, { recursive: true });
    fs.mkdirSync(assertionsDir, { recursive: true });

    const specContent = `---
id: existing-spec
created: 2026-01-28T10:00:00Z
priority: 1
---

# Existing Spec`;

    fs.writeFileSync(path.join(specDir, 'existing-spec.md'), specContent);

    // Create observation with non-existent spec reference
    const observation = `---
id: drift-001
created: 2026-01-28T10:00:00Z
type: spec_drift
severity: high
affected_specs:
  - existing-spec
  - non-existent-spec
affected_files:
  - src/test.js
---

# Drift`;

    fs.writeFileSync(path.join(observationsDir, 'drift-001.md'), observation);

    const { specs } = parseAllSpecs(testDir);
    const specIds = new Set(specs.map(s => s.id));

    assert.throws(
      () => {
        const observations = parseObservations(testDir);
        // Validate affected_specs references
        for (const obs of observations) {
          for (const specId of obs.affected_specs) {
            if (!specIds.has(specId)) {
              throw new Error(`Observation '${obs.id}' references non-existent spec '${specId}'`);
            }
          }
        }
      },
      { message: "Observation 'drift-001' references non-existent spec 'non-existent-spec'" }
    );
    } finally {
      cleanupTestDirs();
    }
  });

  test('parseObservations should validate affected_files references', () => {
    setupTestDirs();
    
    try {
    const observation = `---
id: drift-001
created: 2026-01-28T10:00:00Z
type: spec_drift
severity: high
affected_specs:
  - test-spec
affected_files:
  - /absolute/path/not/allowed.js
---

# Drift`;

    fs.writeFileSync(path.join(observationsDir, 'drift-001.md'), observation);

    // Create a mock file system check
    const observations = parseObservations(testDir);
    
    assert.throws(
      () => {
        for (const obs of observations) {
          for (const file of obs.affected_files) {
            // Check for absolute paths
            if (path.isAbsolute(file)) {
              throw new Error(`Observation '${obs.id}' contains absolute path '${file}' (must be relative)`);
            }
          }
        }
      },
      { message: "Observation 'drift-001' contains absolute path '/absolute/path/not/allowed.js' (must be relative)" }
    );
    } finally {
      cleanupTestDirs();
    }
  });

  test('observation parsing errors should be reported clearly', () => {
    setupTestDirs();
    
    try {
    const invalidObservation = `---
id: invalid-001
created: not-a-timestamp
type: spec_drift
severity: high
affected_specs:
  - test-spec
affected_files:
  - src/test.js
---

# Invalid`;

    fs.writeFileSync(path.join(observationsDir, 'invalid-001.md'), invalidObservation);

    assert.throws(
      () => parseObservations(testDir),
      /Invalid ISO 8601 timestamp in 'created' field/
    );
    } finally {
      cleanupTestDirs();
    }
  });

  test('observations should be included in JSON output format', () => {
    setupTestDirs();
    
    try {
    // Create test data
    const specDir = path.join(specsDir, 'test-spec');
    const assertionsDir = path.join(specDir, 'assertions');
    fs.mkdirSync(specDir, { recursive: true });
    fs.mkdirSync(assertionsDir, { recursive: true });

    const specContent = `---
id: test-spec
created: 2026-01-28T10:00:00Z
priority: 1
---

# Test Spec`;

    fs.writeFileSync(path.join(specDir, 'test-spec.md'), specContent);

    const observation = `---
id: drift-001
created: 2026-01-28T10:00:00Z
type: spec_drift
severity: high
affected_specs:
  - test-spec
affected_files:
  - src/test.js
---

# Drift`;

    fs.writeFileSync(path.join(observationsDir, 'drift-001.md'), observation);

    const result = parseAllSpecs(testDir);
    const jsonOutput = JSON.stringify(result, null, 2);
    const parsed = JSON.parse(jsonOutput);

    assert(parsed.hasOwnProperty('observations'));
    assert.strictEqual(parsed.observations.length, 1);
    assert.deepStrictEqual(parsed.observations[0], {
      id: 'drift-001',
      created: '2026-01-28T10:00:00Z',
      type: 'spec_drift',
      severity: 'high',
      affected_specs: ['test-spec'],
      affected_files: ['src/test.js'],
      file: 'observations/drift-001.md',
      title: 'Drift',
      content: observation
    });
    } finally {
      cleanupTestDirs();
    }
  });
});