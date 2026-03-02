/**
 * Spec Context Gatherer
 *
 * Dynamically gathers the current state of specs in the working directory
 * and formats it as a context block that gets prepended to the first user
 * message in a coach session. This gives the coach real data about the
 * project's spec landscape without baking it into the static system prompt.
 */

import fs from 'fs';
import path from 'path';
import { execSync } from 'node:child_process';

/**
 * Parse YAML frontmatter from a markdown file's content.
 * Minimal parser — only extracts key: value pairs (no nested objects/arrays).
 *
 * @param {string} content - Raw file content
 * @returns {{ data: object, content: string }} Parsed frontmatter and body
 */
function parseFrontmatter(content) {
  const lines = content.split('\n');
  if (lines[0] !== '---') return { data: {}, content };

  let end = -1;
  for (let i = 1; i < lines.length; i++) {
    if (lines[i] === '---') { end = i; break; }
  }
  if (end === -1) return { data: {}, content };

  const data = {};
  for (const line of lines.slice(1, end)) {
    const m = line.match(/^([^:]+):\s*(.*)$/);
    if (!m) continue;
    const key = m[1].trim();
    let val = m[2].trim();
    if (val === 'true') val = true;
    else if (val === 'false') val = false;
    else if (/^\d+$/.test(val)) val = parseInt(val, 10);
    // Convert kebab-case keys used in spekk specs
    const jsKey = key === 'depends-on' ? 'dependsOn' : key;
    data[jsKey] = val;
  }

  return { data, content: lines.slice(end + 1).join('\n') };
}

/**
 * Extract the first H1 title from markdown content.
 *
 * @param {string} content - Markdown body (after frontmatter)
 * @returns {string}
 */
function extractTitle(content) {
  const m = content.match(/^# (.+)$/m);
  return m ? m[1] : 'Untitled';
}

/**
 * Compute a parent spec's status from its child assertions.
 *
 * @param {string} parentId
 * @param {object[]} assertions
 * @returns {string}
 */
function computeParentStatus(parentId, assertions) {
  const children = assertions.filter(a => a.parent === parentId);
  if (children.length === 0) return 'not_started';

  const active = children.filter(a => a.status !== 'draft');
  if (active.length === 0) return 'not_started';
  if (active.some(a => a.status === 'failed')) return 'failed';
  if (active.every(a => a.status === 'done')) return 'done';
  if (active.some(a => a.status === 'in_progress' || a.status === 'not_started')) return 'in_progress';
  return 'not_started';
}

/**
 * Gather the current git branch name.
 *
 * @param {string} [cwd] - Working directory (defaults to process.cwd())
 * @returns {string}
 */
function getCurrentBranch(cwd) {
  try {
    return execSync('git branch --show-current', {
      encoding: 'utf8',
      cwd: cwd || process.cwd(),
      stdio: ['pipe', 'pipe', 'ignore'],
    }).trim() || 'unknown';
  } catch {
    return 'unknown';
  }
}

/**
 * Get the last N commits that touched the specs/ directory.
 *
 * @param {number} [n=5]
 * @param {string} [cwd]
 * @returns {string[]} Array of one-line commit descriptions
 */
function getRecentSpecCommits(n = 5, cwd) {
  try {
    const out = execSync(`git log --oneline -${n} -- specs/`, {
      encoding: 'utf8',
      cwd: cwd || process.cwd(),
      stdio: ['pipe', 'pipe', 'ignore'],
    }).trim();
    return out ? out.split('\n') : [];
  } catch {
    return [];
  }
}

/**
 * Read all spec groups and assertions from a specs/ directory, returning
 * a lightweight summary (no full file content).
 *
 * @param {string} specsDir - Absolute path to the specs/ directory
 * @returns {{ groups: object[] }}
 */
function readSpecGroups(specsDir) {
  const groups = [];

  if (!fs.existsSync(specsDir)) return { groups };

  let entries;
  try {
    entries = fs.readdirSync(specsDir);
  } catch {
    return { groups };
  }

  for (const entry of entries) {
    const groupDir = path.join(specsDir, entry);
    try {
      if (!fs.statSync(groupDir).isDirectory()) continue;
    } catch { continue; }

    // Read the parent spec file
    const parentFile = path.join(groupDir, `${entry}.md`);
    if (!fs.existsSync(parentFile)) continue;

    let parentContent;
    try {
      parentContent = fs.readFileSync(parentFile, 'utf8');
    } catch { continue; }

    if (!parentContent.trimStart().startsWith('---')) continue;

    const { data: parentData, content: parentBody } = parseFrontmatter(parentContent);
    const parentTitle = extractTitle(parentBody);

    // Read assertions
    const assertionsDir = path.join(groupDir, 'assertions');
    const assertions = [];

    if (fs.existsSync(assertionsDir)) {
      let assertionFiles;
      try {
        assertionFiles = fs.readdirSync(assertionsDir).filter(f => f.endsWith('.md'));
      } catch {
        assertionFiles = [];
      }

      for (const af of assertionFiles) {
        try {
          const content = fs.readFileSync(path.join(assertionsDir, af), 'utf8');
          if (!content.trimStart().startsWith('---')) continue;
          const { data, content: body } = parseFrontmatter(content);
          assertions.push({
            id: data.id,
            parent: data.parent,
            status: data.status || 'not_started',
            title: extractTitle(body),
          });
        } catch { continue; }
      }
    }

    const doneCount = assertions.filter(a => a.status === 'done').length;
    const computedStatus = computeParentStatus(parentData.id, assertions);

    groups.push({
      id: parentData.id,
      title: parentTitle,
      status: computedStatus,
      totalAssertions: assertions.length,
      doneAssertions: doneCount,
      assertions,
    });
  }

  return { groups };
}

/**
 * Gather full spec context for the current working directory and format
 * it as a text block suitable for prepending to a user message.
 *
 * @param {string} [cwd] - Working directory (defaults to process.cwd())
 * @returns {string} Formatted context block, or empty string if no specs found
 */
export function gatherSpecContext(cwd) {
  const dir = cwd || process.cwd();
  const specsDir = path.join(dir, 'specs');
  const branch = getCurrentBranch(dir);
  const recentCommits = getRecentSpecCommits(5, dir);
  const { groups } = readSpecGroups(specsDir);

  if (groups.length === 0 && recentCommits.length === 0) {
    // Still include branch even when no specs exist — the coach should know
    return [
      '[Spec Context]',
      `Git branch: ${branch}`,
      'No specs found in this project yet.',
      '[/Spec Context]',
    ].join('\n');
  }

  const lines = ['[Spec Context]'];
  lines.push(`Git branch: ${branch}`);

  if (groups.length > 0) {
    lines.push('');
    lines.push('Spec groups:');
    for (const g of groups) {
      lines.push(`  - ${g.id} (${g.status}) — "${g.title}" [${g.doneAssertions}/${g.totalAssertions} done]`);
    }
  }

  if (recentCommits.length > 0) {
    lines.push('');
    lines.push('Recent spec changes (last 5 commits):');
    for (const c of recentCommits) {
      lines.push(`  - ${c}`);
    }
  }

  lines.push('[/Spec Context]');
  return lines.join('\n');
}
