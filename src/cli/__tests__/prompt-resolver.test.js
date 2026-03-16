import { test, describe, beforeEach, afterEach } from 'node:test';
import assert from 'node:assert';
import path from 'node:path';
import { mkdtempSync, mkdirSync, writeFileSync, rmSync } from 'fs';
import { tmpdir } from 'os';
import { fileURLToPath } from 'url';
import { PromptResolver } from '../prompt-resolver.js';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const projectRoot = path.join(__dirname, '../../..');

describe('PromptResolver', () => {
  test('uses simplified agent names: coach, builder, observer', () => {
    const resolver = new PromptResolver();
    const names = resolver.promptFiles.map(p => p.name);
    assert.deepStrictEqual(names, ['coach', 'builder', 'observer']);
  });

  test('base prompt files use <agent>.prompt.md naming convention', () => {
    const resolver = new PromptResolver();
    for (const promptFile of resolver.promptFiles) {
      const filename = path.basename(promptFile.path);
      assert.strictEqual(filename, `${promptFile.name}.prompt.md`,
        `Expected ${promptFile.name} prompt file to be named ${promptFile.name}.prompt.md, got ${filename}`);
    }
  });

  test('base prompt files are located in specs/<agent>-agent/ directories', () => {
    const resolver = new PromptResolver();
    for (const promptFile of resolver.promptFiles) {
      const expectedDir = path.join(projectRoot, `specs/${promptFile.name}-agent`);
      const actualDir = path.dirname(promptFile.path);
      assert.strictEqual(actualDir, expectedDir,
        `Expected ${promptFile.name} prompt to be in ${expectedDir}, got ${actualDir}`);
    }
  });

  test('getPromptContent loads content for simplified agent names', () => {
    const resolver = new PromptResolver();
    for (const name of ['coach', 'builder', 'observer']) {
      const content = resolver.getPromptContent(name);
      assert.ok(content.length > 0, `${name} prompt should have content`);
    }
  });

  test('getPromptContent throws for unknown agent name', () => {
    const resolver = new PromptResolver();
    assert.throws(
      () => resolver.getPromptContent('unknown-agent'),
      /Unknown agent: unknown-agent/
    );
  });

  test('getPromptContent throws for old-style agent names', () => {
    const resolver = new PromptResolver();
    assert.throws(
      () => resolver.getPromptContent('builder-agent'),
      /Unknown agent: builder-agent/
    );
  });

  test('createActivationMessage uses capitalized agent name in greeting', () => {
    const resolver = new PromptResolver();
    const message = resolver.createActivationMessage('builder');
    assert.ok(message.includes('You are the Builder Agent'),
      'Should capitalize agent name in activation message');
  });

  test('verifyPromptFilesExist returns empty array when all prompts exist', () => {
    const resolver = new PromptResolver();
    const missing = resolver.verifyPromptFilesExist();
    assert.deepStrictEqual(missing, [],
      'All base prompt files should exist');
  });
});

