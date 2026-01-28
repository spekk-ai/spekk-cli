---
id: code-spec-misalignment-test-syntax
created: 2026-01-28T22:50:45Z
type: code_spec_misalignment
severity: medium
affected_specs:
  - coach-skills-system
affected_files:
  - src/coach/__tests__/skill-registry.test.js
---

# Test File Contains Syntax Errors Mixing Assertion Styles

## Issue Description
The skill-registry test file contains multiple syntax errors where Node.js assert module methods are incorrectly mixed with Jest-style assertion syntax. This will cause test failures when the test suite runs.

## Evidence
Multiple instances of incorrect assertion syntax in `src/coach/__tests__/skill-registry.test.js`:

**Line 137:** `assert(session).toBeInstanceOf(SkillSession);`
- Should be: `assert(session instanceof SkillSession);`

**Line 142:** `assert(() => registry.createSession('unknown-skill')).toThrow("Skill 'unknown-skill' not found");`
- Should be: `assert.throws(() => registry.createSession('unknown-skill'), /Skill 'unknown-skill' not found/);`

**Line 167:** `assert(session.getCurrentQuestion()).toBeNull();`
- Should be: `assert.strictEqual(session.getCurrentQuestion(), null);`

**Line 229:** `assert(result.data).toEqual({ q1: 'Answer 1', q2: 'Answer 2' });`
- Should be: `assert.deepStrictEqual(result.data, { q1: 'Answer 1', q2: 'Answer 2' });`

**Line 230:** `assert(session.completedAt).toBeTruthy();`
- Should be: `assert(session.completedAt);`

**Line 236:** `assert(() => session.process()).toThrow('Session is not complete');`
- Should be: `assert.throws(() => session.process(), /Session is not complete/);`

**Line 250:** `assert(metadata.completedAt).toBeNull();`
- Should be: `assert.strictEqual(metadata.completedAt, null);`

## Impact
- Tests will fail to run due to syntax errors
- CI/CD pipeline will fail when these tests are executed
- The skills system functionality cannot be properly validated
- Contradicts the "done" status of the coach-skills-system spec

## Recommendation
1. Update all assertions to use correct Node.js assert module syntax
2. Run the test file to ensure all tests pass
3. Consider adding a linting rule to prevent mixing assertion styles
4. Verify CI pipeline includes these tests in the test suite