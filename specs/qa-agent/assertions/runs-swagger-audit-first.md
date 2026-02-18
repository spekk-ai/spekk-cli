---
id: runs-swagger-audit-first
parent: qa-agent
created: 2026-02-18T12:00:00Z
priority: 1
status: not_started
---

# QA Agent Runs Swagger Audit First

## What Must Be True

Before running any other validation, QA agent must verify the OpenAPI schema is accurate by running `/swagger-audit`.

If the source of truth (OpenAPI) is wrong, all downstream validation is unreliable.

## Why This Matters

```
Django models → Serializers → Viewsets → OpenAPI → Everything else
                                              ↑
                                    If this is wrong,
                                    all checks below fail
```

Example failure mode:
1. Dev adds `max_length=50` to Django model
2. Forgets to regenerate swagger or add `@extend_schema`
3. OpenAPI still shows no maxLength
4. `/tn-services-validator` passes (Zod matches wrong swagger)
5. Form validation check passes (matches wrong swagger)
6. User submits 100 chars, DB rejects → runtime error

## Success Criteria

- QA agent runs `/swagger-audit` as first step
- If swagger-audit finds errors, QA agent stops and reports them
- If swagger-audit passes, QA agent continues to other checks
- Report clearly indicates swagger validation status

## Workflow

```
1. Run /swagger-audit
   ├── ERRORS → Stop, report "Fix swagger first"
   └── PASS → Continue to next checks
2. Run /tn-services-validator
3. ... (other checks)
```