describe('PromptResolver global prompt layer', () => {
  let tempHome;

  beforeEach(() => {
    tempHome = mkdtempSync(path.join(tmpdir(), 'spekk-test-'));
  });

  afterEach(() => {
    rmSync(tempHome, { recursive: true, force: true });
  });

  test('returns base prompt silently when no global prompts exist', () => {
    const resolver = new PromptResolver({ homeDir: tempHome });
    const content = resolver.getPromptContent('builder');
    assert.ok(content.length > 0, 'Should return base prompt content');
    // No separator should be present when there are no extra layers
    assert.ok(!content.includes('\n\n---\n\n'), 'Should not contain separator when no layers appended');
  });

  test('appends global extend prompt after base with separator', () => {
    const globalDir = path.join(tempHome, '.spekk');
    mkdirSync(globalDir, { recursive: true });
    writeFileSync(path.join(globalDir, 'builder.prompt.md'), '## Custom Extensions\n\nUse strict mode.');

    const resolver = new PromptResolver({ homeDir: tempHome });
    const content = resolver.getPromptContent('builder');

    const parts = content.split('\n\n---\n\n');
    assert.strictEqual(parts.length, 2, 'Should have 2 layers (base + global extend)');
    assert.ok(parts[0].length > 0, 'First layer should be the base prompt');
    assert.ok(parts[1].includes('## Custom Extensions'), 'Second layer should be the global extend');
  });

  test('global override replaces the base prompt', () => {
    const globalDir = path.join(tempHome, '.spekk');
    mkdirSync(globalDir, { recursive: true });
    writeFileSync(path.join(globalDir, 'builder.prompt.override.md'), '# Completely Custom Builder');

    const resolver = new PromptResolver({ homeDir: tempHome });
    const content = resolver.getPromptContent('builder');

    assert.strictEqual(content, '# Completely Custom Builder');
  });

  test('global override as base with global extend appended', () => {
    const globalDir = path.join(tempHome, '.spekk');
    mkdirSync(globalDir, { recursive: true });
    writeFileSync(path.join(globalDir, 'builder.prompt.override.md'), '# Custom Base');
    writeFileSync(path.join(globalDir, 'builder.prompt.md'), '## Extra Instructions');

    const resolver = new PromptResolver({ homeDir: tempHome });
    const content = resolver.getPromptContent('builder');

    const parts = content.split('\n\n---\n\n');
    assert.strictEqual(parts.length, 2);
    assert.strictEqual(parts[0], '# Custom Base');
    assert.strictEqual(parts[1], '## Extra Instructions');
  });

  test('tilde expansion uses os.homedir by default', () => {
    const resolver = new PromptResolver();
    assert.ok(resolver.homeDir, 'Should have a homeDir set from os.homedir()');
    assert.ok(!resolver.homeDir.includes('~'), 'homeDir should be an absolute path, not contain tilde');
  });

  test('throws when base prompt is missing and no override exists', () => {
    const resolver = new PromptResolver({ homeDir: tempHome });
    // Point base to a nonexistent file
    resolver.promptFiles[1].path = path.join(projectRoot, 'specs/builder-agent/nonexistent.prompt.md');
    assert.throws(
      () => resolver.getPromptContent('builder'),
      /Prompt file not found/,
      'Should throw with clear error identifying the missing file'
    );
  });

  test('succeeds when base prompt is missing but global override exists', () => {
    const globalDir = path.join(tempHome, '.spekk');
    mkdirSync(globalDir, { recursive: true });
    writeFileSync(path.join(globalDir, 'builder.prompt.override.md'), '# Override Builder');

    const resolver = new PromptResolver({ homeDir: tempHome });
    // Point base to a nonexistent file
    resolver.promptFiles[1].path = path.join(projectRoot, 'specs/builder-agent/nonexistent.prompt.md');

    const content = resolver.getPromptContent('builder');
    assert.ok(content.includes('# Override Builder'),
      'Should use global override when base prompt is missing');
  });
});

