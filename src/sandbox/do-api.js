const BASE_URL = 'https://api.digitalocean.com';

function getToken() {
  const token = process.env.DO_API_TOKEN;
  if (!token) {
    throw new Error(
      'DO_API_TOKEN environment variable is not set. ' +
      'Get a token from https://cloud.digitalocean.com/account/api/tokens'
    );
  }
  return token;
}

function headers() {
  return {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${getToken()}`,
  };
}

async function doFetch(path, options = {}) {
  const url = `${BASE_URL}${path}`;
  const res = await fetch(url, { ...options, headers: headers() });

  if (res.status === 204) {
    return null;
  }

  const body = await res.json().catch(() => null);

  if (!res.ok) {
    const message = body?.message || body?.id || `HTTP ${res.status}`;
    throw new Error(`DigitalOcean API error: ${message} (${res.status} ${res.statusText})`);
  }

  return body;
}

export async function createDroplet({ name, region, size, userData, sshKeyIds, vpcUuid }) {
  const payload = {
    name,
    region,
    size,
    image: 'ubuntu-24-04-x64',
    ssh_keys: sshKeyIds || [],
    user_data: userData || undefined,
    tags: ['spekk-sandbox'],
    ...(vpcUuid ? { vpc_uuid: vpcUuid } : {}),
  };

  const body = await doFetch('/v2/droplets', {
    method: 'POST',
    body: JSON.stringify(payload),
  });

  return body.droplet;
}

export async function getDroplet(id) {
  const body = await doFetch(`/v2/droplets/${id}`);
  return body.droplet;
}

export async function listDroplets(tag) {
  const query = tag ? `?tag_name=${encodeURIComponent(tag)}` : '';
  const body = await doFetch(`/v2/droplets${query}`);
  return body.droplets;
}

export async function deleteDroplet(id) {
  await doFetch(`/v2/droplets/${id}`, { method: 'DELETE' });
  return { success: true };
}

export async function listSSHKeys() {
  const body = await doFetch('/v2/account/keys');
  return body.ssh_keys;
}

export async function listProjects() {
  const body = await doFetch('/v2/projects');
  return body.projects;
}

export async function assignToProject(projectId, resourceUrns) {
  const body = await doFetch(`/v2/projects/${projectId}/resources`, {
    method: 'POST',
    body: JSON.stringify({ resources: resourceUrns }),
  });
  return body;
}
