import { createDroplet, getDroplet, listSSHKeys, listProjects, assignToProject } from './do-api.js';
import { readTemplate, getTemplatePath } from './templates.js';
import { saveSandbox } from './store.js';
import { spawn } from 'child_process';
import net from 'net';

const REQUIRED_ENV_VARS = ['AWS_ACCESS_KEY_ID', 'AWS_SECRET_ACCESS_KEY', 'AWS_DEFAULT_REGION', 'GITHUB_TOKEN', 'SPEKK_AGENT_TOKEN', 'SPEKK_HOST'];
const DROPLET_READY_TIMEOUT = 5 * 60 * 1000; // 5 minutes
const PROVISION_TIMEOUT = 10 * 60 * 1000; // 10 minutes
const POLL_INTERVAL = 5000; // 5 seconds

function checkRequiredEnv() {
  const missing = REQUIRED_ENV_VARS.filter((v) => !process.env[v]);
  if (missing.length > 0) {
    throw new Error(`Missing required environment variables: ${missing.join(', ')}`);
  }
}

function runSSH(ip, command) {
  return new Promise((resolve, reject) => {
    const proc = spawn('ssh', [
      '-o', 'StrictHostKeyChecking=no',
      '-o', 'UserKnownHostsFile=/dev/null',
      '-o', 'ConnectTimeout=10',
      `root@${ip}`,
      command,
    ], { stdio: ['pipe', 'pipe', 'pipe'] });

    let stdout = '';
    let stderr = '';
    proc.stdout.on('data', (d) => { stdout += d; });
    proc.stderr.on('data', (d) => { stderr += d; });
    proc.on('close', (code) => {
      if (code === 0) resolve(stdout.trim());
      else reject(new Error(`SSH command failed (exit ${code}): ${stderr.trim()}`));
    });
    proc.on('error', reject);
  });
}

function runSCP(localPath, ip, remotePath) {
  return new Promise((resolve, reject) => {
    const proc = spawn('scp', [
      '-o', 'StrictHostKeyChecking=no',
      '-o', 'UserKnownHostsFile=/dev/null',
      localPath,
      `root@${ip}:${remotePath}`,
    ], { stdio: ['pipe', 'pipe', 'pipe'] });

    let stderr = '';
    proc.stderr.on('data', (d) => { stderr += d; });
    proc.on('close', (code) => {
      if (code === 0) resolve();
      else reject(new Error(`SCP failed (exit ${code}): ${stderr.trim()}`));
    });
    proc.on('error', reject);
  });
}

function checkTCPPort(ip, port, timeoutMs = 5000) {
  return new Promise((resolve) => {
    const socket = new net.Socket();
    const timer = setTimeout(() => {
      socket.destroy();
      resolve(false);
    }, timeoutMs);
    socket.on('connect', () => {
      clearTimeout(timer);
      socket.destroy();
      resolve(true);
    });
    socket.on('error', () => {
      clearTimeout(timer);
      socket.destroy();
      resolve(false);
    });
    socket.connect(port, ip);
  });
}

async function waitForDroplet(dropletId) {
  const start = Date.now();
  while (Date.now() - start < DROPLET_READY_TIMEOUT) {
    const droplet = await getDroplet(dropletId);
    if (droplet.status === 'active') {
      const publicV4 = droplet.networks?.v4?.find((n) => n.type === 'public');
      if (publicV4?.ip_address) {
        return { droplet, ip: publicV4.ip_address };
      }
    }
    await new Promise((r) => setTimeout(r, POLL_INTERVAL));
  }
  throw new Error(`Droplet ${dropletId} did not become active within 5 minutes`);
}

async function waitForProvisioning(ip) {
  const start = Date.now();

  // First wait for SSH connectivity
  console.log('Waiting for SSH connectivity...');
  while (Date.now() - start < PROVISION_TIMEOUT) {
    const reachable = await checkTCPPort(ip, 22);
    if (reachable) break;
    await new Promise((r) => setTimeout(r, POLL_INTERVAL));
  }

  // Then wait for provisioning marker
  console.log('Waiting for cloud-init provisioning to complete...');
  while (Date.now() - start < PROVISION_TIMEOUT) {
    try {
      await runSSH(ip, 'test -f /opt/spekk/.provisioned && echo ok');
      return;
    } catch {
      await new Promise((r) => setTimeout(r, POLL_INTERVAL));
    }
  }
  throw new Error(`Provisioning did not complete within 10 minutes on ${ip}`);
}

async function injectCredentials(ip) {
  const envLines = REQUIRED_ENV_VARS
    .map((v) => `${v}=${process.env[v]}`);
  envLines.push('CLAUDE_CODE_USE_BEDROCK=1');
  const envContent = envLines.join('\n');

  await runSSH(ip, `mkdir -p /etc/spekk && cat > /etc/spekk/agent.env << 'ENVEOF'
${envContent}
ENVEOF
chmod 600 /etc/spekk/agent.env`);
}

