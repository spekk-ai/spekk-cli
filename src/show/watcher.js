import { watch } from 'node:fs';

/**
 * Watches a specs directory recursively for .md file changes.
 * Calls onChange() debounced on changes.
 *
 * @param {string} specsDir - Path to the specs directory to watch
 * @param {Function} onChange - Callback invoked when .md files change
 * @returns {Function} Cleanup function that stops the watcher
 */
export function watchSpecs(specsDir, onChange) {
  let debounceTimer = null;
  let stopped = false;

  const watcher = watch(specsDir, { recursive: true }, (eventType, filename) => {
    if (stopped) return;

    // Only react to .md file changes
    if (!filename || !filename.endsWith('.md')) return;

    // Debounce: reset timer on each qualifying event
    if (debounceTimer) {
      clearTimeout(debounceTimer);
    }

    debounceTimer = setTimeout(() => {
      if (!stopped) {
        onChange();
      }
    }, 300);
  });

  // Return cleanup function
  return function cleanup() {
    stopped = true;
    if (debounceTimer) {
      clearTimeout(debounceTimer);
      debounceTimer = null;
    }
    watcher.close();
  };
}
