---
id: prompts-and-docs-use-docs-search
parent: source-document-index
created: 2026-10-17T16:00:00Z
priority: 2
status: done
branch: feature/source-document-index
depends-on: assertion-sources-links
---

# Prompts and User Docs Teach `spekk docs`

## Success criteria

- The coach prompt replaces "No index table holds an assertion body, so this is the one search `spekk query` cannot serve" and its `grep -rl` example with `spekk docs search "<words>" --kind spec,assertion`, and adds one short section on source documents: search first, show the passage, cite it in `sources`, file a new document with `spekk docs add` and write its summary.
- The builder prompt's "Preferred Tool Patterns" names `spekk docs search` for spec prose and docs, after `spekk next` and `spekk query`.
- The tables lists in the coach prompt and in `spekk query --help` add `documents`, `document_tags`, and `passages`.
- `docs/` has a user guide page for source documents (layout, formats and converters, the commands, the output cap, freshness, secrets and pointers, `sources`), linked from the site navigation. `docs/cli-reference.md` has a `spekk docs` section. A release notes file describes the change.
- `spekk help` lists `docs`. `spekk docs --help` and the help of each subcommand print usage.

**Tests:** none for prose. Verify by running every example command in the changed text against this repository.
