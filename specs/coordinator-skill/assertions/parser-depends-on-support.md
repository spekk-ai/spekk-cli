---
id: parser-depends-on-support
parent: coordinator-skill
created: 2026-02-25T00:00:00Z
priority: 2
status: done
---

# Parser Validates and Parses depends-on Field

Spec parser reads and validates the `depends-on` field in assertion frontmatter.

## Success Criteria

### Field Parsing

Parser extracts `depends-on` field from YAML:
```yaml
---
id: chat-message-input
depends-on: chat-session-model
---
```

Result:
```javascript
{
  id: 'chat-message-input',
  dependsOn: 'chat-session-model',  // camelCase in JS
  // ... other fields
}
```

### Field Validation

**Valid values:**
- String (assertion ID in kebab-case)
- `null` (explicit no dependency)
- Omitted (implicit no dependency, same as null)

**Validation rules:**
1. **Type check**: If present, must be string or null
2. **Format check**: If string, must be kebab-case (lowercase, hyphens)
3. **Reference check**: Referenced assertion ID must exist in specs/
4. **No self-reference**: `depends-on` cannot point to same assertion
5. **No circular deps**: Walk dependency chain, detect cycles

### Error Messages

**Invalid type:**
```
Error: Field 'depends-on' must be a string or null in specs/chat-system/assertions/chat-message-input.md
Found: ["chat-session-model", "websocket-connection"]
```

**Invalid format:**
```
Error: Field 'depends-on' must be kebab-case (lowercase with hyphens) in specs/chat-system/assertions/chat-message-input.md
Found: "chatSessionModel"
```

**Missing reference:**
```
Error: Field 'depends-on' references non-existent assertion 'nonexistent-id' in specs/chat-system/assertions/chat-message-input.md
```

**Circular dependency:**
```
Error: Circular dependency detected:
  chat-message-input → chat-session-model → user-auth → chat-message-input

Break the cycle by removing or changing one of the dependencies.
```

## Implementation

### Parser Updates

Update `src/parser/index.js`:

```javascript
function validateAssertionFields(data, filePath, allAssertions) {
  // ... existing validations ...
  
  // Validate depends-on field
  if (data['depends-on'] !== undefined && data['depends-on'] !== null) {
    const dependsOn = data['depends-on'];
    
    // Type check
    if (typeof dependsOn !== 'string') {
      throw new Error(`Field 'depends-on' must be a string or null in ${filePath}`);
    }
    
    // Format check (kebab-case)
    const kebabCasePattern = /^[a-z][a-z0-9]*(-[a-z0-9]+)*$/;
    if (!kebabCasePattern.test(dependsOn)) {
      throw new Error(`Field 'depends-on' must be kebab-case in ${filePath}`);
    }
    
    // Reference check
    if (!allAssertions.some(a => a.id === dependsOn)) {
      throw new Error(`Field 'depends-on' references non-existent assertion '${dependsOn}' in ${filePath}`);
    }
    
    // Self-reference check
    if (dependsOn === data.id) {
      throw new Error(`Field 'depends-on' cannot reference itself in ${filePath}`);
    }
  }
}

// Circular dependency detection (after all assertions parsed)
function detectCircularDependencies(assertions) {
  for (const assertion of assertions) {
    const visited = new Set();
    let current = assertion;
    
    while (current && current.dependsOn) {
      if (visited.has(current.id)) {
        const cycle = [...visited, current.id].join(' → ');
        throw new Error(`Circular dependency detected:\n  ${cycle}\n\nBreak the cycle by removing or changing one of the dependencies.`);
      }
      
      visited.add(current.id);
      current = assertions.find(a => a.id === current.dependsOn);
    }
  }
}
```

### Parsing Logic

Convert YAML field name to camelCase:
```javascript
// In parseFrontmatter()
const frontmatter = {};
// ...
if (key === 'depends-on') {
  frontmatter.dependsOn = value;
} else {
  frontmatter[key] = value;
}
```

## Validation

- Parser reads `depends-on` field correctly
- Parser converts to camelCase `dependsOn` in JS
- Omitted field treated as null (no dependency)
- Invalid types rejected with clear error
- Invalid formats rejected with clear error
- Missing references rejected with clear error
- Self-references rejected with clear error
- Circular dependencies detected and rejected
- Error messages are actionable

**Tests:** `src/parser/__tests__/depends-on-validation.test.js`

All tests passing:
- Parses `depends-on` field and converts to camelCase ✓
- Accepts omitted depends-on field ✓
- Rejects invalid type (arrays) ✓
- Rejects invalid format (non-kebab-case) ✓
- Rejects non-existent assertion references ✓
- Rejects self-references ✓
- Detects circular dependencies ✓
- Accepts valid dependency chains without cycles ✓
- Unit tests for validateDependsOn and detectCircularDependencies ✓
