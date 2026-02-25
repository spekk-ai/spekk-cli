import { test, describe } from 'node:test';
import assert from 'node:assert';
import fs from 'node:fs';
import path from 'node:path';
import os from 'node:os';
import { MeetingNotesToSpecs } from '../meeting-notes-to-specs.js';

describe('Todos Formatted for Action Tracking', () => {
  const skill = new MeetingNotesToSpecs();

  test('formats todo with description, owner, and meeting date', () => {
    const todo = { description: 'Send spreadsheet to team', owner: 'marcy' };
    const result = skill.formatTodo(todo, '2025-02-12');
    assert.strictEqual(result, '- [ ] Send spreadsheet to team (@marcy) - from meeting 2025-02-12');
  });

  test('formats todo without owner', () => {
    const todo = { description: 'Follow up with Kaiser' };
    const result = skill.formatTodo(todo, '2025-02-12');
    assert.strictEqual(result, '- [ ] Follow up with Kaiser - from meeting 2025-02-12');
  });

  test('throws if todo has no description', () => {
    assert.throws(() => skill.formatTodo({}, '2025-02-12'), /description/);
    assert.throws(() => skill.formatTodo(null, '2025-02-12'), /description/);
  });

  test('formats multiple todos', () => {
    const todos = [
      { description: 'Send spreadsheet to team', owner: 'marcy' },
      { description: 'Follow up with Kaiser' },
      { description: 'Review PR #42', owner: 'william', context: 'Urgent fix' }
    ];

    const result = skill.formatTodos(todos, '2025-02-12');

    assert.ok(result.includes('- [ ] Send spreadsheet to team (@marcy) - from meeting 2025-02-12'));
    assert.ok(result.includes('- [ ] Follow up with Kaiser - from meeting 2025-02-12'));
    assert.ok(result.includes('- [ ] Review PR #42 (@william) - from meeting 2025-02-12'));
  });

  test('returns empty string for no todos', () => {
    assert.strictEqual(skill.formatTodos([], '2025-02-12'), '');
    assert.strictEqual(skill.formatTodos(null, '2025-02-12'), '');
    assert.strictEqual(skill.generateTodosUpdate([], '2025-02-12', null), '');
  });

  test('creates new TODOS.md when none exists', () => {
    const todos = [
      { description: 'Send spreadsheet to team', owner: 'marcy' },
      { description: 'Follow up with Kaiser' }
    ];
    const result = skill.generateTodosUpdate(todos, '2025-02-12', null);

    assert.ok(result.includes('# Todos'));
    assert.ok(result.includes('- [ ] Send spreadsheet to team (@marcy) - from meeting 2025-02-12'));
    assert.ok(result.includes('- [ ] Follow up with Kaiser - from meeting 2025-02-12'));
  });

  test('appends to existing TODOS.md', () => {
    const existing = '# Todos\n\n- [ ] Old task (@alice) - from meeting 2025-02-10\n';
    const todos = [{ description: 'New task', owner: 'bob' }];

    const result = skill.generateTodosUpdate(todos, '2025-02-12', existing);

    assert.ok(result.includes('- [ ] Old task (@alice) - from meeting 2025-02-10'));
    assert.ok(result.includes('- [ ] New task (@bob) - from meeting 2025-02-12'));
  });

  test('generates diff showing additions for new file', () => {
    const diff = skill.generateTodosDiff(null, '# Todos\n\n- [ ] Task one\n');
    assert.ok(diff.includes('(new file)'));
    assert.ok(diff.includes('# Todos'));
  });

  test('generates diff showing additions for update', () => {
    const oldContent = '# Todos\n\n- [ ] Old task\n';
    const newContent = '# Todos\n\n- [ ] Old task\n- [ ] New task\n';
    const diff = skill.generateTodosDiff(oldContent, newContent);
    assert.ok(diff.includes('(updated)'));
    assert.ok(diff.includes('New task'));
    assert.ok(!diff.includes('+ # Todos'));
  });

  test('end-to-end: read, update, write TODOS.md', () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'spekk-todos-'));
    try {
      // No file yet
      assert.strictEqual(skill.readTodosFile(tmpDir), null);

      // Create from scratch
      const todos = [
        { description: 'Send spreadsheet to team', owner: 'marcy' },
        { description: 'Follow up with Kaiser' }
      ];
      const content = skill.generateTodosUpdate(todos, '2025-02-12', null);
      skill.writeTodosFile(content, tmpDir);

      // Verify written
      const written = skill.readTodosFile(tmpDir);
      assert.ok(written.includes('- [ ] Send spreadsheet to team (@marcy) - from meeting 2025-02-12'));
      assert.ok(written.includes('- [ ] Follow up with Kaiser - from meeting 2025-02-12'));

      // Append more
      const newTodos = [{ description: 'Review PR #42', owner: 'william' }];
      const updated = skill.generateTodosUpdate(newTodos, '2025-02-14', written);
      skill.writeTodosFile(updated, tmpDir);

      const final = skill.readTodosFile(tmpDir);
      assert.ok(final.includes('Send spreadsheet to team'));
      assert.ok(final.includes('Review PR #42'));
      assert.ok(final.includes('from meeting 2025-02-12'));
      assert.ok(final.includes('from meeting 2025-02-14'));
    } finally {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  test('todos are action items, not product features', () => {
    // Todos like "follow up with Kaiser" should stay as todos
    // They should NOT become specs
    const categories = skill.extractCategories({
      todos: [
        { description: 'Follow up with Kaiser' },
        { description: 'Send pricing doc to client' }
      ],
      features: [
        { id: 'dark-mode', title: 'Dark Mode', description: 'Add dark mode toggle' }
      ],
      decisions: []
    });

    // Todos stay in todos, not features
    assert.strictEqual(categories.todos.length, 2);
    assert.strictEqual(categories.features.length, 1);
    assert.ok(categories.todos.some(t => t.description === 'Follow up with Kaiser'));
    assert.ok(!categories.features.some(f => f.title === 'Follow up with Kaiser'));
  });
});
