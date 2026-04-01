import childProcess from 'child_process';
import { fetchReleaseArtifacts } from './release.js';

function runCommand(cmd, args) {
  return new Promise((resolve, reject) => {
    const proc = childProcess.spawn(cmd, args, { stdio: ['pipe', 'pipe', 'pipe'] });
    let stdout = '';
    let stderr = '';
    proc.stdout.on('data', (d) => { stdout += d; });
    proc.stderr.on('data', (d) => { stderr += d; });
    proc.on('close', (code) => {
      if (code === 0) resolve(stdout.trim());
      else reject(new Error(`${cmd} failed (exit ${code}): ${stderr.trim()}`));
    });
    proc.on('error', reject);
  });
}

/**
 * Deploy the Go agent binary to a sandbox droplet.
 * @param {string} ip - The IP address of the droplet
 */
export async function deployAgent(ip) {
  const { binaryPath, version } = await fetchReleaseArtifacts();

  // rsync binary to remote
  await runCommand('rsync', [
    '-e', 'ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null',
    '-avz',
    binaryPath,
    `root@${ip}:/opt/spekk/agent-client`,
  ]);

  // chmod +x
  await runCommand('ssh', [
    '-o', 'StrictHostKeyChecking=no',
    '-o', 'UserKnownHostsFile=/dev/null',
    `root@${ip}`,
    'chmod +x /opt/spekk/agent-client',
  ]);

  // reload and restart
  await runCommand('ssh', [
    '-o', 'StrictHostKeyChecking=no',
    '-o', 'UserKnownHostsFile=/dev/null',
    `root@${ip}`,
    'systemctl daemon-reload && systemctl restart spekk-agent',
  ]);

  console.log(`Agent ${version} deployed`);
}
