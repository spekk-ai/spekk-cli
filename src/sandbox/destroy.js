import { getSandbox, removeSandbox } from './store.js';
import { deleteDroplet } from './do-api.js';
import readline from 'readline';

export async function destroyCommand(args) {
  const force = args.includes('--force') || args.includes('-f');
  const name = args.find(a => !a.startsWith('-'));

  if (!name) {
    console.error('Usage: spekk sandbox destroy <name> [--force]');
    process.exit(1);
  }

  const sandbox = await getSandbox(name);
  if (!sandbox) {
    console.error(`Sandbox '${name}' not found.`);
    process.exit(1);
  }

  if (!force) {
    const confirmed = await confirm(`Destroy sandbox '${name}' (droplet ${sandbox.dropletId})? [y/N] `);
    if (!confirmed) {
      console.log('Aborted.');
      return;
    }
  }

  try {
    await deleteDroplet(sandbox.dropletId);
  } catch (err) {
    if (err.message && err.message.includes('404')) {
      console.log(`Warning: Droplet ${sandbox.dropletId} was already deleted.`);
    } else {
      throw err;
    }
  }

  await removeSandbox(name);
  console.log(`Sandbox '${name}' destroyed.`);
}

function confirm(prompt) {
  return new Promise((resolve) => {
    const rl = readline.createInterface({
      input: process.stdin,
      output: process.stdout,
    });
    rl.question(prompt, (answer) => {
      rl.close();
      resolve(answer.toLowerCase() === 'y');
    });
  });
}
