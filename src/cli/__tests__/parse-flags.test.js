import { test, describe } from 'node:test';
import assert from 'node:assert';
import { parseFlags } from '../parse-flags.js';

describe('Shared parseFlags utility', () => {
  const sampleDefs = {
    verbose: { flags: ['--verbose', '-v'], type: 'boolean' },
    output:  { flags: ['--output', '-o'],  type: 'string'  },
    force:   { flags: ['--force'],         type: 'boolean' },
  };

  test('returns defaults when no args provided', () => {
    const result = parseFlags([], sampleDefs);
    assert.strictEqual(result.verbose, false);
    assert.strictEqual(result.output, null);
    assert.strictEqual(result.force, false);
  });

  test('parses boolean flags', () => {
    const result = parseFlags(['--verbose'], sampleDefs);
    assert.strictEqual(result.verbose, true);
    assert.strictEqual(result.force, false);
  });

  test('parses short boolean flags', () => {
    const result = parseFlags(['-v'], sampleDefs);
    assert.strictEqual(result.verbose, true);
  });

  test('parses string flags with values', () => {
    const result = parseFlags(['--output', 'file.txt'], sampleDefs);
    assert.strictEqual(result.output, 'file.txt');
  });

  test('parses short string flags with values', () => {
    const result = parseFlags(['-o', 'file.txt'], sampleDefs);
    assert.strictEqual(result.output, 'file.txt');
  });

  test('handles multiple flags together', () => {
    const result = parseFlags(['--verbose', '--force', '--output', 'out.json'], sampleDefs);
    assert.strictEqual(result.verbose, true);
    assert.strictEqual(result.force, true);
    assert.strictEqual(result.output, 'out.json');
  });

  test('ignores unknown flags', () => {
    const result = parseFlags(['--unknown', '--verbose'], sampleDefs);
    assert.strictEqual(result.verbose, true);
    assert.strictEqual(Object.prototype.hasOwnProperty.call(result, 'unknown'), false);
  });

  test('supports custom default values', () => {
    const defs = {
      count: { flags: ['--count'], type: 'string', default: '10' },
    };
    const result = parseFlags([], defs);
    assert.strictEqual(result.count, '10');
  });
});
