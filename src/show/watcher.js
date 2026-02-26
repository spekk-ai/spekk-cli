import { readdirSync, statSync } from 'node:fs';
import { join } from 'node:path';

/**
 * Recursively scan a directory and collect all .md files with their mtimeMs.
 * Returns a Map<relativePath, mtimeMs>.
 */
function scanMdFiles(specsDir) {
  const snapshot = new Map();
  try {
    const entries = readdirSync(specsDir, { recursive: true });
    for (const entry of entries) {
      const rel = typeof entry === 'string' ? entry : entry.toString();
      if (!rel.endsWith('.md')) continue;
      try {
        const fullPath = join(specsDir, rel);
        const stat = statSync(fullPath);
        snapshot.set(rel, stat.mtimeMs);
      } catch {
        // File may have been deleted between readdir and stat; skip it
      }
    }
  } catch {
    // Directory may not exist yet; return empty snapshot
  }
  return snapshot;
}

/**
 * Watches a specs directory for .md file changes using polling.
 * Polls every 500ms, comparing mtimes against a cached snapshot.
 * Calls onChange() when changes are detected (debounced per poll cycle).
 *
 * @param {string} specsDir - Path to the specs directory to watch
 * @param {Function} onChange - Callback invoked when .md files change
 * @returns {Function} Cleanup function that stops polling
 */
export function watchSpecs(specsDir, onChange) {
  let snapshot = scanMdFiles(specsDir);

  const intervalId = setInterval(() => {
    const current = scanMdFiles(specsDir);
    let dirty = false;

    // Check for modified or deleted files
    for (const [path, mtime] of snapshot) {
      if (!current.has(path)) {
        dirty = true;
        break;
      }
      if (current.get(path) !== mtime) {
        dirty = true;
        break;
      }
    }

    // Check for new files
    if (!dirty) {
      for (const path of current.keys()) {
        if (!snapshot.has(path)) {
          dirty = true;
          break;
        }
      }
    }

    if (dirty) {
      snapshot = current;
      onChange();
    }
  }, 500);

  return function cleanup() {
    clearInterval(intervalId);
  };
}
