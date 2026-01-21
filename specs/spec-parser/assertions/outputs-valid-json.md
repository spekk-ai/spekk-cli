---
id: outputs-valid-json
parent: spec-parser
created: 2026-01-20T16:20:00Z
priority: 2
status: done
---

# Parser Must Output Valid JSON

## What Must Be True

The parser must output well-formed, machine-readable JSON to stdout.

## Output Format

### Success Case (work item found)
```json
{
  "type": "assertion",
  "id": "validates-required-fields",
  "parent": "spec-parser",
  "file": "specs/spec-parser/assertions/validates-required-fields.md",
  "priority": 1,
  "status": "not_started",
  "title": "Parser Must Validate Required Fields",
  "content": "... full markdown content including frontmatter ...",
  "spec": {
    "id": "spec-parser",
    "file": "specs/spec-parser/spec-parser.md",
    "title": "Spec Parser"
  }
}
```

### All Complete Case
```json
{
  "status": "complete",
  "message": "All specifications are complete"
}
```

### No Specs Found Case
```json
{
  "status": "empty",
  "message": "No specifications found in specs/ directory"
}
```

### Error Case
```json
{
  "error": true,
  "message": "Missing required field 'id' in specs/foo/bar.md",
  "file": "specs/foo/bar.md"
}
```

## Field Requirements

**Required fields (success case):**
- `type` - Always "assertion" for next work item
- `id` - The assertion's id
- `parent` - The parent spec's id
- `file` - Relative path to assertion file
- `priority` - The assertion's priority
- `status` - Current status
- `title` - Extracted from markdown H1 heading
- `content` - Full file content (for context)
- `spec` - Object with parent spec metadata

## JSON Validation

Output must:
- Be valid JSON (parseable by `JSON.parse()`)
- Be a single JSON object (not an array, not multiple objects)
- Use consistent field names
- Include all required fields
- Use correct types (strings for text, numbers for priority)
- Be written to stdout (not stderr)

## Error Handling

Errors should also be JSON:
- Include `error: true` field
- Include `message` with clear description
- Include `file` if error relates to specific file
- Write to stdout (not stderr) so ralph-loop can parse it

## Success Criteria

- ✅ Parser outputs valid, parseable JSON
- ✅ JSON includes all required fields
- ✅ JSON uses correct types
- ✅ Output goes to stdout
- ✅ Errors are also JSON format
- ✅ Single JSON object per execution (not streaming/multiple)

**Tests:** `src/parser/__tests__/spec-parser.test.js`
