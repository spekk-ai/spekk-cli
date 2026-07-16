# Spekk CLI 1.10.9 — Remove Self-Documenting Spec Headers

Reverts the second half of the 1.10.6 "self-documenting `specs/` tree" feature. The coach and builder agents were instructed to prepend an HTML-comment frontmatter-explainer header to every newly authored spec and assertion `.md` file:

```
<!--
  id       — kebab-case identifier for this spec
  created  — ISO 8601 UTC timestamp of when this spec was authored
  ...
  See specs/README.md for the full concept + frontmatter reference.
-->
```

In practice this cluttered every spec file with boilerplate that repeated what `specs/README.md` already documents. It's gone.

## What changed

- The header-template instructions are removed from the embedded coach and builder prompts, so `spekk prompt coach` / `spekk prompt builder` no longer tell agents to add the header. Installed agents (which fetch their instructions from the binary) stop adding it as soon as you upgrade.
- No CLI code generated these headers — they were purely prompt-driven — so nothing else changes.

## What's kept

The managed `specs/README.md` from 1.10.6 is unchanged: `spekk init` still writes and idempotently regenerates it. It remains the single reference for the concept model and frontmatter schema — generated spec files just no longer carry a redundant inline copy.

Already-authored files that picked up a header are not touched by this change; delete the comment block by hand if you want it gone.

## Upgrade

```bash
spekk update
```
