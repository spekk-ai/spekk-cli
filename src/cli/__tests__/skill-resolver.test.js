import { test, describe, beforeEach, afterEach } from 'node:test';
import assert from 'node:assert';
import path from 'node:path';
import { mkdtempSync, mkdirSync, writeFileSync, rmSync, existsSync } from 'fs';
import { tmpdir } from 'os';
import { fileURLToPath } from 'url';
import { SkillResolver } from '../skill-resolver.js';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const projectRoot = path.join(__dirname, '../../..');

describe('SkillResolver', () => {
  let tempHome;
  let tempCwd;

  beforeEach(() => {
    tempHome = mkdtempSync(path.join(tmpdir(), 'spekk-skill-home-'));
    tempCwd = mkdtempSync(path.join(tmpdir(), 'spekk-skill-cwd-'));
  });

  afterEach(() => {
    rmSync(tempHome, { recursive: true, force: true });
    rmSync(tempCwd, { recursive: true, force: true });
  });

  describe('resolveSkill', () => {
    test('returns null for unknown skill with no directories', () => {
      const resolver = new SkillResolver({ homeDir: tempHome, cwd: tempCwd });
      const result = resolver.resolveSkill('coach', 'nonexistent');
      assert.strictEqual(result, null);
    });

    test('resolves package-shipped coach skill by filename stem', () => {
      const resolver = new SkillResolver({ homeDir: tempHome, cwd: tempCwd });
      const result = resolver.resolveSkill('coach', 'meeting-notes-to-specs-skill');
      assert.ok(result, 'Should find the package skill');
      assert.strictEqual(result.name, 'meeting-notes-to-specs-skill');
      assert.ok(result.content.includes('## Workflow'), 'Should include skill content');
      assert.ok(result.source.includes('specs/coach-skills-system'));
    });

    test('resolves package-shipped coach skill via legacy alias', () => {
      const resolver = new SkillResolver({ homeDir: tempHome, cwd: tempCwd });
      const result = resolver.resolveSkill('coach', 'meeting');
      assert.ok(result, 'Should resolve "meeting" via legacy alias');
      assert.strictEqual(result.name, 'meeting-notes-to-specs-skill');
      assert.ok(result.content.includes('## Workflow'));
    });

    test('resolves coordinator skill via legacy alias', () => {
      const resolver = new SkillResolver({ homeDir: tempHome, cwd: tempCwd });
      const result = resolver.resolveSkill('coach', 'coordinate');
      assert.ok(result, 'Should resolve "coordinate" via legacy alias');
      assert.strictEqual(result.name, 'coordinator-skill');
      assert.ok(result.content.includes('Coordinator'));
    });

    test('resolves skill from global directory', () => {
      const globalSkillDir = path.join(tempHome, '.spekk', 'skills', 'builder');
      mkdirSync(globalSkillDir, { recursive: true });
      writeFileSync(path.join(globalSkillDir, 'my-tool.md'), '# My Tool\nDoes things.');

      const resolver = new SkillResolver({ homeDir: tempHome, cwd: tempCwd });
      const result = resolver.resolveSkill('builder', 'my-tool');
      assert.ok(result, 'Should find global skill');
      assert.strictEqual(result.name, 'my-tool');
      assert.ok(result.content.includes('# My Tool'));
      assert.strictEqual(result.source, globalSkillDir);
    });

    test('resolves skill from local directory', () => {
      const localSkillDir = path.join(tempCwd, '.spekk', 'skills', 'coach');
      mkdirSync(localSkillDir, { recursive: true });
      writeFileSync(path.join(localSkillDir, 'custom.md'), '# Custom Skill');

      const resolver = new SkillResolver({ homeDir: tempHome, cwd: tempCwd });
      const result = resolver.resolveSkill('coach', 'custom');
      assert.ok(result, 'Should find local skill');
      assert.strictEqual(result.name, 'custom');
      assert.ok(result.content.includes('# Custom Skill'));
      assert.strictEqual(result.source, localSkillDir);
    });

    test('local skill overrides global skill with same name', () => {
      const globalSkillDir = path.join(tempHome, '.spekk', 'skills', 'builder');
      mkdirSync(globalSkillDir, { recursive: true });
      writeFileSync(path.join(globalSkillDir, 'my-tool.md'), '# Global version');

      const localSkillDir = path.join(tempCwd, '.spekk', 'skills', 'builder');
      mkdirSync(localSkillDir, { recursive: true });
      writeFileSync(path.join(localSkillDir, 'my-tool.md'), '# Local version');

      const resolver = new SkillResolver({ homeDir: tempHome, cwd: tempCwd });
      const result = resolver.resolveSkill('builder', 'my-tool');
      assert.ok(result);
      assert.ok(result.content.includes('# Local version'), 'Local should win');
      assert.strictEqual(result.source, localSkillDir);
    });

    test('local skill overrides package skill with same name', () => {
      // Create a local skill that matches a package coach skill filename
      const localSkillDir = path.join(tempCwd, '.spekk', 'skills', 'coach');
      mkdirSync(localSkillDir, { recursive: true });
      writeFileSync(path.join(localSkillDir, 'meeting-notes-to-specs-skill.md'), '# Custom meeting skill');

      const resolver = new SkillResolver({ homeDir: tempHome, cwd: tempCwd });
      const result = resolver.resolveSkill('coach', 'meeting');
      assert.ok(result);
      assert.ok(result.content.includes('# Custom meeting skill'), 'Local should override package');
    });

    test('resolves skill by frontmatter id', () => {
      const globalSkillDir = path.join(tempHome, '.spekk', 'skills', 'builder');
      mkdirSync(globalSkillDir, { recursive: true });
      writeFileSync(path.join(globalSkillDir, 'api-audit-tool.md'),
        '---\nid: api-audit\n---\n\n# API Audit Tool');

      const resolver = new SkillResolver({ homeDir: tempHome, cwd: tempCwd });
      const result = resolver.resolveSkill('builder', 'api-audit');
      assert.ok(result, 'Should resolve via frontmatter id');
      assert.strictEqual(result.name, 'api-audit-tool');
      assert.ok(result.content.includes('# API Audit Tool'));
    });

    test('returns null for null subcommand', () => {
      const resolver = new SkillResolver({ homeDir: tempHome, cwd: tempCwd });
      assert.strictEqual(resolver.resolveSkill('coach', null), null);
    });

    test('returns null for empty string subcommand', () => {
      const resolver = new SkillResolver({ homeDir: tempHome, cwd: tempCwd });
      assert.strictEqual(resolver.resolveSkill('coach', ''), null);
    });
  });

  describe('listSkills', () => {
    test('lists package-shipped coach skills', () => {
      const resolver = new SkillResolver({ homeDir: tempHome, cwd: tempCwd });
      const skills = resolver.listSkills('coach');
      assert.ok(skills.length > 0, 'Should find package coach skills');
      const names = skills.map(s => s.name);
      assert.ok(names.includes('meeting-notes-to-specs-skill'), 'Should include meeting skill');
      assert.ok(names.includes('coordinator-skill'), 'Should include coordinator skill');
    });

    test('lists global skills', () => {
      const globalSkillDir = path.join(tempHome, '.spekk', 'skills', 'builder');
      mkdirSync(globalSkillDir, { recursive: true });
      writeFileSync(path.join(globalSkillDir, 'tool-a.md'), '# Tool A');
      writeFileSync(path.join(globalSkillDir, 'tool-b.md'), '# Tool B');

      const resolver = new SkillResolver({ homeDir: tempHome, cwd: tempCwd });
      const skills = resolver.listSkills('builder');
      const names = skills.map(s => s.name);
      assert.ok(names.includes('tool-a'));
      assert.ok(names.includes('tool-b'));
    });

    test('deduplicates skills across layers (local wins)', () => {
      const globalSkillDir = path.join(tempHome, '.spekk', 'skills', 'builder');
      mkdirSync(globalSkillDir, { recursive: true });
      writeFileSync(path.join(globalSkillDir, 'shared.md'), '# Global');
      writeFileSync(path.join(globalSkillDir, 'global-only.md'), '# Global Only');

      const localSkillDir = path.join(tempCwd, '.spekk', 'skills', 'builder');
      mkdirSync(localSkillDir, { recursive: true });
      writeFileSync(path.join(localSkillDir, 'shared.md'), '# Local');
      writeFileSync(path.join(localSkillDir, 'local-only.md'), '# Local Only');

      const resolver = new SkillResolver({ homeDir: tempHome, cwd: tempCwd });
      const skills = resolver.listSkills('builder');
      const names = skills.map(s => s.name);

      // Should contain all unique names
      assert.ok(names.includes('shared'));
      assert.ok(names.includes('global-only'));
      assert.ok(names.includes('local-only'));

      // Should not have duplicates
      const sharedEntries = skills.filter(s => s.name === 'shared');
      assert.strictEqual(sharedEntries.length, 1, 'Should deduplicate "shared"');
      assert.strictEqual(sharedEntries[0].source, localSkillDir, 'Local should win');
    });

    test('returns empty array for builder when no skills exist', () => {
      const resolver = new SkillResolver({ homeDir: tempHome, cwd: tempCwd });
      const skills = resolver.listSkills('builder');
      // builder-skills dir is empty (only .gitkeep), so no .md files
      assert.strictEqual(skills.length, 0);
    });

    test('ignores non-.md files', () => {
      const globalSkillDir = path.join(tempHome, '.spekk', 'skills', 'builder');
      mkdirSync(globalSkillDir, { recursive: true });
      writeFileSync(path.join(globalSkillDir, 'skill.md'), '# Skill');
      writeFileSync(path.join(globalSkillDir, 'notes.txt'), 'not a skill');
      writeFileSync(path.join(globalSkillDir, '.DS_Store'), '');

      const resolver = new SkillResolver({ homeDir: tempHome, cwd: tempCwd });
      const skills = resolver.listSkills('builder');
      assert.strictEqual(skills.length, 1);
      assert.strictEqual(skills[0].name, 'skill');
    });
  });

  describe('constructor defaults', () => {
    test('defaults homeDir to os.homedir()', () => {
      const resolver = new SkillResolver();
      assert.ok(resolver.homeDir);
      assert.ok(!resolver.homeDir.includes('~'));
    });

    test('defaults cwd to process.cwd()', () => {
      const resolver = new SkillResolver();
      assert.strictEqual(resolver.cwd, process.cwd());
    });

    test('accepts custom homeDir and cwd', () => {
      const resolver = new SkillResolver({ homeDir: '/tmp/h', cwd: '/tmp/c' });
      assert.strictEqual(resolver.homeDir, '/tmp/h');
      assert.strictEqual(resolver.cwd, '/tmp/c');
    });
  });
});
