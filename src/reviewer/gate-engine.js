import fs from 'fs';
import path from 'path';
import { execSync } from 'child_process';
import { minimatch } from 'minimatch';

/**
 * Get the current git branch name.
 */
function getCurrentBranch() {
  try {
    return execSync('git branch --show-current', {
      encoding: 'utf8',
      stdio: ['pipe', 'pipe', 'ignore'],
    }).trim();
  } catch {
    return null;
  }
}

/**
 * Get files changed on the current branch vs base.
 * Uses origin/main...HEAD to capture all changes on the branch.
 */
function getChangedFiles(base) {
  try {
    const output = execSync(`git diff ${base}...HEAD --name-only`, {
      encoding: 'utf8',
      stdio: ['pipe', 'pipe', 'ignore'],
    }).trim();
    return output ? output.split('\n') : [];
  } catch {
    return [];
  }
}

/**
 * Detect the base branch for diff comparison.
 * Tries origin/main first, then origin/master, then main, then master.
 */
function detectBaseBranch() {
  const candidates = ['origin/main', 'origin/master', 'main', 'master'];
  for (const candidate of candidates) {
    try {
      execSync(`git rev-parse --verify ${candidate}`, {
        encoding: 'utf8',
        stdio: ['pipe', 'pipe', 'ignore'],
      });
      return candidate;
    } catch {
      continue;
    }
  }
  return 'origin/main';
}

/**
 * Read package.json dependencies from cwd.
 */
function readPackageDeps(cwd) {
  const pkgPath = path.join(cwd, 'package.json');
  if (!fs.existsSync(pkgPath)) return {};

  try {
    const pkg = JSON.parse(fs.readFileSync(pkgPath, 'utf8'));
    return {
      ...pkg.dependencies,
      ...pkg.devDependencies,
    };
  } catch {
    return {};
  }
}

/**
 * Evaluate a single precondition check.
 * Returns { pass: boolean, reason: string }
 */
function evaluateCheck(check, context) {
  switch (check.type) {
    case 'dir-exists': {
      const fullPath = path.resolve(context.cwd, check.path);
      const exists = fs.existsSync(fullPath) && fs.statSync(fullPath).isDirectory();
      return exists
        ? { pass: true }
        : { pass: false, reason: `directory not found: ${check.path}` };
    }

    case 'file-exists': {
      const fullPath = path.resolve(context.cwd, check.path);
      const exists = fs.existsSync(fullPath);
      return exists
        ? { pass: true }
        : { pass: false, reason: `file not found: ${check.path}` };
    }

    case 'file-not-exists': {
      const fullPath = path.resolve(context.cwd, check.path);
      const exists = fs.existsSync(fullPath);
      return exists
        ? { pass: false, reason: `file exists: ${check.path}` }
        : { pass: true };
    }

    case 'has-dependency': {
      const hasDep = check.package in context.dependencies;
      return hasDep
        ? { pass: true }
        : { pass: false, reason: `dependency not found: ${check.package}` };
    }

    case 'branch-matches': {
      if (!context.branch) {
        return { pass: false, reason: 'not in a git repository' };
      }
      const regex = new RegExp('^' + check.pattern.replace(/\*/g, '.*') + '$');
      const matches = regex.test(context.branch);
      return matches
        ? { pass: true }
        : { pass: false, reason: `branch "${context.branch}" does not match "${check.pattern}"` };
    }

    case 'command-succeeds': {
      try {
        execSync(check.command, {
          encoding: 'utf8',
          stdio: ['pipe', 'pipe', 'ignore'],
          cwd: context.cwd,
        });
        return { pass: true };
      } catch {
        return { pass: false, reason: `command failed: ${check.command}` };
      }
    }

    case 'files-changed': {
      const matched = context.changedFiles.filter(f => minimatch(f, check.pattern));
      return matched.length > 0
        ? { pass: true }
        : { pass: false, reason: `no ${check.pattern} files changed on branch` };
    }

    default:
      return { pass: false, reason: `unknown check type: ${check.type}` };
  }
}

/**
 * Detect circular dependencies in gates.
 * Throws with a clear error message if a cycle is found.
 */
function detectCircularDependencies(gates) {
  const gateMap = new Map(gates.map(g => [g.id, g]));

  for (const gate of gates) {
    if (!gate.dependsOn) continue;

    const visited = new Set();
    const chain = [];
    let current = gate;

    while (current && current.dependsOn) {
      if (visited.has(current.id)) {
        chain.push(current.id);
        const cycleStart = chain.indexOf(current.id);
        const cycle = chain.slice(cycleStart).join(' → ');
        throw new Error(`Circular dependency detected in gates: ${cycle}`);
      }

      visited.add(current.id);
      chain.push(current.id);
      current = gateMap.get(current.dependsOn);
    }
  }
}

/**
 * Topological sort of gates by depends-on relationships.
 */
function topologicalSort(gates) {
  const gateMap = new Map(gates.map(g => [g.id, g]));
  const visited = new Set();
  const sorted = [];

  function visit(gate) {
    if (visited.has(gate.id)) return;
    visited.add(gate.id);

    if (gate.dependsOn && gateMap.has(gate.dependsOn)) {
      visit(gateMap.get(gate.dependsOn));
    }

    sorted.push(gate);
  }

  for (const gate of gates) {
    visit(gate);
  }

  return sorted;
}

/**
 * Evaluate all gates and return per-gate results.
 *
 * Options:
 *   - cwd: working directory (default: process.cwd())
 *   - base: base branch for files-changed comparison
 *   - branch: override current branch (for testing)
 *   - changedFiles: override changed files list (for testing)
 *   - dependencies: override package dependencies (for testing)
 *
 * Returns array of { id, status: 'pass'|'skip', reason }
 */
export function evaluateGates(gates, options = {}) {
  const cwd = options.cwd || process.cwd();
  const branch = options.branch !== undefined ? options.branch : getCurrentBranch();
  const base = options.base || detectBaseBranch();
  const changedFiles = options.changedFiles !== undefined ? options.changedFiles : getChangedFiles(base);
  const dependencies = options.dependencies !== undefined ? options.dependencies : readPackageDeps(cwd);

  // Detect circular dependencies
  detectCircularDependencies(gates);

  // Topological sort
  const sorted = topologicalSort(gates);

  const context = { cwd, branch, changedFiles, dependencies };
  const results = new Map();

  for (const gate of sorted) {
    // Check DAG dependency
    if (gate.dependsOn) {
      const depResult = results.get(gate.dependsOn);
      if (depResult && depResult.status === 'skip') {
        results.set(gate.id, {
          id: gate.id,
          status: 'skip',
          reason: `dependency skipped: ${gate.dependsOn}`,
        });
        continue;
      }
    }

    // Evaluate all preconditions (AND logic)
    let skipReason = null;

    for (const check of gate.preconditions) {
      const result = evaluateCheck(check, context);
      if (!result.pass) {
        skipReason = result.reason;
        break;
      }
    }

    if (skipReason) {
      results.set(gate.id, { id: gate.id, status: 'skip', reason: skipReason });
    } else {
      results.set(gate.id, { id: gate.id, status: 'pass', reason: null });
    }
  }

  return Array.from(results.values());
}
