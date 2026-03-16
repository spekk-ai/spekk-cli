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

describe('Builder loop handles parser errors gracefully (src/loops/index.js)', () => {

  test('parser command failure logs warning and retries instead of exiting', () => {
    // The catch block for execSync should NOT call process.exit
    // It should log a warning and continue the loop
    const catchBlock = loopsSource.match(
      /catch\s*\(error\)\s*\{[^}]*?Parser command failed[^}]*?\}/s
    );
    assert.ok(catchBlock, 'Should have a catch block for parser command failure');
    assert.ok(
      !catchBlock[0].includes('process.exit'),
      'Parser command failure catch block must NOT call process.exit'
    );
    assert.ok(
      catchBlock[0].includes('continue'),
      'Parser command failure catch block must continue the loop'
    );
  });

  test('malformed JSON from parser logs warning and retries instead of exiting', () => {
    // The catch block for JSON.parse should NOT call process.exit
    const catchBlock = loopsSource.match(
      /catch\s*\(error\)\s*\{[^}]*?Malformed JSON[^}]*?\}/s
    );
    assert.ok(catchBlock, 'Should have a catch block for malformed JSON');
    assert.ok(
      !catchBlock[0].includes('process.exit'),
      'Malformed JSON catch block must NOT call process.exit'
    );
    assert.ok(
      catchBlock[0].includes('continue'),
      'Malformed JSON catch block must continue the loop'
    );
  });

  test('unexpected result type logs warning and retries instead of exiting', () => {
    // The block handling unexpected type should NOT call process.exit
    const unexpectedBlock = loopsSource.match(
      /if\s*\(parsedResult\.type\s*!==\s*'assertion'\)\s*\{[^}]*?\}/s
    );
    assert.ok(unexpectedBlock, 'Should have a block for unexpected result type');
    assert.ok(
      !unexpectedBlock[0].includes('process.exit'),
      'Unexpected result type block must NOT call process.exit'
    );
    assert.ok(
      unexpectedBlock[0].includes('continue'),
      'Unexpected result type block must continue the loop'
    );
  });

  test('all parser error paths include a retry delay', () => {
    // All three error paths should include a setTimeout for delay
    const parserFailMatch = loopsSource.match(
      /Parser command failed[^}]*?setTimeout[^}]*?(\d+)/s
    );
    assert.ok(parserFailMatch, 'Parser command failure should include a setTimeout delay');
    assert.ok(
      parseInt(parserFailMatch[1]) >= 1000,
      'Retry delay should be at least 1 second'
    );

    const jsonFailMatch = loopsSource.match(
      /Malformed JSON[^}]*?setTimeout[^}]*?(\d+)/s
    );
    assert.ok(jsonFailMatch, 'Malformed JSON should include a setTimeout delay');

    const unexpectedMatch = loopsSource.match(
      /Unexpected result type[^}]*?setTimeout[^}]*?(\d+)/s
    );
    assert.ok(unexpectedMatch, 'Unexpected result type should include a setTimeout delay');
  });
});

describe('Builder loop handles parser errors gracefully (src/builder/cli.js)', () => {

  test('getNextAssertion failure in continuous mode retries instead of exiting', () => {
    // The catch block after getNextAssertion should retry in continuous mode
    // It should contain 'transient' and 'continue'
    assert.ok(
      builderCliSource.includes('Parser error (transient)'),
      'Should label parser errors as transient'
    );
    // Find the section between the catch and the next result.type check
    const section = builderCliSource.match(
      /catch\s*\(error\)\s*\{[\s\S]*?Parser error \(transient\)[\s\S]*?continue;/
    );
    assert.ok(section, 'Should have a catch block that retries on parser errors');
    assert.ok(
      section[0].includes('continue'),
      'Parser error catch block in continuous mode must continue the loop'
    );
  });

  test('error result type in continuous mode retries instead of exiting', () => {
    // When result.type === 'error' in continuous mode, should retry
    const errorBlock = builderCliSource.match(
      /if\s*\(result\.type\s*===\s*'error'\)\s*\{[^]*?continue;/s
    );
    assert.ok(errorBlock, 'Should handle error result type with retry in continuous mode');
    assert.ok(
      errorBlock[0].includes('Retrying'),
      'Error result type handler should mention retrying'
    );
  });

  test('unexpected result type in continuous mode retries instead of exiting', () => {
    // When result.type is not 'assertion' in continuous mode, should retry
    // The block spans multiple lines with a nested if(once) check
    const unexpectedBlock = builderCliSource.match(
      /if\s*\(result\.type\s*!==\s*'assertion'\)\s*\{[\s\S]*?Unexpected result type[\s\S]*?continue;/
    );
    assert.ok(unexpectedBlock, 'Should handle unexpected result type with retry');
    assert.ok(
      unexpectedBlock[0].includes('if (once)'),
      'Unexpected result type block should only exit in --once mode'
    );
    assert.ok(
      unexpectedBlock[0].includes('continue'),
      'Unexpected result type in continuous mode must continue the loop'
    );
  });

  test('--once mode still exits on parser errors (only continuous mode retries)', () => {
    // In --once mode, process.exit(1) is still correct behavior
    // The code should check for `once` before deciding to exit vs retry
    assert.ok(
      builderCliSource.includes('if (once)'),
      'Code should check for once mode before deciding exit vs retry'
    );
  });

  test('all continuous-mode parser error paths include retry delays', () => {
    // Check that retry paths have setTimeout delays
    const parserErrorRetry = builderCliSource.match(
      /Parser error \(transient\)[^}]*?setTimeout[^}]*?(\d+)/s
    );
    assert.ok(parserErrorRetry, 'Parser error retry should include a setTimeout delay');
    assert.ok(
      parseInt(parserErrorRetry[1]) >= 1000,
      'Retry delay should be at least 1 second'
    );
  });

  test('complete type still handled correctly (waits for new work)', () => {
    // The 'complete' type should still work: wait and continue in continuous mode
    const completeBlock = builderCliSource.match(
      /if\s*\(result\.type\s*===\s*'complete'\)\s*\{[^}]*?continue/s
    );
    assert.ok(completeBlock, 'Complete type should still wait and continue in continuous mode');
  });
});
