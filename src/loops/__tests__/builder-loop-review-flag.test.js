import { test, describe } from 'node:test';
import assert from 'node:assert';
import fs from 'node:fs';
import path from 'node:path';
import { execSync } from 'child_process';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const loopsSource = fs.readFileSync(path.join(__dirname, '..', 'index.js'), 'utf8');
const spekkBin = path.join(__dirname, '../../../bin/spekk.js');
const spekkBinSource = fs.readFileSync(spekkBin, 'utf8');

describe('Builder loop --review flag', () => {
  test('runBuilderLoop accepts args parameter', () => {
    assert.ok(
      loopsSource.includes('async function runBuilderLoop(args'),
      'runBuilderLoop should accept args parameter'
    );
  });

  test('review flag is parsed from args', () => {
    assert.ok(
      loopsSource.includes("args.includes('--review')"),
      'Should check for --review flag in args'
    );
  });

  test('review step runs spekk review --no-llm when flag is set', () => {
    assert.ok(
      loopsSource.includes('review --no-llm'),
      'Should run spekk review --no-llm'
    );
  });

  test('review step is conditional on reviewEnabled', () => {
    assert.ok(
      loopsSource.includes('if (reviewEnabled)'),
      'Review step should be gated on reviewEnabled'
    );
  });

  test('review failures do not block the loop (no process.exit in review block)', () => {
    // Extract the review block
    const reviewBlock = loopsSource.match(
      /Running quality gates[\s\S]*?Review gates completed/
    );
    assert.ok(reviewBlock, 'Should have a review gates block');
    assert.ok(
      !reviewBlock[0].includes('process.exit'),
      'Review block must not call process.exit'
    );
  });

  test('review failures are logged as warnings', () => {
    assert.ok(
      loopsSource.includes('non-blocking'),
      'Review failures should be described as non-blocking'
    );
  });

  test('bin/spekk.js passes args to runBuilderLoop', () => {
    assert.ok(
      spekkBinSource.includes('runBuilderLoop(args.slice(2))'),
      'Should pass remaining args to runBuilderLoop'
    );
  });

  test('loop help documents --review flag', () => {
    const output = execSync(`node "${spekkBin}" loop help`, { encoding: 'utf8' });
    assert.ok(output.includes('--review'), 'Loop help should document --review flag');
  });
});
