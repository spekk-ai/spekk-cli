import fs from 'fs';
import path from 'path';
import os from 'os';
import { fileURLToPath } from 'url';
import { parseFrontmatter } from '../parser/index.js';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

/**
 * Get the three-layer gate paths in resolution order (package → global → local).
 * Local overrides global overrides package.
 */
export function getGatePaths(options = {}) {
  const packageRoot = options.packageRoot || path.join(__dirname, '../..');
  const globalDir = options.globalDir || path.join(os.homedir(), '.spekk', 'gates');
  const localDir = options.localDir || path.join(process.cwd(), '.spekk', 'gates');

  return [
    { layer: 'package', path: path.join(packageRoot, 'gates') },
    { layer: 'global', path: globalDir },
    { layer: 'local', path: localDir },
  ];
}

/**
 * Parse the ## Preconditions section into structured check objects.
 */
function parsePreconditions(text) {
  if (!text) return [];

  const checks = [];
  const lines = text.split('\n');

  for (const line of lines) {
    const match = line.match(/^-\s+(\S+):\s+"([^"]+)"$/);
    if (!match) continue;

    const [, checkType, value] = match;

    switch (checkType) {
      case 'files-changed':
        checks.push({ type: 'files-changed', pattern: value });
        break;
      case 'dir-exists':
        checks.push({ type: 'dir-exists', path: value });
        break;
      case 'file-exists':
        checks.push({ type: 'file-exists', path: value });
        break;
      case 'file-not-exists':
        checks.push({ type: 'file-not-exists', path: value });
        break;
      case 'branch-matches':
        checks.push({ type: 'branch-matches', pattern: value });
        break;
      case 'has-dependency':
        checks.push({ type: 'has-dependency', package: value });
        break;
      case 'command-succeeds':
        checks.push({ type: 'command-succeeds', command: value });
        break;
    }
  }

  return checks;
}

/**
 * Parse the ## On Failure section for severity and action.
 */
function parseOnFailure(text) {
  if (!text) return {};

  const result = {};
  const lines = text.split('\n');

  for (const line of lines) {
    const match = line.match(/^-\s+(\S+):\s+(.+)$/);
    if (!match) continue;

    const [, key, value] = match;
    result[key] = value.trim();
  }

  return result;
}

/**
 * Extract named sections from markdown content.
 * Returns a map of section name → section body text.
 */
function parseSections(markdownContent) {
  const sections = {};
  const lines = markdownContent.split('\n');
  let currentSection = null;
  let currentLines = [];

  for (const line of lines) {
    const headingMatch = line.match(/^## (.+)$/);
    if (headingMatch) {
      if (currentSection) {
        sections[currentSection] = currentLines.join('\n').trim();
      }
      currentSection = headingMatch[1];
      currentLines = [];
    } else if (currentSection) {
      currentLines.push(line);
    }
  }

  if (currentSection) {
    sections[currentSection] = currentLines.join('\n').trim();
  }

  return sections;
}

/**
 * Parse a single .gate.md file into a gate object.
 */
function parseGateFile(filePath, layer) {
  const content = fs.readFileSync(filePath, 'utf8');

  if (!content.trimStart().startsWith('---')) {
    return null;
  }

  const { data, content: markdownContent } = parseFrontmatter(content);

  if (!data.id) {
    return null;
  }

  const sections = parseSections(markdownContent);

  // Extract title from first H1
  const titleMatch = markdownContent.match(/^# (.+)$/m);
  const title = titleMatch ? titleMatch[1] : data.id;

  return {
    id: data.id,
    phase: data.phase || null,
    tags: data.tags || [],
    dependsOn: data.dependsOn || null,
    title,
    layer,
    file: filePath,
    preconditions: parsePreconditions(sections['Preconditions'] || ''),
    llmJudgment: sections['LLM Judgment'] || null,
    workflow: sections['Workflow'] || null,
    onFailure: parseOnFailure(sections['On Failure'] || ''),
  };
}

/**
 * Topological sort of gates by depends-on relationships.
 * Gates with no dependencies come first.
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
 * Load all .gate.md files from the three-layer resolution path.
 * Local gates override global gates override package gates (matched by id).
 * Returns an array of gate objects sorted by dependency order.
 */
export function loadGates(options = {}) {
  const paths = getGatePaths(options);
  const gatesById = new Map();

  for (const { layer, path: dirPath } of paths) {
    if (!fs.existsSync(dirPath)) continue;

    const files = fs.readdirSync(dirPath).filter(f => f.endsWith('.gate.md'));

    for (const file of files) {
      const filePath = path.join(dirPath, file);
      const gate = parseGateFile(filePath, layer);

      if (gate) {
        // Later layers override earlier ones (by id)
        gatesById.set(gate.id, gate);
      }
    }
  }

  const gates = Array.from(gatesById.values());
  return topologicalSort(gates);
}
