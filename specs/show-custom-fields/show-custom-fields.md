---
id: show-custom-fields
created: 2026-11-19T14:55:40Z
priority: 2
---

# `spekk show` Displays Custom Frontmatter Fields

## Overview

Users can add their own keys to the frontmatter of a spec or assertion (for example `workflows: [checkout, signup]` or `owner: platform`). `spekk validate` accepts them, and `spekk index` stores them in `frontmatter_fields` (`custom-frontmatter-fields-indexed`). `spekk show` drops them. This spec makes the Spec Explorer and `spekk show --json` carry those keys and display them as additional properties in the detail view.

A custom field is any top-level key outside the known keys of that file type. Values are split by the same rule the index uses, so the index, the JSON, and the UI agree on what a field holds.

Filtering the explorer by a custom field, and showing custom fields on tree cards, are out of scope.

## Assertions

- `show-data-carries-custom-fields`: specs and assertions in the show data carry `customFields`, split by the index rule, and `--json` matches the HTML data.
- `detail-shows-custom-properties`: the detail view lists custom fields under "Properties", sorted, escaped, and omitted when there are none.
- `show-output-stable-with-custom-fields`: two runs over the same tree with custom fields give byte-identical output.
