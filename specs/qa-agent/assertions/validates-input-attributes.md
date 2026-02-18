---
id: validates-input-attributes
parent: qa-agent
created: 2026-02-18T12:00:00Z
priority: 1
status: not_started
---

# QA Agent Validates Input Field Attributes Against OpenAPI

## What Must Be True

QA agent must verify that HTML input field attributes match the constraints defined in OpenAPI schema.

## The Problem

```
OpenAPI: firstName: { maxLength: 50, required: true }
Input:   <input name="firstName" ??? />
```

Even if form validation catches errors, input attributes provide:
- Better UX (user can't type beyond limit)
- Accessibility (screen readers announce requirements)
- Browser-native validation (works without JS)

## What To Check

For each input field that maps to an API field:

| Constraint | OpenAPI | Input Attribute |
|------------|---------|-----------------|
| maxLength | `maxLength: 50` | `maxLength={50}` |
| required | `required: true` | `required` |
| pattern | `pattern: "..."` | `pattern="..."` |
| min | `minimum: 0` | `min={0}` |
| max | `maximum: 100` | `max={100}` |
| type | `format: "email"` | `type="email"` |

## Detection Approach

1. Use `mcp__tnm-openapi__get_schema` to get field constraints
2. Find input elements in form components
3. Parse input attributes from JSX/HTML
4. Compare against OpenAPI constraints

## Success Criteria

- QA agent identifies input elements and their field names
- Extracts attributes from input elements
- Compares against OpenAPI field constraints
- Reports missing or mismatched attributes with file locations

## Example Output

```
Input Attribute Validation
==========================

UserForm.tsx
  <input name="firstName">
    ✗ Missing maxLength (OpenAPI: 50)
    ✗ Missing required (OpenAPI: true)
    Line: 45

  <input name="email">
    ✗ type="text" should be type="email" (OpenAPI format: email)
    Line: 52

  <input name="age">
    ✓ min={0} max={150} matches OpenAPI

OrderForm.tsx
  ✓ All input attributes match OpenAPI
```
