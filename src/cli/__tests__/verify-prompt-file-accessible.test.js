#!/usr/bin/env node

import { join, dirname } from 'path';
import { fileURLToPath } from 'url';
import { existsSync, mkdirSync, rmSync, readdirSync } from 'fs';
import { spawn } from 'child_process';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const projectRoot = join(__dirname, '../../..');
const spekkBinary = join(projectRoot, 'bin/spekk.js');

class VerifyPromptFileAccessibleTest {
  constructor() {
    this.testDir = '/tmp/test-prompt-accessible';
    this.passed = 0;
    this.failed = 0;
  }

  async runAllTests() {
    console.log('🧪 Running Verify Prompt File Accessible Tests...\n');
    
    // Setup test directory
    this.setupTestDirectory();
    
    try {
      // Test core functionality
      await this.testSpekkCoachFromExternal();
      await this.testNoSpecsDirectoryCreated();
      await this.testClaudeCodeReceivesActivationMessage();
      await this.testClaudeCodeCanAccessPrompt();
      await this.testWorkingDirectoryUnmodified();
      await this.testNoTemporaryFilesNeeded();
      
    } catch (error) {
      console.error('❌ Test setup failed:', error.message);
      this.failed++;
    } finally {
      // Cleanup test directory
      this.cleanupTestDirectory();
    }
    
    this.printResults();
  }

  setupTestDirectory() {
    if (existsSync(this.testDir)) {
      rmSync(this.testDir, { recursive: true, force: true });
    }
    mkdirSync(this.testDir, { recursive: true });
  }

  cleanupTestDirectory() {
    if (existsSync(this.testDir)) {
      rmSync(this.testDir, { recursive: true, force: true });
    }
  }

  getDirectoryContents(dir) {
    try {
      return readdirSync(dir, { withFileTypes: true })
        .map(dirent => dirent.name);
    } catch (error) {
      return [];
    }
  }

  async testSpekkCoachFromExternal() {
    console.log('🎯 Testing spekk coach runs from external directory...');
    
    try {
      // Change to external directory for the test
      const originalCwd = process.cwd();
      process.chdir(this.testDir);
      
      try {
        // Test the CLI command setup
        const { launchCoachAgent } = await import('../../coach/cli.js');
        
        // Verify CLI function can be imported without errors
        if (launchCoachAgent && typeof launchCoachAgent === 'function') {
          console.log('   ✅ spekk coach CLI accessible from external directory');
          this.passed++;
        } else {
          console.error('   ❌ spekk coach CLI failed to initialize');
          this.failed++;
        }
        
      } finally {
        process.chdir(originalCwd);
      }
      
    } catch (error) {
      console.error('   ❌ Error testing spekk coach from external directory:', error.message);
      this.failed++;
    }
  }

  async testNoSpecsDirectoryCreated() {
    console.log('\n📁 Testing NO specs directory created in user working directory...');
    
    // Change to test directory
    const originalCwd = process.cwd();
    process.chdir(this.testDir);
    
    try {
      // Record initial directory contents
      const initialContents = this.getDirectoryContents(this.testDir);
      
      // Test the prompt resolution system
      const { PromptResolver } = await import('../prompt-resolver.js');
      const resolver = new PromptResolver();
      
      // Create activation message - this should NOT create any local files
      resolver.createActivationMessage('coach-agent');
      
      // Check that no specs directory was created
      const specsDir = join(this.testDir, 'specs');
      if (!existsSync(specsDir)) {
        console.log('   ✅ No specs directory created in user working directory');
        this.passed++;
      } else {
        console.error('   ❌ specs directory was created in user working directory (prohibited)');
        this.failed++;
      }
      
    } catch (error) {
      console.error('   ❌ Error testing specs directory creation:', error.message);
      this.failed++;
    } finally {
      process.chdir(originalCwd);
    }
  }

  async testClaudeCodeReceivesActivationMessage() {
    console.log('\n💬 Testing Claude Code receives agent activation message successfully...');
    
    // Change to external directory for the test
    const originalCwd = process.cwd();
    process.chdir(this.testDir);
    
    try {
      // Test the prompt resolution system
      const { PromptResolver } = await import('../prompt-resolver.js');
      const resolver = new PromptResolver();
      
      // Create activation message for coach agent
      const activationMessage = resolver.createActivationMessage('coach-agent');
      
      // Verify message contains expected content that Claude Code would receive
      if (activationMessage.includes('You are the Coach Agent') && 
          activationMessage.includes('read the prompt and follow the instructions exactly') &&
          activationMessage.length > 100) {
        console.log('   ✅ Claude Code receives valid agent activation message');
        this.passed++;
      } else {
        console.error('   ❌ Claude Code activation message is invalid or incomplete');
        console.error('   Message length:', activationMessage.length);
        this.failed++;
      }
      
    } catch (error) {
      console.error('   ❌ Error testing Claude Code activation message:', error.message);
      this.failed++;
    } finally {
      process.chdir(originalCwd);
    }
  }

