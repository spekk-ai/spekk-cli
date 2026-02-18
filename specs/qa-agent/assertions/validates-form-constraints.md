---
id: validates-form-constraints
parent: qa-agent
created: 2026-02-18T12:00:00Z
priority: 1
status: not_started
---

# QA Agent Validates Form Constraints Against OpenAPI

## What Must Be True

QA agent must verify that form validation rules match the constraints defined in OpenAPI schema.

## The Problem

```
OpenAPI: firstName: { maxLength: 50, required: true }
Form:    firstName: { validation: ??? }
```

If form validation doesn't enforce the same constraints, users can submit invalid data that the API or DB will reject.

## What To Check

For each form field that maps to an API field:

| Constraint | OpenAPI | Form Validation |
|------------|---------|-----------------|
| maxLength | `maxLength: 50` | `.max(50)` or equivalent |
| required | `required: true` | `.required()` or equivalent |
| enum | `enum: ['a','b']` | `.oneOf(['a','b'])` or equivalent |
| pattern | `pattern: "^[a-z]+$"` | `.matches(/^[a-z]+$/)` or equivalent |
| minimum | `minimum: 0` | `.min(0)` or equivalent |
| maximum | `maximum: 100` | `.max(100)` or equivalent |

## Detection Approach

1. Use `mcp__tnm-openapi__get_schema` to get field constraints from OpenAPI
2. Find form components that submit to that endpoint
3. Parse form validation rules (custom forms library patterns)
4. Compare constraints and report mismatches

## Success Criteria

- QA agent identifies forms and their target API endpoints
- Extracts validation rules from form code
- Compares against OpenAPI field constraints
- Reports mismatches with:
  - Field name
  - OpenAPI constraint
  - Form constraint (or "missing")
  - File location for fix

## Example Output

```
Form Constraint Validation
==========================

UserForm → POST /api/users/
  ✗ firstName: OpenAPI maxLength=50, form has no max validation
    Fix: users/forms/UserForm.tsx:24

  ✗ email: OpenAPI pattern="email", form has no format validation
    Fix: users/forms/UserForm.tsx:31

  ✓ lastName: maxLength=50 matches
  ✓ age: minimum=0, maximum=150 matches

OrderForm → POST /api/orders/
  ✓ All constraints match
```
