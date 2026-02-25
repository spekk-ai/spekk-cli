import { describe, it } from 'node:test';
import assert from 'node:assert';
import { generateSpecExplorerHTML } from '../show/cli.js';

const testSpecs = [{
  id: 'test-spec',
  title: 'Test Spec',
  priority: 1,
  status: 'in_progress',
  created: '2026-01-01T00:00:00Z',
  file: 'specs/test-spec/test-spec.md',
  content: 'Test content'
}];

const testAssertions = [
  {
    id: 'assertion-a',
    title: 'Assertion A',
    parent: 'test-spec',
    priority: 1,
    status: 'done',
    created: '2026-01-01T00:00:00Z',
    file: 'specs/test-spec/assertions/assertion-a.md',
    content: 'Assertion A content',
    branch: 'feature/test'
  },
  {
    id: 'assertion-b',
    title: 'Assertion B',
    parent: 'test-spec',
    priority: 1,
    status: 'in_progress',
    created: '2026-01-02T00:00:00Z',
    file: 'specs/test-spec/assertions/assertion-b.md',
    content: 'Assertion B content',
    branch: 'feature/test',
    dependsOn: 'assertion-a'
  }
];

function getHTML() {
  return generateSpecExplorerHTML(testSpecs, testAssertions);
}

describe('Clickable Metro Map Stations', () => {
  it('should toggle metro-station-current class without re-rendering SVG', () => {
    const html = getHTML();

    // updateMetroMap should toggle .metro-station-current on stations
    assert.ok(html.includes('metro-station-current'), 'Should reference metro-station-current class');
    assert.ok(html.includes("classList.add('metro-station-current')"), 'Should add metro-station-current class to clicked station');
    assert.ok(html.includes("classList.remove('metro-station-current')"), 'Should remove metro-station-current class from other stations');

    // Stations should use data-action="show-detail" for event delegation
    assert.ok(html.includes('data-action="show-detail"'), 'Stations should have data-action=show-detail');
    assert.ok(html.includes('data-assertion-id='), 'Stations should have data-assertion-id');
  });

  it('should have CSS for metro-station-current with glow, stroke, and stroke-width', () => {
    const html = getHTML();

    // Full CSS for current station per spec
    assert.ok(html.includes('.metro-station-current circle'), 'Should have .metro-station-current circle CSS rule');
    assert.ok(html.includes('stroke: #2563eb'), 'Current station should have blue stroke');
    assert.ok(html.includes('stroke-width: 4'), 'Current station should have stroke-width 4');
    assert.ok(html.includes('drop-shadow(0 0 6px rgba(59, 130, 246, 0.6))'), 'Current station should have glow filter');
  });

  it('should have pointer cursor on metro stations', () => {
    const html = getHTML();

    assert.ok(html.includes('.metro-station {'), 'Should have .metro-station CSS rule');
    assert.ok(html.includes('cursor: pointer'), 'Metro stations should have pointer cursor');
  });

  it('should update tree view selection and expand collapsed parent spec', () => {
    const html = getHTML();

    // Tree view selection logic
    assert.ok(html.includes("assertionItem.classList.add('selected')"), 'Should select assertion in tree view');
    assert.ok(html.includes('.closest(\'.assertions-list\')'), 'Should find parent assertions list');
    assert.ok(html.includes("assertionsList.classList.add('expanded')"), 'Should expand collapsed parent spec');
  });

  it('should preserve pan/zoom state when switching stations on same branch', () => {
    const html = getHTML();

    // Pan state preservation
    assert.ok(html.includes('metroMapState.panStates'), 'Should have pan state storage');
    assert.ok(html.includes('metroMapState.currentBranch'), 'Should track current branch');

    // When same branch, should not hide/show branch maps (only toggle classes)
    assert.ok(html.includes('metroMapState.currentBranch !== branch'), 'Should check if branch changed before switching maps');
  });
});
