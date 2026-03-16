import { join, dirname } from 'path';
import { fileURLToPath } from 'url';
import { existsSync, readFileSync } from 'fs';
import { homedir } from 'os';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const projectRoot = join(__dirname, '../..');

const PROMPT_SEPARATOR = '\n\n---\n\n';

export class PromptResolver {
  constructor({ homeDir, cwd } = {}) {
    this.homeDir = homeDir || homedir();
    this.cwd = cwd || process.cwd();
    this.promptFiles = [
      {
        name: 'coach',
        path: join(projectRoot, 'specs/coach-agent/coach.prompt.md')
      },
      {
        name: 'builder',
        path: join(projectRoot, 'specs/builder-agent/builder.prompt.md')
      },
      {
        name: 'observer',
        path: join(projectRoot, 'specs/observer-agent/observer.prompt.md')
      }
    ];
  }

  /**
   * Read a file if it exists, otherwise return null.
   */
  _readIfExists(filePath) {
    if (existsSync(filePath)) {
      return readFileSync(filePath, 'utf8');
    }
    return null;
  }

  /**
   * Get the global prompt directory path (~/.spekk/).
   */
  _globalPromptDir() {
    return join(this.homeDir, '.spekk');
  }

  /**
   * Get the local prompt directory path (.spekk/ relative to cwd).
   */
  _localPromptDir() {
    return join(this.cwd, '.spekk');
  }

  /**
   * Resolve the full prompt content for an agent using layered resolution:
   * 1. Determine base: local override > global override > package base (required)
   * 2. Append global extend if exists
   * 3. Append local extend if exists
   * 4. Concatenate layers with separator
   */
  getPromptContent(agentName) {
    const promptFile = this.promptFiles.find(p => p.name === agentName);
    if (!promptFile) {
      throw new Error(`Unknown agent: ${agentName}`);
    }

    const globalDir = this._globalPromptDir();
    const localDir = this._localPromptDir();

    // Step 1: Determine base prompt (local override > global override > package base)
    const localOverridePath = join(localDir, `${agentName}.prompt.override.md`);
    const localOverride = this._readIfExists(localOverridePath);

    const globalOverridePath = join(globalDir, `${agentName}.prompt.override.md`);
    const globalOverride = this._readIfExists(globalOverridePath);

    let base;
    if (localOverride !== null) {
      base = localOverride;
    } else if (globalOverride !== null) {
      base = globalOverride;
    } else {
      if (!existsSync(promptFile.path)) {
        throw new Error(`Prompt file not found: ${promptFile.path}`);
      }
      base = readFileSync(promptFile.path, 'utf8');
    }

    // Step 2: Collect layers to concatenate
    const layers = [base];

    // Append global extend if exists
    const globalExtendPath = join(globalDir, `${agentName}.prompt.md`);
    const globalExtend = this._readIfExists(globalExtendPath);
    if (globalExtend !== null) {
      layers.push(globalExtend);
    }

    // Append local extend if exists
    const localExtendPath = join(localDir, `${agentName}.prompt.md`);
    const localExtend = this._readIfExists(localExtendPath);
    if (localExtend !== null) {
      layers.push(localExtend);
    }

    // Step 3: Concatenate with separator
    return layers.join(PROMPT_SEPARATOR);
  }

  createActivationMessage(agentName) {
    const agentDisplayName = agentName.charAt(0).toUpperCase() + agentName.slice(1);

    try {
      const promptContent = this.getPromptContent(agentName);

      // Include path information for skill discovery
      const workingDir = process.cwd();
      const spekkInstallation = projectRoot;
      const skillsDir = join(projectRoot, 'specs/coach-skills-system');

      return `You are the ${agentDisplayName} Agent - read the prompt and follow the instructions exactly.

Working directory: ${workingDir}
Spekk installation: ${spekkInstallation}
Skills directory: ${skillsDir}

Here is your prompt:

${promptContent}`;
    } catch (error) {
      console.error(`❌ Error loading prompt for ${agentName}:`, error.message);
      throw error;
    }
  }

  verifyPromptFilesExist() {
    const missing = [];
    for (const prompt of this.promptFiles) {
      if (!existsSync(prompt.path)) {
        missing.push(prompt.path);
      }
    }
    return missing;
  }

}

export function launchAgentWithPrompt(agentName, claudeArgs = ['--dangerously-skip-permissions']) {
  const resolver = new PromptResolver();

  try {
    console.log(`🏃‍♀️ Launching ${agentName.charAt(0).toUpperCase() + agentName.slice(1)} Agent with Claude Code...`);
    console.log('Working directory:', process.cwd());
    console.log('Press Ctrl+C to exit the agent session.');
    console.log('');

    const activationMessage = resolver.createActivationMessage(agentName);

    return {
      activationMessage
    };
  } catch (error) {
    console.error(`❌ Error preparing ${agentName} agent:`, error.message);
    throw error;
  }
}
