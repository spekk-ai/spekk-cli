---
id: telemetry-privacy-doc
parent: telemetry
created: 2026-04-08T17:00:00Z
priority: 2
status: not_started
branch: feature/telemetry
---

# Telemetry Privacy Documentation

## What Must Be True

A dedicated privacy policy document at `docs/telemetry.md` explains in plain language what is captured, why, how it's used, how to opt out, and how to request deletion. The document is the canonical source of truth linked from the consent flow and the CLI help text.

## Success Criteria

- ✅ New file `docs/telemetry.md` with the following sections:
  - **What is captured** — verbatim list matching the consent flow disclosure
  - **What is never captured** — explicit non-capture list
  - **Why we collect this** — PM-vs-engineering signal explained
  - **Where it goes** — endpoint URL, storage, access
  - **How long it is retained** — default retention policy (placeholder: "30 days unless used for training; aggregated metrics retained indefinitely")
  - **How to opt out** — `spekk telemetry disable`
  - **How to delete local data** — `spekk telemetry delete` and `purge`
  - **How to delete uploaded data** — email `privacy@spekk.ai` with install ID
  - **Schema reference** — link to `docs/telemetry-schema.md`
  - **Changelog** — every schema or policy change dated
- ✅ The document is linked from:
  - The consent flow screen
  - `spekk telemetry --help`
  - `README.md` in a short "Telemetry" section
- ✅ `spekk telemetry show-policy` CLI command prints the policy file contents to stdout for offline viewing
- ✅ A test verifies the policy file exists in the binary's expected location (caught at build time or first-run check)
- ✅ Document is version-controlled; policy changes bump `consent_version` in code and trigger re-consent

## Tone and Style

- Plain language, no legalese
- Bullet lists over paragraphs
- Concrete examples ("we capture: the text you typed; we do NOT capture: the contents of files you reference")
- Dated changelog at the bottom so users can diff old versions

## Out of Scope

- Multi-language translations
- Formal legal review (the policy document is a starting point; legal review is separate work)
- GDPR data subject access request (DSAR) automation

## Notes

The document exists because the consent screen can't say everything. The screen says "here's the gist, here's where to read more." This file is the "read more."
