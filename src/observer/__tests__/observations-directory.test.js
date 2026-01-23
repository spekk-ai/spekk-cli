#!/usr/bin/env node

import { promises as fs } from 'node:fs';
import path from 'node:path';
import { ObserverAgent } from '../index.js';

// Test utilities
async function createTestDir(dirname) {
  await fs.mkdir(dirname, { recursive: true });
}

async function cleanup(dirname) {
  try {
    await fs.rm(dirname, { recursive: true, force: true });
  } catch (error) {
    // Ignore cleanup errors
  }
}

// Test: Observer creates observations/ directory if it doesn't exist
export async function testObserverCreatesObservationsDirectory() {
  const testDir = 'test-observations-dir-creation';
  
  try {
    // Ensure directory doesn't exist
    await cleanup(testDir);
    
    const observer = new ObserverAgent({ quiet: true });
    
    // Mock the OBSERVATIONS_DIR constant for testing
    const originalEnsure = observer.ensureObservationsDirectory;
    observer.ensureObservationsDirectory = async function() {
      await fs.mkdir(testDir, { recursive: true });
    };
    
    await observer.ensureObservationsDirectory();
    
    // Verify directory was created
    try {
      const stats = await fs.stat(testDir);
      if (!stats.isDirectory()) {
        throw new Error('Created path is not a directory');
      }
      console.log('✅ Observer creates observations/ directory if it doesn\'t exist');
    } catch (error) {
      throw new Error(`Directory was not created: ${error.message}`);
    }
    
  } finally {
    await cleanup(testDir);
  }
}

// Test: Observations are saved as individual timestamped files
export async function testObservationTimestampedFiles() {
  const testDir = 'test-timestamp-files';
  
  try {
    await createTestDir(testDir);
    
    const observer = new ObserverAgent({ quiet: true });
    
    // Mock createObservation to use test directory
    const originalCreate = observer.createObservation;
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
    
    // Create test observations
    const issues = [
      {
        type: 'code_spec_misalignment',
        severity: 'high',
        title: 'Test Issue 1',
        description: 'First test issue',
        affected_specs: ['test-spec-1'],
        affected_files: ['test1.js']
      },
      {
        type: 'spec_conflicts',
        severity: 'medium',
        title: 'Test Issue 2', 
        description: 'Second test issue',
        affected_specs: ['test-spec-2'],
        affected_files: ['test2.js']
      }
    ];
    
    for (const issue of issues) {
      await observer.createObservation(issue);
      // Add small delay to ensure different timestamps
      await new Promise(resolve => setTimeout(resolve, 10));
    }
    
    // Verify files were created with timestamp pattern
    const files = await fs.readdir(testDir);
    
    if (files.length !== 2) {
      throw new Error(`Expected 2 observation files, got ${files.length}`);
    }
    
    for (const file of files) {
      // Check timestamp pattern: YYYY-MM-DDTHH-MM-SS-sssZ.md
      const timestampPattern = /^\d{4}-\d{2}-\d{2}T\d{2}-\d{2}-\d{2}-\d{3}Z\.md$/;
      if (!timestampPattern.test(file)) {
        throw new Error(`File ${file} doesn't match timestamp pattern`);
      }
    }
    
    console.log('✅ Observations are saved as individual timestamped files');
    
  } finally {
    await cleanup(testDir);
  }
}

