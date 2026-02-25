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

const testAssertions = [{
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

function getHTML() {
  return generateSpecExplorerHTML(testSpecs, testAssertions);
}

describe('Metro Map Pan and Zoom Viewport', () => {
  it('should have metro map section with pan-and-zoom infrastructure', () => {
    const html = getHTML();

    // Core structure
    assert.ok(html.includes('metro-map-section'), 'Should have metro-map-section');
    assert.ok(html.includes('detail-content-section'), 'Should have detail-content-section');
    assert.ok(html.includes('width: 400px'), 'Should have 400px width for tree panel');

    // Pan-and-zoom CSS
    assert.ok(html.includes('overflow: hidden'), 'Should have overflow: hidden');
    assert.ok(html.includes('cursor: grab'), 'Should have cursor: grab');
    assert.ok(html.includes('position: relative'), 'Should have position: relative');
    assert.ok(html.includes('transition: transform'), 'Should have transition: transform for SVG');

    // Panning state styles
    assert.ok(html.includes('.metro-map-container.panning'), 'Should have .panning class styles');
    assert.ok(html.includes('cursor: grabbing'), 'Should have cursor: grabbing when panning');
    assert.ok(html.includes('user-select: none'), 'Should have user-select: none when panning');
  });

  it('should have pan JavaScript functionality', () => {
    const html = getHTML();

    assert.ok(html.includes('Metro map pan functionality'), 'Should have pan JavaScript comment');
    assert.ok(html.includes('metroMapState.panStates'), 'Should track pan state in metroMapState');
    assert.ok(html.includes('metro-map-container'), 'Should reference metro-map-container');
    assert.ok(html.includes('mousedown'), 'Should handle mousedown event');
    assert.ok(html.includes('mousemove'), 'Should handle mousemove event');
    assert.ok(html.includes('mouseup'), 'Should handle mouseup event');
    assert.ok(html.includes('.metro-station'), 'Should check for station clicks');
  });
});

describe('Metro Map Collapsible and Resizable Viewport', () => {
  it('should have default 300px height with min/max constraints', () => {
    const html = getHTML();

    // Default height
    assert.ok(html.includes('height: 300px'), 'Should have 300px default height');

    // Min/max constraints
    assert.ok(html.includes('min-height: 100px'), 'Should have 100px min height');
    assert.ok(html.includes('max-height: 600px'), 'Should have 600px max height');
  });

  it('should have collapsible header with toggle icon and branch name', () => {
    const html = getHTML();

    // Header elements
    assert.ok(html.includes('metro-map-header'), 'Should have collapsible header');
    assert.ok(html.includes('metro-map-toggle-icon'), 'Should have toggle icon');
    assert.ok(html.includes('Branch Dependencies'), 'Should have header title text');
    assert.ok(html.includes('metro-branch-name'), 'Should have branch name display');
  });

  it('should have collapsed state CSS at 36px height', () => {
    const html = getHTML();

    // Collapsed state
    assert.ok(html.includes('.metro-map-section.collapsed'), 'Should have collapsed class styles');
    assert.ok(html.includes('height: 36px'), 'Collapsed state should be 36px');
  });

  it('should have smooth height transition that can be disabled during drag', () => {
    const html = getHTML();

    // Height transition for collapse/expand
    assert.ok(html.includes('transition: height 0.3s ease'), 'Should have height transition');

    // No-transition class for drag responsiveness
    assert.ok(html.includes('.metro-map-section.no-transition'), 'Should have no-transition class');
    assert.ok(html.includes('transition: none'), 'No-transition should disable transitions');
  });

  it('should persist collapse state and height in localStorage', () => {
    const html = getHTML();

    assert.ok(html.includes('spekkMetroMapCollapsed'), 'Should persist collapse state');
    assert.ok(html.includes('spekkMetroMapHeight'), 'Should persist custom height');
  });

  it('should have drag resize handle with ns-resize cursor', () => {
    const html = getHTML();

    // Resize handle element
    assert.ok(html.includes('metro-map-resize-handle'), 'Should have resize handle');
    assert.ok(html.includes('cursor: ns-resize'), 'Resize handle should have ns-resize cursor');
    assert.ok(html.includes('grip-dots'), 'Handle should have grip dots visual');

    // Handle height
    assert.ok(html.includes('height: 6px'), 'Resize handle should be 6px');
  });

  it('should hide resize handle when collapsed', () => {
    const html = getHTML();

    assert.ok(
      html.includes('.metro-map-section.collapsed .metro-map-resize-handle'),
      'Should have rule to hide handle when collapsed'
    );
  });

  it('should have drag resize JavaScript with body cursor and selection control', () => {
    const html = getHTML();

    // Resize JS
    assert.ok(html.includes('initializeMetroMapResize'), 'Should have resize initialization');
    assert.ok(html.includes('isDragging'), 'Should track drag state');
    assert.ok(html.includes('ns-resize'), 'Should set ns-resize cursor during drag');
    assert.ok(html.includes('userSelect'), 'Should disable text selection during drag');
    assert.ok(html.includes('no-transition'), 'Should disable transition during drag');
  });
});
