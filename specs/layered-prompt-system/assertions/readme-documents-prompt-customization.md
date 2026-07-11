---
id: readme-documents-prompt-customization
parent: layered-prompt-system
created: 2026-03-16T00:00:00Z
priority: 1
status: done
depends_on:
  - works-for-all-agents
---

# README Documents Prompt Customization

## Description

The README includes a section early in the document (before detailed CLI usage) explaining how to customize agent prompts using the layered prompt system.

## Success Criteria

- README has a "Customizing Agent Prompts" section near the top (setup/getting-started area)
- Explains the extend vs override distinction with file naming convention
- Shows the global (`~/.config/spekk/`) and local (`.spekk/`) paths
- Includes at least one example of extending a prompt and one of overriding
- Mentions that `.spekk/` can be committed or gitignored per team preference
