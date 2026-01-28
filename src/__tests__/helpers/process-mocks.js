/**
 * Centralized mock system for external process calls in tests
 * This avoids slow real process spawning and makes tests fast and reliable
 */

import { EventEmitter } from 'node:events';

// Mock execSync function to avoid real process spawning
export const mockExecSync = (command, options = {}) => {
  // Mock spekk status command
  if (command.includes('spekk.js status')) {
    return `SPECS STATUS:

1 ✅ Optimize Test Performance (4/4 assertions complete)
  1 ✅ Mock External Processes
  2 ✅ Use In-Memory Fixtures  
  3 ✅ Fast Assertion Validation
  4 ✅ Parallel Test Execution

2 🚧 Builder Agent (2/3 assertions complete)
  1 ✅ Core Implementation
  2 🚧 Test Integration
  3 📋 Documentation

NEXT PRIORITY ITEM:
Priority: 2
Title: Test Integration  
Status: 🚧
File: specs/builder-agent/assertions/test-integration.md`;
  }

  // Mock spekk next command / parser CLI
  if (command.includes('src/parser/cli.js') || command.includes('spekk next')) {
    const baseResponse = {
      type: 'assertion',
      id: 'mock-external-processes', 
      parent: 'optimize-test-performance',
      file: 'specs/optimize-test-performance/assertions/mock-external-processes.md',
      priority: 1,
      status: 'in_progress',
      title: 'Mock External Processes',
      content: '---\nid: mock-external-processes\nparent: optimize-test-performance\npriority: 1\nstatus: in_progress\n---\n\n# Mock External Processes\n\nReplace real process spawning with mocks for faster tests.',
      spec: {
        id: 'optimize-test-performance',
        file: 'specs/optimize-test-performance/optimize-test-performance.md',
        title: 'Optimize Test Performance'
      }
    };
    
    if (command.includes('--all')) {
      return JSON.stringify({
        type: 'hierarchy',
        specs: [
          {
            id: 'optimize-test-performance',
            title: 'Optimize Test Performance',
            status: 'in_progress',
            assertions: [
              {
                id: 'mock-external-processes',
                title: 'Mock External Processes',
                status: 'in_progress'
              }
            ]
          },
          {
            id: 'builder-agent',
            title: 'Builder Agent',
            status: 'in_progress', 
            assertions: [
              {
                id: 'core-implementation',
                title: 'Core Implementation',
                status: 'done'
              }
            ]
          }
        ]
      });
    }
    
    return JSON.stringify(baseResponse);
  }

  // Mock help commands
  if (command.includes('--help')) {
    if (command.includes('loop builder')) {
      return 'Usage: spekk loop builder [options]\nRun builder agent in loop mode';
    }
    if (command.includes('loop coach')) {
      return 'Usage: spekk loop coach [options]\nRun coach agent in loop mode';
    }
    return 'USAGE: spekk [command]\n\nCOMMANDS:\n  loop    Loop commands\n  next    Get next assertion\n  status  Show specs status';
  }

  // Mock git commands
  if (command.includes('git --version')) {
    return 'git version 2.39.0';
  }
  if (command.includes('git status --porcelain')) {
    return '';
  }

  // Mock coach CLI
  if (command.includes('src/coach/cli.js')) {
    return 'Coach Agent ready for input';
  }

  // Mock performance - all commands complete quickly  
  return 'mock-response';
};

// Mock spawn function for child processes
export const createMockChildProcess = (options = {}) => {
  const cp = new EventEmitter();
  cp.stdout = new EventEmitter();
  cp.stderr = new EventEmitter();
  cp.stdin = new EventEmitter();
  
  cp.kill = () => {
    setImmediate(() => cp.emit('exit', 0));
  };
  
  // Simulate process behavior
  if (options.immediate) {
    setImmediate(() => {
      if (options.stdout) {
        cp.stdout.emit('data', Buffer.from(options.stdout));
      }
      if (options.stderr) {
        cp.stderr.emit('data', Buffer.from(options.stderr));
      }
      cp.emit('exit', options.exitCode || 0);
    });
  }
  
  return cp;
};