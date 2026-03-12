import { readFile, writeFile } from 'node:fs/promises'
import { resolve, dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import os from 'node:os'
import { randomUUID } from 'node:crypto'

const __dirname = dirname(fileURLToPath(import.meta.url))
const TEMPLATES_DIR = resolve(__dirname, 'templates')

/**
 * Resolves the absolute path to a bundled template file.
 * @param {string} filename - Template filename (e.g. 'cloud-init.yaml')
 * @returns {string} Absolute path to the template file
 */
export function getTemplatePath(filename) {
  return resolve(TEMPLATES_DIR, filename)
}

/**
 * Reads and returns a template file's contents as a string.
 * @param {string} filename - Template filename (e.g. 'cloud-init.yaml')
 * @returns {Promise<string>} Template contents
 */
export async function readTemplate(filename) {
  const filePath = getTemplatePath(filename)
  return readFile(filePath, 'utf-8')
}

/**
 * Reads the cloud-init template and substitutes the SSH public key placeholder.
 * @param {string} sshPublicKey - The user's SSH public key
 * @returns {Promise<string>} cloud-init.yaml with the SSH key substituted
 */
export async function renderCloudInit(sshPublicKey) {
  const template = await readTemplate('cloud-init.yaml')
  return template.replace(
    'ssh-ed25519 AAAA... your-key-here',
    sshPublicKey
  )
}

/**
 * Fetches the agent client script from the spekk-app GitHub repository.
 * Decodes the base64 content from the GitHub Contents API, writes it to a
 * temporary file, and returns the path for SCP.
 * @returns {Promise<string>} Path to temporary file containing agent-client.py
 */
export async function fetchAgentClient() {
  const url = 'https://api.github.com/repos/spekk-ai/spekk-app/contents/infrastructure/droplet/agent-client.py'
  const ghToken = process.env.GITHUB_TOKEN

  const res = await fetch(url, {
    headers: {
      'Authorization': `token ${ghToken}`,
      'Accept': 'application/vnd.github.v3+json',
    },
  })

  if (!res.ok) {
    console.error(`Failed to fetch agent-client.py from GitHub: HTTP ${res.status}`)
    process.exit(1)
  }

  const data = await res.json()
  const content = Buffer.from(data.content, 'base64').toString('utf-8')

  const tmpPath = join(os.tmpdir(), `agent-client-${randomUUID()}.py`)
  await writeFile(tmpPath, content, 'utf-8')

  return tmpPath
}
