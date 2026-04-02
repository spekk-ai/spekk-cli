import { readFile } from 'node:fs/promises'
import { resolve, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

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

