import { describe, test } from 'node:test';
import assert from 'node:assert';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import { loadGates } from '../gate-loader.js';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const projectRoot = path.join(__dirname, '../../..');

describe('Default gates', () => {
  test('gates/ directory exists at package root', () => {
    assert.ok(fs.existsSync(path.join(projectRoot, 'gates')));
  });

  test('validate-testids.gate.md exists with correct structure', () => {
    const gatePath = path.join(projectRoot, 'gates', 'validate-testids.gate.md');
    assert.ok(fs.existsSync(gatePath));

    const content = fs.readFileSync(gatePath, 'utf8');
    assert.ok(content.includes('id: validate-testids'));
    assert.ok(content.includes('files-changed: "**/*.{tsx,jsx}"'));
    assert.ok(content.includes('## LLM Judgment'));
    assert.ok(content.includes('## Workflow'));
    assert.ok(content.includes('severity: warning'));
    assert.ok(content.includes('action: report'));
  });

  test('test-plan.gate.md exists with correct structure', () => {
    const gatePath = path.join(projectRoot, 'gates', 'test-plan.gate.md');
    assert.ok(fs.existsSync(gatePath));

    const content = fs.readFileSync(gatePath, 'utf8');
    assert.ok(content.includes('id: test-plan'));
    assert.ok(content.includes('command-succeeds: "gh pr view"'));
    assert.ok(content.includes('files-changed: "**/{pages,components,templates}/**"'));
    assert.ok(content.includes('## LLM Judgment'));
    assert.ok(content.includes('## Workflow'));
    assert.ok(content.includes('severity: warning'));
    assert.ok(content.includes('action: report'));
  });

  test('gate loader discovers default gates from package path', () => {
    const gates = loadGates({
      packageRoot: projectRoot,
      globalDir: '/nonexistent',
      localDir: '/nonexistent',
    });

    const ids = gates.map(g => g.id);
    assert.ok(ids.includes('validate-testids'), 'should discover validate-testids gate');
    assert.ok(ids.includes('test-plan'), 'should discover test-plan gate');
  });

  test('package.json files array includes gates/', () => {
    const pkg = JSON.parse(fs.readFileSync(path.join(projectRoot, 'package.json'), 'utf8'));
    assert.ok(pkg.files.includes('gates/'), 'gates/ should be in package.json files array');
  });
});
