import { describe, it, before, after } from 'node:test';
import assert from 'node:assert/strict';
import fs from 'fs';
import path from 'path';
import os from 'os';
import { gatherSpecContext } from '../spec-context.js';

/**
 * Helper: create a minimal spec group directory with assertions.
 */
function createSpecGroup(specsDir, groupId, title, assertions) {
  const groupDir = path.join(specsDir, groupId);
  const assertionsDir = path.join(groupDir, 'assertions');
  fs.mkdirSync(assertionsDir, { recursive: true });

  // Write the parent spec file
  fs.writeFileSync(path.join(groupDir, `${groupId}.md`), [
    '---',
    `id: ${groupId}`,
    'created: 2026-01-20T17:00:00Z',
    'priority: 1',
    '---',
    '',
    `# ${title}`,
    '',
    'Description.',
  ].join('\n'));

  // Write assertion files
  for (const a of assertions) {
    fs.writeFileSync(path.join(assertionsDir, `${a.id}.md`), [
      '---',
      `id: ${a.id}`,
      `parent: ${groupId}`,
      'created: 2026-01-20T17:00:00Z',
      'priority: 1',
      `status: ${a.status}`,
      '---',
      '',
      `# ${a.title}`,
      '',
      '## Success Criteria',
      '',
      '- Something testable',
    ].join('\n'));
  }
}

describe('spec-context', () => {
  let tmpDir;

  before(() => {
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'spekk-ctx-'));
  });

  after(() => {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  });

  it('returns a context block with no-specs message for an empty directory', () => {
    const emptyDir = path.join(tmpDir, 'empty');
    fs.mkdirSync(emptyDir, { recursive: true });
    const ctx = gatherSpecContext(emptyDir);
    assert.ok(ctx.includes('[Spec Context]'), 'should open with [Spec Context]');
    assert.ok(ctx.includes('[/Spec Context]'), 'should close with [/Spec Context]');
    assert.ok(ctx.includes('No specs found'), 'should mention no specs found');
    assert.ok(ctx.includes('Git branch:'), 'should include git branch');
  });

  it('gathers spec groups with computed status and assertion counts', () => {
    const projDir = path.join(tmpDir, 'with-specs');
    const specsDir = path.join(projDir, 'specs');
    fs.mkdirSync(specsDir, { recursive: true });

    createSpecGroup(specsDir, 'auth-flow', 'Authentication Flow', [
      { id: 'login-form', status: 'done', title: 'Login Form' },
      { id: 'logout-button', status: 'not_started', title: 'Logout Button' },
      { id: 'session-expiry', status: 'in_progress', title: 'Session Expiry' },
    ]);

    createSpecGroup(specsDir, 'dashboard', 'Dashboard', [
      { id: 'chart-rendering', status: 'done', title: 'Chart Rendering' },
      { id: 'data-refresh', status: 'done', title: 'Data Refresh' },
    ]);

    const ctx = gatherSpecContext(projDir);

    assert.ok(ctx.includes('[Spec Context]'));
    assert.ok(ctx.includes('[/Spec Context]'));
    assert.ok(ctx.includes('Spec groups:'));

    // auth-flow: 1 done out of 3, has not_started + in_progress => in_progress
    assert.ok(ctx.includes('auth-flow (in_progress)'), 'auth-flow should be in_progress');
    assert.ok(ctx.includes('[1/3 done]'), 'auth-flow should show 1/3 done');

    // dashboard: 2 done out of 2 => done
    assert.ok(ctx.includes('dashboard (done)'), 'dashboard should be done');
    assert.ok(ctx.includes('[2/2 done]'), 'dashboard should show 2/2 done');
  });

  it('includes Git branch in the output', () => {
    const projDir = path.join(tmpDir, 'branch-test');
    fs.mkdirSync(projDir, { recursive: true });
    const ctx = gatherSpecContext(projDir);
    assert.ok(ctx.includes('Git branch:'), 'should include Git branch line');
  });

  it('handles a spec group with no assertions', () => {
    const projDir = path.join(tmpDir, 'no-assertions');
    const specsDir = path.join(projDir, 'specs');
    fs.mkdirSync(specsDir, { recursive: true });

    createSpecGroup(specsDir, 'empty-group', 'Empty Group', []);

    const ctx = gatherSpecContext(projDir);
    assert.ok(ctx.includes('empty-group (not_started)'), 'should show not_started for empty group');
    assert.ok(ctx.includes('[0/0 done]'), 'should show 0/0 done');
  });
});