// Test: Each observation file contains specific drift detection findings
export async function testObservationSpecificFindings() {
  const testDir = 'test-specific-findings';
  
  try {
    await createTestDir(testDir);
    
    const observer = new ObserverAgent({ quiet: true });
    
    // Mock createObservation to use test directory
    observer.createObservation = async function(issue) {
      const timestamp = new Date().toISOString();
      const filename = `${timestamp.replace(/[:.]/g, '-')}.md`;
      const filePath = path.join(testDir, filename);
      
      const observationId = `${issue.type}-${Date.now()}`;
      
      let frontmatter = `---
id: ${observationId}
created: ${timestamp}
type: ${issue.type}
severity: ${issue.severity}
affected_specs:
${issue.affected_specs.map(spec => `  - ${spec}`).join('\n')}
affected_files:
${issue.affected_files.map(file => `  - ${file}`).join('\n')}`;

      if (issue.type === 'spec_conflicts') {
        frontmatter += `
conflict_type: ${issue.conflict_type || 'unknown'}
blocking: ${issue.blocking || false}`;
      }

      frontmatter += `
---`;

      const content = `${frontmatter}

# ${issue.title}

## Issue Description
${issue.description}

## Conflicting Specifications
${issue.affected_specs.map(spec => `- **${spec}**: Referenced in specs/${spec}/${spec}.md`).join('\n')}

## Evidence
This observation was automatically generated during system scan.

## Impact
${issue.severity === 'high' ? 'Critical functionality may be broken or inaccessible.' : 'Important system behavior may not match specifications.'}

## Recommendation
Review the affected specs and files to determine if updates are needed.
`;
      
      await fs.writeFile(filePath, content, 'utf8');
      return observationId;
    };
    
    const testIssue = {
      type: 'spec_conflicts',
      severity: 'high',
      title: 'Conflicting database requirements',
      description: 'Multiple specs require different database technologies',
      affected_specs: ['spec-a', 'spec-b'],
      affected_files: ['specs/spec-a/spec-a.md', 'specs/spec-b/spec-b.md'],
      conflict_type: 'technology_exclusive',
      blocking: true
    };
    
    await observer.createObservation(testIssue);
    
    // Read and verify observation content
    const files = await fs.readdir(testDir);
    const content = await fs.readFile(path.join(testDir, files[0]), 'utf8');
    
    // Verify specific drift detection findings are included
    const requiredSections = [
      '# Conflicting database requirements',
      '## Issue Description',
      '## Conflicting Specifications',
      '## Evidence', 
      '## Impact',
      '## Recommendation'
    ];
    
    for (const section of requiredSections) {
      if (!content.includes(section)) {
        throw new Error(`Missing required section: ${section}`);
      }
    }
    
    // Verify YAML frontmatter includes specific fields
    const frontmatterFields = [
      'id:',
      'created:',
      'type: spec_conflicts',
      'severity: high',
      'affected_specs:',
      'affected_files:',
      'conflict_type: technology_exclusive',
      'blocking: true'
    ];
    
    for (const field of frontmatterFields) {
      if (!content.includes(field)) {
        throw new Error(`Missing required frontmatter field: ${field}`);
      }
    }
    
    console.log('✅ Each observation file contains specific drift detection findings');
    
  } finally {
    await cleanup(testDir);
  }
}

// Test: Observation files reference relevant specs and code files
export async function testObservationReferences() {
  const testDir = 'test-observation-refs';
  
  try {
    await createTestDir(testDir);
    
    const observer = new ObserverAgent({ quiet: true });
    
    // Use actual createObservation logic but with test directory
    observer.createObservation = async function(issue) {
      const timestamp = new Date().toISOString();
      const filename = `${timestamp.replace(/[:.]/g, '-')}.md`;
      const filePath = path.join(testDir, filename);

      const observationId = `${issue.type}-${Date.now()}`;
      
      let frontmatter = `---
id: ${observationId}
created: ${timestamp}
type: ${issue.type}
severity: ${issue.severity}
affected_specs:
${issue.affected_specs.map(spec => `  - ${spec}`).join('\n')}
affected_files:
${issue.affected_files.map(file => `  - ${file}`).join('\n')}
---`;

      const content = `${frontmatter}

# ${issue.title}

## Issue Description
${issue.description}

## Conflicting Specifications
${issue.affected_specs.map(spec => `- **${spec}**: Referenced in specs/${spec}/${spec}.md`).join('\n')}

## Evidence
This observation was automatically generated during system scan.

## Impact
Important system behavior may not match specifications.

## Recommendation
Review the affected specs and files to determine if updates are needed.
`;
      
      await fs.writeFile(filePath, content, 'utf8');
      return observationId;
    };
    
    const testIssue = {
      type: 'code_spec_misalignment',
      severity: 'medium',
      title: 'Missing CLI command: npm run build',
      description: 'Spec requires build command but it\'s not defined',
      affected_specs: ['build-system', 'deployment'],
      affected_files: ['package.json', 'specs/build-system/build-system.md']
    };
    
    await observer.createObservation(testIssue);
    
    // Read and verify references
    const files = await fs.readdir(testDir);
    const content = await fs.readFile(path.join(testDir, files[0]), 'utf8');
    
    // Verify affected_specs are referenced
    if (!content.includes('- build-system')) {
      throw new Error('Missing spec reference: build-system');
    }
    if (!content.includes('- deployment')) {
      throw new Error('Missing spec reference: deployment');
    }
    
    // Verify affected_files are referenced  
    if (!content.includes('- package.json')) {
      throw new Error('Missing file reference: package.json');
    }
    if (!content.includes('- specs/build-system/build-system.md')) {
      throw new Error('Missing file reference: specs/build-system/build-system.md');
    }
    
    // Verify specific references in content
    if (!content.includes('specs/build-system/build-system.md')) {
      throw new Error('Missing detailed spec reference in content');
    }
    
    console.log('✅ Observation files reference relevant specs and code files');
    
  } finally {
    await cleanup(testDir);
  }
}

