import childProcess from 'child_process';
import { getSandbox } from './store.js';
import { getTemplatePath } from './templates.js';

/**
 * Runs a command and returns a promise that resolves with { code, stdout, stderr }.
 */
function run(cmd, args, opts = {}) {
  return new Promise((resolve, reject) => {
    const child = childProcess.spawn(cmd, args, { stdio: ['ignore', 'pipe', 'pipe'], ...opts });
    let stdout = '';
    let stderr = '';
    child.stdout.on('data', (d) => { stdout += d; });
    child.stderr.on('data', (d) => { stderr += d; });
    child.on('error', reject);
    child.on('close', (code) => resolve({ code, stdout, stderr }));
  });
}

/**
 * Deploy the agent client to an existing sandbox.
 * @param {string[]} args - Command arguments, first element is sandbox name
 */
export async function deployCommand(args) {
  const name = args[0];
  if (!name) {
    console.error('Usage: spekk sandbox deploy <name>');
    process.exit(1);
  }

  const sandbox = await getSandbox(name);
  if (!sandbox) {
    console.error(`Sandbox '${name}' not found.`);
    process.exit(1);
  }

  const ip = sandbox.ip;

  // Step 1: Get bundled agent-client.py path and copy via SCP
  const templatePath = getTemplatePath('agent-client.py');
  const remoteDest = `root@${ip}:/opt/spekk/agent-client.py`;

  console.log(`Copying agent-client.py to ${ip}...`);
  const scpResult = await run('scp', [
    '-o', 'StrictHostKeyChecking=no',
    templatePath,
    remoteDest,
  ]);

  if (scpResult.code !== 0) {
    console.error(`SCP failed (${ip}): ${scpResult.stderr.trim()}`);
    process.exit(1);
  }

  // Step 2: Install/upgrade websockets package
  console.log('Installing websockets package...');
  const pipResult = await run('ssh', [
    '-o', 'StrictHostKeyChecking=no',
    `root@${ip}`,
    'uv pip install --python /opt/spekk/.venv/bin/python --upgrade websockets',
  ]);

  if (pipResult.code !== 0) {
    console.error(`SSH uv install failed (${ip}): ${pipResult.stderr.trim()}`);
    process.exit(1);
  }

  // Step 3: Restart the spekk-agent systemd service
  console.log('Restarting spekk-agent service...');
  const restartResult = await run('ssh', [
    '-o', 'StrictHostKeyChecking=no',
    `root@${ip}`,
    'systemctl restart spekk-agent',
  ]);

  if (restartResult.code !== 0) {
    console.error(`SSH systemctl restart failed (${ip}): ${restartResult.stderr.trim()}`);
    process.exit(1);
  }

  // Step 4: Check service status
  const statusResult = await run('ssh', [
    '-o', 'StrictHostKeyChecking=no',
    `root@${ip}`,
    'systemctl is-active spekk-agent',
  ]);

  const serviceStatus = statusResult.stdout.trim();
  console.log(`Service status: ${serviceStatus}`);
  console.log(`Agent redeployed to '${name}'.`);
}
