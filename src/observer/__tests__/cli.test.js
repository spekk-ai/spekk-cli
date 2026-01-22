import { spawn } from 'node:child_process';
import { promises as fs } from 'node:fs';
import path from 'node:path';

// Test utilities
async function runCommand(command, args = [], timeout = 5000) {
  return new Promise((resolve, reject) => {
    const proc = spawn(command, args, {
      stdio: ['pipe', 'pipe', 'pipe']
    });
    
    let stdout = '';
    let stderr = '';
    
    proc.stdout.on('data', (data) => {
      stdout += data.toString();
    });
    
    proc.stderr.on('data', (data) => {
      stderr += data.toString();
    });
    
    const timer = setTimeout(() => {
      proc.kill('SIGTERM');
      reject(new Error(`Command timed out after ${timeout}ms`));
    }, timeout);
    
    proc.on('close', (code) => {
      clearTimeout(timer);
      resolve({ code, stdout, stderr });
    });
    
    proc.on('error', (error) => {
      clearTimeout(timer);
      reject(error);
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
    // Start the observer with a custom interval and kill it quickly
    const proc = spawn('npm', ['run', 'observer', '--', '--interval', '5'], {
      stdio: ['pipe', 'pipe', 'pipe']
    });
    
    let stdout = '';
    
    proc.stdout.on('data', (data) => {
      stdout += data.toString();
    });
    
    // Wait a moment for startup, then kill
    setTimeout(() => {
      proc.kill('SIGTERM');
    }, 1000);
    
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
    // Start the observer in quiet mode and kill it quickly
    const proc = spawn('npm', ['run', 'observer', '--', '--quiet'], {
      stdio: ['pipe', 'pipe', 'pipe']
    });
    
    let stdout = '';
    
    proc.stdout.on('data', (data) => {
      stdout += data.toString();
    });
    
    // Wait a moment for startup, then kill
    setTimeout(() => {
      proc.kill('SIGTERM');
    }, 1000);
    
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
    
    // Start the observer and kill it quickly
    const proc = spawn('npm', ['run', 'observer'], {
      stdio: ['pipe', 'pipe', 'pipe']
    });
    
    // Wait a moment for startup, then kill
    setTimeout(() => {
      proc.kill('SIGTERM');
    }, 2000);
    
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