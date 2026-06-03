---
id: security-audit-remediation
created: 2026-06-03T12:00:00Z
priority: 1
---

# Security Audit Remediation

Fixes for vulnerabilities identified during a security audit of the spekk-cli codebase. Covers shell injection in sandbox credential handling, prompt injection via CLI flags, XSS in the spec explorer, path traversal bypass in the transcript handler, skill content tag breakout, SSH host key verification, and WebSocket session authentication.

Builds on the earlier `golang-security-hardening` spec which addressed the initial Go migration security issues. This spec covers gaps and new attack vectors found in the follow-up audit.
