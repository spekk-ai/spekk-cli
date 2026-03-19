---
id: test-plan
phase: pre-merge
tags:
  - testing
  - qa
---

# Test Plan

## Preconditions
- command-succeeds: "gh pr view"
- files-changed: "**/{pages,components,templates}/**"

## LLM Judgment
Skip if the PR is:
- Documentation-only (only `.md`, `.txt`, or comment changes)
- Trivial changes (version bumps, dependency updates, config-only)
- Changes only to test files themselves (no production code affected)

If the PR modifies user-facing pages, components, or templates, proceed with the gate.

## Workflow
1. Get the PR description: `gh pr view --json title,body,labels`
2. Get the list of changed files: `git diff origin/main...HEAD --name-only`
3. Read the changed page/component/template files to understand what functionality changed
4. Generate a non-technical QA testing plan that covers:
   - **What changed**: Brief summary of the user-facing changes
   - **Test scenarios**: Step-by-step manual test cases a QA tester can follow
   - **Edge cases**: Boundary conditions, error states, empty states to verify
   - **Device/browser considerations**: If changes affect responsive layout or specific platforms
   - **Regression areas**: Existing functionality that could be affected by these changes
5. Format the test plan as a clear checklist that a non-technical QA tester can follow
6. Report the test plan as the gate output

## On Failure
- severity: warning
- action: report
