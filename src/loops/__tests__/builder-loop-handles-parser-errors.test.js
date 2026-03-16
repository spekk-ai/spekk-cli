import { test, describe } from 'node:test';
import assert from 'node:assert';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Read source files to verify error handling behavior
const loopsSource = fs.readFileSync(path.join(__dirname, '..', 'index.js'), 'utf8');
const builderCliSource = fs.readFileSync(path.join(__dirname, '..', '..', 'builder', 'cli.js'), 'utf8');

describe('Builder loop handles parser errors gracefully', () => {

  test('loops/index.js: all error paths retry with delay instead of exiting', () => {
    // Parser command failure
    const parserFail = loopsSource.match(
      /catch\s*\(error\)\s*\{[^}]*?Parser command failed[^}]*?\}/s
    );
    assert.ok(parserFail, 'Should have a catch block for parser command failure');
    assert.ok(!parserFail[0].includes('process.exit'), 'Parser failure must not call process.exit');
    assert.ok(parserFail[0].includes('continue'), 'Parser failure must continue the loop');

    // Malformed JSON
    const jsonFail = loopsSource.match(
      /catch\s*\(error\)\s*\{[^}]*?Malformed JSON[^}]*?\}/s
    );
    assert.ok(jsonFail, 'Should have a catch block for malformed JSON');
    assert.ok(!jsonFail[0].includes('process.exit'), 'Malformed JSON must not call process.exit');
    assert.ok(jsonFail[0].includes('continue'), 'Malformed JSON must continue the loop');

    // Unexpected result type
    const unexpectedType = loopsSource.match(
      /if\s*\(parsedResult\.type\s*!==\s*'assertion'\)\s*\{[^}]*?\}/s
    );
    assert.ok(unexpectedType, 'Should have a block for unexpected result type');
    assert.ok(!unexpectedType[0].includes('process.exit'), 'Unexpected type must not call process.exit');
    assert.ok(unexpectedType[0].includes('continue'), 'Unexpected type must continue the loop');
  });

  test('builder/cli.js: continuous mode retries on all error paths', () => {
    // Parser error catch block retries
    const parserRetry = builderCliSource.match(
      /catch\s*\(error\)\s*\{[\s\S]*?Parser error \(transient\)[\s\S]*?continue;/
    );
    assert.ok(parserRetry, 'Parser errors in continuous mode must retry');

    // Error result type retries
    const errorRetry = builderCliSource.match(
      /if\s*\(result\.type\s*===\s*'error'\)\s*\{[^]*?continue;/s
    );
    assert.ok(errorRetry, 'Error result type in continuous mode must retry');

    // Unexpected result type retries
    const unexpectedRetry = builderCliSource.match(
      /if\s*\(result\.type\s*!==\s*'assertion'\)\s*\{[\s\S]*?continue;/
    );
    assert.ok(unexpectedRetry, 'Unexpected result type in continuous mode must retry');
  });

  test('builder/cli.js: --once mode still exits on errors', () => {
    assert.ok(
      builderCliSource.includes('if (once)'),
      'Code must check for --once mode before deciding exit vs retry'
    );
  });
});
