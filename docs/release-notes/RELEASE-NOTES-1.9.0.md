# Spekk CLI 1.9.0 — Cross-Branch Merge Preview

This release covers changes since v1.8.0.

## Cross-Branch / Merge-Preview Mode for `spekk show` (PRs #118, #120)

`spekk show --cross-branch` previews what merging every other branch into your current branch would do to the spec corpus — without touching the working tree or index.

```bash
spekk show --cross-branch                          # Preview all branches
spekk show --cross-branch --branch-filter 'feat/*' # Only feature branches
spekk show --cross-branch --watch                   # Live reload
```

What you see in the explorer:

- **Inline badges** on affected specs/assertions: `+` (addition), change, `⚠` (conflict), `✕` (deletion)
- **Branch checkbox dropdown** in the header — filter which branches contribute, persisted per project in `localStorage`
- **Foreign item synthesis** — incoming additions from other branches appear with real metadata, flagged `foreign: true` so they disappear when all contributing branches are deselected

Under the hood:

- Uses `git merge-tree --write-tree` (git >= 2.38) for honest conflict detection; degrades gracefully on older git with a warning badge
- All git operations are read-only — funnelled through an allowlist chokepoint, no checkout/merge/index mutation
- Watch mode caches classification on a git ref-state fingerprint; working-tree edits re-render cheaply, ref/commit moves invalidate the cache and reclassify
- CDN dependencies (marked, DOMPurify) inlined for offline-safe operation

## Observer Skill Discovery and Dynamic Help (PR #116)

Observer now supports skills, matching coach and builder:

```bash
spekk observer --help          # Shows AVAILABLE SKILLS section
spekk observer coverage-gap    # Launch with a specific skill
```

Details:

- **Layered skill resolution**: local (`.spekk/skills/observer/`) > global (`~/.config/spekk/skills/observer/`) > package > embedded — same precedence as coach and builder
- **`coverage-gap` seed skill**: Ships embedded in the binary; scans `internal/` for code with no spec backing (inverse of the default spec-to-code lens)
- **Per-skill observation output**: Each skill writes to `observations/{skill-name}/` with required frontmatter and body sections
- Observer flags (`--interval`, `--quiet`) now work correctly alongside skill arguments

## Show Markdown Rendering Fix (PR #119)

The `spekk show` detail panel previously rendered raw file content including YAML frontmatter. Now:

- Only the markdown body is rendered (frontmatter stripped)
- Prose renders with proper typography — headings, paragraphs, lists, blockquotes, tables
- Monospace reserved for inline code and code blocks

## Upgrade

```bash
spekk update        # if installed to a user-writable directory
sudo spekk update   # if installed to /usr/local/bin
```
