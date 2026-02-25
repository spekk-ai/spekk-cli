import { describe, it } from 'node:test';
import assert from 'node:assert';
import { generateSpecExplorerHTML } from '../show/cli.js';

describe('Metro Map Pan and Zoom Viewport', () => {
  it('should have metro map in dedicated third column panel', () => {
    const specs = [{
      id: 'test-spec',
      title: 'Test Spec',
      priority: 1,
      status: 'in_progress',
      created: '2026-01-01T00:00:00Z',
      file: 'specs/test-spec/test-spec.md',
      content: 'Test content'
    }];

    const assertions = [{
      id: 'test-assertion',
      title: 'Test Assertion',
      parent: 'test-spec',
      priority: 1,
      status: 'not_started',
      created: '2026-01-01T00:00:00Z',
      file: 'specs/test-spec/assertions/test-assertion.md',
      content: 'Test assertion content',
      branch: 'feature/test'
    }];

    const html = generateSpecExplorerHTML(specs, assertions);

    // Verify metro map section exists inside detail panel
    assert.ok(html.includes('metro-map-section'), 'Should have metro-map-section');
    assert.ok(html.includes('detail-content-section'), 'Should have detail-content-section');
    assert.ok(html.includes('width: 400px'), 'Should have 400px width for tree panel');
  });

  it('should have overflow hidden for panning', () => {
    const specs = [{
      id: 'test-spec',
      title: 'Test Spec',
      priority: 1,
      status: 'in_progress',
      created: '2026-01-01T00:00:00Z',
      file: 'specs/test-spec/test-spec.md',
      content: 'Test content'
    }];

    const assertions = [{
      id: 'test-assertion',
      title: 'Test Assertion',
      parent: 'test-spec',
      priority: 1,
      status: 'not_started',
      created: '2026-01-01T00:00:00Z',
      file: 'specs/test-spec/assertions/test-assertion.md',
      content: 'Test assertion content',
      branch: 'feature/test'
    }];

    const html = generateSpecExplorerHTML(specs, assertions);

    // Verify overflow is hidden (replaced scrolling with panning)
    assert.ok(html.includes('overflow: hidden'), 'Should have overflow: hidden');
  });

  it('should have grab cursor for pan interaction', () => {
    const specs = [{
      id: 'test-spec',
      title: 'Test Spec',
      priority: 1,
      status: 'in_progress',
      created: '2026-01-01T00:00:00Z',
      file: 'specs/test-spec/test-spec.md',
      content: 'Test content'
    }];

    const assertions = [{
      id: 'test-assertion',
      title: 'Test Assertion',
      parent: 'test-spec',
      priority: 1,
      status: 'not_started',
      created: '2026-01-01T00:00:00Z',
      file: 'specs/test-spec/assertions/test-assertion.md',
      content: 'Test assertion content',
      branch: 'feature/test'
    }];

    const html = generateSpecExplorerHTML(specs, assertions);

    // Verify cursor: grab is set for pan interaction
    assert.ok(html.includes('cursor: grab'), 'Should have cursor: grab');
  });

  it('should have panning state styles', () => {
    const specs = [{
      id: 'test-spec',
      title: 'Test Spec',
      priority: 1,
      status: 'in_progress',
      created: '2026-01-01T00:00:00Z',
      file: 'specs/test-spec/test-spec.md',
      content: 'Test content'
    }];

    const assertions = [{
      id: 'test-assertion',
      title: 'Test Assertion',
      parent: 'test-spec',
      priority: 1,
      status: 'not_started',
      created: '2026-01-01T00:00:00Z',
      file: 'specs/test-spec/assertions/test-assertion.md',
      content: 'Test assertion content',
      branch: 'feature/test'
    }];

    const html = generateSpecExplorerHTML(specs, assertions);

    // Verify panning state styles for metro-map-container
    assert.ok(html.includes('.metro-map-container.panning'), 'Should have .panning class styles');
    assert.ok(html.includes('cursor: grabbing'), 'Should have cursor: grabbing when panning');
    assert.ok(html.includes('user-select: none'), 'Should have user-select: none when panning');
  });

  it('should position metro-map-section relatively for panning', () => {
    const specs = [{
      id: 'test-spec',
      title: 'Test Spec',
      priority: 1,
      status: 'in_progress',
      created: '2026-01-01T00:00:00Z',
      file: 'specs/test-spec/test-spec.md',
      content: 'Test content'
    }];

    const assertions = [{
      id: 'test-assertion',
      title: 'Test Assertion',
      parent: 'test-spec',
      priority: 1,
      status: 'not_started',
      created: '2026-01-01T00:00:00Z',
      file: 'specs/test-spec/assertions/test-assertion.md',
      content: 'Test assertion content',
      branch: 'feature/test'
    }];

    const html = generateSpecExplorerHTML(specs, assertions);

    // Verify position: relative is set for panning context
    assert.ok(html.includes('position: relative'), 'Should have position: relative');
  });

  it('should have smooth transition for SVG transform', () => {
    const specs = [{
      id: 'test-spec',
      title: 'Test Spec',
      priority: 1,
      status: 'in_progress',
      created: '2026-01-01T00:00:00Z',
      file: 'specs/test-spec/test-spec.md',
      content: 'Test content'
    }];

    const assertions = [{
      id: 'test-assertion',
      title: 'Test Assertion',
      parent: 'test-spec',
      priority: 1,
      status: 'not_started',
      created: '2026-01-01T00:00:00Z',
      file: 'specs/test-spec/assertions/test-assertion.md',
      content: 'Test assertion content',
      branch: 'feature/test'
    }];

    const html = generateSpecExplorerHTML(specs, assertions);

    // Verify smooth transition for transform during panning
    assert.ok(html.includes('transition: transform'), 'Should have transition: transform');
  });

  it('should have pan JavaScript functionality', () => {
    const specs = [{
      id: 'test-spec',
      title: 'Test Spec',
      priority: 1,
      status: 'in_progress',
      created: '2026-01-01T00:00:00Z',
      file: 'specs/test-spec/test-spec.md',
      content: 'Test content'
    }];

    const assertions = [{
      id: 'test-assertion',
      title: 'Test Assertion',
      parent: 'test-spec',
      priority: 1,
      status: 'not_started',
      created: '2026-01-01T00:00:00Z',
      file: 'specs/test-spec/assertions/test-assertion.md',
      content: 'Test assertion content',
      branch: 'feature/test'
    }];

    const html = generateSpecExplorerHTML(specs, assertions);

    // Verify pan JavaScript functionality is included
    assert.ok(html.includes('Metro map pan functionality'), 'Should have pan JavaScript comment');
    assert.ok(html.includes('metroMapState.panStates'), 'Should track pan state in metroMapState');
    assert.ok(html.includes('metro-map-container'), 'Should reference metro-map-container');
    assert.ok(html.includes('mousedown'), 'Should handle mousedown event');
    assert.ok(html.includes('mousemove'), 'Should handle mousemove event');
    assert.ok(html.includes('mouseup'), 'Should handle mouseup event');
    assert.ok(html.includes('.metro-station'), 'Should check for station clicks');
  });
});
