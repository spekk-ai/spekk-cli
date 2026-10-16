---
id: show-order-matches-list
parent: show-command
created: 2026-10-15T00:00:00Z
priority: 1
status: done
---

# `spekk show` Output Order Is Deterministic and Matches `spekk list`

## Description

The explorer data that `spekk show` writes must list specs and assertions in one fixed order. Before this fix, cross-branch mode added the specs and assertions that exist only on other branches by iterating a Go map, so their order changed from run to run. The default mode was stable but used directory order, not the order of `spekk list`.

**Tests:** internal/show/show_test.go (`TestShowOrderMatchesList`)

## Success Criteria

- The `specs` and `assertions` arrays in the explorer data are sorted by priority (ascending), then by ID (alphabetical), which is the order `spekk list` uses. When two items have the same priority and ID, the file path decides, so the order is total.
- The same ordering rule applies with and without `--cross-branch`, and in `--watch` renders. Cross-branch mode adds the items that exist only on other branches; the rule sorts them together with the local items, not at the end in map order. An item that could not be parsed has no priority (0) and sorts after every item that has one.
- Two runs over the same specs (and, in cross-branch mode, the same branches) give byte-identical explorer data.
