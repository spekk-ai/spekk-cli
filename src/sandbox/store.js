import fs from 'fs/promises';
import path from 'path';
import os from 'os';

function spekkDir() {
  return path.join(os.homedir(), '.spekk');
}

function sandboxesFile() {
  return path.join(spekkDir(), 'sandboxes.json');
}

async function ensureDir() {
  await fs.mkdir(spekkDir(), { recursive: true });
}

export async function loadSandboxes() {
  try {
    const data = await fs.readFile(sandboxesFile(), 'utf8');
    return JSON.parse(data);
  } catch (err) {
    if (err.code === 'ENOENT') return {};
    throw err;
  }
}

export async function saveSandbox(name, data) {
  await ensureDir();
  const sandboxes = await loadSandboxes();
  sandboxes[name] = { ...sandboxes[name], ...data };
  await fs.writeFile(sandboxesFile(), JSON.stringify(sandboxes, null, 2) + '\n');
}

export async function removeSandbox(name) {
  await ensureDir();
  const sandboxes = await loadSandboxes();
  delete sandboxes[name];
  await fs.writeFile(sandboxesFile(), JSON.stringify(sandboxes, null, 2) + '\n');
}

export async function getSandbox(name) {
  const sandboxes = await loadSandboxes();
  return sandboxes[name] || null;
}
