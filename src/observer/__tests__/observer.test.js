import { promises as fs } from 'node:fs';
import path from 'node:path';
import { ObserverAgent } from '../index.js';
import { parseAllSpecs } from '../../parser/index.js';

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

// Test: Observer detects spec conflicts - mutually exclusive requirements
export async function testObserverDetectsExclusiveRequirements() {
  const testDir = 'test-spec-conflicts';
  
  try {
    await createTestDir(testDir);
    
    // Create conflicting specs
    await createTestDir(path.join(testDir, 'specs', 'spec-a', 'assertions'));
    await createTestDir(path.join(testDir, 'specs', 'spec-b', 'assertions'));
    
    const specA = `---
id: spec-a
created: 2026-01-01T10:00:00Z
priority: 1
status: not_started
---

# Spec A
This spec requires the system to use SQLite database.`;

    const specB = `---
id: spec-b
created: 2026-01-01T11:00:00Z
priority: 1
status: not_started
---

# Spec B
This spec requires the system to use PostgreSQL database.`;

    await createTestFile(path.join(testDir, 'specs', 'spec-a', 'spec-a.md'), specA);
    await createTestFile(path.join(testDir, 'specs', 'spec-b', 'spec-b.md'), specB);
    
    const originalCwd = process.cwd();
    process.chdir(testDir);
    
    const observer = new ObserverAgent({ quiet: true });
    const issues = [];
    
    // Parse the test specs
    const { specs, assertions } = parseAllSpecs();
    
    await observer.checkSpecConflicts(specs, assertions, issues);
    
    const conflictIssues = issues.filter(i => i.type === 'spec_conflicts');
    
    if (conflictIssues.length === 0) {
      throw new Error('Expected to find database requirement conflicts');
    }
    
    console.log('✅ Observer detects exclusive requirements test passed');
    
    process.chdir(originalCwd);
    
  } finally {
    await cleanup(testDir);
  }
}

// Test: Observer detects conflicting file structure requirements
export async function testObserverDetectsFileStructureConflicts() {
  const testDir = 'test-file-conflicts';
  
  try {
    await createTestDir(testDir);
    await createTestDir(path.join(testDir, 'specs', 'spec-c', 'assertions'));
    await createTestDir(path.join(testDir, 'specs', 'spec-d', 'assertions'));
    
    const specC = `---
id: spec-c
created: 2026-01-01T10:00:00Z
priority: 1
status: not_started
---

# Spec C
Create src/config/database.js for database configuration.`;

    const specD = `---
id: spec-d
created: 2026-01-01T11:00:00Z
priority: 1
status: not_started
---

# Spec D
Create src/config/database.ts for database configuration using TypeScript.`;

    await createTestFile(path.join(testDir, 'specs', 'spec-c', 'spec-c.md'), specC);
    await createTestFile(path.join(testDir, 'specs', 'spec-d', 'spec-d.md'), specD);
    
    const originalCwd = process.cwd();
    process.chdir(testDir);
    
    const observer = new ObserverAgent({ quiet: true });
    const issues = [];
    
    // Parse the test specs
    const { specs, assertions } = parseAllSpecs();
    
    await observer.checkSpecConflicts(specs, assertions, issues);
    
    const conflictIssues = issues.filter(i => i.type === 'spec_conflicts');
    
    if (conflictIssues.length === 0) {
      throw new Error('Expected to find file structure conflicts');
    }
    
    console.log('✅ Observer detects file structure conflicts test passed');
    
    process.chdir(originalCwd);
    
  } finally {
    await cleanup(testDir);
  }
}

