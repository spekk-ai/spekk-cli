---
id: telemetry-redaction
parent: telemetry
created: 2026-04-08T17:00:00Z
priority: 1
status: not_started
depends-on: telemetry-local-queue
branch: feature/telemetry
---

# Telemetry Redaction

## What Must Be True

Before any event is written to the local queue, it passes through a redaction pipeline that strips file contents, secret-like patterns, and any user-configured deny patterns. Events that cannot be fully redacted are dropped rather than queued.

## Success Criteria

- ✅ New package `internal/telemetry/redact/`
- ✅ `Redact(content string, extraPatterns []string) (redacted string, rulesApplied []string)` function
- ✅ Built-in redaction rules (always applied):
  - **file-contents**: any code block containing file content (heuristic: fenced code blocks with more than 3 lines and no spec-related keywords) is replaced with `[file contents redacted]`
  - **api-keys**: patterns matching common API key formats (`sk-...`, `ghp_...`, `xoxb-...`, `AKIA[0-9A-Z]{16}`, etc.)
  - **env-vars**: lines matching `^[A-Z_]+=.*$` where the value looks like a secret (long, random, high-entropy)
  - **private-paths**: absolute paths under `/Users/*/`, `/home/*/`, replaced with `[redacted path]`
  - **email-addresses**: `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}` → `[email redacted]`
  - **bearer-tokens**: `Bearer [A-Za-z0-9+/=._-]{20,}` → `Bearer [redacted]`
- ✅ User-configured extra patterns from `config.Redaction.ExtraPatterns` are applied after built-ins
- ✅ Each applied rule is added to the event's `redaction_rules_applied` list
- ✅ After redaction, the `redacted: true` flag is set on the event envelope
- ✅ `RedactEvent(event []byte, extraPatterns []string) ([]byte, error)` is the main entry point used by capture code
- ✅ Unit tests cover every built-in rule with positive and negative cases
- ✅ Test for a message that is ONLY secrets — verify the message is reduced to the placeholder, not dropped (we still learn "user said something that was all secrets")
- ✅ Test that user-configured patterns from config are applied
- ✅ Test that redaction is deterministic — same input produces same output
- ✅ Golden-file tests in `internal/telemetry/redact/testdata/` showing input/output pairs for documentation

## Redaction Pipeline Order

1. File contents (broadest — removes whole blocks)
2. Private paths
3. API keys (specific formats)
4. Bearer tokens
5. Env vars (structured)
6. Email addresses
7. User-configured extra patterns

Each step processes the output of the previous step.

## Defensive Rules

- Redaction is always-on for telemetry events. There is no way to disable it.
- If a regex in `extra_patterns` is invalid, the event is **dropped** (not queued) and a warning is logged. This prevents bad config from exfiltrating unredacted data.
- Rate-limit redaction warnings to avoid log spam.

## Out of Scope

- NLP-based PII detection (name, address, etc.)
- Binary content redaction (events are text-only)
- Redaction allow-lists (only deny patterns supported)

## Notes

Redaction is the user's safety net. Getting it wrong leaks private data. The test suite should be aggressive — every new pattern added to the built-in list gets golden-file tests demonstrating what it does and doesn't match.
