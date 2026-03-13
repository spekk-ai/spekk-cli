---
id: sandbox-token-generator
parent: sandbox-command
created: 2026-03-13T00:00:00Z
priority: 1
status: done
---

# Sandbox Token Generator

`src/sandbox/tokens.js` exists and exports a `generateAgentToken()` function that produces cryptographically random, URL-safe tokens in a consistent format.

## Success Criteria

- `src/sandbox/tokens.js` exports `generateAgentToken()`
- Uses Node.js built-in `crypto.randomBytes(32).toString('base64url')`
- Returns a 43-character URL-safe base64 string (no padding, no `=`)
- Format matches existing tokens like `aB6Ye_fia5UH8k1KGTbpoPzDtYfaPqyiTcfsxR6X9EE`
- Unit test in `src/sandbox/__tests__/tokens.test.js` verifies:
  - Length is 43 characters
  - Only contains `[A-Za-z0-9_-]` characters
  - Two calls return different values
