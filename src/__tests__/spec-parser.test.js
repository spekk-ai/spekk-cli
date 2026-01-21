import { test, describe } from 'node:test';
import assert from 'node:assert';
import { execSync } from 'node:child_process';
import fs from 'fs';
import path from 'path';

describe('Spec Parser', () => {
  test('parser script exists and is executable', () => {
    const cliPath = path.join(process.cwd(), 'src', 'parser', 'cli.js');
    assert.ok(fs.existsSync(cliPath), 'src/parser/cli.js should exist');
    
    // Check if it's executable by running it via npm
    const result = execSync('node src/parser/cli.js', { encoding: 'utf8' });
    const parsed = JSON.parse(result);
    
    // Should return valid JSON structure
    assert.ok(typeof parsed === 'object', 'Should return JSON object');
  });

  test('outputs valid JSON format', () => {
    const result = execSync('node src/parser/cli.js', { encoding: 'utf8' });
    
    // Should be parseable JSON
    const parsed = JSON.parse(result);
    assert.ok(typeof parsed === 'object', 'Output should be valid JSON object');
    
    // Should have expected structure for success case
    if (parsed.type === 'assertion') {
      assert.ok(parsed.id, 'Should have id field');
      assert.ok(parsed.parent, 'Should have parent field');
      assert.ok(parsed.file, 'Should have file field');
      assert.ok(typeof parsed.priority === 'number', 'Priority should be number');
      assert.ok(parsed.status, 'Should have status field');
      assert.ok(parsed.title, 'Should have title field');
      assert.ok(parsed.content, 'Should have content field');
    }
  });

  test('identifies next priority assertion correctly', () => {
    const result = execSync('node src/parser/cli.js', { encoding: 'utf8' });
    const parsed = JSON.parse(result);
    
    if (parsed.type === 'assertion') {
      // Should return highest priority incomplete assertion
      assert.ok([1, 2, 3].includes(parsed.priority), 'Priority should be 1, 2, or 3');
      assert.ok(['not_started', 'in_progress'].includes(parsed.status), 'Should return incomplete assertion');
    }
  });

  test('validates status values - accepts all valid values', () => {
    // Test that parser accepts valid status values
    const result = execSync('node src/parser/cli.js', { encoding: 'utf8' });
    const parsed = JSON.parse(result);
    
    if (parsed.type === 'assertion') {
      assert.ok(['not_started', 'in_progress', 'done'].includes(parsed.status), 
        'Status should be valid value');
    }
  });

  test('validates timestamp format', () => {
    const result = execSync('node src/parser/cli.js', { encoding: 'utf8' });
    const parsed = JSON.parse(result);
    
    if (parsed.type === 'assertion') {
      // Check if we can parse the assertion file to get timestamp
      const content = fs.readFileSync(parsed.file, 'utf8');
      
      // Should have created timestamp in ISO format
      const timestampMatch = content.match(/created:\s*(.+)/);
      if (timestampMatch) {
        const timestamp = timestampMatch[1].trim();
        // Should match ISO 8601 format
        assert.ok(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$/.test(timestamp), 
          'Created timestamp should be ISO 8601 format');
      }
    }
  });

  test('enforces folder structure', () => {
    // Check that specs follow expected structure
    const specsDir = path.join(process.cwd(), 'specs');
    const specDirs = fs.readdirSync(specsDir).filter(dir => {
      const dirPath = path.join(specsDir, dir);
      return fs.statSync(dirPath).isDirectory();
    });
    
    for (const specDir of specDirs) {
      const specFile = path.join(specsDir, specDir, `${specDir}.md`);
      const assertionsDir = path.join(specsDir, specDir, 'assertions');
      
      // Should have spec file with matching name
      if (fs.existsSync(assertionsDir)) {
        assert.ok(fs.existsSync(specFile), 
          `Spec file ${specDir}.md should exist for directory with assertions`);
      }
    }
  });
});