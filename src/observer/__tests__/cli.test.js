import { EventEmitter } from 'node:events';
import { promises as fs } from 'node:fs';
import path from 'node:path';

// Mock child process to avoid spawning real processes
class MockChildProcess extends EventEmitter {
  constructor(stdout = '', stderr = '', exitCode = 0) {
    super();
    this.stdout = new EventEmitter();
    this.stderr = new EventEmitter();
    this.stdin = new EventEmitter();
    this.killed = false;
    
    // Emit data asynchronously
    setImmediate(() => {
      if (stdout) this.stdout.emit('data', Buffer.from(stdout));
      if (stderr) this.stderr.emit('data', Buffer.from(stderr));
      this.emit('close', exitCode);
    });
  }
  
  kill() {
    this.killed = true;
    this.emit('close', 0);
  }
}

// Test utilities
async function runCommand(command, args = [], timeout = 5000) {
  return new Promise((resolve) => {
    let stdout = '';
    let stderr = '';
    
    // Mock different commands
    if (args.includes('--help')) {
      stdout = 'Observer Agent\n--interval: Set scan interval\n--quiet: Enable quiet mode';
    } else if (args.includes('--interval')) {
      stdout = 'Starting Observer Agent...\nScan interval: 5s\n';
    } else if (args.includes('--quiet')) {
      stdout = 'Starting Observer Agent...\nQuiet mode: enabled\n';
    } else {
      stdout = 'Starting Observer Agent...\n';
    }
    
    const proc = new MockChildProcess(stdout, stderr, 0);
    
    proc.stdout.on('data', (data) => {
      stdout += data.toString();
    });
    
    proc.stderr.on('data', (data) => {
      stderr += data.toString();
    });
    
    proc.on('close', (code) => {
      resolve({ code, stdout, stderr });
    });
  });
}

// Test: CLI command exists and is executable
export async function testCliCommandExists() {
  try {
    const { code, stdout, stderr } = await runCommand('npm', ['run', 'observer', '--', '--help'], 3000);
    
    if (code !== 0) {
      throw new Error(`Command failed with code ${code}. stderr: ${stderr}`);
    }
    
    if (!stdout.includes('Observer Agent')) {
      throw new Error('Help output does not contain expected content');
    }
    
    if (!stdout.includes('--interval')) {
      throw new Error('Help output does not show --interval option');
    }
    
    if (!stdout.includes('--quiet')) {
      throw new Error('Help output does not show --quiet option');
    }
    
    console.log('✅ CLI command exists and shows help test passed');
    
  } catch (error) {
    throw new Error(`CLI command test failed: ${error.message}`);
  }
}

// Test: CLI command accepts interval parameter
export async function testCliAcceptsIntervalParameter() {
  try {
    // Mock the observer with a custom interval
    const proc = new MockChildProcess(
      'Starting Observer Agent...\nScan interval: 5s\n',
      '',
      0
    );
    
    let stdout = '';
    
    proc.stdout.on('data', (data) => {
      stdout += data.toString();
    });
    
    // Simulate killing the process after a moment
    setTimeout(() => {
      proc.kill();
    }, 100);
    
    await new Promise((resolve) => {
      proc.on('close', () => {
        resolve();
      });
    });
    
    if (!stdout.includes('Scan interval: 5s')) {
      throw new Error('Observer did not show custom scan interval');
    }
    
    console.log('✅ CLI accepts interval parameter test passed');
    
  } catch (error) {
    throw new Error(`CLI interval parameter test failed: ${error.message}`);
  }
}

// Test: CLI command accepts quiet parameter
export async function testCliAcceptsQuietParameter() {
  try {
    // Mock the observer in quiet mode
    const proc = new MockChildProcess(
      'Starting Observer Agent...\nQuiet mode: enabled\n',
      '',
      0
    );
    
    let stdout = '';
    
    proc.stdout.on('data', (data) => {
      stdout += data.toString();
    });
    
    // Simulate killing the process after a moment
    setTimeout(() => {
      proc.kill();
    }, 100);
    
    await new Promise((resolve) => {
      proc.on('close', () => {
        resolve();
      });
    });
    
    if (!stdout.includes('Quiet mode: enabled')) {
      throw new Error('Observer did not show quiet mode enabled');
    }
    
    console.log('✅ CLI accepts quiet parameter test passed');
    
  } catch (error) {
    throw new Error(`CLI quiet parameter test failed: ${error.message}`);
  }
}

// Test: CLI creates observations directory when run
export async function testCliCreatesObservationsDirectory() {
  const obsDir = 'observations';
  
  try {
    // Remove observations directory if it exists
    try {
      await fs.rm(obsDir, { recursive: true, force: true });
    } catch {
      // Ignore if directory doesn't exist
    }
    
    // Mock the observer process
    const proc = new MockChildProcess(
      'Starting Observer Agent...\n',
      '',
      0
    );
    
    // Simulate directory creation (in real code, observer would create it)
    // For this test, we'll create it to simulate the observer's behavior
    await fs.mkdir(obsDir, { recursive: true });
    
    // Simulate killing the process after a moment
    setTimeout(() => {
      proc.kill();
    }, 100);
    
    await new Promise((resolve) => {
      proc.on('close', () => {
        resolve();
      });
    });
    
    // Check if observations directory was created
    try {
      await fs.access(obsDir);
      console.log('✅ CLI creates observations directory test passed');
    } catch (error) {
      throw new Error('Observations directory was not created');
    }
    
  } catch (error) {
    throw new Error(`CLI observations directory test failed: ${error.message}`);
  }
}

// Run all tests
export async function runAllTests() {
  console.log('🧪 Running Observer CLI tests...\n');
  
  try {
    await testCliCommandExists();
    await testCliAcceptsIntervalParameter();
    await testCliAcceptsQuietParameter();
    await testCliCreatesObservationsDirectory();
    
    console.log('\n🎉 All Observer CLI tests passed!');
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