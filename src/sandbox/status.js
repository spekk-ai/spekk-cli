import { getSandbox } from './store.js';
import { getDroplet } from './do-api.js';
import { execFile } from 'child_process';
import { promisify } from 'util';

const execFileAsync = promisify(execFile);

export async function statusCommand(args) {
  const name = args[0];
  if (!name) {
    console.error('Usage: spekk sandbox status <name>');
    process.exit(1);
  }

  const sandbox = await getSandbox(name);
  if (!sandbox) {
    console.error(`Sandbox '${name}' not found.`);
    process.exit(1);
  }

  console.log(`Sandbox: ${name}`);
  console.log(`Droplet ID: ${sandbox.dropletId}`);
  console.log(`IP: ${sandbox.ip || 'unknown'}`);
  console.log(`Region: ${sandbox.region || 'unknown'}`);
  console.log(`Size: ${sandbox.size || 'unknown'}`);
  console.log(`Created: ${sandbox.createdAt || 'unknown'}`);

  // Fetch live data from DO API
  let dropletStatus = sandbox.status || 'unknown';
  try {
    const droplet = await getDroplet(sandbox.dropletId);
    if (droplet && droplet.status) {
      dropletStatus = droplet.status;
    }
  } catch (err) {
    console.log(`Warning: Could not fetch live droplet data: ${err.message}`);
  }
  console.log(`Droplet status: ${dropletStatus}`);

  // SSH checks
  const provisioned = await sshCheck(sandbox.ip, 'test -f /opt/spekk/.provisioned && echo yes || echo no');
  console.log(`Provisioned: ${provisioned}`);

  const agentStatus = await sshCheck(sandbox.ip, 'systemctl is-active spekk-agent 2>/dev/null || echo unknown');
  console.log(`Agent service: ${agentStatus}`);
}

async function sshCheck(ip, command) {
  try {
    const { stdout } = await execFileAsync('ssh', [
      '-o', 'ConnectTimeout=5',
      '-o', 'StrictHostKeyChecking=no',
      '-o', 'BatchMode=yes',
      `root@${ip}`,
      command
    ], { timeout: 10000 });
    return stdout.trim();
  } catch {
    return 'unknown';
  }
}
