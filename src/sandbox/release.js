import { writeFile } from 'node:fs/promises'
import { join } from 'node:path'
import os from 'node:os'
import { randomUUID } from 'node:crypto'

/**
 * Fetches the sandbox binary and cloud-init template from a spekk-app GitHub Release.
 * @param {string} tag - Release tag, or 'latest' for the latest release
 * @returns {Promise<{ binaryPath: string, cloudInitPath: string, version: string }>}
 */
export async function fetchReleaseArtifacts(tag = 'latest') {
  const ghToken = process.env.GITHUB_TOKEN

  // Determine the release API URL
  const releaseUrl = tag === 'latest'
    ? 'https://api.github.com/repos/spekk-ai/spekk-app/releases/latest'
    : `https://api.github.com/repos/spekk-ai/spekk-app/releases/tags/${tag}`

  const headers = {
    'Authorization': `token ${ghToken}`,
    'Accept': 'application/vnd.github.v3+json',
  }

  // Step 1: Fetch the release metadata
  const releaseRes = await fetch(releaseUrl, { headers })

  if (!releaseRes.ok) {
    console.error(`Failed to fetch release '${tag}' from GitHub: HTTP ${releaseRes.status}`)
    process.exit(1)
  }

  const release = await releaseRes.json()
  const version = release.tag_name
  const assets = release.assets || []

  // Step 2: Find the two required assets
  const binaryAsset = assets.find(a => a.name === 'sandbox')
  const cloudInitAsset = assets.find(a => a.name === 'cloud-init.yaml')

  if (!binaryAsset) {
    console.error(`Release '${version}' does not contain asset 'sandbox'`)
    process.exit(1)
  }

  if (!cloudInitAsset) {
    console.error(`Release '${version}' does not contain asset 'cloud-init.yaml'`)
    process.exit(1)
  }

  // Step 3: Download both assets
  const [binaryRes, cloudInitRes] = await Promise.all([
    fetch(binaryAsset.browser_download_url, { headers }),
    fetch(cloudInitAsset.browser_download_url, { headers }),
  ])

  if (!binaryRes.ok) {
    console.error(`Failed to download sandbox binary: HTTP ${binaryRes.status}`)
    process.exit(1)
  }

  if (!cloudInitRes.ok) {
    console.error(`Failed to download cloud-init.yaml: HTTP ${cloudInitRes.status}`)
    process.exit(1)
  }

  // Step 4: Write to temp files
  const binaryBuffer = Buffer.from(await binaryRes.arrayBuffer())
  const cloudInitText = await cloudInitRes.text()

  const binaryPath = join(os.tmpdir(), `spekk-sandbox-${randomUUID()}`)
  const cloudInitPath = join(os.tmpdir(), `spekk-cloud-init-${randomUUID()}.yaml`)

  await writeFile(binaryPath, binaryBuffer)
  await writeFile(cloudInitPath, cloudInitText, 'utf-8')

  return { binaryPath, cloudInitPath, version }
}
