import { getSandbox } from './store.js';
import { deployAgent } from './agent.js';

/**
 * Deploy the agent client to an existing sandbox.
 * @param {string[]} args - Command arguments, first element is sandbox name
 */
export async function deployCommand(args) {
  const name = args[0];
  if (!name) {
    console.error('Usage: spekk sandbox deploy <name>');
    process.exit(1);
  }

  const sandbox = await getSandbox(name);
  if (!sandbox) {
    console.error(`Sandbox '${name}' not found.`);
    process.exit(1);
  }

  const ip = sandbox.ip;
  console.log(`Deploying agent to ${ip}...`);
  await deployAgent(ip);
  console.log(`Agent redeployed to '${name}'.`);
}
