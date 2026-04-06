/**
 * Thin shim that delegates all parsing to the Go binary (bin/spekk-go).
 * No Node.js parser logic remains — the Go binary is the sole parser.
 */
import path from 'path';
import { fileURLToPath } from 'url';
import { execSync, spawnSync } from 'child_process';
import { existsSync } from 'fs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Get the spekk-cli installation directory
export function getSpekkInstallationDirectory() {
  return path.join(__dirname, '../..');
}

/**
 * Locate the Go binary. Throws if not found.
 */
function getGoBinary() {
  const installDir = getSpekkInstallationDirectory();
  const candidates = [
    path.join(installDir, 'bin', 'spekk-go'),
    path.join(installDir, 'spekk-go'),
  ];

  for (const candidate of candidates) {
    if (existsSync(candidate)) {
      return candidate;
    }
  }

  throw new Error(
    'Go binary not found. Run "go build -o bin/spekk-go ./cmd/spekk/" to build it.'
  );
}

/**
 * Call the Go binary with --raw and return parsed { specs, assertions, observations }.
 */
export function parseAllSpecs(specsDirectory = null) {
  const goBinary = getGoBinary();
  const args = ['next', '--raw'];

  // If a specific specs directory is provided, pass it to the Go binary
  // so it doesn't rely on git root detection.
  const effectiveSpecsDir = specsDirectory || path.join(process.cwd(), 'specs');
  args.push('--specs-dir', effectiveSpecsDir);

  const result = spawnSync(goBinary, args, {
    stdio: ['pipe', 'pipe', 'pipe'],
    encoding: 'utf8',
  });

  if (result.error) {
    throw new Error(`Failed to execute Go binary: ${result.error.message}`);
  }

  if (result.status !== 0) {
    const stderr = result.stderr ? result.stderr.trim() : '';
    const stdout = result.stdout ? result.stdout.trim() : '';
    // The Go binary outputs JSON errors to stdout
    if (stdout) {
      try {
        const parsed = JSON.parse(stdout);
        if (parsed.error) {
          throw new Error(parsed.message);
        }
      } catch {
        // Not valid JSON or no error field — fall through
      }
    }
    throw new Error(stderr || stdout || 'Go binary exited with non-zero status');
  }

  const output = result.stdout.trim();
  if (!output) {
    return { specs: [], assertions: [], observations: [] };
  }

  const parsed = JSON.parse(output);

  // The --raw output has { specs, assertions, observations }
  return {
    specs: parsed.specs || [],
    assertions: parsed.assertions || [],
    observations: parsed.observations || [],
  };
}

/**
 * Get current git branch.
 */
function getCurrentGitBranch() {
  try {
    const branch = execSync('git branch --show-current', {
      encoding: 'utf8',
      stdio: ['pipe', 'pipe', 'ignore'],
    }).trim();
    return branch || 'main';
  } catch {
    return 'main';
  }
}

/**
 * Check if a lock is stale (>2 hours old).
 */
function isLockStale(lockedBy) {
  if (!lockedBy) return true;
  const parts = lockedBy.split('-');
  const timestamp = parseInt(parts[parts.length - 1]);
  if (isNaN(timestamp)) return true;
  const currentTime = Math.floor(Date.now() / 1000);
  return (currentTime - timestamp) > 7200;
}

/**
 * Find next priority assertion from parsed data.
 * Delegates to the Go binary for the full algorithm, but this function
 * is kept for downstream consumers that call it directly with pre-parsed data.
 */
export function findNextAssertion(assertions, specs = [], options = {}) {
  if (options.assertion) {
    const target = assertions.find(a => a.id === options.assertion);
    if (!target) {
      return { error: true, message: `Assertion '${options.assertion}' not found` };
    }
    return target;
  }

  let incomplete = assertions.filter(a => {
    if (['done', 'draft'].includes(a.status)) return false;
    const parentSpec = specs.find(s => s.id === a.parent);
    if (parentSpec?.status === 'draft') return false;
    return true;
  });

  if (!options.allBranches) {
    const currentBranch = options.currentBranch || getCurrentGitBranch();
    incomplete = incomplete.filter(a => !a.branch || a.branch === currentBranch);
  }

  if (options.spec) {
    if (!specs.some(s => s.id === options.spec)) {
      return { error: true, message: `Spec '${options.spec}' not found` };
    }
    incomplete = incomplete.filter(a => a.parent === options.spec);
  }

  incomplete = incomplete.filter(a => {
    if (!a.dependsOn) return true;
    const dep = assertions.find(d => d.id === a.dependsOn);
    return dep && dep.status === 'done';
  });

  incomplete = incomplete.filter(a => {
    if (a.status !== 'in_progress') return true;
    if (!a.lockedBy) return true;
    return isLockStale(a.lockedBy);
  });

  if (incomplete.length === 0) return null;

  incomplete.sort((a, b) => {
    if (a.priority !== b.priority) return a.priority - b.priority;
    if (a.created !== b.created) return a.created.localeCompare(b.created);
    return a.id.localeCompare(b.id);
  });

  return incomplete[0];
}

/**
 * Run the parser CLI — outputs JSON to stdout.
 * This is the programmatic entry point used by cli.js.
 */
export function run(options = {}) {
  try {
    const { specs, assertions, observations } = parseAllSpecs(options.specsDirectory);

    if (specs.length === 0 && assertions.length === 0) {
      console.log(JSON.stringify({
        status: 'empty',
        message: 'No specifications found in specs/ directory'
      }, null, 2));
      return;
    }

    if (options.all) {
      const specsWithAssertions = specs.map(spec => ({
        id: spec.id,
        title: spec.title,
        status: spec.status,
        priority: spec.priority,
        file: spec.file,
        assertions: assertions
          .filter(a => a.parent === spec.id)
          .map(a => ({ id: a.id, title: a.title, status: a.status, priority: a.priority, file: a.file }))
          .sort((a, b) => a.priority !== b.priority ? a.priority - b.priority : a.id.localeCompare(b.id))
      })).sort((a, b) => a.priority !== b.priority ? a.priority - b.priority : a.id.localeCompare(b.id));

      console.log(JSON.stringify({
        type: 'hierarchy',
        specs: specsWithAssertions,
        observations
      }, null, 2));
      return;
    }

    const next = findNextAssertion(assertions, specs, {
      spec: options.spec,
      assertion: options.assertion,
      allBranches: options.allBranches,
      currentBranch: options.currentBranch,
    });

    if (next?.error) {
      console.log(JSON.stringify({ type: 'error', message: next.message }, null, 2));
      process.exit(1);
      return;
    }

    if (!next) {
      console.log(JSON.stringify({
        type: 'complete',
        status: 'complete',
        message: 'All specifications are complete'
      }, null, 2));
      return;
    }

    const parentSpec = specs.find(s => s.id === next.parent);
    console.log(JSON.stringify({
      type: 'assertion',
      id: next.id,
      parent: next.parent,
      file: next.file,
      priority: next.priority,
      status: next.status,
      branch: next.branch,
      created: next.created,
      dependsOn: next.dependsOn,
      lockedBy: next.lockedBy,
      title: next.title,
      content: next.content,
      spec: parentSpec ? { id: parentSpec.id, file: parentSpec.file, title: parentSpec.title } : null,
    }, null, 2));
  } catch (error) {
    console.log(JSON.stringify({ error: true, message: error.message }, null, 2));
    process.exit(1);
  }
}
