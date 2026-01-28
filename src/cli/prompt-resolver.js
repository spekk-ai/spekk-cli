import { join, dirname } from 'path';
import { fileURLToPath } from 'url';
import { existsSync, readFileSync } from 'fs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const projectRoot = join(__dirname, '../..');

export class PromptResolver {
  constructor() {
    this.promptFiles = [
      {
        name: 'coach-agent',
        path: join(projectRoot, 'specs/coach-agent/coach-agent.prompt.md')
      },
      {
        name: 'builder-agent', 
        path: join(projectRoot, 'specs/builder-agent/builder-agent.prompt.md')
      },
      {
        name: 'observer-agent',
        path: join(projectRoot, 'specs/observer-agent/observer-agent.prompt.md')
      }
    ];
  }

  getPromptContent(agentName) {
    const promptFile = this.promptFiles.find(p => p.name === agentName);
    if (!promptFile) {
      throw new Error(`Unknown agent: ${agentName}`);
    }
    
    if (!existsSync(promptFile.path)) {
      throw new Error(`Prompt file not found: ${promptFile.path}`);
    }
    
    return readFileSync(promptFile.path, 'utf8');
  }

  createActivationMessage(agentName) {
    const agentDisplayName = agentName.split('-').map(word => 
      word.charAt(0).toUpperCase() + word.slice(1)
    ).join(' ');
    
    try {
      const promptContent = this.getPromptContent(agentName);
      
      return `You are the ${agentDisplayName} Agent - read the prompt and follow the instructions exactly.

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
    console.log(`🏃‍♀️ Launching ${agentName.split('-').map(w => w.charAt(0).toUpperCase() + w.slice(1)).join(' ')} Agent with Claude Code...`);
    console.log('Working directory:', process.cwd());
    console.log('Press Ctrl+C to exit the agent session.');
    console.log(''); 
    
    const activationMessage = resolver.createActivationMessage(agentName);
    
    return {
      activationMessage,
      resolver
    };
  } catch (error) {
    console.error(`❌ Error preparing ${agentName} agent:`, error.message);
    throw error;
  }
}