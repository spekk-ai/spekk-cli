import { promises as fs } from 'node:fs';
import path from 'node:path';
import { ObserverAgent } from '../index.js';

// Test utilities
async function createTestDir(dirname) {
  await fs.mkdir(dirname, { recursive: true });
}

async function createTestFile(filepath, content) {
  await fs.mkdir(path.dirname(filepath), { recursive: true });
  await fs.writeFile(filepath, content, 'utf8');
}

async function cleanup(dirname) {
  try {
    await fs.rm(dirname, { recursive: true, force: true });
  } catch (error) {
    // Ignore cleanup errors
  }
}

// Test: Observer can be instantiated with default options
export async function testObserverInstantiation() {
  const observer = new ObserverAgent();
  
  if (observer.scanInterval !== 30) {
    throw new Error(`Expected scanInterval to be 30, got ${observer.scanInterval}`);
  }
  
  if (observer.quiet !== false) {
    throw new Error(`Expected quiet to be false, got ${observer.quiet}`);
  }
  
  if (observer.running !== false) {
    throw new Error(`Expected running to be false, got ${observer.running}`);
  }
  
  console.log('✅ Observer instantiation test passed');
}

// Test: Observer can be instantiated with custom options
export async function testObserverCustomOptions() {
  const observer = new ObserverAgent({ 
    scanInterval: 60, 
    quiet: true 
  });
  
  if (observer.scanInterval !== 60) {
    throw new Error(`Expected scanInterval to be 60, got ${observer.scanInterval}`);
  }
  
  if (observer.quiet !== true) {
    throw new Error(`Expected quiet to be true, got ${observer.quiet}`);
  }
  
  console.log('✅ Observer custom options test passed');
}

// Test: Observer creates observations directory
export async function testObserverCreatesObservationsDir() {
  const testDir = 'test-observations';
  
  // Cleanup first
  await cleanup(testDir);
  
  const observer = new ObserverAgent();
  
  // Mock the OBSERVATIONS_DIR for testing
  const originalDir = observer.constructor.prototype.ensureObservationsDirectory;
  observer.ensureObservationsDirectory = async function() {
    await fs.mkdir(testDir, { recursive: true });
  };
  
  await observer.ensureObservationsDirectory();
  
  // Check if directory exists
  try {
    await fs.access(testDir);
    console.log('✅ Observer creates observations directory test passed');
  } catch (error) {
    throw new Error('Observations directory was not created');
  } finally {
    await cleanup(testDir);
  }
}

// Test: Observer detects missing CLI commands
export async function testObserverDetectsMissingCliCommands() {
  const testDir = 'test-observer-cli';
  
  try {
    await createTestDir(testDir);
    
    // Create a test package.json without observer command
    const packageJsonContent = JSON.stringify({
      "scripts": {
        "test": "echo test",
        "start": "node index.js"
      }
    }, null, 2);
    
    await createTestFile(path.join(testDir, 'package.json'), packageJsonContent);
    
    // Change to test directory
    const originalCwd = process.cwd();
    process.chdir(testDir);
    
    const observer = new ObserverAgent({ quiet: true });
    
    // Test the isCommandMissing method
    const isMissing = await observer.isCommandMissing('observer');
    
    if (!isMissing) {
      throw new Error('Expected observer command to be missing');
    }
    
    // Test with existing command
    const testExists = await observer.isCommandMissing('test');
    
    if (testExists) {
      throw new Error('Expected test command to exist');
    }
    
    console.log('✅ Observer detects missing CLI commands test passed');
    
    // Restore original directory
    process.chdir(originalCwd);
    
  } finally {
    await cleanup(testDir);
  }
}

// Test: Observer creates observation files
export async function testObserverCreatesObservationFiles() {
  const testDir = 'test-observations-output';
  
  try {
    await createTestDir(testDir);
    
    const observer = new ObserverAgent({ quiet: true });
    
    // Mock OBSERVATIONS_DIR for testing
    const originalCreateObservation = observer.createObservation;
    observer.createObservation = async function(issue) {
      const timestamp = new Date().toISOString();
      const filename = `${timestamp.replace(/[:.]/g, '-')}.md`;
      const filePath = path.join(testDir, filename);
      
      const observationId = `${issue.type}-${Date.now()}`;
      
      const content = `---
id: ${observationId}
created: ${timestamp}
type: ${issue.type}
severity: ${issue.severity}
affected_specs:
${issue.affected_specs.map(spec => `  - ${spec}`).join('\n')}
affected_files:
${issue.affected_files.map(file => `  - ${file}`).join('\n')}
---

# ${issue.title}

## Issue Description
${issue.description}
`;
      
      await fs.writeFile(filePath, content, 'utf8');
      return observationId;
    };
    
    const testIssue = {
      type: 'code_spec_misalignment',
      severity: 'high',
      title: 'Test Issue',
      description: 'This is a test issue',
      affected_specs: ['test-spec'],
      affected_files: ['test.js']
    };
    
    await observer.createObservation(testIssue);
    
    // Check if observation file was created
    const files = await fs.readdir(testDir);
    
    if (files.length !== 1) {
      throw new Error(`Expected 1 observation file, got ${files.length}`);
    }
    
    const observationContent = await fs.readFile(path.join(testDir, files[0]), 'utf8');
    
    if (!observationContent.includes('Test Issue')) {
      throw new Error('Observation content does not contain expected title');
    }
    
    if (!observationContent.includes('code_spec_misalignment')) {
      throw new Error('Observation content does not contain expected type');
    }
    
    console.log('✅ Observer creates observation files test passed');
    
  } finally {
    await cleanup(testDir);
  }
}

// Run all tests
export async function runAllTests() {
  console.log('🧪 Running Observer Agent tests...\n');
  
  try {
    await testObserverInstantiation();
    await testObserverCustomOptions();
    await testObserverCreatesObservationsDir();
    await testObserverDetectsMissingCliCommands();
    await testObserverCreatesObservationFiles();
    
    console.log('\n🎉 All Observer Agent tests passed!');
  } catch (error) {
    console.error('❌ Test failed:', error.message);
    throw error;
  }
}

// Run tests if called directly
if (import.meta.url === `file://${process.argv[1]}`) {
  runAllTests().catch(error => {
    console.error('Test suite failed:', error);
    process.exit(1);
  });
}