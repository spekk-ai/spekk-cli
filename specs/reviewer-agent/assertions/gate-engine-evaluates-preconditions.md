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
- Evaluates each precondition type:
  - `files-changed`: runs `git diff --name-only` and matches against glob pattern
  - `dir-exists`: `fs.existsSync()` on the path
  - `file-exists`: `fs.existsSync()` on the path
  - `file-not-exists`: negated `fs.existsSync()`
  - `branch-matches`: regex test against current branch name from `git branch --show-current`
  - `has-dependency`: checks `dependencies` and `devDependencies` in package.json
  - `command-succeeds`: runs command and checks exit code is 0
- ALL preconditions must pass for a gate to proceed (AND logic)
- Returns per-gate result: `{ id, status: 'pass'|'skip', reason: 'why it was skipped' }`
- Handles DAG dependencies: if gate A depends on gate B and B is skipped, A is auto-skipped with reason "dependency skipped: B"
- Topological sort ensures gates are evaluated in dependency order
- Circular dependency detection with clear error message
- Tests exist at `src/reviewer/__tests__/gate-engine.test.js`
