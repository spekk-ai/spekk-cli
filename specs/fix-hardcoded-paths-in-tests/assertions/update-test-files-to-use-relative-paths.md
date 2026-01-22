---
id: update-test-files-to-use-relative-paths
parent: fix-hardcoded-paths-in-tests
created: 2026-01-22T20:45:00Z
priority: 1
status: done
---

# Update Test Files to Use Relative Paths

## What Must Be True

All test files that currently use hardcoded absolute paths must be updated to use relative paths that work in any environment.

### Files That Need Updating

These test files contain hardcoded paths to `/Users/william/thinknimble/spekk-cli/bin/spekk.js`:

1. `src/__tests__/assertions-appear-as-subitems.test.js`
2. `src/__tests__/show-command.test.js`
3. `src/__tests__/clicking-shows-details-panel.test.js`
4. `src/__tests__/web-detail-badges-format.test.js`
5. `src/__tests__/fix-javascript-template-errors.test.js`

### Required Changes

**Pattern to Replace:**
```javascript
execSync('node /Users/william/thinknimble/spekk-cli/bin/spekk.js show', { 
  encoding: 'utf8',
  cwd: testDir,
  timeout: 5000 
});
```

**Replace With:**
```javascript
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

// At top of file, add:
const __dirname = dirname(fileURLToPath(import.meta.url));
const projectRoot = join(__dirname, '../..');
const spekkBin = join(projectRoot, 'bin/spekk.js');

// In test execution:
execSync(`node "${spekkBin}" show`, { 
  encoding: 'utf8',
  cwd: testDir,
  timeout: 5000 
});
```

### Implementation Steps

1. **Add path resolution imports** to each affected test file
2. **Calculate project root path** dynamically using `__dirname` and relative navigation
3. **Replace hardcoded paths** with dynamically resolved paths
4. **Quote the path** to handle spaces in directory names
5. **Test in multiple environments** to ensure compatibility

### Alternative Simpler Approach

Instead of complex path resolution, use the project root as the working directory:

```javascript
// Set working directory to project root instead of temp dir
const originalCwd = process.cwd();
process.chdir(join(__dirname, '../..'));

try {
  execSync('node bin/spekk.js show', { 
    encoding: 'utf8',
    env: { ...process.env, SPEKK_TEST_OUTPUT_DIR: testDir },
    timeout: 5000 
  });
} finally {
  process.chdir(originalCwd);
}
```

This approach:
- ✅ Uses simple relative paths
- ✅ Works in any environment
- ✅ Doesn't require complex path resolution
- ✅ Maintains test isolation

## Success Criteria

- ✅ No test files contain `/Users/william/thinknimble/spekk-cli/` absolute paths
- ✅ All affected test files use dynamic path resolution or relative paths
- ✅ Tests pass on macOS, Linux, and Windows
- ✅ Tests pass in CI environment (GitHub Actions Ubuntu)
- ✅ `npm run test:impl` returns 0 exit code
- ✅ All 5 identified test files are updated and working