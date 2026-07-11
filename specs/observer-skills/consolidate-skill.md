---
id: consolidate
description: Reviews all raw observation files across every observations/*/ subdirectory and maintains a curated digest of at most 5 open items ranked by severity
created: 2026-07-11T14:00:00Z
priority: 2
---

# Consolidate

Maintains a lean, curated view of what the observer has found. Raw observations are written cheaply by each skill or the default loop — this skill is the only thing that reads all of them, prunes noise, and keeps `observations/DIGEST.md` up to date. It is the single output users are expected to read.

## Triggers

- "consolidate"
- "prune observations"
- "update digest"
- "clean up observations"
- "summarize observations"
- One-shot consolidation passes

## Scoped File-Write Exception

This skill is **permitted to move and rewrite files anywhere under `observations/`** — that is its job. All other files (code under `internal/`, specs under `specs/`, configuration) remain untouched. The per-mode output rule applies everywhere except within `observations/`.

## Workflow

**CONTRACT: You must read every open observation file before making any pruning decision. Concluding "nothing to prune" without completing the full review is a contract violation.**

1. **Discover all observation files.**
   - Enumerate every file matching `observations/*/**.md` (excluding `observations/archive/`).
   - Also enumerate `observations/DIGEST.md` if it exists (read it for context; do not treat it as a raw observation).
   - Record the full list before opening any file.

2. **Read every open observation file.**
   - Open and read each file found in step 1 in full.
   - Do not skip, sample, or summarise before reading — partial reads constitute a contract violation.
   - Extract: `id`, `created`, `skill`, `type`, `severity`, `affected_specs`, `affected_files`, and the body's Issue Description.

3. **Identify duplicates.**
   - Two observations are duplicates when they share the same `type` and substantially the same `affected_files` set (exact-path overlap ≥ 50 %).
   - Group duplicates; keep the newest file as the canonical record (latest `created` timestamp wins).
   - The older duplicate(s) are candidates for archiving.

4. **Identify resolved or stale observations.**
   - An observation is **resolved** when all files listed in `affected_files` no longer exist in the working tree, or the affected assertions now have `status: done`.
   - An observation is **stale** when it is more than 30 days old (compare `created` to today) and has not been referenced in any recent commit touching the affected paths.
   - Mark these candidates for archiving.

5. **Archive candidates.**
   - Move each candidate file to `observations/archive/` preserving its filename.
   - Create `observations/archive/` if it does not exist.
   - Never delete originals — move only.

6. **Identify the top 5 open items.**
   - From the remaining (non-archived) observation files, rank by severity: `high` > `medium` > `low`.
   - Break ties by `created` descending (newest first).
   - Select at most 5.

7. **Rewrite `observations/DIGEST.md`.**
   - **Mandatory on every run** — even when nothing changed, the digest is rewritten so its timestamp reflects the latest consolidation pass.
   - Format (see Output Format below).

8. **Print a console summary** (a few lines only):
   - Total observations read, number archived, number remaining open.
   - Path to the written digest.

9. **Exit** — do not start the default monitoring loop.

## Output Format

### `observations/DIGEST.md`

```markdown
---
updated: <ISO-8601 timestamp of this consolidation run>
open_count: <N>   # number of items in the digest (0–5)
---

# Observation Digest

_Consolidated <date>. At most 5 open items, ranked by severity._

## 1. <title from observation's Issue Description — one line> (severity: high)

- **File:** [<observation-id>](./path/to/observation.md)
- **Skill:** <skill name>
- **Affected:** <comma-separated affected_files, truncated to 3 with "… and N more" if longer>
- **Created:** <created date>

## 2. …

<!-- repeated for each ranked item; omit section if open_count is 0 -->

---
_Run `spekk observer consolidate` to refresh._
```

If no open observations remain after archiving, the digest body reads:

```markdown
_No open observations. Run `spekk observer` to generate new ones._
```

## Validation

- Every file matching `observations/*/**.md` (excluding `observations/archive/`) was opened and read before any archiving decision — the count of files read equals the count discovered in step 1
- Duplicate observations (same `type`, ≥ 50 % `affected_files` overlap) are archived rather than left as parallel entries
- Resolved and stale observations are moved to `observations/archive/` with filenames preserved; originals are not deleted
- `observations/DIGEST.md` exists and was rewritten during this run (check `updated` timestamp)
- Digest contains at most 5 items
- Items are ranked high → medium → low; ties broken by newest `created` first
- Each digest item links to its underlying raw observation file with a relative path
- No files outside `observations/` were written or modified
- Console output is a few summary lines; it does not dump observation bodies
- Skill exits after one pass; it does not enter the monitoring loop

## Examples

### Example 1: First consolidation on a noisy observation directory

```
$ spekk observer consolidate
> Reading observations/ ... 14 files across 3 skills
> Duplicates found: 2 pairs → 2 files queued for archive
> Stale (>30 days): 3 files → queued for archive
> Archived 5 files to observations/archive/
> Open items remaining: 9 → top 5 written to digest
> Digest: observations/DIGEST.md
```

### Example 2: Clean repository, nothing to prune

```
$ spekk observer consolidate
> Reading observations/ ... 3 files across 1 skill
> Duplicates found: 0
> Stale: 0
> Archived 0 files
> Open items remaining: 3 → top 3 written to digest
> Digest: observations/DIGEST.md
```

### Example 3: All observations resolved

```
$ spekk observer consolidate
> Reading observations/ ... 6 files across 2 skills
> Resolved: 6 files → queued for archive
> Archived 6 files to observations/archive/
> Open items remaining: 0
> Digest: observations/DIGEST.md (empty)
```
