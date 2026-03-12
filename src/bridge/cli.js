import fs from 'fs';
import path from 'path';
import { execSync } from 'child_process';
import WebSocket from 'ws';
import { parseAllSpecs } from '../parser/index.js';
import { parseFlags } from '../cli/parse-flags.js';

// ANSI color helpers
const green = (s) => `\x1b[32m${s}\x1b[0m`;
const red = (s) => `\x1b[31m${s}\x1b[0m`;
const cyan = (s) => `\x1b[36m${s}\x1b[0m`;
const dim = (s) => `\x1b[2m${s}\x1b[0m`;

const FLAG_DEFS = {
  server: { flags: ['--server', '-s'], type: 'string' },
  token: { flags: ['--token', '-t'], type: 'string' },
  verbose: { flags: ['--verbose', '-v'], type: 'boolean' },
};

/**
 * Read the API token from flag, env var, or ~/.spekk/token file.
 */
function getToken(flagToken) {
  if (flagToken) return flagToken;
  if (process.env.SPEKK_API_TOKEN) return process.env.SPEKK_API_TOKEN;

  const tokenPath = path.join(process.env.HOME || process.env.USERPROFILE, '.spekk', 'token');
  try {
    return fs.readFileSync(tokenPath, 'utf8').trim();
  } catch {
    return null;
  }
}

/**
 * Determine the server WebSocket URL.
 */
function getServerUrl(flagServer) {
  if (flagServer) return flagServer;
  if (process.env.SPEKK_SERVER_URL) return process.env.SPEKK_SERVER_URL;
  // Default to prod
  return 'wss://app.spekk.dev';
}

/**
 * Get repository info from git.
 */
function getRepoInfo() {
  try {
    const toplevel = execSync('git rev-parse --show-toplevel', {
      encoding: 'utf8',
      stdio: ['pipe', 'pipe', 'ignore'],
    }).trim();
    return {
      name: path.basename(toplevel),
      path: toplevel,
    };
  } catch {
    return {
      name: path.basename(process.cwd()),
      path: process.cwd(),
    };
  }
}

/**
 * Get current git branch and dirty files.
 */
function getGitStatus() {
  let branch = 'main';
  let dirtyFiles = [];

  try {
    branch = execSync('git branch --show-current', {
      encoding: 'utf8',
      stdio: ['pipe', 'pipe', 'ignore'],
    }).trim() || 'main';
  } catch {
    // ignore
  }

  try {
    const porcelain = execSync('git status --porcelain', {
      encoding: 'utf8',
      stdio: ['pipe', 'pipe', 'ignore'],
    }).trim();
    if (porcelain) {
      dirtyFiles = porcelain
        .split('\n')
        .map((line) => line.slice(3).trim())
        .filter(Boolean);
    }
  } catch {
    // ignore
  }

  return { branch, dirtyFiles };
}

/**
 * Main bridge runner.
 */
