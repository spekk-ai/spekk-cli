import { parseFlags } from '../cli/parse-flags.js';
import { startServe } from './index.js';

const serveFlagDefs = {
  port: { flags: ['--port', '-p'], type: 'string' },
  host: { flags: ['--host'], type: 'string' },
  idleTimeout: { flags: ['--idle-timeout'], type: 'string' },
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
  --port, -p <port>           Port to listen on (default: 3118)
  --host <host>               Host to bind to (default: localhost)
  --idle-timeout <seconds>    Idle timeout in seconds before auto-disconnect (default: 1800)
  --help, -h                  Show this help message

DESCRIPTION:
  Starts a WebSocket server that bridges the Spekk browser extension
  to a coach agent (Claude Code subprocess). Only one active connection
  is allowed at a time per port. Additional connections receive a
  connection_locked message and can request a force takeover.

  Idle connections are automatically timed out after --idle-timeout seconds.

  The extension connects to ws://localhost:3118 by default.
`);
    return;
  }

  const options = {};
  if (flags.port) options.port = parseInt(flags.port, 10);
  if (flags.host) options.host = flags.host;
  if (flags.idleTimeout) options.idleTimeout = parseInt(flags.idleTimeout, 10);

  startServe(options);
}
