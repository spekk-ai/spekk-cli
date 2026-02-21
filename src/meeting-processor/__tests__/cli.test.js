#!/usr/bin/env node

import { join, dirname } from 'path';
import { fileURLToPath } from 'url';
import { existsSync } from 'fs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const projectRoot = join(__dirname, '../../..');

// Test suite for Meeting Processor Agent CLI
class MeetingProcessorCliTest {
  constructor() {
    this.passed = 0;
    this.failed = 0;
  }

  async runAllTests() {
    console.log('🧪 Running Meeting Processor CLI Tests...\n');

    await this.testPromptFileExists();
    await this.testPromptContentValid();
    await this.testPromptResolverIncludesAgent();
    await this.testCliModuleExists();

    this.printResults();
  }

  async testPromptFileExists() {
    console.log('📁 Testing meeting-processor prompt file exists...');

    const promptPath = join(projectRoot, 'specs/meeting-notes-to-specs/meeting-notes-to-specs.prompt.md');

    if (existsSync(promptPath)) {
      console.log('   ✅ Meeting processor prompt file exists');
      this.passed++;
    } else {
      console.error('   ❌ Meeting processor prompt file not found');
      console.error(`   Expected at: ${promptPath}`);
      this.failed++;
    }
  }

  async testPromptContentValid() {
    console.log('\n📄 Testing meeting-processor prompt content...');

    try {
      const { PromptResolver } = await import('../../cli/prompt-resolver.js');
      const resolver = new PromptResolver();

      const content = resolver.getPromptContent('meeting-processor-agent');

      // Check for required sections in the prompt
      const requiredSections = [
        'Meeting Processor Agent',
        'Features to Spec Files',
        'Wait for User Approval',
        'kebab-case',
        'priority',
        'Success Criteria'
      ];

      let allFound = true;
      for (const section of requiredSections) {
        if (!content.includes(section)) {
          console.error(`   ❌ Missing required content: "${section}"`);
          allFound = false;
        }
      }

      if (allFound && content.length > 1000) {
        console.log('   ✅ Meeting processor prompt has valid content');
        this.passed++;
      } else {
        console.error('   ❌ Meeting processor prompt content incomplete');
        this.failed++;
      }
    } catch (error) {
      console.error('   ❌ Error reading prompt content:', error.message);
      this.failed++;
    }
  }

  async testPromptResolverIncludesAgent() {
    console.log('\n🔧 Testing PromptResolver includes meeting-processor-agent...');

    try {
      const { PromptResolver } = await import('../../cli/prompt-resolver.js');
      const resolver = new PromptResolver();

      // Check that meeting-processor-agent is in the list
      const hasAgent = resolver.promptFiles.some(p => p.name === 'meeting-processor-agent');

      if (hasAgent) {
        console.log('   ✅ PromptResolver includes meeting-processor-agent');
        this.passed++;
      } else {
        console.error('   ❌ PromptResolver missing meeting-processor-agent');
        this.failed++;
      }

      // Test creating activation message
      try {
        const message = resolver.createActivationMessage('meeting-processor-agent');
        if (message.includes('Meeting Processor Agent')) {
          console.log('   ✅ Activation message created successfully');
          this.passed++;
        } else {
          console.error('   ❌ Activation message missing agent name');
          this.failed++;
        }
      } catch (error) {
        console.error('   ❌ Error creating activation message:', error.message);
        this.failed++;
      }
    } catch (error) {
      console.error('   ❌ Error testing PromptResolver:', error.message);
      this.failed++;
    }
  }

  async testCliModuleExists() {
    console.log('\n📦 Testing meeting-processor CLI module...');

    const cliPath = join(projectRoot, 'src/meeting-processor/cli.js');

    if (existsSync(cliPath)) {
      console.log('   ✅ CLI module file exists');
      this.passed++;

      // Test that it can be imported
      try {
        const module = await import('../cli.js');
        if (typeof module.launchMeetingProcessorAgent === 'function') {
          console.log('   ✅ CLI exports launchMeetingProcessorAgent function');
          this.passed++;
        } else {
          console.error('   ❌ launchMeetingProcessorAgent function not exported');
          this.failed++;
        }
      } catch (error) {
        console.error('   ❌ Error importing CLI module:', error.message);
        this.failed++;
      }
    } else {
      console.error('   ❌ CLI module file not found');
      console.error(`   Expected at: ${cliPath}`);
      this.failed++;
    }
  }

  printResults() {
    console.log(`\n📊 Test Results:`);
    console.log(`   ✅ Passed: ${this.passed}`);
    console.log(`   ❌ Failed: ${this.failed}`);

    if (this.failed > 0) {
      console.log('\n❌ Some tests failed.');
      process.exit(1);
    } else {
      console.log('\n✅ All tests passed!');
    }
  }
}

// Run tests if called directly
if (import.meta.url === `file://${process.argv[1]}`) {
  const tester = new MeetingProcessorCliTest();
  await tester.runAllTests();
}

export { MeetingProcessorCliTest };
