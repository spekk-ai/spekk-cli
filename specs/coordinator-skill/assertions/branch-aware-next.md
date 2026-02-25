---
id: branch-aware-next
parent: coordinator-skill
created: 2026-02-25T00:00:00Z
priority: 1
status: done
depends-on: parser-depends-on-support
---

# spekk next is Branch-Aware by Default

The `spekk next` command filters to current git branch and respects dependencies when selecting the next assertion.

## What Must Be True

### Branch Filtering
- [ ] `spekk next` detects the current git branch
- [ ] Only assertions matching the current branch are considered
- [ ] `--all-branches` flag disables branch filtering
- [ ] When not in a git repo, defaults to `branch='main'` with a warning

### Dependency Blocking
- [ ] Assertions with unmet dependencies are not returned
- [ ] An assertion is ready when `dependsOn` is null/omitted OR the referenced assertion has `status: done`
- [ ] Dependencies are checked before selection

### Selection Priority
- [ ] Assertions with `status: done` are excluded
- [ ] Remaining assertions are sorted by priority (lower number = higher priority)
- [ ] Ties are broken by creation timestamp (older first)
- [ ] The first assertion after sorting is returned

### User Feedback
- [ ] When no assertions are ready, a clear message explains why (all done, all blocked, or wrong branch)
- [ ] Blocked assertions are listed with their blocking dependencies
- [ ] Helpful suggestions are provided (e.g., "Try git checkout main")

### Output Format
- [ ] Returns JSON with assertion metadata
- [ ] Includes `id`, `parent`, `branch`, `status`, `dependsOn` fields
- [ ] Clear indication of dependency state (✅ done or ⏳ waiting)