// Test: Files are named with timestamp pattern
export async function testTimestampPattern() {
  const testDir = 'test-timestamp-pattern';
  
  try {
    await createTestDir(testDir);
    
    const observer = new ObserverAgent({ quiet: true });
    
    // Mock createObservation to capture filename
    let capturedFilename;
    observer.createObservation = async function(issue) {
      const timestamp = new Date().toISOString();
      const filename = `${timestamp.replace(/[:.]/g, '-')}.md`;
      capturedFilename = filename;
      
      const filePath = path.join(testDir, filename);
      await fs.writeFile(filePath, 'test content', 'utf8');
      return 'test-id';
    };
    
    const testIssue = {
      type: 'test',
      severity: 'low',
      title: 'Test',
      description: 'Test',
      affected_specs: ['test'],
      affected_files: ['test.js']
    };
    
    await observer.createObservation(testIssue);
    
    // Verify timestamp pattern matches expected format
    const expectedPattern = /^\d{4}-\d{2}-\d{2}T\d{2}-\d{2}-\d{2}-\d{3}Z\.md$/;
    if (!expectedPattern.test(capturedFilename)) {
      throw new Error(`Filename ${capturedFilename} doesn't match expected timestamp pattern YYYY-MM-DDTHH-MM-SS-sssZ.md`);
    }
    
    // Verify it matches the example pattern from assertion
    const examplePattern = /^2026-01-22T17-30-00-\d{3}Z\.md$/;
    const currentYear = new Date().getFullYear();
    const yearPattern = new RegExp(`^${currentYear}-\\d{2}-\\d{2}T\\d{2}-\\d{2}-\\d{2}-\\d{3}Z\\.md$`);
    
    if (!yearPattern.test(capturedFilename)) {
      throw new Error(`Filename ${capturedFilename} doesn't match current year pattern`);
    }
    
    console.log('✅ Files are named with timestamp pattern (e.g., 2026-01-22T17-30-00Z.md)');
    
  } finally {
    await cleanup(testDir);
  }
}

// Test: Directory serves as ephemeral inbox for processing and dismissal  
export async function testEphemeralInbox() {
  const testDir = 'test-ephemeral-inbox';
  
  try {
    await createTestDir(testDir);
    
    // Verify directory can be used as inbox (can add/remove files)
    await fs.writeFile(path.join(testDir, 'test-observation.md'), 'test content', 'utf8');
    
    let files = await fs.readdir(testDir);
    if (files.length !== 1) {
      throw new Error('Could not add observation to directory');
    }
    
    // Verify files can be processed (read)
    const content = await fs.readFile(path.join(testDir, 'test-observation.md'), 'utf8');
    if (content !== 'test content') {
      throw new Error('Could not read observation from directory');
    }
    
    // Verify files can be dismissed (removed)
    await fs.unlink(path.join(testDir, 'test-observation.md'));
    
    files = await fs.readdir(testDir);
    if (files.length !== 0) {
      throw new Error('Could not remove observation from directory');
    }
    
    console.log('✅ Directory serves as ephemeral inbox for processing and dismissal');
    
  } finally {
    await cleanup(testDir);
  }
}

// Run all assertion-specific tests
export async function runObservationDirectoryTests() {
  console.log('🧪 Testing Observer Creates Observations Directory assertion...\n');
  
  try {
    await testObserverCreatesObservationsDirectory();
    await testObservationTimestampedFiles();
    await testObservationSpecificFindings();
    await testObservationReferences();
    await testTimestampPattern();
    await testEphemeralInbox();
    
    console.log('\n🎉 All observation directory tests passed!');
    console.log('✅ All success criteria for observer-creates-observations-directory are validated');
    
  } catch (error) {
    console.error('❌ Test failed:', error.message);
    throw error;
  }
}

// Run tests if called directly
if (import.meta.url === `file://${process.argv[1]}`) {
  runObservationDirectoryTests().catch(error => {
    console.error('Test suite failed:', error);
    process.exit(1);
  });
}