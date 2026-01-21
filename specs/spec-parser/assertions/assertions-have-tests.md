---
id: assertions-have-tests
parent: spec-parser
created: 2026-01-20T16:25:00Z
priority: 2
status: not_started
---

# Assertions Must Have Automated Tests When Possible

## What Must Be True

When an assertion describes behavior that can be automatically validated, it must have associated test files that prove the assertion is true.

## Two Types of Tests

### Type 1: Implementation Tests (JavaScript)
For testing **code behavior** - located in `app/**/__tests__/`

**Use for:**
- Code behavior (functions, scripts, APIs)
- Data validation (parsers, validators)
- Algorithm correctness
- Output format validation
- Any behavior that requires running application code

**Example:** Testing that parser extracts YAML frontmatter correctly

### Type 2: Sidecar Validation Tests (Bash)
For testing **spec system integrity** - located alongside assertions as `*.test.sh`

**Use for:**
- File/directory existence
- Spec structure validation
- Prompt completeness checks
- Any validation that doesn't require running app code

**Example:** Testing that builder-agent prompt file exists

## Testability Categories

### Always Testable (Automated Tests)
- Code behavior → Implementation tests (JS)
- File existence → Sidecar tests (bash)
- Spec structure → Sidecar tests (bash)

### Sometimes Testable (May require tooling)
- UI behavior (with tools like Playwright, Cypress)
- Visual design (screenshot testing)
- Performance requirements (benchmarks)

### Not Automatically Testable (Manual validation)
- Subjective quality ("looks good", "feels responsive")
- Business judgment calls
- Manual processes

## Implementation Test Structure

**Location:** `app/**/__tests__/`

```
app/
├── parser/
│   ├── index.js
│   └── __tests__/
│       ├── parser.test.js
│       └── ...
├── dashboard/
│   └── __tests__/
│       └── dashboard.test.js
└── ...
```

**Framework:** Node.js built-in test runner, Jest, or Vitest

## Sidecar Test Structure

**Location:** Co-located with assertions as `*.test.sh`

```
specs/
├── builder-agent/
│   └── assertions/
│       ├── prompt-exists.md
│       └── prompt-exists.test.sh      # Sidecar test
├── spec-parser/
│   └── assertions/
│       ├── app-directory-structure.md
│       └── app-directory-structure.test.sh
└── ...
```

**Format:** Bash script that exits 0 on success, non-zero on failure

**Example:**
```bash
#!/usr/bin/env bash
# specs/builder-agent/assertions/prompt-exists.test.sh

PROMPT_FILE="specs/builder-agent/builder-agent.prompt.md"

[ -f "$PROMPT_FILE" ] || {
  echo "❌ Builder prompt missing at $PROMPT_FILE"
  exit 1
}

exit 0  # Silent on success
```

## Test Requirements

**When an assertion is testable:**

1. **Test file must exist:**
   - Implementation tests: `src/**/__tests__/*.test.js`
   - Sidecar tests: `specs/**/assertions/*.test.sh`

2. **Test file must be linked** in assertion markdown:
   ```markdown
   **Tests:** `src/parser/__tests__/parser.test.js`
   **Tests:** `specs/builder-agent/assertions/prompt-exists.test.sh`
   ```

3. **Tests must validate success criteria** from assertion
4. **Tests must pass** for assertion to be marked `done`
5. **Tests run on every change** to catch regressions

## Example: Linking Tests to Assertion

In assertion markdown file:

```markdown
## Success Criteria

This assertion is "done" when:
- ✅ Parser extracts YAML frontmatter
- ✅ Parser handles malformed YAML gracefully
- ✅ Parser returns correct data structure

**Tests:** `src/parser/__tests__/parser.test.js`
```

In test file:

```javascript
// src/parser/__tests__/parser.test.js
import { test } from 'node:test';
import { parseSpec } from '../index.js';

test('parses valid YAML frontmatter', () => {
  // Test implementation
});

test('handles malformed YAML', () => {
  // Test implementation
});
```

## Benefits

1. **Automated validation**: No manual checking required
2. **Regression detection**: Tests catch when changes break old assertions
3. **Living documentation**: Tests show exactly what the assertion means
4. **Confidence**: Can mark `done` with certainty
5. **Continuous validation**: Tests run in CI/CD

## Validation Protocol

Before marking an assertion `done`:
1. Check if assertion is testable
2. If yes, ensure test file exists and is linked
3. Run tests: `npm test`
4. All tests must pass
5. Only then update status to `done`

## Success Criteria

This assertion is "done" when:
- ✅ All testable assertions have appropriate test files
- ✅ Implementation tests live in `app/**/__tests__/` directories
- ✅ Sidecar validation tests live as `*.test.sh` alongside assertions
- ✅ Both test types are documented with examples
- ✅ Test frameworks are set up and working (JS + bash)
- ✅ Tests validate assertion success criteria
- ✅ Tests pass consistently
- ✅ Agent workflow includes test validation step
- ✅ Sidecar test runner exists (see sidecar-test-runner assertion)

**Tests:** `src/parser/__tests__/assertions-have-tests.test.js`
