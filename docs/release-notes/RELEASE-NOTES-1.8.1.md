# Spekk CLI 1.8.1 — Show Markdown Rendering Fix

This patch release covers changes since v1.8.0.

## Show Markdown Rendering Fix (PR #119)

The `spekk show` detail panel previously rendered raw file content including YAML frontmatter. Now:

- Only the markdown body is rendered (frontmatter stripped)
- Prose renders with proper typography — headings, paragraphs, lists, blockquotes, tables
- Monospace reserved for inline code and code blocks

## Spec Corpus Cleanup (PR #115)

Removed transition-shaped and superseded specs from the repository's own spec corpus. No effect on the shipped binary.

## Upgrade

```bash
spekk update        # if installed to a user-writable directory
sudo spekk update   # if installed to /usr/local/bin
```
