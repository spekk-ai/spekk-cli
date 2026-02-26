import { test, describe } from 'node:test';
import assert from 'node:assert';
import { mkdirSync, rmSync, writeFileSync, unlinkSync, existsSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { watchSpecs } from '../show/watcher.js';

describe('File Watcher Module (polling)', () => {
  let testDir;
  let specsDir;

  function setup() {
    testDir = join(tmpdir(), `spekk-watcher-test-${Date.now()}-${Math.random().toString(36).slice(2)}`);
    specsDir = join(testDir, 'specs');
    mkdirSync(specsDir, { recursive: true });
  }

  function teardown() {
    if (existsSync(testDir)) {
      rmSync(testDir, { recursive: true, force: true });
    }
  }

  test('detects file modifications', async () => {
    setup();
    try {
      writeFileSync(join(specsDir, 'existing.md'), 'original');
      let callCount = 0;
      const stop = watchSpecs(specsDir, () => { callCount++; });

      // Wait for first poll to pass with no changes
      await new Promise(r => setTimeout(r, 600));
      assert.strictEqual(callCount, 0, 'No changes yet');

      // Modify the file — ensure mtime advances
      await new Promise(r => setTimeout(r, 50));
      writeFileSync(join(specsDir, 'existing.md'), 'modified');

      // Wait for poll to detect
      await new Promise(r => setTimeout(r, 600));
      assert.ok(callCount >= 1, 'onChange should fire after modification');
      stop();
    } finally {
      teardown();
    }
  });

  test('detects new .md files added', async () => {
    setup();
    try {
      let callCount = 0;
      const stop = watchSpecs(specsDir, () => { callCount++; });

      // Add a new .md file after watcher starts
      await new Promise(r => setTimeout(r, 100));
      writeFileSync(join(specsDir, 'new-spec.md'), 'new content');

      await new Promise(r => setTimeout(r, 600));
      assert.ok(callCount >= 1, 'onChange should fire when new .md file appears');
      stop();
    } finally {
      teardown();
    }
  });

  test('detects .md files deleted', async () => {
    setup();
    try {
      writeFileSync(join(specsDir, 'to-delete.md'), 'will be deleted');
      let callCount = 0;
      const stop = watchSpecs(specsDir, () => { callCount++; });

      // Wait for initial poll with no changes
      await new Promise(r => setTimeout(r, 600));
      assert.strictEqual(callCount, 0);

      unlinkSync(join(specsDir, 'to-delete.md'));

      await new Promise(r => setTimeout(r, 600));
      assert.ok(callCount >= 1, 'onChange should fire when .md file is deleted');
      stop();
    } finally {
      teardown();
    }
  });

  test('ignores non-.md files', async () => {
    setup();
    try {
      let callCount = 0;
      const stop = watchSpecs(specsDir, () => { callCount++; });

      await new Promise(r => setTimeout(r, 100));
      writeFileSync(join(specsDir, 'readme.txt'), 'text');
      writeFileSync(join(specsDir, 'data.json'), '{}');
      writeFileSync(join(specsDir, 'script.js'), '//js');

      await new Promise(r => setTimeout(r, 600));
      assert.strictEqual(callCount, 0, 'Non-.md files should be ignored');
      stop();
    } finally {
      teardown();
    }
  });

  test('cleanup stops polling', async () => {
    setup();
    try {
      let callCount = 0;
      const stop = watchSpecs(specsDir, () => { callCount++; });

      stop();

      // Write .md file after cleanup
      writeFileSync(join(specsDir, 'after-stop.md'), 'should not trigger');

      await new Promise(r => setTimeout(r, 700));
      assert.strictEqual(callCount, 0, 'No changes should fire after cleanup');
    } finally {
      teardown();
    }
  });

  test('onChange fires once per poll cycle even with multiple changes', async () => {
    setup();
    try {
      let callCount = 0;
      const stop = watchSpecs(specsDir, () => { callCount++; });

      // Create multiple .md files between polls
      await new Promise(r => setTimeout(r, 100));
      writeFileSync(join(specsDir, 'a.md'), 'a');
      writeFileSync(join(specsDir, 'b.md'), 'b');
      writeFileSync(join(specsDir, 'c.md'), 'c');

      // Wait for exactly one poll cycle to detect all three
      await new Promise(r => setTimeout(r, 600));
      assert.strictEqual(callCount, 1, 'Multiple changes in one cycle should produce one onChange call');
      stop();
    } finally {
      teardown();
    }
  });
});
