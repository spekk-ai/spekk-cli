---
id: spec-format-documentation
parent: coordinator-skill
created: 2026-02-25T00:00:00Z
priority: 3
status: not_started
---

# Spec Format Documentation Updated with New Fields

Documentation specs that define YAML frontmatter format are updated to include `depends-on` and `branch` fields.

## Success Criteria

### Update Coach Agent Prompt

File: `specs/coach-agent/coach-agent.prompt.md`

**Add to "Format Validation" section:**

```markdown
Every assertion file must have:
\`\`\`yaml
---
id: kebab-case-id
parent: parent-spec-id
created: 2026-01-20T17:00:00Z
priority: 1
status: not_started             # not_started | in_progress | done | failed | draft
depends-on: parent-assertion-id # Single parent dependency (optional)
branch: feature/name            # Git branch assignment (optional, defaults to main)
---
\`\`\`

**New fields:**
- \`depends-on\`: Single assertion ID that must be completed first (omit if no dependency)
- \`branch\`: Git branch where this assertion lives (omit to default to main)
```

### Update Nested Spec Organization

File: `specs/nested-spec-organization/assertions/group-specs-have-metadata.md`

**Update "Success Criteria":**

```markdown
- [ ] Group specs include standard fields: id, created, priority, type
- [ ] Assertion specs include: id, parent, created, priority, status
- [ ] Optional assertion fields: depends-on (single parent), branch (git branch)
- [ ] Field \`depends-on\` is validated (must reference existing assertion)
- [ ] Field \`branch\` is validated (must be valid git branch name)
```

### Update Builder Prompt (if exists)

If there's a `specs/builder-agent/builder-agent.prompt.md`, update it to mention:

```markdown
## Dependency-Aware Building

Before starting work on an assertion:
1. Check if \`depends-on\` field exists
2. If yes, verify the referenced assertion has \`status: done\`
3. If dependency is not done, skip this assertion (blocked)

The parser handles this automatically - \`spekk next\` only returns ready assertions.
```

### Add to README (if relevant)

If `README.md` documents frontmatter format, add:

```markdown
### Optional Fields

**\`depends-on\`**: Single assertion ID that must be completed before this one.
- Creates linear dependency chains
- Parser validates reference exists
- \`spekk next\` respects dependencies

**\`branch\`**: Git branch where this assertion lives.
- Defaults to \`main\` if omitted
- \`spekk next\` filters to current branch by default
- Use \`spekk next --all-branches\` to see all assertions

\`\`\`yaml
---
id: feature-x
depends-on: prerequisite-y  # Must complete prerequisite-y first
branch: feature/my-feature   # Lives on feature branch
---
\`\`\`
```

## Implementation

### Files to Update

1. `specs/coach-agent/coach-agent.prompt.md` - Coach format validation section
2. `specs/nested-spec-organization/assertions/group-specs-have-metadata.md` - Metadata spec
3. `specs/builder-agent/builder-agent.prompt.md` - Builder dependency awareness (if exists)
4. `README.md` - Frontmatter documentation (if relevant)

### Documentation Style

- **Concise** - Don't over-explain, give clear examples
- **Actionable** - Show WHAT the fields do, not just that they exist
- **Consistent** - Use same terminology across all docs

## Validation

- Coach prompt documents new fields
- Nested spec organization updated
- Builder prompt mentions dependency awareness (if exists)
- README includes examples (if relevant)
- All examples use correct format
- No conflicting documentation

**Tests:** Manual review of updated documentation
