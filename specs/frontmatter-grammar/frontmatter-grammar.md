---
id: frontmatter-grammar
created: 2026-08-28T22:00:00Z
priority: 2
---

# Frontmatter Grammar — One Reading of a Frontmatter Line

## Overview

`internal/parser` reads YAML frontmatter with a line scanner, not a YAML library. That is a deliberate trade: the scanner is small, it never fails on a shape it does not understand, and it keeps the parse of a spec tree free of a dependency. The cost is that every YAML rule it does implement has to be written out, and a rule it half-implements is worse than one it does not implement at all — a value that is read wrongly is indistinguishable from a value that was written wrongly.

This spec holds the rules the scanner does implement, so that a later change adds to one list instead of guessing.

## Scope

The scanner serves both the known keys (`id`, `parent`, `created`, `priority`, `status`, `branch`, `depends-on`, `locked-by`) and the custom fields that reach `frontmatter_fields`. A rule that reads a value must read it the same way for both, or one file parses two ways.

## Assertions

See `assertions/` for what must be true.
