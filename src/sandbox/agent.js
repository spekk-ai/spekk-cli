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
 * @param {object} [artifacts] - Pre-fetched artifacts from fetchReleaseArtifacts(); fetched fresh if omitted
 */
export async function deployAgent(ip, artifacts = null) {
  const { binaryPath, version } = artifacts ?? await fetchReleaseArtifacts();

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

  // Ensure log directory exists
  await runCommand('ssh', [
    '-o', 'StrictHostKeyChecking=no',
    '-o', 'UserKnownHostsFile=/dev/null',
    `root@${ip}`,
    'mkdir -p /var/log/spekk && chown agent:agent /var/log/spekk',
  ]);

  // Update systemd unit to point to Go binary (replaces any legacy Python unit)
  const unitFile = [
    '[Unit]',
    'Description=Spekk Agent Client',
    'After=network.target',
    '',
    '[Service]',
    'Type=simple',
    'User=agent',
    'WorkingDirectory=/opt/spekk',
    'EnvironmentFile=/etc/spekk/agent.env',
    'ExecStart=/opt/spekk/agent-client',
    'Restart=always',
    'RestartSec=5',
    'StandardOutput=append:/var/log/spekk/agent.log',
    'StandardError=append:/var/log/spekk/agent.log',
    '',
    '[Install]',
    'WantedBy=multi-user.target',
  ].join('\\n');
  await runCommand('ssh', [
    '-o', 'StrictHostKeyChecking=no',
    '-o', 'UserKnownHostsFile=/dev/null',
    `root@${ip}`,
    `printf '${unitFile}' > /etc/systemd/system/spekk-agent.service`,
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
