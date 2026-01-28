#!/usr/bin/env node

import { join, dirname } from 'path';
import { fileURLToPath } from 'url';
import { existsSync, mkdirSync, rmSync, writeFileSync } from 'fs';
import { execSync } from 'child_process';
import { spawn } from 'child_process';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const projectRoot = join(__dirname, '../../..');

// Test suite for CLI Prompt Resolution
class PromptResolutionTest {
  constructor() {
    this.testDir = join(projectRoot, 'tmp-test-dir');
    this.passed = 0;
    this.failed = 0;
  }

  async runAllTests() {
    console.log('🧪 Running CLI Prompt Resolution Tests...\n');
    
    // Setup test directory
    this.setupTestDirectory();
    
    try {
      // Test that prompt files exist in the spekk-cli installation
      await this.testPromptFilesExist();
      
      // Test that agent CLI commands can find prompt files from different directories
      await this.testPromptAccessFromDifferentDirectory();
      
      // Test cleanup is now included in the previous test
      
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

  async testPromptFilesExist() {
    console.log('📁 Testing prompt files exist in spekk-cli installation...');
    
    const promptFiles = [
      join(projectRoot, 'specs/coach-agent/coach-agent.prompt.md'),
      join(projectRoot, 'specs/builder-agent/builder-agent.prompt.md'), 
      join(projectRoot, 'specs/observer-agent/observer-agent.prompt.md')
    ];
    
    let allExist = true;
    for (const file of promptFiles) {
      if (!existsSync(file)) {
        console.error(`   ❌ Missing prompt file: ${file}`);
        allExist = false;
      }
    }
    
    if (allExist) {
      console.log('   ✅ All prompt files exist');
      this.passed++;
    } else {
      console.error('   ❌ Some prompt files are missing');
      this.failed++;
    }
  }

  async testPromptAccessFromDifferentDirectory() {
    console.log('\n📂 Testing prompt file access from different directory...');
    
    // Change to test directory
    const originalCwd = process.cwd();
    process.chdir(this.testDir);
    
    try {
      // Import and test the PromptResolver utility
      const { PromptResolver } = await import('../prompt-resolver.js');
      const resolver = new PromptResolver();
      
      const expectedPromptFiles = [
        'specs/coach-agent/coach-agent.prompt.md',
        'specs/builder-agent/builder-agent.prompt.md', 
        'specs/observer-agent/observer-agent.prompt.md'
      ];
      
      // Test setup of prompt files
      resolver.setupPromptFiles();
      
      // Check if all expected files are now accessible
      let allAccessible = expectedPromptFiles.every(file => {
        const localPath = join(process.cwd(), file);
        return existsSync(localPath);
      });
      
      if (allAccessible) {
        console.log('   ✅ All prompt files accessible after setup');
        this.passed++;
        
        // Test cleanup
        resolver.cleanupPromptFiles();
        
        // Verify files are cleaned up
        let allCleaned = expectedPromptFiles.every(file => {
          const localPath = join(process.cwd(), file);
          return !existsSync(localPath);
        });
        
        if (allCleaned) {
          console.log('   ✅ All prompt files cleaned up properly');
          this.passed++;
        } else {
          console.error('   ❌ Some prompt files not cleaned up');
          this.failed++;
        }
        
      } else {
        console.error('   ❌ Some prompt files not accessible after setup');
        this.failed++;
      }
      
    } catch (error) {
      console.error('   ❌ Error testing prompt access:', error.message);
      this.failed++;
    } finally {
      process.chdir(originalCwd);
    }
  }

  async testPromptFileCleanup() {
    console.log('\n🧹 Testing prompt file cleanup...');
    console.log('   ✅ Cleanup testing integrated with access test');
    this.passed++;
  }

  printResults() {
    console.log(`\n📊 Test Results:`);
    console.log(`   ✅ Passed: ${this.passed}`);
    console.log(`   ❌ Failed: ${this.failed}`);
    
    if (this.failed > 0) {
      console.log('\n❌ Some tests failed. Implementation needed.');
      process.exit(1);
    } else {
      console.log('\n✅ All tests passed!');
    }
  }
}

// Run tests if called directly
if (import.meta.url === `file://${process.argv[1]}`) {
  const tester = new PromptResolutionTest();
  await tester.runAllTests();
}

export { PromptResolutionTest };