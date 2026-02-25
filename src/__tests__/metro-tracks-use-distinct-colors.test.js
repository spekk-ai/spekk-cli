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

describe('Uniform Gray Dependency Lines', () => {
  it('should use uniform gray #94a3b8 on all dependency lines', () => {
    const html = generateSpecExplorerHTML(testSpecs, branchingAssertions);
    const colors = extractDependencyLineColors(html);

    assert.ok(colors.length > 0, 'Should have dependency lines');

    // Every dependency line must be gray #94a3b8
    colors.forEach(color => {
      assert.strictEqual(color, '#94a3b8',
        `All dependency lines should be gray #94a3b8, but found ${color}`);
    });
  });

  it('should not contain per-track color assignment logic', async () => {
    const fs = await import('node:fs');
    const source = fs.readFileSync(new URL('../show/cli.js', import.meta.url), 'utf8');

    assert.ok(!source.includes('assertionToColor'), 'Should not contain assertionToColor map');
    assert.ok(!source.includes('trackPalette'), 'Should not contain trackPalette array');
    assert.ok(!source.includes('assignTrackColors'), 'Should not contain assignTrackColors function');
    assert.ok(!source.includes('getBranchColor'), 'Should not contain getBranchColor function');
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
