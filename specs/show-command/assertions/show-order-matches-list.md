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
- The order is the same with and without `--cross-branch`, and in `--watch` renders. Items that exist only on other branches are sorted with the local items, not added at the end in map order.
- Two runs over the same specs (and, in cross-branch mode, the same branches) give byte-identical explorer data.
