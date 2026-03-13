import { loadSandboxes } from './store.js';

export async function listCommand() {
  const sandboxes = await loadSandboxes();
  const entries = Object.entries(sandboxes);

  if (entries.length === 0) {
    console.log('No sandboxes found.');
    return;
  }

  // Print table header
  const header = padRow('Name', 'IP', 'Region', 'Status', 'Created');
  console.log(header);
  console.log('-'.repeat(header.length));

  for (const [name, data] of entries) {
    console.log(padRow(
      name,
      data.ip || '-',
      data.region || '-',
      data.status || '-',
      data.createdAt || '-'
    ));
  }
}

function padRow(name, ip, region, status, created) {
  return [
    name.padEnd(20),
    ip.padEnd(18),
    region.padEnd(10),
    status.padEnd(12),
    created
  ].join('  ');
}
