/**
 * Shared flag parser for CLI commands.
 *
 * @param {string[]} args - Raw CLI argument array (e.g. process.argv.slice(2))
 * @param {Object} flagDefs - Flag definitions keyed by destination name.
 *   Each value is an object with:
 *     - flags: string[] of flag strings (e.g. ['--spec', '-s'])
 *     - type: 'boolean' | 'string' (default 'boolean')
 *     - default: default value (false for boolean, null for string)
 *
 * @returns {Object} Parsed flags with defaults applied
 *
 * Example:
 *   parseFlags(['--once', '--spec', 'auth'], {
 *     once:  { flags: ['--once'],       type: 'boolean' },
 *     spec:  { flags: ['--spec', '-s'], type: 'string'  },
 *   })
 *   // => { once: true, spec: 'auth' }
 */
export function parseFlags(args, flagDefs) {
  // Build defaults
  const result = {};
  for (const [key, def] of Object.entries(flagDefs)) {
    if ('default' in def) {
      result[key] = def.default;
    } else {
      result[key] = def.type === 'string' ? null : false;
    }
  }

  // Build a lookup: flag string → { key, type }
  const lookup = {};
  for (const [key, def] of Object.entries(flagDefs)) {
    const type = def.type || 'boolean';
    for (const flag of def.flags) {
      lookup[flag] = { key, type };
    }
  }

  // Parse
  for (let i = 0; i < args.length; i++) {
    const match = lookup[args[i]];
    if (!match) continue;

    if (match.type === 'string') {
      result[match.key] = args[++i];
    } else {
      result[match.key] = true;
    }
  }

  return result;
}
