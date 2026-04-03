import { join, basename } from 'path';
import { existsSync, readFileSync, readdirSync } from 'fs';
import { homedir } from 'os';
import { fileURLToPath } from 'url';
import { dirname } from 'path';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const projectRoot = join(__dirname, '../..');

/**
 * Legacy aliases: subcommand shorthand → full skill filename stem.
 * These map old `spekk coach meeting` to the actual skill file name.
 */
const LEGACY_ALIASES = {
  coach: {
    meeting: 'meeting-notes-to-specs-skill',
    coordinate: 'coordinator-skill',
  },
  builder: {},
};

/**
 * Package-shipped skill directories per agent.
 */
const PACKAGE_SKILL_DIRS = {
  coach: join(projectRoot, 'specs/coach-skills-system'),
  builder: join(projectRoot, 'specs/builder-skills'),
};

export class SkillResolver {
  constructor({ homeDir, cwd } = {}) {
    this.homeDir = homeDir || homedir();
    this.cwd = cwd || process.cwd();
  }

  /**
   * Get the layered skill directories for an agent, in resolution order
   * (first match wins): local → global → package.
   */
  _skillDirs(agentName) {
    return [
      join(this.cwd, '.spekk', 'skills', agentName),
      join(this.homeDir, '.spekk', 'skills', agentName),
      PACKAGE_SKILL_DIRS[agentName],
    ].filter(Boolean);
  }

  /**
   * Parse frontmatter `id` field from a markdown file's YAML header.
   * Returns the id string or null.
   */
  _parseFrontmatterId(content) {
    const match = content.match(/^---\n([\s\S]*?)\n---/);
    if (!match) return null;
    const idMatch = match[1].match(/^id:\s*(.+)$/m);
    return idMatch ? idMatch[1].trim() : null;
  }

  /**
   * Resolve a skill by subcommand name for an agent.
   *
   * Checks legacy aliases first, then searches layered directories
   * by filename stem and frontmatter id.
   *
   * @returns {{ name: string, content: string, source: string }} | null
   */
  resolveSkill(agentName, subcommand) {
    if (!subcommand) return null;

    // Resolve legacy alias to the actual filename stem
    const aliases = LEGACY_ALIASES[agentName] || {};
    const resolvedName = aliases[subcommand] || subcommand;

    const dirs = this._skillDirs(agentName);

    for (const dir of dirs) {
      if (!existsSync(dir)) continue;

      // Direct filename match: <resolvedName>.md
      const directPath = join(dir, `${resolvedName}.md`);
      if (existsSync(directPath)) {
        const content = readFileSync(directPath, 'utf8');
        return { name: resolvedName, content, source: dir };
      }

      // If the original subcommand differs from resolvedName, also try <subcommand>.md
      if (subcommand !== resolvedName) {
        const aliasPath = join(dir, `${subcommand}.md`);
        if (existsSync(aliasPath)) {
          const content = readFileSync(aliasPath, 'utf8');
          return { name: subcommand, content, source: dir };
        }
      }

      // Scan directory for frontmatter id match
      let files;
      try {
        files = readdirSync(dir).filter(f => f.endsWith('.md'));
      } catch {
        continue;
      }

      for (const file of files) {
        const filePath = join(dir, file);
        const content = readFileSync(filePath, 'utf8');
        const fmId = this._parseFrontmatterId(content);
        if (fmId === resolvedName || fmId === subcommand) {
          const stem = basename(file, '.md');
          return { name: stem, content, source: dir };
        }
      }
    }

    return null;
  }

  /**
   * Return the legacy alias map for an agent.
   * Keys are the alias names (e.g. "meeting"), values are the underlying skill filename stems.
   *
   * @returns {Record<string, string>}
   */
  listAliases(agentName) {
    return LEGACY_ALIASES[agentName] || {};
  }

  /**
   * List all available skills for an agent, with later layers
   * not duplicating earlier ones (local wins over global wins over package).
   *
   * @returns {Array<{ name: string, source: string }>}
   */
  listSkills(agentName) {
    const seen = new Set();
    const skills = [];
    const dirs = this._skillDirs(agentName);

    for (const dir of dirs) {
      if (!existsSync(dir)) continue;

      let files;
      try {
        files = readdirSync(dir).filter(f => f.endsWith('.md'));
      } catch {
        continue;
      }

      for (const file of files) {
        const stem = basename(file, '.md');
        if (seen.has(stem)) continue;
        seen.add(stem);
        skills.push({ name: stem, source: dir });
      }
    }

    return skills;
  }
}
