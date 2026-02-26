import { test, describe } from 'node:test';
import assert from 'node:assert';
import { mkdirSync, rmSync, writeFileSync, existsSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { watchSpecs } from '../show/watcher.js';

describe('File Watcher Module', () => {
  const testDir = join(tmpdir(), `spekk-watcher-test-${Date.now()}`);

  function setup() {
    if (existsSync(testDir)) {
      rmSync(testDir, { recursive: true, force: true });
    }
    mkdirSync(join(testDir, 'specs'), { recursive: true });
  }

  function cleanup() {
    if (existsSync(testDir)) {
      rmSync(testDir, { recursive: true, force: true });
    }
  }

  test('exports watchSpecs function', () => {
    assert.strictEqual(typeof watchSpecs, 'function');
  });

  test('returns a cleanup function', () => {
    setup();
    try {
      const stop = watchSpecs(join(testDir, 'specs'), () => {});
      assert.strictEqual(typeof stop, 'function');
      stop();
    } finally {
      cleanup();
    }
  });

  test('invokes onChange for .md file changes with debounce', async () => {
    setup();
    try {
      const specsDir = join(testDir, 'specs');
      let callCount = 0;

      const stop = watchSpecs(specsDir, () => {
        callCount++;
      });

      // Write multiple .md files in rapid succession
      writeFileSync(join(specsDir, 'spec-a.md'), 'content a');
      writeFileSync(join(specsDir, 'spec-b.md'), 'content b');
      writeFileSync(join(specsDir, 'spec-c.md'), 'content c');

      // Wait for debounce to fire (300ms debounce + buffer)
      await new Promise(resolve => setTimeout(resolve, 600));

      // Should have been debounced into a single call
      assert.strictEqual(callCount, 1, 'Rapid changes should be debounced into one call');

      stop();
    } finally {
      cleanup();
    }
  });

  test('ignores non-.md file changes', async () => {
    setup();
    try {
      const specsDir = join(testDir, 'specs');
      let callCount = 0;

      const stop = watchSpecs(specsDir, () => {
        callCount++;
      });

      // Write non-md files
      writeFileSync(join(specsDir, 'readme.txt'), 'text content');
      writeFileSync(join(specsDir, 'data.json'), '{}');

      // Wait longer than debounce window
      await new Promise(resolve => setTimeout(resolve, 600));

      assert.strictEqual(callCount, 0, 'Non-.md file changes should be ignored');

      stop();
    } finally {
      cleanup();
    }
  });

  test('cleanup function stops watching', async () => {
    setup();
    try {
      const specsDir = join(testDir, 'specs');
      let callCount = 0;

      const stop = watchSpecs(specsDir, () => {
        callCount++;
      });

      // Stop watching immediately
      stop();

      // Write .md file after cleanup
      writeFileSync(join(specsDir, 'after-stop.md'), 'should not trigger');

      // Wait longer than debounce window
      await new Promise(resolve => setTimeout(resolve, 600));

      assert.strictEqual(callCount, 0, 'No changes should fire after cleanup');
    } finally {
      cleanup();
    }
  });
});
