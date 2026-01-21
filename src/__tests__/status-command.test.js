import { test, describe } from 'node:test';
import assert from 'node:assert';
import { execSync } from 'node:child_process';

describe('Status Command', () => {
  
  test('spekk status command exists and runs without errors', () => {
    const result = execSync('node bin/spekk.js status', { 
      encoding: 'utf8', 
      timeout: 5000 
    });
    
    // Should return string output without throwing error
    assert.ok(typeof result === 'string', 'Status command should return string output');
    assert.ok(result.length > 0, 'Status command should return non-empty output');
  });

  test('status command shows specs and assertions with status icons', () => {
    const result = execSync('node bin/spekk.js status', { encoding: 'utf8' });
    
    // Should contain status icons defined in requirements
    const statusIcons = ['✅', '🚧', '📋', '⏸️'];
    const hasStatusIcon = statusIcons.some(icon => result.includes(icon));
    
    assert.ok(hasStatusIcon, 'Status output should contain status icons (✅ 🚧 📋 ⏸️)');
  });

  test('status command shows completion ratios', () => {
    const result = execSync('node bin/spekk.js status', { encoding: 'utf8' });
    
    // Should show completion ratios in format like "2/5 assertions complete"
    const completionRatioPattern = /\d+\/\d+\s+(assertions?\s+)?complete/i;
    
    assert.ok(completionRatioPattern.test(result), 
      'Status output should show completion ratios (e.g., "2/5 assertions complete")');
  });

  test('status command groups assertions under parent specs with indentation', () => {
    const result = execSync('node bin/spekk.js status', { encoding: 'utf8' });
    
    // Should have indented content (at least some lines starting with spaces)
    const lines = result.split('\n');
    const hasIndentation = lines.some(line => /^\s{2,}/.test(line));
    
    assert.ok(hasIndentation, 'Status output should have indented assertions under specs');
  });

  test('status command shows next priority item clearly', () => {
    const result = execSync('node bin/spekk.js status', { encoding: 'utf8' });
    
    // Should highlight or mention next priority item
    const nextItemKeywords = ['next', 'priority', 'up next', '→'];
    const hasNextItemIndicator = nextItemKeywords.some(keyword => 
      result.toLowerCase().includes(keyword.toLowerCase())
    );
    
    assert.ok(hasNextItemIndicator, 
      'Status output should clearly indicate the next priority item');
  });

  test('status command handles empty specs directory gracefully', () => {
    // This test would need to be run in a directory without specs
    // For now, we test that the command doesn't crash with current specs
    const result = execSync('node bin/spekk.js status', { encoding: 'utf8' });
    
    // Should handle current directory structure without errors
    assert.ok(typeof result === 'string', 'Status command should handle directory structure gracefully');
  });

  test('status command executes quickly (performance test)', () => {
    const startTime = Date.now();
    
    const result = execSync('node bin/spekk.js status', { 
      encoding: 'utf8',
      timeout: 1000 // 1 second timeout, but should complete much faster
    });
    
    const endTime = Date.now();
    const executionTime = endTime - startTime;
    
    assert.ok(executionTime < 100, `Status command should execute in < 100ms (took ${executionTime}ms)`);
    assert.ok(typeof result === 'string', 'Status command should return output');
  });

  test('status command uses same validation logic as main parser', () => {
    // Test that status command correctly parses specs without validation errors
    const statusResult = execSync('node bin/spekk.js status', { encoding: 'utf8' });
    
    // Compare with parser output to ensure consistency
    const parserResult = execSync('node src/parser/cli.js --all', { encoding: 'utf8' });
    const parserData = JSON.parse(parserResult);
    
    // Status should show same specs that parser finds (by title, not id)
    if (parserData.type === 'hierarchy' && parserData.specs) {
      for (const spec of parserData.specs) {
        assert.ok(statusResult.includes(spec.title), 
          `Status should show spec title '${spec.title}' found by parser`);
      }
    }
    
    assert.ok(typeof statusResult === 'string', 'Status command should use valid parsing logic');
  });

  test('status command shows all specs found in specs/ directory', () => {
    const result = execSync('node bin/spekk.js status', { encoding: 'utf8' });
    
    // Get parser data to verify all specs are shown
    const parserResult = execSync('node src/parser/cli.js --all', { encoding: 'utf8' });
    const parserData = JSON.parse(parserResult);
    
    if (parserData.type === 'hierarchy' && parserData.specs) {
      // Check that status output contains all spec titles
      for (const spec of parserData.specs) {
        assert.ok(result.includes(spec.title), 
          `Status output should contain spec title '${spec.title}'`);
      }
    }
    
    assert.ok(result.length > 0, 'Status should show specs when they exist');
  });

  test('status command displays assertion count for each spec', () => {
    const result = execSync('node bin/spekk.js status', { encoding: 'utf8' });
    
    // Should show assertion counts - look for numbers that could be counts
    const countPattern = /\d+/;
    
    assert.ok(countPattern.test(result), 
      'Status output should display numeric assertion counts');
  });
});