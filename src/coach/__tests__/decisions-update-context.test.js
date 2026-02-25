import { test, describe } from 'node:test';
import assert from 'node:assert';
import fs from 'node:fs';
import path from 'node:path';
import os from 'node:os';
import { MeetingNotesToSpecs } from '../meeting-notes-to-specs.js';

describe('Decisions Update CONTEXT.md', () => {
  const skill = new MeetingNotesToSpecs();

  test('formats decisions with date stamp and context', () => {
    const decisions = [
      { decision: 'Use deep-link searches instead of scraping', context: 'Discussed alternatives' },
      { decision: 'Partner with platforms rather than replacing them' }
    ];

    const formatted = skill.formatDecisions(decisions, '2025-02-12');

    assert.ok(formatted.includes('Decision from meeting 2025-02-12: Use deep-link searches instead of scraping'));
    assert.ok(formatted.includes('Discussed alternatives'));
    assert.ok(formatted.includes('Decision from meeting 2025-02-12: Partner with platforms rather than replacing them'));
  });

  test('returns empty string for no decisions', () => {
    assert.strictEqual(skill.formatDecisions([], '2025-02-12'), '');
    assert.strictEqual(skill.generateContextUpdate([], '2025-02-12', null), '');
  });

  test('creates new CONTEXT.md when none exists', () => {
    const decisions = [{ decision: 'Use deep-link searches', context: 'Technical discussion' }];
    const result = skill.generateContextUpdate(decisions, '2025-02-12', null);

    assert.ok(result.includes('# Project Context'));
    assert.ok(result.includes('## Architectural Decisions'));
    assert.ok(result.includes('Decision from meeting 2025-02-12: Use deep-link searches'));
  });

  test('appends to existing CONTEXT.md preserving structure', () => {
    const existing = '# Project Context\n\nIntro text.\n\n## Architectural Decisions\n\n- Decision from meeting 2025-02-10: Use React\n  - *Context: Framework choice*\n\n## Other Section\n\nOther content.\n';
    const decisions = [{ decision: 'Add caching layer', context: 'Performance discussion' }];

    const result = skill.generateContextUpdate(decisions, '2025-02-12', existing);

    assert.ok(result.includes('Decision from meeting 2025-02-10: Use React'));
    assert.ok(result.includes('Decision from meeting 2025-02-12: Add caching layer'));
    assert.ok(result.includes('## Other Section'));
    assert.ok(result.includes('Other content.'));
  });

  test('generates diff showing additions', () => {
    const diff = skill.generateContextDiff(null, '# Project Context\n\n## Architectural Decisions\n');
    assert.ok(diff.includes('(new file)'));
    assert.ok(diff.includes('# Project Context'));

    const oldContent = '# Project Context\n';
    const newContent = '# Project Context\n\n- New decision\n';
    const appendDiff = skill.generateContextDiff(oldContent, newContent);
    assert.ok(appendDiff.includes('New decision'));
  });

  test('end-to-end: read, update, write CONTEXT.md', () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'spekk-ctx-'));
    try {
      // No file yet
      assert.strictEqual(skill.readContextFile(tmpDir), null);

      // Create from scratch
      const decisions = [
        { decision: 'Use deep-link searches', context: 'Technical discussion' },
        { decision: 'Keep todos separate from specs', context: 'Workflow design' }
      ];
      const content = skill.generateContextUpdate(decisions, '2025-02-12', null);
      skill.writeContextFile(content, tmpDir);

      // Verify written
      const written = skill.readContextFile(tmpDir);
      assert.ok(written.includes('Decision from meeting 2025-02-12: Use deep-link searches'));
      assert.ok(written.includes('Decision from meeting 2025-02-12: Keep todos separate from specs'));

      // Append more
      const newDecisions = [{ decision: 'Add Redis caching', context: 'Perf review' }];
      const updated = skill.generateContextUpdate(newDecisions, '2025-02-14', written);
      skill.writeContextFile(updated, tmpDir);

      const final = skill.readContextFile(tmpDir);
      assert.ok(final.includes('Use deep-link searches'));
      assert.ok(final.includes('Add Redis caching'));
    } finally {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });
});
