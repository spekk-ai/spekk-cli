import { parseFlags } from '../cli/parse-flags.js';
import { startServe } from './index.js';

const serveFlagDefs = {
  port: { flags: ['--port', '-p'], type: 'string' },
  host: { flags: ['--host'], type: 'string' },
  help: { flags: ['--help', '-h'], type: 'boolean' },
};

export function launchServe(args = []) {
  const flags = parseFlags(args, serveFlagDefs);

  if (flags.help) {
    console.log(`
spekk serve - Start WebSocket server for browser extension

USAGE:
  spekk serve [OPTIONS]

OPTIONS:
  --port, -p <port>   Port to listen on (default: 3118)
  --host <host>       Host to bind to (default: localhost)
  --help, -h          Show this help message

DESCRIPTION:
  Starts a WebSocket server that bridges the Spekk browser extension
  to a coach agent (Claude Code subprocess). Each browser connection
  gets its own agent instance.

  The extension connects to ws://localhost:3118 by default.
`);
    return;
  }

  const options = {};
  if (flags.port) options.port = parseInt(flags.port, 10);
  if (flags.host) options.host = flags.host;

  startServe(options);
}
