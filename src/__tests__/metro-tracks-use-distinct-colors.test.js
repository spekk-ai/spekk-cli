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

// Two independent roots (A and D), where A has two children (B, C) branching from it
const branchingAssertions = [
  {
    id: 'root-a',
    title: 'Root A',
    parent: 'test-spec',
    priority: 1,
    status: 'done',
    created: '2026-01-01T00:00:00Z',
    file: 'specs/test-spec/assertions/root-a.md',
    content: 'Root A content',
    branch: 'feature/test'
  },
  {
    id: 'child-b',
    title: 'Child B',
    parent: 'test-spec',
    priority: 1,
    status: 'not_started',
    created: '2026-01-02T00:00:00Z',
    file: 'specs/test-spec/assertions/child-b.md',
    content: 'Child B content',
    branch: 'feature/test',
    dependsOn: 'root-a'
  },
  {
    id: 'child-c',
    title: 'Child C',
    parent: 'test-spec',
    priority: 1,
    status: 'not_started',
    created: '2026-01-03T00:00:00Z',
    file: 'specs/test-spec/assertions/child-c.md',
    content: 'Child C content',
    branch: 'feature/test',
    dependsOn: 'root-a'
  },
  {
    id: 'root-d',
    title: 'Root D',
    parent: 'test-spec',
    priority: 1,
    status: 'not_started',
    created: '2026-01-04T00:00:00Z',
    file: 'specs/test-spec/assertions/root-d.md',
    content: 'Root D content',
    branch: 'feature/test'
  }
];

function extractPositions(html) {
  // Extract transform="translate(x, y)" from metro-station groups
  const positions = new Map();
  const stationRegex = /data-assertion-id="([^"]+)"[^>]*transform="translate\((\d+(?:\.\d+)?),\s*(\d+(?:\.\d+)?)\)"/g;
  let match;
  while ((match = stationRegex.exec(html)) !== null) {
    positions.set(match[1], { x: parseFloat(match[2]), y: parseFloat(match[3]) });
  }
  return positions;
}

function extractDependencyLineColors(html) {
  // Extract stroke colors from dependency lines
  const colors = [];
  const lineRegex = /class="metro-dependency"[^>]*stroke="([^"]+)"/g;
  let match;
  while ((match = lineRegex.exec(html)) !== null) {
    colors.push(match[1]);
  }
  return colors;
}

describe('Tree-Stacking Layout: Independent Trees Do Not Share Y-Space', () => {
  it('should place independent roots in separate vertical bands', () => {
    const html = generateSpecExplorerHTML(testSpecs, branchingAssertions);
    const positions = extractPositions(html);

    const rootAPos = positions.get('root-a');
    const rootDPos = positions.get('root-d');
    const childBPos = positions.get('child-b');
    const childCPos = positions.get('child-c');

    assert.ok(rootAPos, 'root-a should have a position');
    assert.ok(rootDPos, 'root-d should have a position');
    assert.ok(childBPos, 'child-b should have a position');
    assert.ok(childCPos, 'child-c should have a position');

    // Tree 1 (root-a with children B, C) should all be above tree 2 (root-d)
    const tree1MaxY = Math.max(rootAPos.y, childBPos.y, childCPos.y);
    const tree2MinY = rootDPos.y;
    assert.ok(tree1MaxY < tree2MinY,
      `Tree 1 max Y (${tree1MaxY}) should be below Tree 2 min Y (${tree2MinY}) - trees should not share Y-space`);
  });

  it('should fan children out vertically when parent has multiple children', () => {
    const html = generateSpecExplorerHTML(testSpecs, branchingAssertions);
    const positions = extractPositions(html);

    const childBPos = positions.get('child-b');
    const childCPos = positions.get('child-c');

    // Children B and C should have different Y positions (fanned out)
    assert.notStrictEqual(childBPos.y, childCPos.y,
      'Children of same parent should have different Y positions');
  });

  it('should center parent at vertical midpoint of its children', () => {
    const html = generateSpecExplorerHTML(testSpecs, branchingAssertions);
    const positions = extractPositions(html);

    const rootAPos = positions.get('root-a');
    const childBPos = positions.get('child-b');
    const childCPos = positions.get('child-c');

    const expectedParentY = (childBPos.y + childCPos.y) / 2;
    assert.strictEqual(rootAPos.y, expectedParentY,
      `Parent Y (${rootAPos.y}) should be centered between children (${expectedParentY})`);
  });

  it('should position nodes left-to-right by dependency depth', () => {
    const html = generateSpecExplorerHTML(testSpecs, branchingAssertions);
    const positions = extractPositions(html);

    const rootAPos = positions.get('root-a');
    const childBPos = positions.get('child-b');
    const rootDPos = positions.get('root-d');

    // Root nodes at depth 0 should have same X
    assert.strictEqual(rootAPos.x, rootDPos.x,
      'Root nodes should share the same X position');

    // Children at depth 1 should be further right than root
    assert.ok(childBPos.x > rootAPos.x,
      'Child should be to the right of its parent');
  });
});

describe('Tree-Stacking Layout: Track Colors Are Distinct', () => {
  it('should use distinct track colors from the palette on dependency lines', () => {
    const html = generateSpecExplorerHTML(testSpecs, branchingAssertions);
    const colors = extractDependencyLineColors(html);

    // Should have dependency lines with colors (not all gray)
    assert.ok(colors.length > 0, 'Should have dependency line colors');

    // At least some colors should be from the palette (not gray #94a3b8)
    const paletteColors = ['#3b82f6', '#f97316', '#10b981', '#a855f7', '#ec4899', '#14b8a6', '#eab308', '#ef4444'];
    const hasTrackColor = colors.some(c => paletteColors.includes(c));
    assert.ok(hasTrackColor, 'Dependency lines should use colors from the track palette');
  });

  it('should assign different colors to different terminal tracks', () => {
    const html = generateSpecExplorerHTML(testSpecs, branchingAssertions);

    // Extract colors from the dependency lines for child-b and child-c
    // They are different terminals so should get different colors
    const childBLineMatch = html.match(/Dependency: root-a .* child-b[^>]*stroke="([^"]+)"/);
    const childCLineMatch = html.match(/Dependency: root-a .* child-c[^>]*stroke="([^"]+)"/);

    if (childBLineMatch && childCLineMatch) {
      assert.notStrictEqual(childBLineMatch[1], childCLineMatch[1],
        'Different terminal tracks should have different colors');
    }
  });
});

describe('No Dead Layout Code', () => {
  it('should not contain unused Sugiyama layout functions', async () => {
    // Read the source file to check for dead code
    const fs = await import('node:fs');
    const source = fs.readFileSync(new URL('../show/cli.js', import.meta.url), 'utf8');

    assert.ok(!source.includes('function assignLayers'), 'Should not contain assignLayers function');
    assert.ok(!source.includes('function minimizeCrossings'), 'Should not contain minimizeCrossings function');
    assert.ok(!source.includes('function assignCoordinatesWithSugiyama'), 'Should not contain assignCoordinatesWithSugiyama function');
    assert.ok(!source.includes('function calculateDependencyDepth'), 'Should not contain calculateDependencyDepth function');
  });
});
