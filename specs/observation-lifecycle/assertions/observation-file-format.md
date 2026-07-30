---
id: observation-file-format
parent: observation-lifecycle
created: 2026-07-26T12:00:00Z
priority: 1
status: done
---

# Observations Are Markdown Files With a Defined Frontmatter Schema

## Description

Every observation is a markdown file under `observations/` whose YAML
frontmatter carries the full lifecycle record. The schema is documented in the
parent spec and in the observer prompt, and anything that parses observations
(the indexer, the announce subcommand, validation) agrees on it.

## Success Criteria

- The documented observation frontmatter schema is exactly:
  - `slug` — kebab-case identifier; matches the `observer/<slug>` branch name
  - `type` — one of `code_spec_misalignment` | `outdated_specs`
  - `severity` — one of `high` | `medium` | `low`
  - `status` — one of `open` | `resolved` | `dismissed`
  - `created` — ISO 8601 timestamp
  - `announced` — ISO 8601 timestamp; **absent** until a Slack conversation
    has been opened for this observation (absence is meaningful — it is the
    "not yet announced" marker; no separate ledger exists)
  - `pr` — URL, optional
  - `affected` — list of repo file paths (the evidence); required and
    non-empty
- **Evidence gate:** an observation with a missing or empty `affected` list is
  invalid — tooling that consumes observations (indexer, announce) rejects or
  skips it rather than treating it as announceable. No evidence, no
  observation.
- A field outside this set does not break parsing (unknown fields are
  ignored), so the format can grow without flag days.
- `specs/observer-agent/observer.prompt.md` (or its successor) instructs the
  observer to emit observations in exactly this format.

**Note:** `announced` is a timestamp, not a boolean — its value records *when*
the conversation opened, and its absence (not `announced: false`) encodes
"unannounced". Parsers must treat a present-but-empty value as invalid rather
than as either state.

**Tests:** internal/observation/observation_test.go (TestParseValid,
TestParseValidation, TestParseIgnoresUnknownFields)
