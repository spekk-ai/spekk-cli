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
 * Normalize a server URL to a filesystem-safe key for token storage.
 * ws://localhost:8000 → localhost_8000
 * wss://app.spekk.ai → app.spekk.ai
 */
function normalizeServerHost(url) {
  return (url || '')
    .replace(/^(wss?|https?):\/\//, '')
    .replace(/\/.*$/, '')
    .replace(/:/g, '_')
    .toLowerCase();
}

/**
 * Read the API token from flag, env var, or ~/.spekk/tokens/{server} file.
 * Falls back to legacy ~/.spekk/token and migrates it.
 */
function getToken(flagToken, serverUrl) {
  if (flagToken) return flagToken;
  if (process.env.SPEKK_API_TOKEN) return process.env.SPEKK_API_TOKEN;

  const home = process.env.HOME || process.env.USERPROFILE;
  const host = normalizeServerHost(serverUrl);

  // Try per-server token first
  if (host) {
    const perServerPath = path.join(home, '.spekk', 'tokens', host);
    try {
      return fs.readFileSync(perServerPath, 'utf8').trim();
    } catch {}
  }

  // Fall back to legacy single token file and migrate it
  const legacyPath = path.join(home, '.spekk', 'token');
  try {
    const token = fs.readFileSync(legacyPath, 'utf8').trim();
    if (token && host) {
      // Migrate to per-server storage
      const tokensDir = path.join(home, '.spekk', 'tokens');
      if (!fs.existsSync(tokensDir)) fs.mkdirSync(tokensDir, { recursive: true });
      fs.writeFileSync(path.join(tokensDir, host), token);
      fs.unlinkSync(legacyPath);
      console.log(dim(`  Migrated token to ~/.spekk/tokens/${host}`));
    }
    return token;
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
  return 'wss://app.spekk.ai';
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
  // Handle subcommands: up, dashboard, help
  const subcommand = args[0] && !args[0].startsWith('-') ? args[0] : null;
  const subArgs = subcommand ? args.slice(1) : args;
  const flags = parseFlags(subArgs, FLAG_DEFS);

  // --local is shorthand for --server ws://localhost:8000
  if (subArgs.includes('--local') || subArgs.includes('-l')) {
    flags.server = flags.server || 'ws://localhost:8000';
  }

  if (subcommand === 'dashboard') {
    const serverUrl = getServerUrl(flags.server);
    // Derive HTTP URL from WebSocket URL
    let dashUrl = serverUrl
      .replace('wss://', 'https://')
      .replace('ws://', 'http://');
    // For local dev, open the Vite client
    if (dashUrl.includes('localhost:8000')) {
      dashUrl = 'http://localhost:8080';
    }
    console.log(cyan(`Opening dashboard: ${dashUrl}`));
    const { execSync: exec } = await import('child_process');
    exec(`open "${dashUrl}"`);
    return;
  }

  if (subcommand === 'up') {
    const serverUrl = getServerUrl(flags.server);
    const isLocal = serverUrl.includes('localhost') || serverUrl.includes('127.0.0.1');

    // For local dev: start the full stack (pg/redis, server, client, bridge, browser)
    if (isLocal) {
      const { spawn, execSync: exec } = await import('child_process');
      const repo = getRepoInfo();

      // Check for justfile (indicates spekk-app repo)
      const justfilePath = path.join(repo.path, 'justfile');
      if (fs.existsSync(justfilePath)) {
        console.log(cyan('spekk bridge up --local'));
        console.log(dim('  Starting full local dev stack...'));
        console.log('');

        // Run `just bridge-dev` which handles everything
        const child = spawn('just', ['bridge-dev'], {
          cwd: repo.path,
          stdio: 'inherit',
          env: { ...process.env },
        });

        // Forward Ctrl+C to child
        process.on('SIGINT', () => child.kill('SIGINT'));
        process.on('SIGTERM', () => child.kill('SIGTERM'));

        await new Promise((resolve) => child.on('close', resolve));
        return;
      }
      // Fall through to bridge-only launch if no justfile
    }

    // Check if we have a token for this server — if not, prompt login
    const existingToken = getToken(flags.token, serverUrl);
    if (!existingToken) {
      console.log(dim(`  No token found for ${serverUrl}. Let's log in first.`));
      console.log('');
      const { execSync: execLogin } = await import('child_process');
      const httpUrl = serverUrl.replace('wss://', 'https://').replace('ws://', 'http://');
      try {
        execLogin(`node ${path.join(getRepoInfo().path, 'bridge-app', 'cli', 'index.js')} login --server ${httpUrl}`, { stdio: 'inherit' });
      } catch {
        console.error(red('Login failed. Run `spekk bridge login` manually.'));
        process.exit(1);
      }
    }

    // Staging/production: just launch the Electron menubar app
    try {
      const { spawn } = await import('child_process');
      // Look for bridge-app relative to the repo root
      const repo = getRepoInfo();
      const bridgeAppDir = path.join(repo.path, 'bridge-app');
      if (!fs.existsSync(bridgeAppDir)) {
        console.log(dim('  bridge-app not found, falling back to CLI bridge'));
      } else {
        const env = { ...process.env, NODE_ENV: 'development' };
        if (serverUrl !== 'wss://app.spekk.ai') {
          env.SPEKK_SERVER_URL = serverUrl;
        }
        const electronPath = path.join(bridgeAppDir, 'node_modules', '.bin', 'electron');
        if (fs.existsSync(electronPath)) {
          const child = spawn(electronPath, [bridgeAppDir], { env, stdio: 'ignore', detached: true });
          child.unref();
          console.log(green('✓') + ' Bridge app launched');
          if (flags.server) console.log(dim(`  server: ${serverUrl}`));
          return;
        }
        // Try global electron
        try {
          const child = spawn('npx', ['electron', bridgeAppDir], { env, stdio: 'ignore', detached: true });
          child.unref();
          console.log(green('✓') + ' Bridge app launched');
          if (flags.server) console.log(dim(`  server: ${serverUrl}`));
          return;
        } catch {
          console.log(dim('  Electron not available, falling back to CLI bridge'));
        }
      }
    } catch {
      // Fall through to CLI bridge
    }
    // Fall through — run the CLI bridge below
  }

  if (subcommand === 'help' || subcommand === '--help') {
    console.log('');
    console.log(cyan('spekk bridge') + ' — Connect to Spekk cloud');
    console.log('');
    console.log('  ' + cyan('up') + '          Launch the bridge menubar app');
    console.log('  ' + cyan('up --local') + '  Launch pointed at localhost:8000');
    console.log('  ' + cyan('dashboard') + '   Open the web dashboard in browser');
    console.log('  ' + cyan('[no cmd]') + '    Run CLI-only bridge (no menubar)');
    console.log('');
    console.log('  --server, -s   WebSocket server URL');
    console.log('  --local, -l    Shorthand for --server ws://localhost:8000');
    console.log('  --token, -t    API token (or set SPEKK_API_TOKEN)');
    console.log('  --verbose, -v  Verbose output');
    console.log('');
    return;
  }

  const serverUrl = getServerUrl(flags.server);
  const token = getToken(flags.token, serverUrl);
  if (!token) {
    console.error(red('Error: No API token found.'));
    console.error(`Set SPEKK_API_TOKEN, pass --token, or run: spekk bridge login`);
    process.exit(1);
  }
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

      // Authenticate first — server requires this before any other messages
      sendJson({ type: 'authenticate', token });

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

        case 'git_log': {
          const requestId = msg.request_id || '';
          const specId = msg.spec_id || '';
          const filePath = msg.file_path || '';
          const maxCount = msg.max_count || 50;

          try {
            // Determine which file paths to filter by
            let filterPaths = [];
            if (filePath) {
              filterPaths = [filePath];
            } else if (specId) {
              // Resolve spec_id to file paths (spec dir + assertions dir)
              const specDir = path.join('specs', specId);
              filterPaths = [specDir];
            }

            const pathArg = filterPaths.length > 0 ? `-- ${filterPaths.map(p => `"${p}"`).join(' ')}` : '';
            const logCmd = `git log --format="%H%n%h%n%s%n%an%n%aI%n%D%n---END---" -n ${maxCount} ${pathArg}`;
            const logOutput = execSync(logCmd, {
              encoding: 'utf8',
              stdio: ['pipe', 'pipe', 'ignore'],
            }).trim();

            const commits = [];
            if (logOutput) {
              const entries = logOutput.split('---END---\n').filter(Boolean);
              for (const entry of entries) {
                const lines = entry.trim().split('\n');
                if (lines.length < 5) continue;
                const refs = lines[5] || '';
                // Extract branch name from refs like "HEAD -> main, origin/main"
                const branchMatch = refs.match(/(?:HEAD -> |origin\/)([^,\s]+)/);
                const branch = branchMatch ? branchMatch[1] : '';

                // Get changed files for this commit
                let filesChanged = [];
                try {
                  const diffCmd = `git diff-tree --no-commit-id --name-only -r ${lines[0]}`;
                  const diffOutput = execSync(diffCmd, {
                    encoding: 'utf8',
                    stdio: ['pipe', 'pipe', 'ignore'],
                  }).trim();
                  filesChanged = diffOutput ? diffOutput.split('\n') : [];
                } catch {
                  // ignore
                }

                commits.push({
                  sha: lines[0],
                  short_sha: lines[1],
                  message: lines[2],
                  author: lines[3],
                  date: lines[4],
                  branch,
                  files_changed: filesChanged,
                });
              }
            }

            sendJson({
              type: 'git_log_result',
              request_id: requestId,
              spec_id: specId,
              file_path: filePath,
              total_commits: commits.length,
              commits,
            });
            if (verbose) console.log(dim(`  → sent git_log (${commits.length} commits)`));
          } catch (err) {
            sendJson({
              type: 'git_log_result',
              request_id: requestId,
              spec_id: specId,
              file_path: filePath,
              total_commits: 0,
              commits: [],
              error: err.message,
            });
          }
          break;
        }

        case 'git_show': {
          const requestId = msg.request_id || '';
          const specId = msg.spec_id || '';
          const filePath = msg.file_path || '';
          const commitSha = msg.commit_sha || '';

          if (!commitSha) {
            sendJson({
              type: 'git_show_result',
              request_id: requestId,
              error: 'commit_sha is required',
            });
            break;
          }

          try {
            // Get commit info
            const commitInfo = execSync(
              `git log -1 --format="%s%n%aI" ${commitSha}`,
              { encoding: 'utf8', stdio: ['pipe', 'pipe', 'ignore'] }
            ).trim().split('\n');

            // Determine paths to filter diff by
            let filterPaths = [];
            if (filePath) {
              filterPaths = [filePath];
            } else if (specId) {
              filterPaths = [path.join('specs', specId)];
            }

            // Get diff for this commit
            const pathArg = filterPaths.length > 0 ? `-- ${filterPaths.map(p => `"${p}"`).join(' ')}` : '';
            const diffCmd = `git diff-tree -p --no-commit-id -r ${commitSha} ${pathArg}`;
            const diffOutput = execSync(diffCmd, {
              encoding: 'utf8',
              stdio: ['pipe', 'pipe', 'ignore'],
            }).trim();

            // Parse diff into files
            const files = [];
            if (diffOutput) {
              const fileDiffs = diffOutput.split(/^diff --git /m).filter(Boolean);
              for (const fileDiff of fileDiffs) {
                const headerMatch = fileDiff.match(/^a\/(.+?) b\/(.+)/);
                if (!headerMatch) continue;
                const filename = headerMatch[2];

                let status = 'modified';
                if (fileDiff.includes('new file mode')) status = 'added';
                else if (fileDiff.includes('deleted file mode')) status = 'deleted';
                else if (fileDiff.includes('rename from')) status = 'renamed';

                // Extract patch (everything after the @@ line)
                const patchStart = fileDiff.indexOf('@@');
                const patch = patchStart >= 0 ? fileDiff.slice(patchStart) : '';

                files.push({ filename, status, patch });
              }
            }

            // For spec mode, try to read spec and assertion state at this commit
            let spec = null;
            let assertions = [];
            if (specId) {
              try {
                const specFile = path.join('specs', specId, `${specId}.md`);
                const specContent = execSync(
                  `git show ${commitSha}:${specFile}`,
                  { encoding: 'utf8', stdio: ['pipe', 'pipe', 'ignore'] }
                );
                // Parse frontmatter
                const fmMatch = specContent.match(/^---\n([\s\S]*?)\n---\n([\s\S]*)$/);
                if (fmMatch) {
                  const fm = fmMatch[1];
                  const body = fmMatch[2].trim();
                  const getId = (s) => { const m = s.match(/^id:\s*(.+)/m); return m ? m[1].trim() : specId; };
                  const getPriority = (s) => { const m = s.match(/^priority:\s*(\d+)/m); return m ? parseInt(m[1]) : 1; };
                  const getStatus = (s) => { const m = s.match(/^status:\s*(.+)/m); return m ? m[1].trim() : 'not_started'; };
                  const getTitle = (body) => { const m = body.match(/^#\s+(.+)/m); return m ? m[1].trim() : specId; };
                  spec = {
                    id: getId(fm),
                    title: getTitle(body),
                    priority: getPriority(fm),
                    status: getStatus(fm),
                    content_markdown: body,
                  };
                }
              } catch {
                // spec may not exist at this commit
              }

              // Try to read assertions
              try {
                const assertionsDir = path.join('specs', specId, 'assertions');
                const lsOutput = execSync(
                  `git ls-tree --name-only ${commitSha} ${assertionsDir}/`,
                  { encoding: 'utf8', stdio: ['pipe', 'pipe', 'ignore'] }
                ).trim();
                if (lsOutput) {
                  const changedFiles = files.map(f => f.filename);
                  for (const aFile of lsOutput.split('\n')) {
                    if (!aFile.endsWith('.md')) continue;
                    try {
                      const aContent = execSync(
                        `git show ${commitSha}:${aFile}`,
                        { encoding: 'utf8', stdio: ['pipe', 'pipe', 'ignore'] }
                      );
                      const aFmMatch = aContent.match(/^---\n([\s\S]*?)\n---\n([\s\S]*)$/);
                      if (aFmMatch) {
                        const fm = aFmMatch[1];
                        const body = aFmMatch[2].trim();
                        const getId = (s) => { const m = s.match(/^id:\s*(.+)/m); return m ? m[1].trim() : ''; };
                        const getPriority = (s) => { const m = s.match(/^priority:\s*(\d+)/m); return m ? parseInt(m[1]) : 1; };
                        const getStatus = (s) => { const m = s.match(/^status:\s*(.+)/m); return m ? m[1].trim() : 'not_started'; };
                        const getBranch = (s) => { const m = s.match(/^branch:\s*(.+)/m); return m ? m[1].trim() : ''; };
                        const getTitle = (body) => { const m = body.match(/^#\s+(.+)/m); return m ? m[1].trim() : ''; };
                        assertions.push({
                          id: getId(fm),
                          title: getTitle(body),
                          priority: getPriority(fm),
                          status: getStatus(fm),
                          branch: getBranch(fm),
                          changed_in_this_commit: changedFiles.includes(aFile),
                        });
                      }
                    } catch {
                      // assertion may not exist at this commit
                    }
                  }
                }
              } catch {
                // assertions dir may not exist
              }
            }

            // For file mode, try to read the file content at this commit
            let fileContent = null;
            if (filePath) {
              try {
                fileContent = execSync(
                  `git show ${commitSha}:${filePath}`,
                  { encoding: 'utf8', stdio: ['pipe', 'pipe', 'ignore'] }
                );
              } catch {
                // file may not exist at this commit
              }
            }

            sendJson({
              type: 'git_show_result',
              request_id: requestId,
              spec_id: specId,
              file_path: filePath,
              commit_sha: commitSha,
              commit_message: commitInfo[0] || '',
              commit_date: commitInfo[1] || '',
              spec,
              assertions,
              diff: { files },
              file_content: fileContent,
            });
            if (verbose) console.log(dim(`  → sent git_show for ${commitSha.slice(0, 7)}`));
          } catch (err) {
            sendJson({
              type: 'git_show_result',
              request_id: requestId,
              error: err.message,
            });
          }
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
