# Test Issues to Fix

## Critical: Test Failures (Environment Coupling)

### branch-aware-next.test.js
**Problem:** Tests assume fixtures match current git branch, but tests run on feature branches.

**Issue:**
```javascript
test('returns assertion matching current git branch', () => {
  const currentBranch = execSync('git branch --show-current', { encoding: 'utf8' }).trim();
  // Test expects fixture with assertions for currentBranch
  // But fixtures only have feature/chat-system and main
});
```

**Fix:** Mock `getCurrentGitBranch()` to return controlled values:
```javascript
import { test } from 'node:test';
import * as parserModule from '../index.js';

test('returns assertion matching current git branch', (t) => {
  t.mock.method(parserModule, 'getCurrentGitBranch', () => 'feature/chat-system');
  // Now test with known fixture branch
});
```

### parser-basic.test.js (Next Priority Identification)
**Problem:** Likely broken by branch-aware filtering changes.

**Fix:** Review and update to account for branch-aware default behavior.

---

## Non-Critical: Test Bloat (Lean Testing Violations)

Per builder prompt: "Tests should be lean - one test per meaningful behavior, no redundant coverage."

### 1. Trivial Getter Tests (Low Value)

**File:** `src/coach/__tests__/coordinator.test.js`

**Remove these (lines 38-68):**
```javascript
it('should have correct ID', () => {
  assert.strictEqual(coordinator.getId(), 'coordinator');
});

it('should have correct name', () => {
  assert.strictEqual(coordinator.getName(), 'Coordinator');
});

it('should have description', () => {
  const desc = coordinator.getDescription();
  assert.ok(desc);
  assert.ok(desc.toLowerCase().includes('dependencies'));
});

it('should have questions', () => {
  const questions = coordinator.getQuestions();
  assert.ok(Array.isArray(questions));
  assert.ok(questions.length > 0);
});
```

**Why:** These test trivial getters that return constants. No meaningful behavior.

**Keep:** The trigger tests (actually test regex logic).

### 2. Redundant Integration Tests

**File:** `src/parser/__tests__/depends-on-validation.test.js`

**Problem:** Lines 45-80 test "accepts omitted depends-on field" with full file I/O and symlinks.

**Fix:** This is already covered by unit test on line 37. Remove the integration version:
```javascript
// REMOVE (redundant):
test('accepts omitted depends-on field', () => {
  const tempDir = path.join(process.cwd(), 'temp-depends-omitted-test');
  // ... 35 lines of file I/O setup
});

// KEEP (sufficient):
test('parses depends-on field and converts to camelCase', () => {
  const { data } = parseFrontmatter(yaml);
  assert.equal(data.dependsOn, 'other-assertion');
});
```

### 3. Over-Detailed Error Message Tests

**File:** `src/parser/__tests__/depends-on-validation.test.js` (lines 150-250)

**Tests that check exact error message wording:**
```javascript
test('provides clear error message for non-existent reference', () => {
  // ...
  assert.ok(error.message.includes("references non-existent assertion 'nonexistent'"));
  assert.ok(error.message.includes('test-depends-invalid.md'));
});
```

**Why redundant:** We already test that the error is thrown. Exact wording is implementation detail.

**Fix:** Consolidate to one error message quality test instead of 5.

### 4. Excessive Edge Case Coverage

**File:** `src/parser/__tests__/branch-validation.test.js` (213 lines)

**14 tests for branch validation, including:**
- 4 tests for valid cases (main, feature/, bugfix/, hotfix/)
- 5 tests for invalid cases (spaces, special chars, etc.)
- 3 tests for warnings
- 2 tests for defaults

**Fix:** Reduce to 5 tests:
- 1 test: valid branches (multiple examples in one test)
- 1 test: invalid characters rejected
- 1 test: warnings for non-standard (doesn't fail)
- 1 test: defaults to main when omitted
- 1 test: integration (parseAllSpecs handles branch field)

### 5. Redundant Setup/Teardown

**File:** `src/coach/__tests__/coordinator.test.js`

**Problem:** Every test creates full git repo with execSync:
```javascript
beforeEach(() => {
  testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'coordinator-test-'));
  execSync('git init', { cwd: testDir, stdio: 'pipe' });
  execSync('git config user.email "test@example.com"', { cwd: testDir });
  execSync('git config user.name "Test User"', { cwd: testDir });
  // ... more git commands
});
```

**Why redundant:** Most tests don't need actual git. Only tests that call git commands need this.

**Fix:** 
- Remove git setup from global beforeEach
- Add git setup only to tests that actually use git
- Use mocks for tests that just need branch name

---

## Summary

**Critical fixes (must do):**
1. Mock `getCurrentGitBranch()` in branch-aware tests
2. Fix parser-basic.test.js for branch-aware behavior

**Lean testing fixes (should do):**
1. Remove trivial getter tests (save ~30 lines)
2. Remove redundant integration tests (save ~100 lines)
3. Consolidate error message tests (save ~50 lines)
4. Reduce branch validation tests (save ~100 lines)
5. Remove unnecessary git setup (faster tests)

**Expected outcome:**
- All tests pass (decoupled from environment)
- ~280 lines of low-value tests removed
- Test suite runs faster
- Maintains meaningful behavior coverage

**Estimated effort:** 2-3 hours to fix properly
