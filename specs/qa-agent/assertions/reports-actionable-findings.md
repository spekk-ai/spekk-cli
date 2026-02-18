---
id: reports-actionable-findings
parent: qa-agent
created: 2026-02-18T12:00:00Z
priority: 1
status: not_started
---

# QA Agent Reports Actionable Findings

## What Must Be True

QA agent must produce a clear, actionable report that developers can use to fix issues.

## Report Requirements

### 1. Summary Section

```
QA Validation Summary
=====================
Errors:   3 (must fix)
Warnings: 5 (should fix)
Passed:   12

Status: FAILED - Fix errors before marking done
```

### 2. Categorized Issues

Group by severity and type:
- **Errors**: Block marking assertion as done
- **Warnings**: Should fix but don't block
- **Info**: Observations, suggestions

### 3. Actionable Details

Each issue must include:
- What's wrong (clear description)
- Where to fix (file path + line number)
- How to fix (code suggestion when possible)

### 4. Fix Suggestions

Provide copy-paste fixes when possible:

```
Issue: firstName missing maxLength validation
File:  users/forms/UserForm.tsx:24
Fix:   Add .max(50) to validation schema

  // Current:
  firstName: z.string()

  // Should be:
  firstName: z.string().max(50)
```

## Success Criteria

- Report has clear summary with counts
- Issues categorized by severity
- Each issue has file location
- Each issue has fix suggestion (when determinable)
- Report can be output to stdout or file
- Exit code reflects pass/fail status

## Example Full Report

```
QA Validation Report
====================

Summary: 2 errors, 3 warnings, 8 passed
Status:  FAILED

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
ERRORS (must fix)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

[E1] Form constraint mismatch
     Field:    firstName
     OpenAPI:  maxLength=50
     Form:     no max validation
     File:     users/forms/UserForm.tsx:24
     Fix:      firstName: z.string().max(50)

[E2] E2E test failure
     Test:     user-can-submit-order.spec.ts
     Error:    Validation error: firstName too long
     File:     tests/e2e/user-can-submit-order.spec.ts:45
     Fix:      Update form to enforce maxLength

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
WARNINGS (should fix)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

[W1] Input missing attribute
     Field:    firstName
     Missing:  maxLength={50}
     File:     users/forms/UserForm.tsx:31
     Fix:      <input maxLength={50} ... />

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
PASSED
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

✓ Swagger audit
✓ TN services validator (Zod ↔ OpenAPI)
✓ Validate test IDs
✓ 5 other checks...
```
