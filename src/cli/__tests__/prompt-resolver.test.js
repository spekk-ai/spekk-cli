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
  test('uses simplified agent names: coach, builder, observer, reviewer', () => {
    const resolver = new PromptResolver();
    const names = resolver.promptFiles.map(p => p.name);
    assert.deepStrictEqual(names, ['coach', 'builder', 'observer', 'reviewer']);
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
    for (const name of ['coach', 'builder', 'observer', 'reviewer']) {
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

  test('final prompt is always a single string regardless of layer count', () => {
    const globalDir = path.join(tempHome, '.spekk');
    mkdirSync(globalDir, { recursive: true });
    writeFileSync(path.join(globalDir, 'builder.prompt.md'), '## Global Extend');

    const localDir = path.join(tempCwd, '.spekk');
    mkdirSync(localDir, { recursive: true });
    writeFileSync(path.join(localDir, 'builder.prompt.md'), '## Local Extend');

    const resolver = new PromptResolver({ homeDir: tempHome, cwd: tempCwd });
    const content = resolver.getPromptContent('builder');

    assert.strictEqual(typeof content, 'string', 'getPromptContent must return a string');
    assert.ok(!Array.isArray(content), 'Result must not be an array');
  });

  test('cwd defaults to process.cwd when not provided', () => {
    const resolver = new PromptResolver({ homeDir: tempHome });
    assert.strictEqual(resolver.cwd, process.cwd());
  });
});

describe('PromptResolver works for all agents', () => {
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

  for (const agentName of ['coach', 'builder', 'observer', 'reviewer']) {
    test(`${agentName} supports global extend layer`, () => {
      const globalDir = path.join(tempHome, '.spekk');
      mkdirSync(globalDir, { recursive: true });
      writeFileSync(path.join(globalDir, `${agentName}.prompt.md`), `## Global ${agentName} extend`);

      const resolver = new PromptResolver({ homeDir: tempHome, cwd: tempCwd });
      const content = resolver.getPromptContent(agentName);

      // Verify the extend content is present and follows a separator
      assert.ok(content.includes(`## Global ${agentName} extend`), `${agentName} global extend should be appended`);
      assert.ok(content.includes('\n\n---\n\n'), `${agentName} should contain separator when extend is appended`);
      // The extend should appear after the last separator
      const lastSepIdx = content.lastIndexOf('\n\n---\n\n');
      const extendIdx = content.indexOf(`## Global ${agentName} extend`);
      assert.ok(extendIdx > lastSepIdx, `${agentName} global extend should follow the last separator`);
    });

    test(`${agentName} supports local override layer`, () => {
      const localDir = path.join(tempCwd, '.spekk');
      mkdirSync(localDir, { recursive: true });
      writeFileSync(path.join(localDir, `${agentName}.prompt.override.md`), `# Custom ${agentName}`);

      const resolver = new PromptResolver({ homeDir: tempHome, cwd: tempCwd });
      const content = resolver.getPromptContent(agentName);

      assert.strictEqual(content, `# Custom ${agentName}`);
    });

    test(`${agentName} supports all three layers combined`, () => {
      const globalDir = path.join(tempHome, '.spekk');
      mkdirSync(globalDir, { recursive: true });
      writeFileSync(path.join(globalDir, `${agentName}.prompt.md`), `## Global ${agentName}`);

      const localDir = path.join(tempCwd, '.spekk');
      mkdirSync(localDir, { recursive: true });
      writeFileSync(path.join(localDir, `${agentName}.prompt.md`), `## Local ${agentName}`);

      const resolver = new PromptResolver({ homeDir: tempHome, cwd: tempCwd });
      const content = resolver.getPromptContent(agentName);

      // Verify both extends are present and in correct order
      assert.ok(content.includes(`## Global ${agentName}`), `${agentName} should include global extend`);
      assert.ok(content.includes(`## Local ${agentName}`), `${agentName} should include local extend`);
      const globalIdx = content.indexOf(`## Global ${agentName}`);
      const localIdx = content.indexOf(`## Local ${agentName}`);
      assert.ok(globalIdx < localIdx, `${agentName} global extend should appear before local extend`);
    });
  }

  test('adding a new agent only requires adding an entry to promptFiles', () => {
    const resolver = new PromptResolver({ homeDir: tempHome, cwd: tempCwd });

    // Simulate adding a new agent by pushing to promptFiles
    const newAgentPromptPath = path.join(tempCwd, 'new-agent.prompt.md');
    writeFileSync(newAgentPromptPath, '# New Agent Base Prompt');
    resolver.promptFiles.push({ name: 'planner', path: newAgentPromptPath });

    // The new agent should work with layered prompts immediately
    const globalDir = path.join(tempHome, '.spekk');
    mkdirSync(globalDir, { recursive: true });
    writeFileSync(path.join(globalDir, 'planner.prompt.md'), '## Global planner extend');

    const content = resolver.getPromptContent('planner');
    const parts = content.split('\n\n---\n\n');
    assert.strictEqual(parts.length, 2);
    assert.ok(parts[0].includes('# New Agent Base Prompt'));
    assert.ok(parts[1].includes('## Global planner extend'));
  });

  test('all CLI launchers use simplified names via launchAgentWithPrompt or PromptResolver', () => {
    // This test documents the contract: each CLI uses the simplified name
    const resolver = new PromptResolver();
    const agentNames = resolver.promptFiles.map(p => p.name);

    // All four agents must be registered
    assert.ok(agentNames.includes('coach'), 'coach must be registered');
    assert.ok(agentNames.includes('builder'), 'builder must be registered');
    assert.ok(agentNames.includes('observer'), 'observer must be registered');
    assert.ok(agentNames.includes('reviewer'), 'reviewer must be registered');

    // None should use the old -agent suffix
    for (const name of agentNames) {
      assert.ok(!name.includes('-agent'), `Agent name "${name}" should not contain -agent suffix`);
    }
  });
});

describe('PromptResolver per-agent skills directories', () => {
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

  test('coach activation message includes coach skills directory', () => {
    const resolver = new PromptResolver({ homeDir: tempHome, cwd: tempCwd });
    const message = resolver.createActivationMessage('coach');
    assert.ok(message.includes('Skills directory:'), 'Coach should have Skills directory');
    assert.ok(message.includes('coach-skills-system'), 'Coach skills dir should reference coach-skills-system');
  });

  test('builder activation message includes builder skills directory', () => {
    const resolver = new PromptResolver({ homeDir: tempHome, cwd: tempCwd });
    const message = resolver.createActivationMessage('builder');
    assert.ok(message.includes('Skills directory:'), 'Builder should have Skills directory');
    assert.ok(message.includes('builder-skills-system'), 'Builder skills dir should reference builder-skills-system');
  });

  test('observer activation message does NOT include a skills directory', () => {
    const resolver = new PromptResolver({ homeDir: tempHome, cwd: tempCwd });
    const message = resolver.createActivationMessage('observer');
    assert.ok(!message.includes('Skills directory:'), 'Observer should not have Skills directory');
  });

  test('getSkillsPaths returns null for agents without skills', () => {
    const resolver = new PromptResolver({ homeDir: tempHome, cwd: tempCwd });
    assert.strictEqual(resolver.getSkillsPaths('observer'), null);
  });

  test('getSkillsPaths returns three paths for coach', () => {
    const resolver = new PromptResolver({ homeDir: tempHome, cwd: tempCwd });
    const paths = resolver.getSkillsPaths('coach');
    assert.ok(paths.packageDir.includes('coach-skills-system'));
    assert.ok(paths.globalDir.includes('coach-skills'));
    assert.ok(paths.localDir.includes('coach-skills'));
  });

  test('getSkillsPaths returns three paths for builder', () => {
    const resolver = new PromptResolver({ homeDir: tempHome, cwd: tempCwd });
    const paths = resolver.getSkillsPaths('builder');
    assert.ok(paths.packageDir.includes('builder-skills-system'));
    assert.ok(paths.globalDir.includes('builder-skills'));
    assert.ok(paths.localDir.includes('builder-skills'));
  });

  test('local skills dir is included in activation message when it exists', () => {
    const localSkillsDir = path.join(tempCwd, '.spekk', 'builder-skills');
    mkdirSync(localSkillsDir, { recursive: true });

    const resolver = new PromptResolver({ homeDir: tempHome, cwd: tempCwd });
    const message = resolver.createActivationMessage('builder');

    assert.ok(message.includes('Local skills:'), 'Should include Local skills line when dir exists');
    assert.ok(message.includes(localSkillsDir), 'Should include the actual local skills path');
  });

  test('global skills dir is included in activation message when it exists', () => {
    const globalSkillsDir = path.join(tempHome, '.spekk', 'coach-skills');
    mkdirSync(globalSkillsDir, { recursive: true });

    const resolver = new PromptResolver({ homeDir: tempHome, cwd: tempCwd });
    const message = resolver.createActivationMessage('coach');

    assert.ok(message.includes('Global skills:'), 'Should include Global skills line when dir exists');
    assert.ok(message.includes(globalSkillsDir), 'Should include the actual global skills path');
  });

  test('activation message omits global/local skills lines when dirs do not exist', () => {
    const resolver = new PromptResolver({ homeDir: tempHome, cwd: tempCwd });
    const message = resolver.createActivationMessage('builder');

    assert.ok(!message.includes('Global skills:'), 'Should not include Global skills when dir missing');
    assert.ok(!message.includes('Local skills:'), 'Should not include Local skills when dir missing');
  });
});