  async testClaudeCodeCanAccessPrompt() {
    console.log('\n📖 Testing Claude Code can access coach prompt from spekk-cli installation directory...');
    
    // Change to external directory for the test
    const originalCwd = process.cwd();
    process.chdir(this.testDir);
    
    try {
      // Test the prompt resolution system
      const { PromptResolver } = await import('../prompt-resolver.js');
      const resolver = new PromptResolver();
      
      // Test that Claude Code can access the actual prompt content
      const promptContent = resolver.getPromptContent('coach-agent');
      
      // Verify prompt content is accessible and valid
      if (promptContent && 
          promptContent.includes('You are the **Coach Agent**') &&
          promptContent.length > 500) {
        console.log('   ✅ Claude Code can access coach prompt from installation directory');
        this.passed++;
      } else {
        console.error('   ❌ Claude Code cannot access coach prompt or content is invalid');
        console.error('   Content length:', promptContent ? promptContent.length : 0);
        this.failed++;
      }
      
    } catch (error) {
      console.error('   ❌ Error testing Claude Code prompt access:', error.message);
      this.failed++;
    } finally {
      process.chdir(originalCwd);
    }
  }

  async testWorkingDirectoryUnmodified() {
    console.log('\n🧹 Testing user working directory remains completely unmodified...');
    
    // Change to test directory
    const originalCwd = process.cwd();
    process.chdir(this.testDir);
    
    try {
      // Record initial directory contents
      const initialContents = this.getDirectoryContents(this.testDir);
      
      // Test the prompt resolution system
      const { PromptResolver } = await import('../prompt-resolver.js');
      const resolver = new PromptResolver();
      
      // Perform multiple operations that might create files
      resolver.createActivationMessage('coach-agent');
      resolver.getPromptContent('coach-agent');
      resolver.createActivationMessage('builder-agent');
      resolver.getPromptContent('builder-agent');
      
      // Check that directory contents remain unchanged
      const finalContents = this.getDirectoryContents(this.testDir);
      
      if (JSON.stringify(initialContents.sort()) === JSON.stringify(finalContents.sort())) {
        console.log('   ✅ User working directory remains completely unmodified');
        this.passed++;
      } else {
        console.error('   ❌ User working directory was modified');
        console.error('   Initial:', initialContents);
        console.error('   Final:', finalContents);
        this.failed++;
      }
      
    } catch (error) {
      console.error('   ❌ Error testing working directory modification:', error.message);
      this.failed++;
    } finally {
      process.chdir(originalCwd);
    }
  }

  async testNoTemporaryFilesNeeded() {
    console.log('\n🗂️ Testing no temporary files or cleanup needed in user directory...');
    
    // Change to test directory
    const originalCwd = process.cwd();
    process.chdir(this.testDir);
    
    try {
      // Test the prompt resolution system
      const { PromptResolver } = await import('../prompt-resolver.js');
      const resolver = new PromptResolver();
      
      // Perform operations multiple times to test for temp file accumulation
      for (let i = 0; i < 3; i++) {
        resolver.createActivationMessage('coach-agent');
        resolver.getPromptContent('coach-agent');
      }
      
      // Check for any temporary files (common patterns)
      const tempPatterns = ['.tmp', '.temp', '.cache', 'tmp_', 'temp_'];
      const allFiles = this.getDirectoryContents(this.testDir);
      const tempFiles = allFiles.filter(file => 
        tempPatterns.some(pattern => file.includes(pattern))
      );
      
      if (tempFiles.length === 0) {
        console.log('   ✅ No temporary files created or cleanup needed');
        this.passed++;
      } else {
        console.error('   ❌ Temporary files found in user directory:', tempFiles);
        this.failed++;
      }
      
    } catch (error) {
      console.error('   ❌ Error testing temporary files:', error.message);
      this.failed++;
    } finally {
      process.chdir(originalCwd);
    }
  }

  printResults() {
    console.log(`\n📊 Test Results:`);
    console.log(`   ✅ Passed: ${this.passed}`);
    console.log(`   ❌ Failed: ${this.failed}`);
    
    if (this.failed > 0) {
      console.log('\n❌ Some tests failed. Prompt file accessibility not verified.');
      process.exit(1);
    } else {
      console.log('\n✅ All tests passed! Claude Code can access prompt files from any directory.');
    }
  }
}

// Run tests if called directly
if (import.meta.url === `file://${process.argv[1]}`) {
  const tester = new VerifyPromptFileAccessibleTest();
  await tester.runAllTests();
}

export { VerifyPromptFileAccessibleTest };