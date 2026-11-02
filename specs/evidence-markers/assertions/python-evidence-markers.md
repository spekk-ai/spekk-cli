---
id: python-evidence-markers
parent: evidence-markers
created: 2026-11-02T21:30:00Z
priority: 1
status: not_started
---

# Spekk Finds Each pytest Evidence Marker and the Test It Marks

## Description

A Python marker is a `@spekk("<id>")` or `@spekk("<id>", evidence="<level>")` decorator on a pytest test function, in a file whose name matches `test_*.py` or `*_test.py`. The function can be at module level or a method of a class whose name starts with `Test`.

## Success criteria

- After `spekk index`, the `evidence` table has one row for each Python marker: `language = 'python'`, `test` is `Class.function` for a method and `function` at module level, and `file` is the project-relative path.
- Single and double quotes both work, and `evidence` can be a keyword argument in any position after the ID.
- A function with other decorators above or below `@spekk` keeps its marker. Two `@spekk` decorators on one function give two rows.
- A marker with no `evidence` has level `example`.
- A `@spekk` decorator on a function that is not a test function (its name does not start with `test`) adds no row, and `spekk validate` prints a warning with the file and line.
- **Tests:** a module-level test; a method of a `Test` class; a stacked decorator; a marker with no `evidence`.
