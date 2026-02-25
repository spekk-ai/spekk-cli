---
id: parser-branch-support
parent: coordinator-skill
created: 2026-02-25T00:00:00Z
priority: 2
status: not_started
---

# Parser Validates and Parses branch Field

Spec parser reads and validates the `branch` field in assertion frontmatter.

## Success Criteria

### Field Parsing

Parser extracts `branch` field from YAML:
```yaml
---
id: chat-message-input
branch: feature/chat-system
---
```

Result:
```javascript
{
  id: 'chat-message-input',
  branch: 'feature/chat-system',
  // ... other fields
}
```

### Field Validation

**Valid values:**
- String matching git branch naming conventions
- Common patterns: `main`, `feature/<name>`, `bugfix/<name>`, `hotfix/<name>`

**Validation rules:**
1. **Type check**: Must be string if present
2. **Format check**: Valid git branch name (no spaces, no special chars except `/`, `-`, `_`)
3. **Convention check** (warning): Should match common patterns

### Git Branch Name Rules

Valid according to git:
- No spaces
- No control characters
- No `..`, `@{`, `\`, `~`, `^`, `:`, `?`, `*`, `[`
- Cannot start or end with `/`
- Cannot end with `.`
- Cannot contain consecutive dots `..`

**Simplified check** (covers 99% of cases):
```regex
^[a-zA-Z0-9][a-zA-Z0-9/_-]*$
```

### Warning vs Error

**Error (invalid git branch):**
```
Error: Field 'branch' contains invalid characters in specs/chat-system/assertions/chat-message-input.md
Found: "feature/chat system"
Git branch names cannot contain spaces.
```

**Warning (uncommon pattern):**
```
Warning: Field 'branch' uses non-standard pattern in specs/chat-system/assertions/chat-message-input.md
Found: "my-custom-branch"
Consider using standard patterns: main, feature/<name>, bugfix/<name>, hotfix/<name>
```

## Implementation

### Parser Updates

Update `src/parser/index.js`:

```javascript
function validateAssertionFields(data, filePath) {
  // ... existing validations ...
  
  // Validate branch field
  if (data.branch !== undefined && data.branch !== null) {
    const branch = data.branch;
    
    // Type check
    if (typeof branch !== 'string') {
      throw new Error(`Field 'branch' must be a string in ${filePath}`);
    }
    
    // Format check (valid git branch name)
    const validBranchPattern = /^[a-zA-Z0-9][a-zA-Z0-9/_-]*$/;
    if (!validBranchPattern.test(branch)) {
      throw new Error(`Field 'branch' contains invalid characters in ${filePath}\nFound: "${branch}"\nGit branch names can only contain letters, numbers, slashes, hyphens, and underscores.`);
    }
    
    // Cannot start or end with /
    if (branch.startsWith('/') || branch.endsWith('/')) {
      throw new Error(`Field 'branch' cannot start or end with '/' in ${filePath}`);
    }
    
    // Warning for non-standard patterns (don't throw, just warn)
    const standardPatterns = /^(main|master|develop|feature\/|bugfix\/|hotfix\/|release\/)/;
    if (!standardPatterns.test(branch)) {
      console.warn(`Warning: Field 'branch' uses non-standard pattern in ${filePath}\nFound: "${branch}"\nConsider using standard patterns: main, feature/<name>, bugfix/<name>, hotfix/<name>`);
    }
  }
}
```

### Default Value

If `branch` field is omitted, default to `main`:
```javascript
// After parsing
if (!assertion.branch) {
  assertion.branch = 'main';
}
```

## Validation

- Parser reads `branch` field correctly
- Invalid characters rejected with clear error
- Invalid formats (starts/ends with `/`) rejected
- Non-standard patterns show warning but don't fail
- Omitted field defaults to `main`
- Error messages are actionable
- Warnings are informative but non-blocking

**Tests:**
- `src/parser/__tests__/branch-validation.test.js`
- Test valid cases (main, feature/x, bugfix/y)
- Test invalid cases (spaces, special chars, bad format)
- Test warnings for non-standard patterns
- Test default value when omitted
