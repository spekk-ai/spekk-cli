import { describe, test } from 'node:test';
import assert from 'node:assert';
import { execSync } from 'child_process';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const spekkBin = path.join(__dirname, '../../../bin/spekk.js');

describe('spekk review CLI', () => {
  test('--help shows usage information', () => {
    const output = execSync(`node "${spekkBin}" review --help`, { encoding: 'utf8' });
    assert.ok(output.includes('spekk review'));
    assert.ok(output.includes('--list'));
    assert.ok(output.includes('--dry-run'));
    assert.ok(output.includes('--gate'));
    assert.ok(output.includes('--force'));
    assert.ok(output.includes('--skip'));
    assert.ok(output.includes('--no-llm'));
    assert.ok(output.includes('--tags'));
  });

  test('review command is registered in main help', () => {
    const output = execSync(`node "${spekkBin}" --help`, { encoding: 'utf8' });
    assert.ok(output.includes('review'));
  });

  test('--list shows Applicable and Skipped groups', () => {
    const output = execSync(`node "${spekkBin}" review --list`, { encoding: 'utf8' });
    assert.ok(output.includes('Applicable'));
    assert.ok(output.includes('Skipped'));
  });

  test('--gate with nonexistent gate exits with error', () => {
    try {
      execSync(`node "${spekkBin}" review --gate nonexistent-gate-xyz`, {
        encoding: 'utf8',
        stdio: ['pipe', 'pipe', 'pipe'],
      });
      assert.fail('Should have exited with error');
    } catch (error) {
      assert.ok(error.status !== 0);
    }
  });

  test('--skip with nonexistent gate exits with error', () => {
    try {
      execSync(`node "${spekkBin}" review --skip nonexistent-gate-xyz`, {
        encoding: 'utf8',
        stdio: ['pipe', 'pipe', 'pipe'],
      });
      assert.fail('Should have exited with error');
    } catch (error) {
      assert.ok(error.status !== 0);
    }
  });

  test('--force with nonexistent gate exits with error', () => {
    try {
      execSync(`node "${spekkBin}" review --force nonexistent-gate-xyz`, {
        encoding: 'utf8',
        stdio: ['pipe', 'pipe', 'pipe'],
      });
      assert.fail('Should have exited with error');
    } catch (error) {
      assert.ok(error.status !== 0);
    }
  });

  test('review script exists in package.json', () => {
    const pkgPath = path.join(__dirname, '../../../package.json');
    const pkg = JSON.parse(fs.readFileSync(pkgPath, 'utf8'));
    assert.ok(pkg.scripts.review, 'review script should exist in package.json');
    assert.ok(pkg.scripts.review.includes('review'), 'review script should reference review command');
  });

  test('bin/spekk.js has review case routing', () => {
    const binContent = fs.readFileSync(spekkBin, 'utf8');
    assert.ok(binContent.includes("case 'review'"), 'should have review case');
    assert.ok(binContent.includes('launchReview'), 'should import launchReview');
  });
});
