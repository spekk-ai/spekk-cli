import fs from 'fs/promises';
import path from 'path';
import os from 'os';

const SPEKK_DIR = path.join(os.homedir(), '.spekk');
const SANDBOXES_FILE = path.join(SPEKK_DIR, 'sandboxes.json');

async function ensureDir() {
  await fs.mkdir(SPEKK_DIR, { recursive: true });
}

export async function loadSandboxes() {
  try {
    const data = await fs.readFile(SANDBOXES_FILE, 'utf8');
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
  await fs.writeFile(SANDBOXES_FILE, JSON.stringify(sandboxes, null, 2) + '\n');
}

export async function removeSandbox(name) {
  await ensureDir();
  const sandboxes = await loadSandboxes();
  delete sandboxes[name];
  await fs.writeFile(SANDBOXES_FILE, JSON.stringify(sandboxes, null, 2) + '\n');
}

export async function getSandbox(name) {
  const sandboxes = await loadSandboxes();
  return sandboxes[name] || null;
}