// Test: Observer detects priority conflicts
export async function testObserverDetectsPriorityConflicts() {
  const testDir = 'test-priority-conflicts';
  
  try {
    await createTestDir(testDir);
    await createTestDir(path.join(testDir, 'specs', 'spec-e', 'assertions'));
    await createTestDir(path.join(testDir, 'specs', 'spec-f', 'assertions'));
    
    const specE = `---
id: spec-e
created: 2026-01-01T10:00:00Z
priority: 1
status: not_started
---

# Spec E
Must be completed before any database work. Removes all database functionality.`;

    const specF = `---
id: spec-f
created: 2026-01-01T11:00:00Z
priority: 1
status: not_started
---

# Spec F
High priority database implementation. Requires database setup.`;

    await createTestFile(path.join(testDir, 'specs', 'spec-e', 'spec-e.md'), specE);
    await createTestFile(path.join(testDir, 'specs', 'spec-f', 'spec-f.md'), specF);
    
    const originalCwd = process.cwd();
    process.chdir(testDir);
    
    const observer = new ObserverAgent({ quiet: true });
    const issues = [];
    
    // Parse the test specs
    const { specs, assertions } = parseAllSpecs();
    
    await observer.checkSpecConflicts(specs, assertions, issues);
    
    const conflictIssues = issues.filter(i => i.type === 'spec_conflicts');
    
    if (conflictIssues.length === 0) {
      throw new Error('Expected to find priority conflicts');
    }
    
    console.log('✅ Observer detects priority conflicts test passed');
    
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

// Test: Observer compares assertion success criteria against actual code
export async function testObserverComparesAssertionSuccessCriteria() {
  const testDir = 'test-assertion-criteria';
  
  try {
    await createTestDir(testDir);
    await createTestDir(path.join(testDir, 'specs', 'test-spec', 'assertions'));
    
    // Create assertion with specific success criteria
    const assertionContent = `---
id: test-assertion
parent: test-spec
created: 2026-01-01T10:00:00Z
priority: 1
status: not_started
---

# Test Assertion

## Success Criteria

- [ ] Function \`calculateTotal\` exists in src/utils/math.js
- [ ] Function accepts two parameters (a, b)
- [ ] Function returns the sum of a + b
- [ ] Function has proper error handling for non-numeric inputs
`;

    // Create main spec file (required by parser)
    const specContent = `---
id: test-spec
created: 2026-01-01T10:00:00Z
priority: 1
status: not_started
---

# Test Spec

Main spec file.
`;

    await createTestFile(path.join(testDir, 'specs', 'test-spec', 'test-spec.md'), specContent);
    await createTestFile(path.join(testDir, 'specs', 'test-spec', 'assertions', 'test-assertion.md'), assertionContent);
    
    // Create incomplete implementation
    const incompleteCode = `// Incomplete implementation
export function calculateTotal(a) {
  return a; // Missing second parameter and logic
}`;
    
    await createTestFile(path.join(testDir, 'src', 'utils', 'math.js'), incompleteCode);
    
    const originalCwd = process.cwd();
    process.chdir(testDir);
    
    const observer = new ObserverAgent({ quiet: true });
    const issues = [];
    
    // Parse the test specs
    const { specs, assertions } = parseAllSpecs();
    
    await observer.checkAssertionSuccessCriteria(specs, assertions, issues);
    
    const misalignmentIssues = issues.filter(i => i.type === 'code_spec_misalignment');
    
    if (misalignmentIssues.length === 0) {
      throw new Error('Expected to find assertion success criteria misalignment');
    }
    
    const issue = misalignmentIssues[0];
    if (!issue.description.includes('calculateTotal')) {
      throw new Error('Issue should reference the specific function mentioned in criteria');
    }
    
    console.log('✅ Observer compares assertion success criteria test passed');
    
    process.chdir(originalCwd);
    
  } finally {
    await cleanup(testDir);
  }
}

// Test: Observer detects spec compression opportunities
export async function testObserverDetectsSpecCompressionOpportunities() {
  const testDir = 'test-compression-opportunities';
  
  try {
    await createTestDir(testDir);
    await createTestDir(path.join(testDir, 'specs', 'user-login', 'assertions'));
    await createTestDir(path.join(testDir, 'specs', 'user-auth', 'assertions'));
    await createTestDir(path.join(testDir, 'specs', 'authentication-system', 'assertions'));
    
    // Create three specs with overlapping functionality
    const userLoginSpec = `---
id: user-login
created: 2026-01-01T10:00:00Z
priority: 2
status: not_started
---

# User Login

Implements user login functionality with email and password validation.

## Success Criteria

- [ ] User can enter email and password
- [ ] System validates credentials against database
- [ ] Successful login redirects to dashboard
- [ ] Failed login shows error message
`;

    const userAuthSpec = `---
id: user-auth
created: 2026-01-01T11:00:00Z
priority: 2
status: not_started
---

# User Authentication

User authentication system with session management.

## Success Criteria

- [ ] User credentials are validated securely
- [ ] JWT tokens are generated for authenticated users
- [ ] Session management handles login/logout
- [ ] Password reset functionality available
`;

    const authSystemSpec = `---
id: authentication-system
created: 2026-01-01T12:00:00Z
priority: 2  
status: not_started
---

# Authentication System

Complete authentication system for user management.

## Success Criteria

- [ ] Login and logout functionality
- [ ] Password validation and reset
- [ ] Session token management
- [ ] User profile management
`;

    // Create assertion files for each spec (required by parser)
    const loginAssertion = `---
id: login-validation
parent: user-login
created: 2026-01-01T10:30:00Z
priority: 1
status: not_started
---

# Login Validation

User can login with email and password.
`;

    const authAssertion = `---
id: jwt-tokens
parent: user-auth
created: 2026-01-01T11:30:00Z
priority: 1
status: not_started
---

# JWT Token Generation

System generates JWT tokens for authenticated users.
`;

    const systemAssertion = `---
id: session-management
parent: authentication-system
created: 2026-01-01T12:30:00Z
priority: 1
status: not_started
---

# Session Management

Complete session management with login/logout.
`;

    await createTestFile(path.join(testDir, 'specs', 'user-login', 'user-login.md'), userLoginSpec);
    await createTestFile(path.join(testDir, 'specs', 'user-auth', 'user-auth.md'), userAuthSpec);
    await createTestFile(path.join(testDir, 'specs', 'authentication-system', 'authentication-system.md'), authSystemSpec);
    
    await createTestFile(path.join(testDir, 'specs', 'user-login', 'assertions', 'login-validation.md'), loginAssertion);
    await createTestFile(path.join(testDir, 'specs', 'user-auth', 'assertions', 'jwt-tokens.md'), authAssertion);
    await createTestFile(path.join(testDir, 'specs', 'authentication-system', 'assertions', 'session-management.md'), systemAssertion);
    
    const originalCwd = process.cwd();
    process.chdir(testDir);
    
    const observer = new ObserverAgent({ quiet: true });
    const issues = [];
    
    // Parse the test specs
    const { specs, assertions } = parseAllSpecs();
    
    await observer.checkSpecCompressionOpportunities(specs, assertions, issues);
    
    const compressionIssues = issues.filter(i => i.type === 'compression_opportunity');
    
    if (compressionIssues.length === 0) {
      throw new Error('Expected to find spec compression opportunities');
    }
    
    const issue = compressionIssues[0];
    
    // Should find overlap between any of the authentication-related specs
    const hasAuthSpecs = issue.affected_specs.some(spec => 
      ['user-login', 'user-auth', 'authentication-system'].includes(spec)
    );
    
    if (!hasAuthSpecs) {
      throw new Error('Compression issue should reference authentication-related specs');
    }
    
    if (!issue.description.includes('overlapping functionality') && !issue.description.includes('consolidated')) {
      throw new Error('Issue should reference overlapping functionality');
    }
    
    console.log('✅ Observer detects spec compression opportunities test passed');
    
    process.chdir(originalCwd);
    
  } finally {
    await cleanup(testDir);
  }
}

// Test: Observer detects when functions exist but don't meet spec requirements  
export async function testObserverDetectsFunctionRequirementMismatch() {
  const testDir = 'test-function-requirements';
  
  try {
    await createTestDir(testDir);
    await createTestDir(path.join(testDir, 'specs', 'api-spec', 'assertions'));
    
    // Create assertion with API endpoint requirements
    const assertionContent = `---
id: api-endpoints
parent: api-spec
created: 2026-01-01T10:00:00Z
priority: 1
status: not_started
---

# API Endpoints

## Success Criteria

- [ ] GET /users endpoint returns user list with pagination
- [ ] POST /users endpoint creates new user with validation
- [ ] Each user object includes id, name, email fields
- [ ] API responds with proper HTTP status codes
`;

    // Create main spec file (required by parser)
    const specContent = `---
id: api-spec
created: 2026-01-01T10:00:00Z
priority: 1
status: not_started
---

# API Spec

Main API spec file.
`;

    await createTestFile(path.join(testDir, 'specs', 'api-spec', 'api-spec.md'), specContent);
    await createTestFile(path.join(testDir, 'specs', 'api-spec', 'assertions', 'api-endpoints.md'), assertionContent);
    
    // Create implementation that exists but doesn't fully meet requirements
    const incompleteApiCode = `// API implementation exists but incomplete
app.get('/users', (req, res) => {
  // Missing pagination
  res.json(users);
});

// Missing POST endpoint entirely
// Missing proper error handling
`;
    
    await createTestFile(path.join(testDir, 'src', 'routes', 'users.js'), incompleteApiCode);
    
    const originalCwd = process.cwd();
    process.chdir(testDir);
    
    const observer = new ObserverAgent({ quiet: true });
    const issues = [];
    
    // Parse the test specs
    const { specs, assertions } = parseAllSpecs();
    
    await observer.checkFunctionRequirements(specs, assertions, issues);
    
    const misalignmentIssues = issues.filter(i => i.type === 'code_spec_misalignment');
    
    if (misalignmentIssues.length === 0) {
      throw new Error('Expected to find function requirement misalignment');
    }
    
    console.log('✅ Observer detects function requirement mismatch test passed');
    
    process.chdir(originalCwd);
    
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
    await testObserverDetectsExclusiveRequirements();
    await testObserverDetectsFileStructureConflicts();
    await testObserverDetectsPriorityConflicts();
    await testObserverCreatesObservationFiles();
    await testObserverComparesAssertionSuccessCriteria();
    await testObserverDetectsSpecCompressionOpportunities();
    await testObserverDetectsFunctionRequirementMismatch();
    
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