async function configureGitCredentials(ip) {
  const ghToken = process.env.GITHUB_TOKEN;
  await runSSH(ip, [
    // Configure git for agent user
    `su - agent -c 'git config --global credential.helper store'`,
    `su - agent -c 'echo "https://x-access-token:${ghToken}@github.com" > ~/.git-credentials'`,
    `su - agent -c 'chmod 600 ~/.git-credentials'`,
    // Configure gh CLI for agent user
    `su - agent -c 'echo "${ghToken}" | gh auth login --with-token 2>/dev/null || true'`,
  ].join(' && '));
}

async function deployAgentClient(ip) {
  const agentClientPath = getTemplatePath('agent-client.py');
  await runSCP(agentClientPath, ip, '/opt/spekk/agent-client.py');
  await runSSH(ip, 'uv pip install --python /opt/spekk/.venv/bin/python websockets');
  await runSSH(ip, 'systemctl enable spekk-agent && systemctl start spekk-agent');
}

async function registerAgent(name, ip) {
  const spekkHost = process.env.SPEKK_HOST;
  const agentToken = process.env.SPEKK_AGENT_TOKEN;

  const res = await fetch(`${spekkHost}/api/agents/`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Token ${agentToken}`,
    },
    body: JSON.stringify({ name, ip }),
  });

  if (!res.ok) {
    const body = await res.text().catch(() => '');
    throw new Error(`Failed to register agent: ${res.status} ${body}`);
  }

  return res.json();
}

// UUID v4 pattern for detecting project IDs
const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

async function resolveProject(projectValue) {
  // If it looks like a UUID, use it directly
  if (UUID_RE.test(projectValue)) {
    return { id: projectValue, name: projectValue };
  }

  // Otherwise, look up by name
  const projects = await listProjects();
  const match = projects.find((p) => p.name === projectValue);
  if (!match) {
    const names = projects.map((p) => `  - ${p.name}`).join('\n');
    console.error(`Error: No project found with name "${projectValue}"`);
    console.error(`Available projects:\n${names}`);
    process.exitCode = 1;
    return null;
  }
  return { id: match.id, name: match.name };
}

export async function createSandbox({ name, region = 'nyc1', size = 's-2vcpu-4gb', project }) {
  const dropletName = `spekk-${name}`;
  let dropletId = null;
  let ip = null;

  try {
    // Validate environment
    checkRequiredEnv();

    // Resolve project before creating anything
    let resolvedProject = null;
    if (project) {
      resolvedProject = await resolveProject(project);
      if (!resolvedProject) return; // error already printed
    }

    // Get SSH keys
    console.log('Fetching SSH keys...');
    const sshKeys = await listSSHKeys();
    if (!sshKeys || sshKeys.length === 0) {
      throw new Error('No SSH keys found on your DigitalOcean account. Add one at https://cloud.digitalocean.com/account/security');
    }
    const sshKeyIds = sshKeys.map((k) => k.id);

    // Read cloud-init template
    const userData = await readTemplate('cloud-init.yaml');

    // Create droplet
    console.log(`Creating droplet "${dropletName}" in ${region} (${size})...`);
    const droplet = await createDroplet({
      name: dropletName,
      region,
      size,
      userData,
      sshKeyIds,
    });
    dropletId = droplet.id;
    console.log(`Droplet created (ID: ${dropletId}). Waiting for it to become active...`);

    // Wait for droplet to be active with public IP
    const result = await waitForDroplet(dropletId);
    ip = result.ip;
    console.log(`Droplet active at ${ip}`);

    // Wait for cloud-init provisioning
    await waitForProvisioning(ip);
    console.log('Provisioning complete.');

    // Inject credentials
    console.log('Injecting credentials...');
    await injectCredentials(ip);

    // Configure git credentials
    console.log('Configuring git credentials...');
    await configureGitCredentials(ip);

    // Deploy agent client
    console.log('Deploying agent client...');
    await deployAgentClient(ip);

    // Register with Spekk API
    console.log('Registering agent...');
    let agentConnected = true;
    try {
      await registerAgent(name, ip);
    } catch (err) {
      agentConnected = false;
      console.error(`Warning: Agent registration failed: ${err.message}`);
    }

    // Assign to project if specified
    if (resolvedProject) {
      console.log(`Assigning droplet to project "${resolvedProject.name}"...`);
      await assignToProject(resolvedProject.id, [`do:droplet:${dropletId}`]);
    }

    // Save metadata locally
    const metadata = {
      dropletId,
      ip,
      region,
      size,
      createdAt: new Date().toISOString(),
      status: 'active',
    };
    if (resolvedProject) {
      metadata.project = resolvedProject.name;
    }
    await saveSandbox(name, metadata);

    // Print summary
    console.log(`
Sandbox created successfully:
  Name:    ${name}
  IP:      ${ip}
  Region:  ${region}
  Size:    ${size}
  ${agentConnected ? 'Agent connected' : 'Agent connection error (registered locally)'}
`);
  } catch (err) {
    if (dropletId && ip) {
      console.error(`\nError: ${err.message}`);
      console.error(`Droplet IP: ${ip} (ID: ${dropletId}) -- not auto-destroyed, debug manually.`);
    } else if (dropletId) {
      console.error(`\nError: ${err.message}`);
      console.error(`Droplet ID: ${dropletId} -- not auto-destroyed, debug manually.`);
    } else {
      console.error(`\nError: ${err.message}`);
    }
    process.exitCode = 1;
  }
}
