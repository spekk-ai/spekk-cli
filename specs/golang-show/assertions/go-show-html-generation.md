---
id: go-show-html-generation
parent: golang-show
created: 2026-04-05T12:22:00Z
priority: 2
status: not_started
depends-on: go-parser-json-matches-node
branch: feature/golang-migration
---

# Go show command generates spec explorer HTML

The `spekk show` command in Go generates a self-contained HTML file with the spec explorer web interface and opens it in the default browser.

## Success Criteria

- Parses specs using Go parser
- Generates single HTML file at `.spekk/show/index.html` (creates directory if needed)
- HTML contains embedded CSS and JS (no external dependencies)
- Spec tree displayed with collapsible dropdowns per spec group
- Assertions appear as sub-items under their parent spec
- Status icons display correctly for all status values
- Clicking a spec or assertion shows detail panel with full content
- Search bar filters the spec tree
- Metro map visualization shows branch-based dependency graph
- Opens generated HTML in default browser (platform-aware: `open` on macOS, `xdg-open` on Linux)
