const SUBCOMMANDS = {
  create: 'Create a new sandbox droplet',
  list: 'List all sandbox droplets',
  status: 'Show status of a sandbox',
  ssh: 'SSH into a sandbox',
  destroy: 'Destroy a sandbox droplet',
  deploy: 'Deploy agent client to a sandbox',
};

function printHelp() {
  console.log(`
spekk sandbox - Manage cloud sandbox environments

USAGE:
  spekk sandbox <subcommand> [options]

SUBCOMMANDS:
${Object.entries(SUBCOMMANDS)
  .map(([name, desc]) => `  ${name.padEnd(12)}${desc}`)
  .join('\n')}

OPTIONS:
  --help, -h   Show this help message

Use "spekk sandbox <subcommand> --help" for more information about a subcommand.
`);
}

function parseCreateFlags(args) {
  const flags = { region: 'nyc1', size: 's-2vcpu-4gb', name: null, project: null, vpc: null };
  for (let i = 0; i < args.length; i++) {
    switch (args[i]) {
      case '--name':
        flags.name = args[++i];
        break;
      case '--region':
        flags.region = args[++i];
        break;
      case '--size':
        flags.size = args[++i];
        break;
      case '--project':
        flags.project = args[++i];
        break;
      case '--vpc':
        flags.vpc = args[++i];
        break;
    }
  }
  return flags;
}

export async function launchSandbox(args) {
  const subcommand = args[0];

  if (!subcommand || subcommand === '--help' || subcommand === '-h' || subcommand === 'help') {
    printHelp();
    process.exitCode = 0;
    return;
  }

  if (!SUBCOMMANDS[subcommand]) {
    console.error(`Unknown sandbox command: ${subcommand}`);
    console.error('Run "spekk sandbox --help" for available subcommands.');
    process.exitCode = 1;
    return;
  }

  const subArgs = args.slice(1);

  switch (subcommand) {
    case 'create': {
      const flags = parseCreateFlags(subArgs);
      if (subArgs.includes('--help') || subArgs.includes('-h')) {
        console.log(`
spekk sandbox create - Create a new sandbox droplet

USAGE:
  spekk sandbox create --name <name> [options]

OPTIONS:
  --name <name>        Sandbox name (required)
  --region <region>    DigitalOcean region (default: nyc1)
  --size <size>        Droplet size slug (default: s-2vcpu-4gb)
  --project <project>  Assign to a DigitalOcean project (name or UUID)
  --vpc <uuid>         Place droplet in a specific DigitalOcean VPC (optional)
`);
        return;
      }
      if (!flags.name) {
        console.error('Error: --name is required for sandbox create');
        process.exitCode = 1;
        return;
      }
      const { createSandbox } = await import('./create.js');
      await createSandbox(flags);
      break;
    }
    case 'list': {
      const { listCommand } = await import('./list.js');
      await listCommand();
      break;
    }
    case 'status': {
      const { statusCommand } = await import('./status.js');
      await statusCommand(subArgs);
      break;
    }
    case 'ssh': {
      const { sshCommand } = await import('./ssh.js');
      await sshCommand(subArgs);
      break;
    }
    case 'destroy': {
      const { destroyCommand } = await import('./destroy.js');
      await destroyCommand(subArgs);
      break;
    }
    case 'deploy': {
      const { deployCommand } = await import('./deploy.js');
      await deployCommand(subArgs);
      break;
    }
  }
}
