---
id: install-source-flag-fetches-arbitrary-url
parent: skill-install-system
created: 2026-05-22T12:00:00Z
priority: 1
status: not_started
depends-on: install-fetches-from-official-registry
branch: feature/skill-install-system
---

# --source Flag Fetches from an Arbitrary URL

## Description

`--source <URL>` bypasses the official registry and fetches a markdown file from any http(s) URL. This lets users install skills from gists, internal company servers, forks, or commit-pinned raw URLs.

## Success Criteria

- `spekk install coach my-skill --source https://example.com/foo.md` fetches from the provided URL (not the registry)
- The destination filename uses the `<skill>` positional arg, not the URL's basename — so the example above writes to `.spekk/skills/coach/my-skill.md`
- If `<skill>` is omitted, the filename is derived from the URL's basename minus `.md` (e.g. `https://example.com/foo.md` → `foo`)
- If the URL's basename is empty (URL ends in `/`) or otherwise unusable, the command exits non-zero with an error asking for an explicit `<skill>` argument
- URLs without `http://` or `https://` scheme are rejected
- Malformed URLs (parse errors, no host) are rejected with a clear message
- The fetched body is written verbatim — no validation that it's "really" markdown, no Content-Type sniffing
