#!/usr/bin/env node

import { join, dirname } from 'path';
import { fileURLToPath } from 'url';
import { existsSync, mkdirSync, rmSync } from 'fs';
import { spawn } from 'child_process';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const projectRoot = join(__dirname, '../../..');
const spekkBinary = join(projectRoot, 'bin/spekk.js');

class ExternalDirectoryIntegrationTest {
  constructor() {
    this.testDir = '/tmp/test-spekk-external';
    this.passed = 0;
    this.failed = 0;
  }

  async runAllTests() {
    console.log('🧪 Running External Directory Integration Tests...\n');
    
    // Setup test directory
    this.setupTestDirectory();
    
    try {
      // Test spekk coach from external directory
      await this.testSpekkCoachFromExternal();
      
      // Test spekk builder from external directory  
      await this.testSpekkBuilderFromExternal();
      
      // Test spekk observer from external directory
      await this.testSpekkObserverFromExternal();
      
      // Test default spekk command (parser)
      await this.testSpekkParserFromExternal();
      
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

  async runCommandInDirectory(command, args = [], cwd = this.testDir, timeout = 5000) {
    return new Promise((resolve, reject) => {
      const process = spawn(command, args, { 
        cwd,
        stdio: ['pipe', 'pipe', 'pipe']
      });
      
      let stdout = '';
      let stderr = '';
      
      process.stdout.on('data', (data) => {
        stdout += data.toString();
      });
      
      process.stderr.on('data', (data) => {
        stderr += data.toString();
      });
      
      // Set a timeout to prevent hanging
      const timer = setTimeout(() => {
        process.kill('SIGTERM');
        reject(new Error('Command timed out'));
      }, timeout);
      
      process.on('exit', (code) => {
        clearTimeout(timer);
        resolve({
          code,
          stdout,
          stderr,
          success: code === 0
        });
      });
      
      process.on('error', (error) => {
        clearTimeout(timer);
        reject(error);
      });
      
      // Send a quick exit for interactive commands
      setTimeout(() => {
        if (process.stdin && !process.killed) {
          process.stdin.write('\x03'); // Ctrl+C
        }
      }, 1000);
    });
  }

  async testSpekkCoachFromExternal() {
    console.log('🎯 Testing spekk coach prompt resolution from external directory...');
    
    try {
      // Test the prompt resolution directly without launching Claude Code
      const { PromptResolver } = await import('../prompt-resolver.js');
      const resolver = new PromptResolver();
      
      // Change to external directory for the test
      const originalCwd = process.cwd();
      process.chdir(this.testDir);
      
      try {
        const activationMessage = resolver.createActivationMessage('coach-agent');
        
        if (activationMessage.includes('You are the Coach Agent') && activationMessage.length > 100) {
          console.log('   ✅ Coach agent prompt resolved from external directory');
          this.passed++;
        } else {
          console.error('   ❌ Coach agent prompt resolution failed');
          this.failed++;
        }
        
      } finally {
        process.chdir(originalCwd);
      }
      
    } catch (error) {
      console.error('   ❌ Error testing coach prompt resolution:', error.message);
      this.failed++;
    }
  }

  async testSpekkBuilderFromExternal() {
    console.log('\n🔨 Testing spekk builder prompt resolution from external directory...');
    
    try {
      // Test the prompt resolution directly without launching Claude Code
      const { PromptResolver } = await import('../prompt-resolver.js');
      const resolver = new PromptResolver();
      
      // Change to external directory for the test
      const originalCwd = process.cwd();
      process.chdir(this.testDir);
      
      try {
        const activationMessage = resolver.createActivationMessage('builder-agent');
        
        if (activationMessage.includes('You are the Builder Agent') && activationMessage.length > 100) {
          console.log('   ✅ Builder agent prompt resolved from external directory');
          this.passed++;
        } else {
          console.error('   ❌ Builder agent prompt resolution failed');
          this.failed++;
        }
        
      } finally {
        process.chdir(originalCwd);
      }
      
    } catch (error) {
      console.error('   ❌ Error testing builder prompt resolution:', error.message);
      this.failed++;
    }
  }

  async testSpekkObserverFromExternal() {
    console.log('\n👀 Testing spekk observer prompt resolution from external directory...');
    
    try {
      // Test the prompt resolution directly without launching Claude Code
      const { PromptResolver } = await import('../prompt-resolver.js');
      const resolver = new PromptResolver();
      
      // Change to external directory for the test
      const originalCwd = process.cwd();
      process.chdir(this.testDir);
      
      try {
        const activationMessage = resolver.createActivationMessage('observer-agent');
        
        if (activationMessage.includes('You are the Observer Agent') && activationMessage.length > 100) {
          console.log('   ✅ Observer agent prompt resolved from external directory');
          this.passed++;
        } else {
          console.error('   ❌ Observer agent prompt resolution failed');
          this.failed++;
        }
        
      } finally {
        process.chdir(originalCwd);
      }
      
    } catch (error) {
      console.error('   ❌ Error testing observer prompt resolution:', error.message);
      this.failed++;
    }
  }

  async testSpekkParserFromExternal() {
    console.log('\n📋 Testing spekk parser (default) from external directory...');
    
    try {
      const result = await this.runCommandInDirectory('node', [spekkBinary], this.testDir, 3000);
      
      // Parser should try to parse specs but fail gracefully since no specs in external dir
      // Check that it doesn't crash with path resolution errors
      if (!result.stderr.includes('Prompt file not found') && 
          !result.stderr.includes('file not found') &&
          !result.stderr.includes('ENOENT')) {
        console.log('   ✅ Parser runs without path resolution errors');
        this.passed++;
      } else {
        console.error('   ❌ Parser has path resolution errors');
        console.error('   stderr:', result.stderr);
        this.failed++;
      }
      
    } catch (error) {
      console.error('   ❌ Error running spekk parser:', error.message);
      this.failed++;
    }
  }

  printResults() {
    console.log(`\n📊 Integration Test Results:`);
    console.log(`   ✅ Passed: ${this.passed}`);
    console.log(`   ❌ Failed: ${this.failed}`);
    
    if (this.failed > 0) {
      console.log('\n❌ Some integration tests failed. CLI commands may not work from external directories.');
      process.exit(1);
    } else {
      console.log('\n✅ All integration tests passed! CLI works from any directory.');
    }
  }
}

// Run tests if called directly
if (import.meta.url === `file://${process.argv[1]}`) {
  const tester = new ExternalDirectoryIntegrationTest();
  await tester.runAllTests();
}

export { ExternalDirectoryIntegrationTest };