describe('PromptResolver local prompt layer', () => {
  let tempHome;
  let tempCwd;

  beforeEach(() => {
    tempHome = mkdtempSync(path.join(tmpdir(), 'spekk-home-'));
    tempCwd = mkdtempSync(path.join(tmpdir(), 'spekk-cwd-'));
  });

  afterEach(() => {
    rmSync(tempHome, { recursive: true, force: true });
    rmSync(tempCwd, { recursive: true, force: true });
  });

  test('returns base prompt silently when no local prompts exist', () => {
    const resolver = new PromptResolver({ homeDir: tempHome, cwd: tempCwd });
    const content = resolver.getPromptContent('builder');
    assert.ok(content.length > 0, 'Should return base prompt content');
    assert.ok(!content.includes('\n\n---\n\n'), 'No separator when no extra layers');
  });

  test('local extend is appended after global extend', () => {
    const globalDir = path.join(tempHome, '.spekk');
    mkdirSync(globalDir, { recursive: true });
    writeFileSync(path.join(globalDir, 'builder.prompt.md'), '## Global Extend');

    const localDir = path.join(tempCwd, '.spekk');
    mkdirSync(localDir, { recursive: true });
    writeFileSync(path.join(localDir, 'builder.prompt.md'), '## Local Extend');

    const resolver = new PromptResolver({ homeDir: tempHome, cwd: tempCwd });
    const content = resolver.getPromptContent('builder');

    const parts = content.split('\n\n---\n\n');
    assert.strictEqual(parts.length, 3, 'Should have 3 layers (base + global extend + local extend)');
    assert.ok(parts[1].includes('## Global Extend'), 'Second layer should be global extend');
    assert.ok(parts[2].includes('## Local Extend'), 'Third layer should be local extend');
  });

  test('local override takes precedence over global override', () => {
    const globalDir = path.join(tempHome, '.spekk');
    mkdirSync(globalDir, { recursive: true });
    writeFileSync(path.join(globalDir, 'builder.prompt.override.md'), '# Global Override');

    const localDir = path.join(tempCwd, '.spekk');
    mkdirSync(localDir, { recursive: true });
    writeFileSync(path.join(localDir, 'builder.prompt.override.md'), '# Local Override');

    const resolver = new PromptResolver({ homeDir: tempHome, cwd: tempCwd });
    const content = resolver.getPromptContent('builder');

    assert.strictEqual(content, '# Local Override');
  });

  test('local override as base with both global and local extends appended', () => {
    const globalDir = path.join(tempHome, '.spekk');
    mkdirSync(globalDir, { recursive: true });
    writeFileSync(path.join(globalDir, 'builder.prompt.override.md'), '# Global Override (ignored)');
    writeFileSync(path.join(globalDir, 'builder.prompt.md'), '## Global Extend');

    const localDir = path.join(tempCwd, '.spekk');
    mkdirSync(localDir, { recursive: true });
    writeFileSync(path.join(localDir, 'builder.prompt.override.md'), '# Local Override');
    writeFileSync(path.join(localDir, 'builder.prompt.md'), '## Local Extend');

    const resolver = new PromptResolver({ homeDir: tempHome, cwd: tempCwd });
    const content = resolver.getPromptContent('builder');

    const parts = content.split('\n\n---\n\n');
    assert.strictEqual(parts.length, 3, 'Should have 3 layers');
    assert.strictEqual(parts[0], '# Local Override', 'Base should be local override');
    assert.strictEqual(parts[1], '## Global Extend', 'Second layer should be global extend');
    assert.strictEqual(parts[2], '## Local Extend', 'Third layer should be local extend');
  });

  test('local extend only (no global prompts) appends after base', () => {
    const localDir = path.join(tempCwd, '.spekk');
    mkdirSync(localDir, { recursive: true });
    writeFileSync(path.join(localDir, 'builder.prompt.md'), '## Project-Specific Rules');

    const resolver = new PromptResolver({ homeDir: tempHome, cwd: tempCwd });
    const content = resolver.getPromptContent('builder');

    const parts = content.split('\n\n---\n\n');
    assert.strictEqual(parts.length, 2, 'Should have 2 layers (base + local extend)');
    assert.ok(parts[1].includes('## Project-Specific Rules'), 'Second layer should be local extend');
  });

  test('local override only (no global prompts) replaces base', () => {
    const localDir = path.join(tempCwd, '.spekk');
    mkdirSync(localDir, { recursive: true });
    writeFileSync(path.join(localDir, 'builder.prompt.override.md'), '# Project Custom Builder');

    const resolver = new PromptResolver({ homeDir: tempHome, cwd: tempCwd });
    const content = resolver.getPromptContent('builder');

    assert.strictEqual(content, '# Project Custom Builder');
  });

  test('cwd defaults to process.cwd when not provided', () => {
    const resolver = new PromptResolver({ homeDir: tempHome });
    assert.strictEqual(resolver.cwd, process.cwd());
  });
});
