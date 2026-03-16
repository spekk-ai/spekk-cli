---
id: fix-hardcoded-paths-in-tests
created: 2026-01-22T20:45:00Z
priority: 1
---

# Fix Hardcoded Paths in Tests

## Problem

CI tests are failing because test files use hardcoded absolute paths like `/Users/william/thinknimble/spekk-cli/bin/spekk.js` which only exist on the developer's local machine, not in CI environments.

## What Must Be True

All test files must use relative paths or dynamically resolved paths that work across all environments (local development, CI, other developers' machines).

### Current Failing Pattern

Tests currently use:
```javascript
execSync('node /Users/william/thinknimble/spekk-cli/bin/spekk.js show', { 
  encoding: 'utf8',
  cwd: testDir,
  timeout: 5000 
});
```

### Required Solution Pattern

Tests must use relative paths from the project root:
```javascript
execSync('node bin/spekk.js show', { 
  encoding: 'utf8',
  cwd: process.cwd(), // Use project root as working directory
  timeout: 5000 
});
```

Or dynamically resolve the project root:
```javascript
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const projectRoot = join(__dirname, '../..');
const spekkBin = join(projectRoot, 'bin/spekk.js');

execSync(`node ${spekkBin} show`, { 
  encoding: 'utf8',
  cwd: testDir,
  timeout: 5000 
});
```

## Success Criteria

- ✅ All tests pass in CI environment (GitHub Actions)
- ✅ All tests pass on different developer machines
- ✅ No hardcoded absolute paths in any test files
- ✅ Tests use relative paths or dynamically resolved paths
- ✅ Tests work regardless of where the project is cloned
- ✅ `npm run test:impl` returns 0 exit code in CI