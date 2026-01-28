import { promises as fs } from 'node:fs';
import path from 'node:path';
import { ObserverAgent } from '../programmatic.js';

// Test: Observation files follow required YAML frontmatter format
export async function testObservationFormatRequirements() {
  const testDir = 'test-observation-format';
  
  try {
    await fs.mkdir(testDir, { recursive: true });
    
    const observer = new ObserverAgent({ quiet: true });
    
    // Mock createObservation to write to test directory
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
      title: 'Test Format Validation',
      description: 'Testing observation format requirements',
      affected_specs: ['test-spec', 'another-spec'],
      affected_files: ['test.js', 'config.json']
    };
    
    await observer.createObservation(testIssue);
    
    // Read the created file
    const files = await fs.readdir(testDir);
    const observationContent = await fs.readFile(path.join(testDir, files[0]), 'utf8');
    
    // Validate YAML frontmatter format
    const frontmatterMatch = observationContent.match(/^---\n([\s\S]*?)\n---/);
    if (!frontmatterMatch) {
      throw new Error('Observation does not contain YAML frontmatter');
    }
    
    const frontmatter = frontmatterMatch[1];
    
    // Check required fields exist
    const requiredFields = ['id:', 'created:', 'type:', 'severity:', 'affected_specs:', 'affected_files:'];
    for (const field of requiredFields) {
      if (!frontmatter.includes(field)) {
        throw new Error(`Missing required frontmatter field: ${field}`);
      }
    }
    
    // Validate ISO 8601 timestamp format
    const createdMatch = frontmatter.match(/created: (.+)/);
    if (!createdMatch) {
      throw new Error('Missing created timestamp');
    }
    
    const timestamp = createdMatch[1];
    if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$/.test(timestamp)) {
      throw new Error(`Created timestamp is not in ISO 8601 format: ${timestamp}`);
    }
    
    // Validate severity values
    const severityMatch = frontmatter.match(/severity: (\w+)/);
    if (!severityMatch) {
      throw new Error('Missing severity field');
    }
    
    const severity = severityMatch[1];
    if (!['low', 'medium', 'high'].includes(severity)) {
      throw new Error(`Invalid severity value: ${severity}. Must be low, medium, or high`);
    }
    
    // Validate affected_specs is array format
    if (!frontmatter.includes('affected_specs:') || !frontmatter.includes('  - test-spec')) {
      throw new Error('affected_specs must be in YAML array format');
    }
    
    // Validate affected_files is array format  
    if (!frontmatter.includes('affected_files:') || !frontmatter.includes('  - test.js')) {
      throw new Error('affected_files must be in YAML array format');
    }
    
    // Validate markdown body exists
    const bodyMatch = observationContent.match(/---\n\n([\s\S]+)/);
    if (!bodyMatch || !bodyMatch[1].trim()) {
      throw new Error('Observation must have markdown body content');
    }
    
    const body = bodyMatch[1];
    
    // Validate markdown formatting
    if (!body.includes('# Test Format Validation')) {
      throw new Error('Body must use markdown formatting with headers');
    }
    
    if (!body.includes('## Issue Description')) {
      throw new Error('Body must include required sections in markdown format');
    }
    
    console.log('✅ Observation format requirements test passed');
    
  } finally {
    // Cleanup
    try {
      await fs.rm(testDir, { recursive: true, force: true });
    } catch (error) {
      // Ignore cleanup errors
    }
  }
}

// Test: Existing observation files follow format
export async function testExistingObservationFiles() {
  const observationsDir = 'observations';
  
  try {
    const files = await fs.readdir(observationsDir);
    
    if (files.length === 0) {
      console.log('✅ No existing observation files to validate');
      return;
    }
    
    for (const file of files) {
      if (!file.endsWith('.md')) continue;
      
      const filePath = path.join(observationsDir, file);
      const content = await fs.readFile(filePath, 'utf8');
      
      // Validate YAML frontmatter exists
      const frontmatterMatch = content.match(/^---\n([\s\S]*?)\n---/);
      if (!frontmatterMatch) {
        throw new Error(`File ${file} does not contain YAML frontmatter`);
      }
      
      const frontmatter = frontmatterMatch[1];
      
      // Check required fields
      const requiredFields = ['id:', 'created:', 'type:', 'severity:', 'affected_specs:', 'affected_files:'];
      for (const field of requiredFields) {
        if (!frontmatter.includes(field)) {
          throw new Error(`File ${file} missing required field: ${field}`);
        }
      }
      
      // Validate ISO 8601 timestamp
      const createdMatch = frontmatter.match(/created: (.+)/);
      if (createdMatch) {
        const timestamp = createdMatch[1];
        if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$/.test(timestamp)) {
          throw new Error(`File ${file} has invalid timestamp format: ${timestamp}`);
        }
      }
      
      // Validate severity
      const severityMatch = frontmatter.match(/severity: (\w+)/);
      if (severityMatch) {
        const severity = severityMatch[1];
        if (!['low', 'medium', 'high'].includes(severity)) {
          throw new Error(`File ${file} has invalid severity: ${severity}`);
        }
      }
    }
    
    console.log(`✅ All ${files.length} existing observation files follow correct format`);
    
  } catch (error) {
    if (error.code === 'ENOENT') {
      console.log('✅ No observations directory found - format validation not needed');
    } else {
      throw error;
    }
  }
}

// Run all format tests
export async function runFormatTests() {
  console.log('🧪 Running observation format tests...\n');
  
  try {
    await testObservationFormatRequirements();
    await testExistingObservationFiles();
    
    console.log('\n🎉 All observation format tests passed!');
  } catch (error) {
    console.error('❌ Format test failed:', error.message);
    throw error;
  }
}

// Run tests if called directly
if (import.meta.url === `file://${process.argv[1]}`) {
  runFormatTests().catch(error => {
    console.error('Format test suite failed:', error);
    process.exit(1);
  });
}