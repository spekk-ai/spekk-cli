import { test, describe } from 'node:test';
import assert from 'node:assert';
import { generateAgentToken } from '../tokens.js';

describe('generateAgentToken', () => {
  test('returns a 43-character string', () => {
    const token = generateAgentToken();
    assert.strictEqual(token.length, 43, `Expected length 43, got ${token.length}: "${token}"`);
  });

  test('only contains URL-safe characters [A-Za-z0-9_-]', () => {
    const token = generateAgentToken();
    assert.match(token, /^[A-Za-z0-9_-]+$/, `Token contains invalid characters: "${token}"`);
  });

  test('returns different values on successive calls', () => {
    const a = generateAgentToken();
    const b = generateAgentToken();
    assert.notStrictEqual(a, b, `Expected unique tokens, but got the same value twice: "${a}"`);
  });
});
