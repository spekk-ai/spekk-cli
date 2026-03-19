---
id: validate-testids
phase: post-build
tags:
  - frontend
  - testing
---

# Validate Test IDs

## Preconditions
- files-changed: "**/*.{tsx,jsx}"

## LLM Judgment
Skip if the only JSX/TSX changes are:
- Styling-only changes (CSS classes, inline styles, Tailwind utilities)
- Changes exclusively in test files (`**/*.test.{tsx,jsx}`, `**/*.spec.{tsx,jsx}`)
- Changes exclusively in storybook files (`**/*.stories.{tsx,jsx}`)

If any changed files contain interactive elements (buttons, inputs, links, forms) or data-display components that a QA tester would need to target, proceed with the gate.

## Workflow
1. Get the list of changed `.tsx` and `.jsx` files on this branch: `git diff origin/main...HEAD --name-only | grep -E '\.(tsx|jsx)$'`
2. For each changed file, read the file content
3. Scan for interactive and targetable elements that are missing `data-testid` attributes:
   - `<button>` without `data-testid`
   - `<input>` without `data-testid`
   - `<a>` (links) without `data-testid`
   - `<form>` without `data-testid`
   - `<select>` without `data-testid`
   - `<textarea>` without `data-testid`
   - Custom components that render interactive elements
4. Report each missing `data-testid` with:
   - File path and line number
   - The element type
   - A suggested `data-testid` value following the naming convention: `{page}-{component}-{element}`

## On Failure
- severity: warning
- action: report
