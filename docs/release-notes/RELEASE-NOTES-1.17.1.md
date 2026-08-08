# Spekk CLI 1.17.1 — Announcements Drop the Evidence Path List

The first observer announcements in production carried an `Evidence:` line with every `affected` path. On a finding that touches ten assertion files, that line is longer than the finding itself. It repeats what the PR already shows, and it pushes the pointer line out of view.

The line is gone, from both message shapes: the single-finding body, and each numbered section of a batch.

## Before

```
The reporting-exports spec header (line 10) states that no assertion has been
confirmed with the team. In reality, all 10 assertions carry a Confirmed section.

Evidence: specs/reporting-exports/assertions/a-report-can-be-saved-as-a-preset.md,
specs/reporting-exports/assertions/exports-honour-the-active-filters.md, ... (11 paths)

Proposed fix in PR: https://github.com/... — merge to accept, close to dismiss. Reply here to discuss.

⚠️ Severity: medium — meaningful drift; review when convenient.
```

## After

```
The reporting-exports spec header (line 10) states that no assertion has been
confirmed with the team. In reality, all 10 assertions carry a Confirmed section.

Proposed fix in PR: https://github.com/... — merge to accept, close to dismiss. Reply here to discuss.

⚠️ Severity: medium — meaningful drift; review when convenient.
```

## Evidence keeps its other two roles

Nothing about evidence collection changes:

- An observation with no `affected` path stays invalid. Selection removes it, so a finding with no evidence still never announces.
- The observation file and the PR body still carry the paths, where a reader sees them with their context.

The announcement points at the PR. A reader who wants the paths opens it.

## Upgrade

```bash
spekk update
```
