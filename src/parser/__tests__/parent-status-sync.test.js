#!/usr/bin/env node

import { test } from 'node:test';
import assert from 'node:assert';
import { computeParentStatus } from '../index.js';

test('Parent Status Synchronization', async (t) => {
  
  await t.test('parent status with all done children', () => {
    const assertions = [
      { id: 'a1', parent: 'spec1', status: 'done' },
      { id: 'a2', parent: 'spec1', status: 'done' },
      { id: 'a3', parent: 'spec1', status: 'done' }
    ];
    
    const result = computeParentStatus('spec1', assertions);
    assert.strictEqual(result, 'done');
  });

  await t.test('parent status with some in_progress children', () => {
    const assertions = [
      { id: 'a1', parent: 'spec1', status: 'done' },
      { id: 'a2', parent: 'spec1', status: 'in_progress' },
      { id: 'a3', parent: 'spec1', status: 'done' }
    ];
    
    const result = computeParentStatus('spec1', assertions);
    assert.strictEqual(result, 'in_progress');
  });

  await t.test('parent status with some not_started children', () => {
    const assertions = [
      { id: 'a1', parent: 'spec1', status: 'done' },
      { id: 'a2', parent: 'spec1', status: 'not_started' },
      { id: 'a3', parent: 'spec1', status: 'done' }
    ];
    
    const result = computeParentStatus('spec1', assertions);
    assert.strictEqual(result, 'in_progress');
  });

  await t.test('parent status with failed children has priority', () => {
    const assertions = [
      { id: 'a1', parent: 'spec1', status: 'done' },
      { id: 'a2', parent: 'spec1', status: 'failed' },
      { id: 'a3', parent: 'spec1', status: 'in_progress' }
    ];
    
    const result = computeParentStatus('spec1', assertions);
    assert.strictEqual(result, 'failed');
  });

  await t.test('parent status with no children', () => {
    const assertions = [];
    
    const result = computeParentStatus('spec1', assertions);
    assert.strictEqual(result, 'not_started');
  });

  await t.test('parent status ignores draft children', () => {
    const assertions = [
      { id: 'a1', parent: 'spec1', status: 'done' },
      { id: 'a2', parent: 'spec1', status: 'draft' },
      { id: 'a3', parent: 'spec1', status: 'done' }
    ];
    
    const result = computeParentStatus('spec1', assertions);
    assert.strictEqual(result, 'done');
  });

  await t.test('parent status with only draft children', () => {
    const assertions = [
      { id: 'a1', parent: 'spec1', status: 'draft' },
      { id: 'a2', parent: 'spec1', status: 'draft' }
    ];
    
    const result = computeParentStatus('spec1', assertions);
    assert.strictEqual(result, 'not_started');
  });

  await t.test('parent status mixed with failed takes precedence over in_progress', () => {
    const assertions = [
      { id: 'a1', parent: 'spec1', status: 'not_started' },
      { id: 'a2', parent: 'spec1', status: 'failed' },
      { id: 'a3', parent: 'spec1', status: 'in_progress' },
      { id: 'a4', parent: 'spec1', status: 'done' }
    ];
    
    const result = computeParentStatus('spec1', assertions);
    assert.strictEqual(result, 'failed');
  });

});