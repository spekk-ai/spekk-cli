import { describe, it } from 'node:test';
import assert from 'node:assert';
import { generateSpecExplorerHTML } from '../show/cli.js';

describe('Metro Map Scrollable Viewport', () => {
  it('should have max-height constraint on metro-map-section', () => {
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

    // Verify max-height is set
    assert.ok(html.includes('max-height: 300px'), 'Should have max-height: 300px');
  });

  it('should have horizontal overflow scrolling', () => {
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

    // Verify overflow-x is auto for horizontal scrolling
    assert.ok(html.includes('overflow-x: auto'), 'Should have overflow-x: auto');
  });

  it('should have vertical overflow hidden', () => {
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

    // Verify overflow-y is hidden
    assert.ok(html.includes('overflow-y: hidden'), 'Should have overflow-y: hidden');
  });

  it('should have smooth scroll behavior', () => {
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

    // Verify smooth scroll behavior
    assert.ok(html.includes('scroll-behavior: smooth'), 'Should have scroll-behavior: smooth');
  });

  it('should have visual fade edge indicator', () => {
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

    // Verify ::after pseudo-element for fade edge
    assert.ok(html.includes('.metro-map-section::after'), 'Should have ::after pseudo-element');
    assert.ok(html.includes('linear-gradient(to right, transparent, #f8fafc)'), 'Should have fade gradient');
    assert.ok(html.includes('pointer-events: none'), 'Fade edge should not block pointer events');
  });

  it('should position metro-map-section relatively for fade edge', () => {
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

    // Verify position: relative is set for containing the ::after pseudo-element
    assert.ok(html.includes('position: relative'), 'Should have position: relative');
  });
});
