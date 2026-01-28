#!/usr/bin/env node

import { join, dirname } from 'path';
import { fileURLToPath } from 'url';
import { existsSync, mkdirSync, rmSync } from 'fs';
import { readdirSync } from 'fs';

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
      
      // Test that PromptResolver can create activation messages from different directories
      await this.testPromptContentAccessFromDifferentDirectory();
      
      // Test that user directory stays clean (no copying)
      await this.testUserDirectoryStaysClean();
      
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
      console.log('   ✅ All prompt files exist in installation');
      this.passed++;
    } else {
      console.error('   ❌ Some prompt files are missing from installation');
      this.failed++;
    }
  }

  async testPromptContentAccessFromDifferentDirectory() {
    console.log('\n📂 Testing prompt content access from different directory...');
    
    // Change to test directory
    const originalCwd = process.cwd();
    process.chdir(this.testDir);
    
    try {
      // Import and test the PromptResolver utility
      const { PromptResolver } = await import('../prompt-resolver.js');
      const resolver = new PromptResolver();
      
      // Test that resolver can create activation messages
      const coachMessage = resolver.createActivationMessage('coach-agent');
      const builderMessage = resolver.createActivationMessage('builder-agent');
      const observerMessage = resolver.createActivationMessage('observer-agent');
      
      // Verify messages contain expected content
      if (coachMessage.includes('You are the Coach Agent') && coachMessage.length > 100) {
        console.log('   ✅ Coach agent activation message created');
        this.passed++;
      } else {
        console.error('   ❌ Coach agent activation message invalid');
        this.failed++;
      }
      
      if (builderMessage.includes('You are the Builder Agent') && builderMessage.length > 100) {
        console.log('   ✅ Builder agent activation message created');
        this.passed++;
      } else {
        console.error('   ❌ Builder agent activation message invalid');
        this.failed++;
      }
      
      if (observerMessage.includes('You are the Observer Agent') && observerMessage.length > 100) {
        console.log('   ✅ Observer agent activation message created');
        this.passed++;
      } else {
        console.error('   ❌ Observer agent activation message invalid');
        this.failed++;
      }
      
      // Test individual prompt content access
      try {
        const coachContent = resolver.getPromptContent('coach-agent');
        if (coachContent.length > 50) {
          console.log('   ✅ Coach agent prompt content readable');
          this.passed++;
        } else {
          console.error('   ❌ Coach agent prompt content too short or empty');
          this.failed++;
        }
      } catch (error) {
        console.error('   ❌ Error reading coach agent prompt:', error.message);
        this.failed++;
      }
      
    } catch (error) {
      console.error('   ❌ Error testing prompt content access:', error.message);
      this.failed++;
    } finally {
      process.chdir(originalCwd);
    }
  }

  async testUserDirectoryStaysClean() {
    console.log('\n🧹 Testing user directory stays clean (no copying)...');
    
    // Change to test directory
    const originalCwd = process.cwd();
    process.chdir(this.testDir);
    
    try {
      // Record initial directory contents
      const initialContents = this.getDirectoryContents(this.testDir);
      
      // Import and use PromptResolver 
      const { PromptResolver } = await import('../prompt-resolver.js');
      const resolver = new PromptResolver();
      
      // Create activation messages (this should NOT create any local files)
      resolver.createActivationMessage('coach-agent');
      resolver.createActivationMessage('builder-agent'); 
      resolver.createActivationMessage('observer-agent');
      
      // Check that no new files were created
      const finalContents = this.getDirectoryContents(this.testDir);
      
      if (JSON.stringify(initialContents.sort()) === JSON.stringify(finalContents.sort())) {
        console.log('   ✅ User directory remains clean - no files copied');
        this.passed++;
      } else {
        console.error('   ❌ User directory was modified');
        console.error('   Initial:', initialContents);
        console.error('   Final:', finalContents);
        this.failed++;
      }
      
      // Specifically check that no 'specs' directory was created
      const specsDir = join(this.testDir, 'specs');
      if (!existsSync(specsDir)) {
        console.log('   ✅ No specs directory created in user directory');
        this.passed++;
      } else {
        console.error('   ❌ specs directory was created in user directory (prohibited)');
        this.failed++;
      }
      
    } catch (error) {
      console.error('   ❌ Error testing directory cleanliness:', error.message);
      this.failed++;
    } finally {
      process.chdir(originalCwd);
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