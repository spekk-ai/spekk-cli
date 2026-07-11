---
id: branch-selection-ui
parent: show-cross-branch
created: 2026-06-15T12:00:00Z
priority: 3
status: done
branch: feat/show-cross-branch
depends-on: html-state-rendering
---

# A Branch Checkbox Dropdown Filters Contributions Client-Side, Persisted in localStorage

## Description

In cross-branch mode the explorer offers an interactive control to deselect
comparison branches without re-running `spekk show`. This is distinct from the
server-side `--branch-filter` glob (which decides, at classification time, which
branches are diffed at all): the dropdown narrows the *already-rendered* set
purely in the browser, instantly, and the user's selection is remembered across
reloads via `localStorage`.

## Success Criteria

- In cross-branch mode (and only then), the explorer renders a **checkbox
  dropdown / multi-select** listing every compared branch present in the data,
  each independently toggleable. The control is absent on the non-cross-branch
  path, which renders exactly as today.
- Deselecting a branch hides that branch's contributions **everywhere**, computed
  client-side from the per-branch contribution lists (no server round-trip):
  - per-item contribution lists in the detail panel show only selected branches;
  - each item's inline state badge and the spec-level summary badge **recompute**
    from only the selected branches (e.g. a spec whose only conflict came from a
    now-deselected branch loses its conflict badge);
  - a **local** item whose every contribution came from deselected branches
    reverts to its normal, non-cross-branch rendering;
  - a **foreign** item (no local file — see `cross-branch-data-model`) whose
    every contributing branch is deselected is **removed from the view** entirely
    (it has nothing to show without a contributing branch).
- The banner's list of compared branches reflects the active selection (deselected
  branches are visually distinguished or omitted), so the view is self-describing.
- The selection is persisted in `localStorage`, keyed per project, and restored on
  the next load — including the watch-mode auto-reload — so a refresh or a live
  reload keeps the user's chosen branches. (This is separate from the
  `sessionStorage` watch-state used to preserve expansion/scroll.)
- Defaults are sensible: with no stored selection, **all** branches are selected.
  A branch newly appearing in the data (e.g. created while watching) defaults to
  selected; a branch that disappears is dropped from the stored selection without
  error.
- Toggling is **instant** — re-deriving badges/rollups and re-rendering the tree
  (and the open detail panel) without reparsing or recontacting the server, since
  all per-branch contributions are already present in the page data.
- The control composes with `--branch-filter`: the server glob bounds which
  branches exist in the data; the dropdown can only select among those.
