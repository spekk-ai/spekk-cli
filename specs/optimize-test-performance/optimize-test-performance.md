---
id: optimize-test-performance
created: 2026-01-28T21:25:00Z
priority: 1
---

# Optimize Test Performance

## Overview

Tests are currently taking too long (timing out after 30s) and failing in CI. Individual tests are taking 5+ seconds when they should complete in milliseconds. The goal is to get the entire test suite running in under 10 seconds.

## Current Issues

**Performance Problems:**
- Tests timing out after 30 seconds in CI
- Individual tests taking 5+ seconds each
- Some tests taking over 5000ms (orchestration loops)
- 37 test files found, but too slow to complete

**Root Causes:**
- Tests may be spawning real processes instead of mocking
- File system operations without proper cleanup
- Network calls or external dependencies
- Inefficient test setup/teardown

## Target Performance

- **Total test suite:** < 30 seconds
- **Individual tests:** < 500ms each
- **Test suites:** < 3 seconds each
- **CI pipeline:** Reliable completion without timeouts