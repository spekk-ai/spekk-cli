import { join, dirname } from 'path';
import { fileURLToPath } from 'url';
import { existsSync, mkdirSync, copyFileSync, rmSync } from 'fs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const projectRoot = join(__dirname, '../..');

export class PromptResolver {
  constructor() {
    this.tempPromptFiles = [];
    this.promptFiles = [
      {
        source: join(projectRoot, 'specs/coach-agent/coach-agent.prompt.md'),
        target: 'specs/coach-agent/coach-agent.prompt.md'
      },
      {
        source: join(projectRoot, 'specs/builder-agent/builder-agent.prompt.md'), 
        target: 'specs/builder-agent/builder-agent.prompt.md'
      },
      {
        source: join(projectRoot, 'specs/observer-agent/observer-agent.prompt.md'),
        target: 'specs/observer-agent/observer-agent.prompt.md'
      }
    ];
  }

  setupPromptFiles() {
    console.log('📋 Setting up prompt files for Claude Code access...');
    
    try {
      for (const prompt of this.promptFiles) {
        // Ensure target directory exists
        const targetDir = dirname(prompt.target);
        if (!existsSync(targetDir)) {
          mkdirSync(targetDir, { recursive: true });
        }
        
        // Copy source prompt file to target location
        if (existsSync(prompt.source)) {
          copyFileSync(prompt.source, prompt.target);
          this.tempPromptFiles.push(prompt.target);
          console.log(`   ✅ Copied ${prompt.target}`);
        } else {
          console.warn(`   ⚠️  Warning: Source prompt file not found: ${prompt.source}`);
        }
      }
      
      if (this.tempPromptFiles.length > 0) {
        console.log(`   📁 ${this.tempPromptFiles.length} prompt files ready for Claude Code`);
      }
      
    } catch (error) {
      console.error('❌ Error setting up prompt files:', error.message);
      // Clean up any partially created files
      this.cleanupPromptFiles();
      throw error;
    }
  }

  cleanupPromptFiles() {
    if (this.tempPromptFiles.length === 0) {
      return;
    }
    
    console.log('🧹 Cleaning up temporary prompt files...');
    
    try {
      for (const tempFile of this.tempPromptFiles) {
        if (existsSync(tempFile)) {
          // Remove the file
          rmSync(tempFile, { force: true });
          
          // Remove parent directories if they're empty
          const dir = dirname(tempFile);
          this.removeEmptyDirectories(dir);
        }
      }
      
      console.log(`   ✅ Cleaned up ${this.tempPromptFiles.length} temporary files`);
      this.tempPromptFiles = [];
      
    } catch (error) {
      console.error('❌ Error cleaning up prompt files:', error.message);
      // Don't throw here - cleanup errors shouldn't crash the process
    }
  }

  removeEmptyDirectories(dir) {
    try {
      // Only remove specs/ subdirectories we created, not user's existing directories
      if (!dir.includes('specs/') || dir === 'specs') {
        return;
      }
      
      // Try to remove if empty
      rmSync(dir, { force: true });
      
      // Recursively try parent directory
      const parentDir = dirname(dir);
      if (parentDir !== dir && parentDir.includes('specs/')) {
        this.removeEmptyDirectories(parentDir);
      }
    } catch (error) {
      // Directory not empty or other error - ignore silently
    }
  }
}

export async function withPromptFiles(callback) {
  const resolver = new PromptResolver();
  
  try {
    resolver.setupPromptFiles();
    await callback();
  } finally {
    resolver.cleanupPromptFiles();
  }
}