---
id: gate-engine-evaluates-preconditions
parent: reviewer-agent
created: 2026-03-19T18:09:00Z
priority: 1
status: not_started
depends-on: gate-loader-parses-gate-files
branch: feature/code-quality-qa
---

# Gate engine evaluates deterministic preconditions

A gate engine module (`src/reviewer/gate-engine.js`) evaluates the deterministic preconditions from loaded gates and produces a pass/skip report for each gate.

## Success Criteria

- Module exists at `src/reviewer/gate-engine.js`
- Evaluates two categories of preconditions:

### Project capability checks (static, is the tooling/directory present?)
  - `dir-exists`: `fs.existsSync()` on the path
  - `file-exists`: `fs.existsSync()` on the path
  - `file-not-exists`: negated `fs.existsSync()`
  - `has-dependency`: checks `dependencies` and `devDependencies` in package.json
  - `branch-matches`: regex test against current branch name from `git branch --show-current`
  - `command-succeeds`: runs command and checks exit code is 0

### Branch impact checks (dynamic, what did this branch actually change?)
  - `files-changed`: runs `git diff <base>...HEAD --name-only` and matches against glob pattern
  - Base branch detection: uses `git merge-base` to find the fork point from main/master, or accepts explicit `--base` flag
  - Compares against `origin/main...HEAD` to capture both local and remote changes on the branch
  - Example: `files-changed: "**/*.tsx"` only passes if `.tsx` files were actually modified on this branch vs main

### Evaluation logic
- ALL preconditions must pass for a gate to proceed (AND logic)
- This means a gate can combine both: `dir-exists: ios/` AND `files-changed: "ios/**"` — project has mobile AND this branch touched it
- Returns per-gate result: `{ id, status: 'pass'|'skip', reason: 'why it was skipped' }`
- Reasons are specific: "skipped: no .tsx files changed on branch" not just "precondition failed"
- Handles DAG dependencies: if gate A depends on gate B and B is skipped, A is auto-skipped with reason "dependency skipped: B"
- Topological sort ensures gates are evaluated in dependency order
- Circular dependency detection with clear error message
- Tests exist at `src/reviewer/__tests__/gate-engine.test.js`
