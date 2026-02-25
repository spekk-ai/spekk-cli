import { test, describe } from 'node:test';
import assert from 'node:assert';
import fs from 'node:fs';
import path from 'node:path';
import os from 'node:os';
import { execSync } from 'node:child_process';
import { MeetingNotesToSpecs } from '../meeting-notes-to-specs.js';

describe('Outputs Committed Together', () => {
  const skill = new MeetingNotesToSpecs();

  describe('buildCommitMessage', () => {
    test('should format commit message with all three categories', () => {
      const message = skill.buildCommitMessage({
        meetingDate: '2025-02-12',
        summary: 'Job scraping alternatives and match score improvements',
        todos: ['Follow up with Kaiser team', 'Schedule design review'],
        specIds: ['job-scraping', 'match-score'],
        decisions: ['Decided to use deep-link searches over scraping']
      });

      // First line is the subject
      const lines = message.split('\n');
      assert.strictEqual(
        lines[0],
        'Process meeting: 2025-02-12 - Job scraping alternatives and match score improvements'
      );

      // Body has categorized sections
      assert.ok(message.includes('Todos:'));
      assert.ok(message.includes('- Follow up with Kaiser team'));
      assert.ok(message.includes('- Schedule design review'));
      assert.ok(message.includes('Specs created:'));
      assert.ok(message.includes('- job-scraping'));
      assert.ok(message.includes('- match-score'));
      assert.ok(message.includes('Context updates:'));
      assert.ok(message.includes('- Decided to use deep-link searches over scraping'));
    });

    test('should omit empty categories from body', () => {
      const message = skill.buildCommitMessage({
        meetingDate: '2025-03-01',
        summary: 'Quick sync on todos',
        todos: ['Send report'],
        specIds: [],
        decisions: []
      });

      assert.ok(message.includes('Todos:'));
      assert.ok(message.includes('- Send report'));
      assert.ok(!message.includes('Specs created:'));
      assert.ok(!message.includes('Context updates:'));
    });

    test('should handle only specs', () => {
      const message = skill.buildCommitMessage({
        meetingDate: '2025-04-10',
        summary: 'Feature planning',
        todos: [],
        specIds: ['new-feature'],
        decisions: []
      });

      assert.ok(!message.includes('Todos:'));
      assert.ok(message.includes('Specs created:'));
      assert.ok(message.includes('- new-feature'));
      assert.ok(!message.includes('Context updates:'));
    });

    test('should handle only decisions', () => {
      const message = skill.buildCommitMessage({
        meetingDate: '2025-05-20',
        summary: 'Architecture decisions',
        todos: [],
        specIds: [],
        decisions: ['Use PostgreSQL for persistence']
      });

      assert.ok(!message.includes('Todos:'));
      assert.ok(!message.includes('Specs created:'));
      assert.ok(message.includes('Context updates:'));
      assert.ok(message.includes('- Use PostgreSQL for persistence'));
    });

    test('should require meetingDate and summary', () => {
      assert.throws(
        () => skill.buildCommitMessage({ summary: 'test', todos: [], specIds: [], decisions: [] }),
        /meetingDate is required/
      );
      assert.throws(
        () => skill.buildCommitMessage({ meetingDate: '2025-01-01', todos: [], specIds: [], decisions: [] }),
        /summary is required/
      );
    });
  });

  describe('commitAllOutputs (git integration)', () => {
    let tmpDir;

    function setupGitRepo() {
      tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'spekk-commit-test-'));
      execSync('git init', { cwd: tmpDir, stdio: 'pipe' });
      execSync('git config user.email "test@test.com"', { cwd: tmpDir, stdio: 'pipe' });
      execSync('git config user.name "Test"', { cwd: tmpDir, stdio: 'pipe' });
      // Initial commit so we have a HEAD
      fs.writeFileSync(path.join(tmpDir, 'README.md'), '# Test\n');
      execSync('git add . && git commit -m "Initial commit"', { cwd: tmpDir, stdio: 'pipe' });
      return tmpDir;
    }

    function teardown() {
      if (tmpDir) {
        fs.rmSync(tmpDir, { recursive: true, force: true });
        tmpDir = null;
      }
    }

    test('should commit all outputs in a single commit', () => {
      const dir = setupGitRepo();
      try {
        // Create output files
        fs.writeFileSync(path.join(dir, 'TODOS.md'), '- [ ] Follow up with team\n');
        fs.mkdirSync(path.join(dir, 'specs/new-feature/assertions'), { recursive: true });
        fs.writeFileSync(path.join(dir, 'specs/new-feature/new-feature.md'), '---\nid: new-feature\n---\n');
        fs.writeFileSync(path.join(dir, 'CONTEXT.md'), '# Project Context\n');

        const result = skill.commitAllOutputs({
          meetingDate: '2025-02-12',
          summary: 'Sprint planning',
          todos: ['Follow up with team'],
          specIds: ['new-feature'],
          decisions: ['Use React for frontend'],
          filePaths: [
            'TODOS.md',
            'specs/new-feature/new-feature.md',
            'CONTEXT.md'
          ],
          baseDir: dir
        });

        assert.ok(result.success);

        // Verify single commit was made (2 total: initial + ours)
        const log = execSync('git log --oneline', { cwd: dir, encoding: 'utf8' });
        const commits = log.trim().split('\n');
        assert.strictEqual(commits.length, 2);

        // Verify commit message format
        const fullMessage = execSync('git log -1 --format=%B', { cwd: dir, encoding: 'utf8' });
        assert.ok(fullMessage.startsWith('Process meeting: 2025-02-12 - Sprint planning'));
        assert.ok(fullMessage.includes('Todos:'));
        assert.ok(fullMessage.includes('Specs created:'));
        assert.ok(fullMessage.includes('Context updates:'));

        // Verify all files in the commit
        const filesInCommit = execSync('git diff-tree --no-commit-id --name-only -r HEAD', {
          cwd: dir, encoding: 'utf8'
        });
        assert.ok(filesInCommit.includes('TODOS.md'));
        assert.ok(filesInCommit.includes('specs/new-feature/new-feature.md'));
        assert.ok(filesInCommit.includes('CONTEXT.md'));
      } finally {
        teardown();
      }
    });

    test('should commit only provided file paths', () => {
      const dir = setupGitRepo();
      try {
        // Create files but only commit some
        fs.writeFileSync(path.join(dir, 'TODOS.md'), '- [ ] Do thing\n');
        fs.writeFileSync(path.join(dir, 'unrelated.txt'), 'should not be committed\n');

        skill.commitAllOutputs({
          meetingDate: '2025-06-01',
          summary: 'Todos only',
          todos: ['Do thing'],
          specIds: [],
          decisions: [],
          filePaths: ['TODOS.md'],
          baseDir: dir
        });

        const filesInCommit = execSync('git diff-tree --no-commit-id --name-only -r HEAD', {
          cwd: dir, encoding: 'utf8'
        });
        assert.ok(filesInCommit.includes('TODOS.md'));
        assert.ok(!filesInCommit.includes('unrelated.txt'));
      } finally {
        teardown();
      }
    });

    test('should throw if no filePaths provided', () => {
      const dir = setupGitRepo();
      try {
        assert.throws(
          () => skill.commitAllOutputs({
            meetingDate: '2025-01-01',
            summary: 'Empty',
            todos: [],
            specIds: [],
            decisions: [],
            filePaths: [],
            baseDir: dir
          }),
          /filePaths must be a non-empty array/
        );
      } finally {
        teardown();
      }
    });

    test('should return commit hash on success', () => {
      const dir = setupGitRepo();
      try {
        fs.writeFileSync(path.join(dir, 'CONTEXT.md'), '# Context\n');

        const result = skill.commitAllOutputs({
          meetingDate: '2025-07-01',
          summary: 'Context update',
          todos: [],
          specIds: [],
          decisions: ['Use microservices'],
          filePaths: ['CONTEXT.md'],
          baseDir: dir
        });

        assert.ok(result.success);
        assert.ok(result.commitHash);
        assert.ok(result.commitHash.length >= 7);
      } finally {
        teardown();
      }
    });
  });
});
