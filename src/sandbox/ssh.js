import { spawn } from 'child_process';
import { getSandbox } from './store.js';

export async function sshCommand(args) {
  const name = args[0];
  if (!name) {
    console.error('Usage: spekk sandbox ssh <name> [ssh-flags...]');
    process.exit(1);
  }

  const sandbox = await getSandbox(name);
  if (!sandbox) {
    console.error(`Sandbox '${name}' not found.`);
    process.exit(1);
  }

  const extraFlags = args.slice(1);
  const sshArgs = [`root@${sandbox.ip}`, ...extraFlags];

  const child = spawn('ssh', sshArgs, { stdio: 'inherit' });

  child.on('error', (err) => {
    console.error(`Failed to start SSH: ${err.message}`);
    process.exit(1);
  });

  child.on('close', (code) => {
    process.exit(code ?? 1);
  });
}
