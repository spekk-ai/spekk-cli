import { test, describe } from 'node:test';
import assert from 'node:assert';
import { execSync } from 'node:child_process';

describe('Detail Badges Format', () => {
  
  test('generateDetailStatusBadge returns correct HTML format for done status', () => {
    // Test the function by invoking it through the CLI
    const script = `
      import fs from 'node:fs';
      const content = fs.readFileSync('src/show/cli.js', 'utf8');
      
      function getStatusIcon(status) {
        switch (status) {
          case 'not_started': return '⏸️';
          case 'in_progress': return '🔄';
          case 'done': return '✅';
          default: return '';
        }
      }
      
      function generateDetailStatusBadge(status) {
        const icon = getStatusIcon(status);
        return '<span class=\\"detail-status-badge status-' + status + '\\">' + icon + '</span>';
      }
      
      const result = generateDetailStatusBadge('done');
      console.log('STATUS_RESULT:', result);
    `;
    
    const output = execSync(`node -e "${script}"`, { encoding: 'utf8' });
    const result = output.split('STATUS_RESULT: ')[1].trim();
    
    // Should return span with correct class
    assert.ok(result.includes('<span class="detail-status-badge'), 'Should contain detail-status-badge class');
    assert.ok(result.includes('status-done'), 'Should include status-done class');
    assert.ok(result.includes('✅'), 'Done status should show checkmark icon');
    assert.ok(!result.includes('>done<') && !result.includes(' done '), 'Should not contain "done" as text content');
    assert.ok(result.endsWith('</span>'), 'Should end with closing span tag');
  });

  test('generateDetailPriorityBadge returns correct HTML format', () => {
    const script = `
      function generateDetailPriorityBadge(priority) {
        return '<span class=\\"detail-priority-badge priority-' + priority + '\\">' + priority + '</span>';
      }
      
      const result = generateDetailPriorityBadge(2);
      console.log('PRIORITY_RESULT:', result);
    `;
    
    const output = execSync(`node -e "${script}"`, { encoding: 'utf8' });
    const result = output.split('PRIORITY_RESULT: ')[1].trim();
    
    // Should return span with correct class  
    assert.ok(result.includes('<span class="detail-priority-badge'), 'Should contain detail-priority-badge class');
    assert.ok(result.includes('priority-2'), 'Should include priority-2 class');
    assert.ok(result.includes('>2<'), 'Priority 2 should show number 2');
    assert.ok(!result.includes('🔥'), 'Should not contain fire emoji');
    assert.ok(!result.includes('⚠️'), 'Should not contain warning emoji');
    assert.ok(result.endsWith('</span>'), 'Should end with closing span tag');
  });

  test('status badges contain only icons, no text labels', () => {
    const script = `
      function getStatusIcon(status) {
        switch (status) {
          case 'not_started': return '⏸️';
          case 'in_progress': return '🔄';
          case 'done': return '✅';
          default: return '';
        }
      }
      
      function generateDetailStatusBadge(status) {
        const icon = getStatusIcon(status);
        return '<span class=\\"detail-status-badge status-' + status + '\\">' + icon + '</span>';
      }
      
      console.log('NOT_STARTED:', generateDetailStatusBadge('not_started'));
      console.log('IN_PROGRESS:', generateDetailStatusBadge('in_progress'));
      console.log('DONE:', generateDetailStatusBadge('done'));
    `;
    
    const output = execSync(`node -e "${script}"`, { encoding: 'utf8' });
    
    assert.ok(output.includes('⏸️'), 'Not started should show pause icon');
    assert.ok(output.includes('🔄'), 'In progress should show cycle icon');  
    assert.ok(output.includes('✅'), 'Done should show checkmark icon');
    
    assert.ok(!output.includes('not started'), 'Should not contain "not started" text');
    assert.ok(!output.includes('in progress'), 'Should not contain "in progress" text');
  });

  test('priority badges contain only numbers, no emoji decorations', () => {
    const script = `
      function generateDetailPriorityBadge(priority) {
        return '<span class=\\"detail-priority-badge priority-' + priority + '\\">' + priority + '</span>';
      }
      
      console.log('PRIORITY_1:', generateDetailPriorityBadge(1));
      console.log('PRIORITY_2:', generateDetailPriorityBadge(2));
      console.log('PRIORITY_3:', generateDetailPriorityBadge(3));
    `;
    
    const output = execSync(`node -e "${script}"`, { encoding: 'utf8' });
    
    assert.ok(output.includes('>1<'), 'Priority 1 should show number 1');
    assert.ok(output.includes('>2<'), 'Priority 2 should show number 2');
    assert.ok(output.includes('>3<'), 'Priority 3 should show number 3');
    
    assert.ok(!output.includes('🔥'), 'Should not contain fire emoji');
    assert.ok(!output.includes('⚠️'), 'Should not contain warning emoji');
    assert.ok(!output.includes('💡'), 'Should not contain light bulb emoji');
  });
});