---
id: spec-explorer-sanitizes-markdown
parent: security-audit-remediation
created: 2026-06-03T12:00:00Z
priority: 2
status: not_started
depends-on: sandbox-credentials-use-safe-transport
branch: feature/spekk-sandbox-vulnrabilities
---

# Spec explorer sanitizes rendered markdown before DOM insertion

The spec explorer web interface in `internal/show/template.html` sanitizes HTML output from `marked.parse()` before inserting it into the DOM via `innerHTML`. Script tags, event handlers, and other XSS vectors in spec/assertion markdown content do not execute.

## Success Criteria

- `marked.parse()` output is sanitized before assignment to `innerHTML` — either via DOMPurify, marked's built-in sanitizer, or equivalent
- A spec containing `<script>alert('xss')</script>` does not execute the script
- A spec containing `<img src=x onerror=alert('xss')>` does not execute the handler
- Legitimate markdown formatting (headers, lists, code blocks, links, bold, italic) still renders correctly
- Both spec detail panels (line 883) and assertion detail panels (line 904) are protected
