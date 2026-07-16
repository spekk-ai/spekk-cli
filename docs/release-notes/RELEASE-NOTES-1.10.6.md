# Spekk CLI 1.10.6 — Self-Documenting `specs/` Tree

This release makes a `specs/` directory explain itself — to a developer reading it on GitHub, to an AI agent grepping the tree, and to anyone who opens a single assertion file and wonders what `status:` or `parent:` mean.

## A CLI-managed block in `specs/README.md`

`spekk init` now writes a `specs/README.md` containing a delimited, CLI-owned region:

```
<!-- spekk:managed:begin -->
## Concepts
...
## Frontmatter reference
  id          required, kebab-case
  status      optional, defaults to not_started
  ...
## Schema version
spekk_schema_version: 1
<!-- spekk:managed:end -->
```

The managed region documents the concept model, every spec and assertion frontmatter field (kept accurate against the parser — valid `status` values, `priority` range, `branch` default, timestamp format), and a `spekk_schema_version`.

Key behaviors:

- **Idempotent** — regenerating with no schema change produces a byte-identical file. Running `spekk init` twice never churns the README.
- **Human prose is never touched** — anything outside the two markers is preserved byte-for-byte, so you can add your own project notes above or below the managed block.
- **Legacy upgrade** — an existing README with no managed block is upgraded in place: your content is preserved and one managed region is appended.
- **Corrupt-fence recovery** — a hand-broken fence (missing, duplicated, or out-of-order markers) is recovered to exactly one clean region.
- **No live state** — the managed block is a pure function of the schema version; it holds no spec names, counts, or statuses. `spekk status` remains the authoritative live view.

Drift detection is just a diff: bump the schema version and the next `spekk init` shows exactly what changed.

## Self-documenting file headers

The coach and builder agents now add a brief HTML-comment header to newly authored spec and assertion files, documenting each frontmatter field inline (invisible when rendered on GitHub) with a pointer back to `specs/README.md`. A developer opening one assertion file no longer has to guess what `parent:` or `depends-on:` mean.

## Upgrade

```bash
spekk update
```