export async function run(args) {
  const flags = parseFlags(args, FLAG_DEFS);

  const token = getToken(flags.token);
  if (!token) {
    console.error(red('Error: No API token found.'));
    console.error('Set SPEKK_API_TOKEN env var, pass --token, or create ~/.spekk/token');
    process.exit(1);
  }

  const serverUrl = getServerUrl(flags.server);
  const repo = getRepoInfo();
  const verbose = flags.verbose;

  // Content directories to watch
  const WATCHED_DIRS = ['specs', 'docs', 'vision', 'feedback'];
  const watchedPaths = WATCHED_DIRS.map((dir) => ({
    name: dir,
    path: path.join(process.cwd(), dir),
  }));
  const specsDir = path.join(process.cwd(), 'specs');

  // Print banner
  console.log('');
  console.log(cyan('spekk bridge'));
  console.log(dim(`  repo:    ${repo.name}`));
  console.log(dim(`  server:  ${serverUrl}`));
  console.log(dim(`  watching: ${WATCHED_DIRS.join(', ')}`));
  console.log('');

  // State
  let ws = null;
  let stopWatcher = null;
  let gitPollInterval = null;
  let heartbeatInterval = null;
  let lastGitStatus = null;
  let selfWrittenPaths = new Set(); // paths we wrote ourselves (to ignore in watcher)
  let shouldReconnect = true;

  function parseSpecTree() {
    try {
      const { specs, assertions } = parseAllSpecs(specsDir);
      return { specs, assertions };
    } catch (err) {
      if (verbose) console.error(dim(`  parser error: ${err.message}`));
      return { specs: [], assertions: [] };
    }
  }

  function sendJson(data) {
    if (ws && ws.readyState === ws.OPEN) {
      ws.send(JSON.stringify(data));
    }
  }

  function sendSpecTree() {
    const { specs, assertions } = parseSpecTree();
    sendJson({ type: 'spec_tree', specs, assertions, repo: { name: repo.name, path: repo.path } });
  }

  function sendGitStatus() {
    const status = getGitStatus();
    // Only send if changed
    const statusKey = `${status.branch}:${status.dirtyFiles.join(',')}`;
    if (statusKey !== lastGitStatus) {
      lastGitStatus = statusKey;
      sendJson({ type: 'git_status', branch: status.branch, dirty_files: status.dirtyFiles });
      if (verbose) {
        console.log(cyan(`→ Branch: ${status.branch} (${status.dirtyFiles.length} dirty files)`));
      }
    }
  }

  // --- File watcher using fs.watch recursive ---

  function startFileWatcher() {
    let debounceTimer = null;
    const pendingChanges = new Map(); // filename -> event type
    const watchers = [];

    for (const { name, path: dirPath } of watchedPaths) {
      if (!fs.existsSync(dirPath)) {
        if (verbose) console.log(dim(`  ${name}/ directory not found, skipping`));
        continue;
      }

      const watcher = fs.watch(dirPath, { recursive: true }, (eventType, filename) => {
        if (!filename || !filename.endsWith('.md')) return;

        const relativePath = path.join(name, filename);

        // Skip self-triggered writes
        if (selfWrittenPaths.has(relativePath)) {
          selfWrittenPaths.delete(relativePath);
          return;
        }

        pendingChanges.set(relativePath, eventType);

        // Debounce: wait 300ms after last change before processing
        if (debounceTimer) clearTimeout(debounceTimer);
        debounceTimer = setTimeout(() => {
          processPendingChanges(new Map(pendingChanges));
          pendingChanges.clear();
        }, 300);
      });

      watcher.on('error', (err) => {
        if (verbose) console.error(dim(`  watcher error (${name}): ${err.message}`));
      });

      watchers.push(watcher);
      if (verbose) console.log(dim(`  watching ${name}/`));
    }

    if (watchers.length === 0) {
      if (verbose) console.log(dim('  no content directories found, skipping file watcher'));
      return null;
    }

    return () => {
      if (debounceTimer) clearTimeout(debounceTimer);
      watchers.forEach((w) => w.close());
    };
  }

  function processPendingChanges(changes) {
    let needsFullTree = false;

    for (const [relativePath, eventType] of changes) {
      const fullPath = path.join(process.cwd(), relativePath);

      if (eventType === 'rename') {
        // File added or deleted — need full tree re-parse
        needsFullTree = true;
        const exists = fs.existsSync(fullPath);
        console.log(exists ? green(`→ Added: ${relativePath}`) : red(`→ Removed: ${relativePath}`));
      } else {
        // File changed
        if (fs.existsSync(fullPath)) {
          try {
            const content = fs.readFileSync(fullPath, 'utf8');
            // Extract frontmatter for the message
            let frontmatter = {};
            if (content.trimStart().startsWith('---')) {
              try {
                const lines = content.split('\n');
                const endIdx = lines.indexOf('---', 1);
                if (endIdx > 0) {
                  const yamlLines = lines.slice(1, endIdx);
                  for (const line of yamlLines) {
                    const match = line.match(/^([^:]+):\s*(.*)$/);
                    if (match) {
                      frontmatter[match[1].trim()] = match[2].trim();
                    }
                  }
                }
              } catch {
                // ignore frontmatter parse errors for the change message
              }
            }
            sendJson({ type: 'file_changed', path: relativePath, content, frontmatter });
            console.log(green(`→ Synced: ${relativePath}`));
          } catch (err) {
            if (verbose) console.error(dim(`  read error: ${err.message}`));
          }
        } else {
          needsFullTree = true;
        }
      }
    }

    if (needsFullTree) {
      sendSpecTree();
    }
  }

  // --- WebSocket connection ---

  function connect() {
    const wsUrl = `${serverUrl}/ws/bridge/?token=${token}`;
    if (verbose) console.log(dim(`  connecting to ${serverUrl}/ws/bridge/...`));

    // Derive HTTP origin from WS URL for AllowedHostsOriginValidator
    const origin = wsUrl.replace(/^ws/, 'http').replace(/\/ws\/.*$/, '');
    ws = new WebSocket(wsUrl, { headers: { Origin: origin } });

    ws.on('open', () => {
      console.log(green('✓ Connected to spekk cloud'));

      // Send initial spec tree
      sendSpecTree();

      // Send initial git status
      lastGitStatus = null; // reset so it always sends
      sendGitStatus();

      // Start file watcher
      if (stopWatcher) stopWatcher();
      stopWatcher = startFileWatcher();

      // Start git status polling (every 5 seconds)
      if (gitPollInterval) clearInterval(gitPollInterval);
      gitPollInterval = setInterval(sendGitStatus, 5000);

      // Start heartbeat (every 30 seconds)
      if (heartbeatInterval) clearInterval(heartbeatInterval);
      heartbeatInterval = setInterval(() => {
        sendJson({ type: 'ping' });
      }, 30000);
    });

    ws.on('message', (data) => {
      let msg;
      try {
        msg = JSON.parse(data.toString());
      } catch {
        if (verbose) console.error(dim('  received non-JSON message'));
        return;
      }

      switch (msg.type) {
        case 'file_edit': {
          if (!msg.path || msg.content === undefined) {
            if (verbose) console.error(dim('  file_edit missing path or content'));
            break;
          }
          const fullPath = path.join(process.cwd(), msg.path);
          // Ensure directory exists
          const dir = path.dirname(fullPath);
          if (!fs.existsSync(dir)) {
            fs.mkdirSync(dir, { recursive: true });
          }
          // Mark this path so the watcher ignores it
          selfWrittenPaths.add(msg.path);
          fs.writeFileSync(fullPath, msg.content, 'utf8');
          console.log(cyan(`← Received edit: ${msg.path}`));
          break;
        }

        case 'request_spec_tree':
          sendSpecTree();
          if (verbose) console.log(dim('  → sent spec tree (requested)'));
          break;

        case 'request_git_status':
          lastGitStatus = null; // force send
          sendGitStatus();
          if (verbose) console.log(dim('  → sent git status (requested)'));
          break;

        case 'read_local_file': {
          const filePath = msg.path;
          const requestId = msg.request_id || '';
          if (!filePath) break;
          const fullPath = path.join(process.cwd(), filePath);
          try {
            const content = fs.readFileSync(fullPath, 'utf8');
            const stats = fs.statSync(fullPath);
            sendJson({
              type: 'local_file_content',
              request_id: requestId,
              path: filePath,
              content,
              size: stats.size,
              error: '',
            });
            if (verbose) console.log(dim(`  → sent file: ${filePath}`));
          } catch (err) {
            sendJson({
              type: 'local_file_content',
              request_id: requestId,
              path: filePath,
              content: '',
              size: 0,
              error: err.message,
            });
          }
          break;
        }

        case 'list_local_files': {
          const listPath = msg.path || '.';
          const requestId = msg.request_id || '';
          const files = [];
          for (const { name, path: dirPath } of watchedPaths) {
            if (!fs.existsSync(dirPath)) continue;
            const walk = (dir, prefix) => {
              for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
                const rel = path.join(prefix, entry.name);
                if (entry.isDirectory()) {
                  walk(path.join(dir, entry.name), rel);
                } else if (entry.name.endsWith('.md')) {
                  const stats = fs.statSync(path.join(dir, entry.name));
                  files.push({ path: rel, size: stats.size, modified: stats.mtimeMs });
                }
              }
            };
            walk(dirPath, name);
          }
          sendJson({
            type: 'local_file_list',
            request_id: requestId,
            path: listPath,
            files,
          });
          if (verbose) console.log(dim(`  → sent file list (${files.length} files)`));
          break;
        }

        case 'pong':
          // heartbeat response, no action
          break;

        default:
          if (verbose) console.log(dim(`  received unknown message type: ${msg.type}`));
      }
    });

    ws.on('close', () => {
      cleanup(false);
      if (shouldReconnect) {
        console.log(red('✗ Disconnected. Reconnecting in 3s...'));
        setTimeout(() => {
          if (shouldReconnect) connect();
        }, 3000);
      }
    });

    ws.on('error', (err) => {
      if (verbose) console.error(dim(`  ws error: ${err.message || 'connection error'}`));
      // The close event will fire after this, triggering reconnect
    });
  }

  function cleanup(full = true) {
    if (heartbeatInterval) {
      clearInterval(heartbeatInterval);
      heartbeatInterval = null;
    }
    if (gitPollInterval) {
      clearInterval(gitPollInterval);
      gitPollInterval = null;
    }
    if (stopWatcher) {
      stopWatcher();
      stopWatcher = null;
    }
    if (full && ws) {
      shouldReconnect = false;
      ws.close();
      ws = null;
    }
  }

  // Graceful shutdown
  function shutdown() {
    console.log('');
    console.log(dim('Shutting down...'));
    shouldReconnect = false;
    cleanup(true);
    process.exit(0);
  }

  process.on('SIGINT', shutdown);
  process.on('SIGTERM', shutdown);

  // Start
  connect();

  // Keep the process alive
  await new Promise(() => {});
}
