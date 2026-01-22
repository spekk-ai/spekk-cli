---
id: project-name-in-title
parent: simplified-display-formatting
created: 2026-01-22T22:55:00Z
priority: 2
status: done
---

# Include Project Name in Page Title

Browser tab title should show which project the spekk explorer is for when multiple projects are open.

## What Must Be True

Page title includes the current project's directory name to distinguish between different spekk explorers.

### Title Format

- **Current:** "Spec Explorer - Spekk"
- **New:** "Spec Explorer - {project-name}"

### Examples

- `~/thinknimble/spekk-cli/` → "Spec Explorer - spekk-cli"
- `~/thinknimble/spekk/` → "Spec Explorer - spekk"
- `~/my-project/` → "Spec Explorer - my-project"

### Implementation

- Derive project name from `process.cwd()` - use the last directory segment
- Update both `<title>` tag and page header `<h1>` to include project name
- Maintain "Spec Explorer" prefix for consistency

### HTML Changes

```html
<!-- Before -->
<title>Spec Explorer - Spekk</title>
<h1>Spec Tree</h1>

<!-- After -->  
<title>Spec Explorer - spekk-cli</title>
<h1>Spec Tree - spekk-cli</h1>
```

## Success Criteria

- ✅ Run `spekk show` in different project directories
- ✅ Browser tab shows unique title for each project
- ✅ Page header includes project name
- ✅ Can easily distinguish between multiple spekk explorer tabs