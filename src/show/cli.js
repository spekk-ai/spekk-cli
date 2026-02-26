import { mkdirSync, existsSync, writeFileSync } from 'node:fs';
import { join, basename } from 'node:path';
import { spawn } from 'node:child_process';
import { platform } from 'node:os';
import { parseAllSpecs } from '../parser/index.js';

export async function showSpekk() {
  // Create .spekk directory if it doesn't exist
  const spekkDir = join(process.cwd(), '.spekk');
  
  if (!existsSync(spekkDir)) {
    mkdirSync(spekkDir, { recursive: true });
    console.log('Created .spekk directory');
  }
  
  // Parse all specs and assertions
  const { specs, assertions } = parseAllSpecs();
  
  // Generate HTML content
  const htmlContent = generateSpecExplorerHTML(specs, assertions);
  
  // Write HTML file (overwrite if exists)
  const htmlFilePath = join(spekkDir, 'index.html');
  writeFileSync(htmlFilePath, htmlContent, 'utf8');
  
  console.log('Generated spec explorer at .spekk/index.html');
  
  // Open the HTML file in the default browser (skip in test/CI environments)
  if (process.env.NODE_ENV !== 'test' && !process.env.CI) {
    openInBrowser(htmlFilePath);
  }
}

export function openInBrowser(htmlFilePath) {
  // Convert to file:// URL for proper browser handling
  const fileUrl = `file://${htmlFilePath}`;
  
  // Determine the correct command based on the operating system
  let command;
  let args = [fileUrl];
  
  switch (platform()) {
    case 'darwin': // macOS
      command = 'open';
      break;
    case 'win32': // Windows
      command = 'start';
      args = ['', fileUrl]; // start command requires empty string as first arg
      break;
    default: // Linux and other Unix-like systems
      command = 'xdg-open';
      break;
  }
  
  // Spawn the browser command
  // Use detached: true to prevent hanging and stdio: 'ignore' to prevent output
  const child = spawn(command, args, {
    detached: true,
    stdio: 'ignore'
  });
  
  // Handle errors gracefully - don't let browser opening failure crash the command
  child.on('error', (error) => {
    // Silently ignore browser opening errors - command should still succeed
    // Could log this in debug mode if needed in the future
  });
  
  // Unref the child process so it doesn't keep the parent process alive
  child.unref();
}

function getStatusIcon(status) {
  switch (status) {
    case 'not_started': return '○';
    case 'in_progress': return '🔄';
    case 'done': return '✅';
    case 'failed': return '❌';
    case 'draft': return '⏸️';
    default: return '';
  }
}

function generateDetailStatusBadge(status) {
  const icon = getStatusIcon(status);
  return `<span class="detail-status-badge">${icon}</span>`;
}

function generateDetailPriorityBadge(priority) {
  return `<span class="detail-priority-badge">${priority}</span>`;
}

function escapeHTML(str) {
  return str
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
    .replace(/`/g, '&#96;');
}

function getStatusColor(status) {
  switch (status) {
    case 'not_started': return '#64748b';
    case 'in_progress': return '#f59e0b';
    case 'done': return '#10b981';
    case 'failed': return '#ef4444';
    case 'draft': return '#94a3b8';
    default: return '#64748b';
  }
}

// Find terminal assertions (assertions with no children depending on them)
function findTerminalAssertions(assertions) {
  return assertions.filter(assertion =>
    !assertions.some(child => child.dependsOn === assertion.id)
  );
}

function shouldShowMetroMap(assertion, allAssertions) {
  const assertionBranch = assertion.branch || 'main';

  // Always show for feature branches
  if (assertionBranch !== 'main') {
    return true;
  }

  // For main branch, only show if there are dependencies
  const branchAssertions = allAssertions.filter(a => (a.branch || 'main') === assertionBranch);
  const hasDependencies = branchAssertions.some(a => a.dependsOn);
  return hasDependencies;
}

function generateNoDependenciesNotice() {
  return `
    <div class="no-dependencies-notice">
      <div class="notice-icon">ℹ️</div>
      <div class="notice-content">
        <div class="notice-title">No branch dependencies to visualize</div>
        <div class="notice-text">
          This assertion is on the main branch with no related dependencies.
          Branch dependencies are shown for feature branches and main branch assertions with dependency chains.
        </div>
      </div>
    </div>
  `;
}

function generateMetroMapSVG(currentAssertion, allAssertions) {
  // Filter assertions in the same branch
  const assertionBranch = currentAssertion.branch || 'main';
  const branchAssertions = allAssertions
    .filter(a => (a.branch || 'main') === assertionBranch);

  // Find terminal assertions for terminus dot rendering
  const terminalAssertions = findTerminalAssertions(branchAssertions);

  if (branchAssertions.length === 0) {
    return '';
  }

  const stationRadius = 8;
  const currentStationRadius = 10;
  const terminusRadius = 5;

  // Build parent-to-children map for tree traversal
  const childrenMap = new Map(); // parentId -> [child assertions]
  branchAssertions.forEach(a => {
    if (a.dependsOn) {
      if (!childrenMap.has(a.dependsOn)) {
        childrenMap.set(a.dependsOn, []);
      }
      childrenMap.get(a.dependsOn).push(a);
    }
  });

  // Find true root nodes: assertions with no dependsOn (or whose dependsOn target is not in this branch)
  const branchIds = new Set(branchAssertions.map(a => a.id));
  const rootNodes = branchAssertions.filter(a =>
    !a.dependsOn || !branchIds.has(a.dependsOn)
  );

  // Layout constants
  const positions = new Map();
  const layerSpacing = 120; // Horizontal spacing between dependency levels
  const nodeSpacing = 45;   // Vertical spacing between sibling rows
  const treeGap = 45;       // Vertical gap between independent trees
  const startX = 60;
  let nextTreeY = 80;       // Running Y cursor for stacking trees

  // Calculate dependency depth for a node (distance from its tree root)
  function getDepth(nodeId, visited = new Set()) {
    if (visited.has(nodeId)) return 0;
    const node = branchAssertions.find(a => a.id === nodeId);
    if (!node || !node.dependsOn || !branchIds.has(node.dependsOn)) return 0;
    visited.add(nodeId);
    return 1 + getDepth(node.dependsOn, visited);
  }

  // Recursively layout a tree rooted at `node`, starting at yStart.
  // Returns the total vertical extent (height) consumed by this subtree.
  function layoutTree(node, yStart) {
    const depth = getDepth(node.id);
    const x = startX + (depth * layerSpacing);
    const children = childrenMap.get(node.id) || [];

    if (children.length === 0) {
      // Leaf node: place at yStart
      positions.set(node.id, { x, y: yStart });
      return nodeSpacing; // Single row height
    }

    // Sort children by created date for stable ordering
    children.sort((a, b) => a.created.localeCompare(b.created));

    // Layout each child subtree, stacking them vertically
    let childY = yStart;
    let totalChildHeight = 0;
    const childYPositions = []; // Track each child's Y for centering parent

    children.forEach((child, i) => {
      const childHeight = layoutTree(child, childY);
      // Record the child's actual Y position (already set by recursive call)
      childYPositions.push(positions.get(child.id).y);
      childY += childHeight;
      totalChildHeight += childHeight;
    });

    // Position parent at vertical center of its children's Y range
    const minChildY = Math.min(...childYPositions);
    const maxChildY = Math.max(...childYPositions);
    const parentY = (minChildY + maxChildY) / 2;
    positions.set(node.id, { x, y: parentY });

    return totalChildHeight;
  }

  // Sort roots by created date for stable ordering
  rootNodes.sort((a, b) => a.created.localeCompare(b.created));

  // Layout each independent tree, stacking vertically
  rootNodes.forEach((root, i) => {
    const treeHeight = layoutTree(root, nextTreeY);
    nextTreeY += treeHeight + treeGap;
  });

  // Give each terminal assertion its own "Done" node (only if multiple terminals)
  const showTerminus = terminalAssertions.length > 1;
  const terminusPositions = new Map(); // terminal.id -> {x, y}

  if (showTerminus) {
    const maxX = Math.max(...Array.from(positions.values()).map(p => p.x));
    const terminusX = maxX + 120;

    // Each terminal gets its own Done node at the same Y position
    terminalAssertions.forEach(terminal => {
      const terminalPos = positions.get(terminal.id);
      if (terminalPos) {
        terminusPositions.set(terminal.id, {
          x: terminusX,
          y: terminalPos.y
        });
      }
    });
  }

  // Compute SVG dimensions from actual node positions with 60px edge padding on all sides
  // Labels extend ~30px below stations, so add that to the bottom padding
  const EDGE_PADDING = 60;
  const LABEL_EXTRA = 30; // metro-label y="28" plus some buffer
  const rawMaxX = Math.max(...Array.from(positions.values()).map(p => p.x),
    ...(showTerminus ? Array.from(terminusPositions.values()).map(p => p.x) : [0]));
  const rawMaxY = Math.max(...Array.from(positions.values()).map(p => p.y));
  const svgWidth = Math.max(rawMaxX + EDGE_PADDING * 2, 800);
  const svgHeight = Math.max(rawMaxY + EDGE_PADDING + LABEL_EXTRA, 200);

  // Generate SVG
  let svg = `
<svg class="metro-map" width="${svgWidth}" height="${svgHeight}" viewBox="0 0 ${svgWidth} ${svgHeight}">
  `;

  // Add dependency lines - all uniform gray
  branchAssertions.forEach(assertion => {
    if (assertion.dependsOn) {
      const parentPos = positions.get(assertion.dependsOn);
      const childPos = positions.get(assertion.id);

      if (parentPos && childPos) {
        // Draw curved line if y positions differ, otherwise straight line
        if (parentPos.y !== childPos.y) {
          const midX = (parentPos.x + childPos.x) / 2;
          svg += `
  <!-- Dependency: ${escapeHTML(assertion.dependsOn)} → ${escapeHTML(assertion.id)} -->
  <path class="metro-dependency" d="M${parentPos.x},${parentPos.y} Q${midX},${parentPos.y} ${midX},${(parentPos.y + childPos.y) / 2} T${childPos.x},${childPos.y}"
        fill="none" stroke="#94a3b8" stroke-width="3" opacity="0.4"/>`;
        } else {
          svg += `
  <!-- Dependency: ${escapeHTML(assertion.dependsOn)} → ${escapeHTML(assertion.id)} -->
  <line class="metro-dependency" x1="${parentPos.x}" y1="${parentPos.y}" x2="${childPos.x}" y2="${childPos.y}"
        stroke="#94a3b8" stroke-width="3" opacity="0.4"/>`;
        }
      }
    }
  });

  // Add lines from terminal assertions to their individual "Done" nodes
  if (showTerminus) {
    terminalAssertions.forEach(assertion => {
      const terminalPos = positions.get(assertion.id);
      const donePos = terminusPositions.get(assertion.id);

      if (terminalPos && donePos) {
        svg += `
  <!-- Convergence: ${escapeHTML(assertion.id)} → Done -->
  <line class="metro-dependency" x1="${terminalPos.x}" y1="${terminalPos.y}" x2="${donePos.x}" y2="${donePos.y}"
        stroke="#94a3b8" stroke-width="3" opacity="0.4"/>`;
      }
    });
  }

  // Add stations
  branchAssertions.forEach(assertion => {
    const pos = positions.get(assertion.id);
    if (!pos) return;

    const isCurrent = assertion.id === currentAssertion.id;
    const radius = isCurrent ? currentStationRadius : stationRadius;
    const statusColor = getStatusColor(assertion.status);

    // Use status color for fill and border
    const strokeWidth = isCurrent ? 4 : 3;
    const glowFilter = isCurrent ? ' filter="drop-shadow(0 0 6px rgba(59, 130, 246, 0.6))"' : '';

    // Truncate long titles
    let displayTitle = assertion.title;
    if (displayTitle.length > 20) {
      displayTitle = displayTitle.substring(0, 17) + '...';
    }

    svg += `
  <!-- Station: ${escapeHTML(assertion.id)} -->
  <g class="metro-station" data-action="show-detail" data-assertion-id="${escapeHTML(assertion.id)}" data-type="assertion" transform="translate(${pos.x}, ${pos.y})">
    <title>${escapeHTML(assertion.title)}</title>
    <circle r="${radius}" fill="${statusColor}" stroke="#fff" stroke-width="${strokeWidth}"${glowFilter}/>
    <text class="metro-label" y="28" style="font-size: 10px; fill: #1e293b; text-anchor: middle; font-weight: ${isCurrent ? '700' : '400'};">${escapeHTML(displayTitle)}</text>
  </g>`;
  });

  // Add "Done" nodes - one for each terminal assertion
  if (showTerminus) {
    terminalAssertions.forEach(assertion => {
      const donePos = terminusPositions.get(assertion.id);
      if (donePos) {
        // Check if all assertions in this tree are done by tracing back to root
        const treeNodes = [];
        let currentNode = assertion;
        while (currentNode) {
          treeNodes.push(currentNode);
          const parent = branchAssertions.find(a => a.id === currentNode.dependsOn);
          currentNode = parent;
        }

        const allDone = treeNodes.every(node => node.status === 'done');
        const doneColor = allDone ? '#10b981' : '#94a3b8'; // Green if all done, gray otherwise

        svg += `
  <!-- Done Terminus -->
  <g class="metro-terminus" transform="translate(${donePos.x}, ${donePos.y})">
    <circle r="${terminusRadius}" fill="${doneColor}" stroke="#fff" stroke-width="2"/>
  </g>`;
      }
    });
  }

  svg += `
</svg>`;

  return svg;
}

export function generateSpecExplorerHTML(specs, assertions) {
  // Get project name from current working directory
  const projectName = basename(process.cwd());

  // Group assertions by branch and generate metro maps
  const branchGroups = new Map();
  assertions.forEach(assertion => {
    const branch = assertion.branch || 'main';
    if (!branchGroups.has(branch)) {
      branchGroups.set(branch, []);
    }
    branchGroups.get(branch).push(assertion);
  });

  // Generate metro map for each branch
  const branchMetroMaps = new Map();
  branchGroups.forEach((branchAssertions, branch) => {
    if (branchAssertions.length > 0) {
      // Use the first assertion as reference point for generation
      const refAssertion = branchAssertions[0];
      const shouldShow = shouldShowMetroMap(refAssertion, assertions);
      const metroMapHTML = shouldShow ?
        generateMetroMapSVG(refAssertion, assertions) :
        generateNoDependenciesNotice();
      branchMetroMaps.set(branch, metroMapHTML);
    }
  });

  // Group assertions by parent spec
  const specHierarchy = specs.map(spec => {
    const specAssertions = assertions
      .filter(assertion => assertion.parent === spec.id)
      .sort((a, b) => {
        // Sort by priority first, then by creation date
        if (a.priority !== b.priority) {
          return a.priority - b.priority;
        }
        return a.created.localeCompare(b.created);
      });

    return {
      ...spec,
      assertions: specAssertions
    };
  }).sort((a, b) => {
    // Sort specs by priority first, then by creation date
    if (a.priority !== b.priority) {
      return a.priority - b.priority;
    }
    return a.created.localeCompare(b.created);
  });

  return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Spec Explorer - ${projectName}</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: #f8fafc;
            color: #1e293b;
            line-height: 1.5;
        }
        
        .container {
            display: flex;
            height: 100vh;
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
        }

        .tree-panel {
            width: 400px;
            padding: 20px;
            border-right: 1px solid #e2e8f0;
            overflow-y: auto;
        }

        .detail-panel {
            flex: 1;
            display: flex;
            flex-direction: column;
            overflow: hidden;
        }

        .metro-map-section {
            display: flex;
            flex-direction: column;
            background: #f8fafc;
            overflow: hidden;
            position: relative;
            transition: height 0.3s ease;
            height: 300px;
            min-height: 100px;
            max-height: 600px;
        }

        .metro-map-section.collapsed {
            height: 36px !important;
            min-height: 36px !important;
            cursor: default;
            overflow: hidden;
        }

        .metro-map-section.no-transition {
            transition: none !important;
        }

        .metro-map-header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 8px 20px;
            background: #f1f5f9;
            cursor: pointer;
            user-select: none;
            flex-shrink: 0;
            height: 36px;
        }

        .metro-map-header:hover {
            background: #e2e8f0;
        }

        .metro-map-header-left {
            display: flex;
            align-items: center;
            gap: 8px;
        }

        .metro-map-toggle-icon {
            font-size: 12px;
            color: #64748b;
            transition: transform 0.2s;
        }

        .metro-map-section.collapsed .metro-map-toggle-icon {
            transform: rotate(-90deg);
        }

        .metro-map-header-title {
            font-size: 14px;
            font-weight: 600;
            color: #475569;
        }

        .metro-map-header .branch-name {
            font-size: 12px;
            color: #64748b;
            font-family: 'Courier New', monospace;
        }

        .metro-map-container {
            flex: 1;
            overflow: hidden;
            position: relative;
            cursor: grab;
            background: #f8fafc;
        }

        .metro-map-container.panning {
            cursor: grabbing;
            user-select: none;
        }

        .metro-map-container .metro-map {
            transition: transform 0.1s ease-out;
        }

        .metro-map-resize-handle {
            height: 6px;
            cursor: ns-resize;
            background: #e2e8f0;
            border-bottom: 1px solid #cbd5e1;
            display: flex;
            align-items: center;
            justify-content: center;
            flex-shrink: 0;
        }

        .metro-map-resize-handle:hover {
            background: #cbd5e1;
        }

        .metro-map-resize-handle .grip-dots {
            display: flex;
            gap: 3px;
        }

        .metro-map-resize-handle .grip-dots span {
            width: 3px;
            height: 3px;
            border-radius: 50%;
            background: #94a3b8;
        }

        .metro-map-section.collapsed .metro-map-resize-handle {
            display: none;
        }

        .detail-content-section {
            flex: 1;
            padding: 20px;
            overflow-y: auto;
        }
        
        .header {
            margin-bottom: 24px;
            padding-bottom: 12px;
            border-bottom: 2px solid #e2e8f0;
        }
        
        .header h1 {
            font-size: 24px;
            color: #1e293b;
            margin-bottom: 4px;
        }
        
        .header p {
            color: #64748b;
            font-size: 14px;
        }
        
        .spec-tree {
            list-style: none;
        }
        
        .spec-item {
            margin-bottom: 12px;
        }

        /* Hide completed specs by default */
        .spec-item.completed {
            display: none;
        }

        /* Show completed specs when toggle is checked */
        .spec-tree.show-completed .spec-item.completed {
            display: block;
        }

        .toggle-container {
            margin-top: 16px;
            padding-top: 12px;
            border-top: 1px solid #e2e8f0;
        }

        .toggle-container label {
            display: flex;
            align-items: center;
            cursor: pointer;
            font-size: 14px;
            color: #475569;
            user-select: none;
        }

        .toggle-container input[type="checkbox"] {
            margin-right: 8px;
            cursor: pointer;
        }

        .header .stats {
            color: #64748b;
            font-size: 14px;
        }

        .header .hidden-count {
            color: #94a3b8;
            font-size: 13px;
            font-style: italic;
        }

        .spec-header {
            display: flex;
            align-items: center;
            padding: 12px;
            background: #f1f5f9;
            border-radius: 8px;
            cursor: pointer;
            transition: background-color 0.2s;
        }
        
        .spec-header:hover {
            background: #e2e8f0;
        }
        
        .spec-header.expanded {
            background: #dbeafe;
        }
        
        .spec-header.selected {
            background: #eff6ff;
            border-left: 3px solid #3b82f6;
        }
        
        .toggle-icon {
            margin-right: 8px;
            font-size: 12px;
            color: #64748b;
            transition: transform 0.2s;
        }
        
        .toggle-icon.expanded {
            transform: rotate(90deg);
        }
        
        .status-badge {
            padding: 3px 8px;
            border-radius: 6px;
            font-size: 11px;
            font-weight: 600;
            text-transform: uppercase;
            margin-right: 8px;
            display: inline-flex;
            align-items: center;
            gap: 4px;
            box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
        }
        
        .status-not_started {
            background: #f8fafc;
            color: #64748b;
            border: 1px solid #e2e8f0;
        }
        .status-not_started::before {
            content: "○";
            font-size: 10px;
        }
        
        .status-in_progress {
            background: #fef3c7;
            color: #92400e;
            border: 1px solid #f59e0b;
            animation: pulse-yellow 2s infinite;
        }
        .status-in_progress::before {
            content: "🔄";
            font-size: 10px;
        }
        
        .status-failed {
            background: #fecaca;
            color: #dc2626;
            border: 1px solid #ef4444;
            box-shadow: 0 0 0 2px rgba(239, 68, 68, 0.2);
        }
        .status-failed::before {
            content: "❌";
            font-size: 10px;
        }
        
        .status-done {
            background: #d1fae5;
            color: #065f46;
            border: 1px solid #10b981;
            opacity: 0.8;
        }
        .status-done::before {
            content: "✅";
            font-size: 10px;
        }
        
        .status-draft {
            background: #f1f5f9;
            color: #475569;
            border: 1px solid #cbd5e1;
        }
        .status-draft::before {
            content: "⏸️";
            font-size: 10px;
        }
        
        @keyframes pulse-yellow {
            0%, 100% { box-shadow: 0 0 0 0 rgba(245, 158, 11, 0.7); }
            50% { box-shadow: 0 0 0 4px rgba(245, 158, 11, 0); }
        }
        
        .priority-badge {
            padding: 3px 8px;
            border-radius: 6px;
            font-size: 11px;
            font-weight: 700;
            color: white;
            margin-right: 8px;
            display: inline-flex;
            align-items: center;
            gap: 3px;
            box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
        }
        
        .priority-1, .priority-2, .priority-3 {
            background: #e2e8f0;
            color: #1e293b;
            border: 1px solid #cbd5e1;
        }
        
        /* Neutral styling for all items */
        
        .spec-title {
            font-weight: 600;
            flex: 1;
        }
        
        .assertions-list {
            margin-top: 8px;
            margin-left: 20px;
            display: none;
        }
        
        .assertions-list.expanded {
            display: block;
        }
        
        .assertion-item {
            padding: 8px 12px;
            margin-bottom: 4px;
            background: white;
            border-radius: 6px;
            border-left: 3px solid #e2e8f0;
            cursor: pointer;
            transition: all 0.2s;
        }
        
        .assertion-item:hover {
            background: #f8fafc;
            border-left-color: #3b82f6;
        }
        
        .assertion-item.selected {
            background: #eff6ff;
            border-left-color: #3b82f6;
        }
        
        .detail-content {
            display: none;
        }
        
        .detail-content.active {
            display: block;
        }
        
        .detail-header {
            margin-bottom: 20px;
        }
        
        .detail-title {
            font-size: 20px;
            margin-bottom: 8px;
        }
        
        .detail-meta {
            display: flex;
            gap: 16px;
            margin-bottom: 20px;
            padding: 12px;
            background: #f8fafc;
            border-radius: 8px;
            border: 1px solid #e2e8f0;
        }
        
        .meta-item {
            font-size: 13px;
            color: #475569;
            font-weight: 500;
        }
        
        .meta-item strong {
            font-weight: 600;
            color: #1e293b;
        }
        
        .detail-status-badge {
            padding: 4px 10px;
            border-radius: 6px;
            font-size: 12px;
            font-weight: 600;
            text-transform: uppercase;
            display: inline-flex;
            align-items: center;
            gap: 6px;
            box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
        }
        
        .detail-priority-badge {
            padding: 4px 10px;
            border-radius: 6px;
            font-size: 12px;
            font-weight: 700;
            color: #1e293b;
            background: #e2e8f0;
            border: 1px solid #cbd5e1;
            display: inline-flex;
            align-items: center;
            gap: 4px;
            box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
        }
        
        .detail-body {
            prose: true;
            font-family: Consolas, Monaco, 'Courier New', monospace;
        }
        
        .detail-body pre {
            font-family: Consolas, Monaco, 'Courier New', monospace;
        }
        
        .empty-state {
            text-align: center;
            color: #64748b;
            margin-top: 100px;
        }
        
        .empty-state h3 {
            margin-bottom: 8px;
            color: #374151;
        }

        .metro-map-branch {
            width: 100%;
            height: 100%;
        }

        .metro-map {
            display: block;
            margin: 0 auto;
        }

        .metro-track {
            fill: none;
            stroke: #cbd5e1;
            stroke-width: 6;
            stroke-linecap: round;
        }

        .metro-dependency {
            fill: none;
            stroke-linecap: round;
        }

        .metro-station {
            cursor: pointer;
        }

        .metro-station circle {
            transition: fill 0.15s;
        }

        .metro-station:hover circle {
            fill: #2563eb;
        }

        .metro-station-current circle {
            stroke: #2563eb;
            stroke-width: 4;
            filter: drop-shadow(0 0 6px rgba(59, 130, 246, 0.6));
        }

        .metro-station-current .metro-label {
            font-weight: 700 !important;
        }

        .metro-terminus {
            cursor: default;
            pointer-events: none;
        }

        .metro-label {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            pointer-events: none;
        }

        .metro-station-tooltip {
            position: fixed;
            background: #1e293b;
            color: white;
            padding: 6px 12px;
            border-radius: 6px;
            font-size: 11px;
            white-space: nowrap;
            pointer-events: none;
            z-index: 1000;
            opacity: 0;
            transition: opacity 0.2s;
        }

        .metro-station-tooltip.visible {
            opacity: 1;
        }

        .metro-station-tooltip::after {
            content: '';
            position: absolute;
            top: 100%;
            left: 50%;
            transform: translateX(-50%);
            border: 5px solid transparent;
            border-top-color: #1e293b;
        }

        .no-dependencies-notice {
            display: flex;
            align-items: flex-start;
            gap: 12px;
            padding: 16px;
            background: #eff6ff;
            border: 1px solid #bfdbfe;
            border-radius: 8px;
            color: #1e40af;
        }

        .notice-icon {
            font-size: 20px;
            flex-shrink: 0;
            line-height: 1;
        }

        .notice-content {
            flex: 1;
        }

        .notice-title {
            font-size: 14px;
            font-weight: 600;
            margin-bottom: 6px;
            color: #1e40af;
        }

        .notice-text {
            font-size: 13px;
            line-height: 1.5;
            color: #1e40af;
        }

        .search-container {
            margin-top: 12px;
            position: relative;
        }

        .search-container input {
            width: 100%;
            padding: 8px 12px 8px 32px;
            border: 1px solid #e2e8f0;
            border-radius: 6px;
            font-size: 14px;
            color: #1e293b;
            background: #f8fafc;
            outline: none;
            transition: border-color 0.2s, box-shadow 0.2s;
        }

        .search-container input:focus {
            border-color: #3b82f6;
            box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.15);
            background: white;
        }

        .search-container .search-icon {
            position: absolute;
            left: 10px;
            top: 50%;
            transform: translateY(-50%);
            font-size: 14px;
            color: #94a3b8;
            pointer-events: none;
        }

        .spec-item.search-hidden {
            display: none !important;
        }

        .assertion-item.search-hidden {
            display: none !important;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="tree-panel">
            <div class="header">
                <h1>Spec Tree - ${projectName}</h1>
                <p class="stats">${specs.length} specs, ${assertions.length} assertions</p>
                <p class="hidden-count" id="hidden-count"></p>

                <div class="toggle-container">
                    <label>
                        <input type="checkbox" id="toggle-completed-specs">
                        Show completed specs
                    </label>
                </div>

                <div class="search-container">
                    <span class="search-icon">&#128269;</span>
                    <input type="text" id="spec-search" placeholder="Search specs..." autocomplete="off">
                </div>
            </div>

            <ul class="spec-tree">
                ${specHierarchy.map(spec => `
                    <li class="spec-item" data-search-text="${escapeHTML((spec.id + ' ' + spec.title + ' ' + spec.status + ' ' + spec.priority).toLowerCase())}">
                        <div class="spec-header" data-action="toggle-spec" data-spec-id="${escapeHTML(spec.id)}" data-show-detail="true" data-type="spec">
                            <span class="toggle-icon" id="toggle-${spec.id}">▶</span>
                            <span class="priority-badge priority-${spec.priority}">${spec.priority}</span>
                            <span class="status-badge status-${spec.status}"></span>
                            <span class="spec-title">${escapeHTML(spec.title)}</span>
                        </div>
                        
                        <ul class="assertions-list" id="assertions-${spec.id}">
                            ${spec.assertions.map(assertion => `
                                <li class="assertion-item" data-action="show-detail" data-assertion-id="${escapeHTML(assertion.id)}" data-type="assertion" data-search-text="${escapeHTML((assertion.id + ' ' + assertion.title + ' ' + assertion.status + ' ' + assertion.priority).toLowerCase())}">
                                    <div style="display: flex; align-items: center;">
                                        <span class="priority-badge priority-${assertion.priority}">${assertion.priority}</span>
                                        <span class="status-badge status-${assertion.status}"></span>
                                        <span>${escapeHTML(assertion.title)}</span>
                                    </div>
                                </li>
                            `).join('')}
                        </ul>
                    </li>
                `).join('')}
            </ul>
        </div>
        
        <div class="detail-panel">
            <!-- Metro Map Section: Collapsible, at top of detail panel -->
            <div class="metro-map-section" id="metro-map-section" style="display: none;">
                <div class="metro-map-header" id="metro-map-toggle">
                    <div class="metro-map-header-left">
                        <span class="metro-map-toggle-icon">▼</span>
                        <span class="metro-map-header-title">Branch Dependencies</span>
                        <span class="branch-name" id="metro-branch-name"></span>
                    </div>
                </div>
                <div class="metro-map-container" id="metro-map-container">
                    ${Array.from(branchMetroMaps.entries()).map(([branch, mapHTML]) => `
                        <div class="metro-map-branch" id="metro-map-${escapeHTML(branch)}" data-branch="${escapeHTML(branch)}" style="display: none;">
                            ${mapHTML}
                        </div>
                    `).join('')}
                </div>
                <div class="metro-map-resize-handle" id="metro-map-resize-handle">
                    <div class="grip-dots"><span></span><span></span><span></span></div>
                </div>
            </div>

            <!-- Detail Content Section: Scrollable assertion/spec content -->
            <div class="detail-content-section">
                <div class="empty-state" id="empty-state">
                    <h3>Spec Explorer</h3>
                    <p>Click on any spec or assertion to view details</p>
                </div>

                ${specHierarchy.map(spec => `
                    <div class="detail-content" id="detail-spec-${spec.id}">
                        <div class="detail-header">
                            <div class="detail-title">${escapeHTML(spec.title)}</div>
                            <div class="detail-meta">
                                <span class="meta-item">Status: ${generateDetailStatusBadge(spec.status)}</span>
                                <span class="meta-item">Priority: ${generateDetailPriorityBadge(spec.priority)}</span>
                                <span class="meta-item">File: <strong>${spec.file}</strong></span>
                            </div>
                        </div>
                        <div class="detail-body">
                            <pre style="white-space: pre-wrap; font-family: inherit;">${escapeHTML(spec.content)}</pre>
                        </div>
                    </div>
                `).join('')}

                ${assertions.map(assertion => `
                    <div class="detail-content" id="detail-assertion-${assertion.id}" data-branch="${assertion.branch || 'main'}">
                        <div class="detail-header">
                            <div class="detail-title">${escapeHTML(assertion.title)}</div>
                            <div class="detail-meta">
                                <span class="meta-item">Status: ${generateDetailStatusBadge(assertion.status)}</span>
                                <span class="meta-item">Priority: ${generateDetailPriorityBadge(assertion.priority)}</span>
                                <span class="meta-item">Parent: <strong>${assertion.parent}</strong></span>
                                ${assertion.branch ? `<span class="meta-item">Branch: <strong>${assertion.branch}</strong></span>` : ''}
                                <span class="meta-item">File: <strong>${assertion.file}</strong></span>
                            </div>
                        </div>
                        <div class="detail-body">
                            <pre style="white-space: pre-wrap; font-family: inherit;">${escapeHTML(assertion.content)}</pre>
                        </div>
                    </div>
                `).join('')}
            </div>
        </div>
    </div>
    
    <script>
        // Initialize completed specs toggle, metro map collapse, and search on page load
        document.addEventListener('DOMContentLoaded', function() {
            initializeCompletedSpecsToggle();
            initializeMetroMapCollapse();
            initializeSearch();
        });

        function initializeCompletedSpecsToggle() {
            const specTree = document.querySelector('.spec-tree');
            const toggleCheckbox = document.getElementById('toggle-completed-specs');
            const hiddenCountElement = document.getElementById('hidden-count');

            // Mark all completed specs with the 'completed' class
            const specItems = document.querySelectorAll('.spec-item');
            specItems.forEach(specItem => {
                const statusBadge = specItem.querySelector('.spec-header .status-badge');
                if (statusBadge && statusBadge.classList.contains('status-done')) {
                    specItem.classList.add('completed');
                }
            });

            // Load saved preference from localStorage (default: false = hidden)
            const showCompleted = localStorage.getItem('spekkShowCompleted') === 'true';
            toggleCheckbox.checked = showCompleted;
            if (showCompleted) {
                specTree.classList.add('show-completed');
            }

            // Update hidden count
            updateHiddenCount();

            // Handle toggle changes
            toggleCheckbox.addEventListener('change', function() {
                const isChecked = this.checked;
                localStorage.setItem('spekkShowCompleted', isChecked.toString());

                if (isChecked) {
                    specTree.classList.add('show-completed');
                } else {
                    specTree.classList.remove('show-completed');
                }

                updateHiddenCount();
            });
        }

        function updateHiddenCount() {
            const hiddenCountElement = document.getElementById('hidden-count');
            const completedSpecs = document.querySelectorAll('.spec-item.completed');
            const isShowing = document.querySelector('.spec-tree').classList.contains('show-completed');

            const completedCount = completedSpecs.length;

            if (completedCount > 0 && !isShowing) {
                hiddenCountElement.textContent = \`(\${completedCount} completed \${completedCount === 1 ? 'spec' : 'specs'} hidden)\`;
            } else {
                hiddenCountElement.textContent = '';
            }
        }

        function initializeSearch() {
            const searchInput = document.getElementById('spec-search');
            const specTree = document.querySelector('.spec-tree');
            if (!searchInput || !specTree) return;

            searchInput.addEventListener('input', function() {
                const query = this.value.trim().toLowerCase();
                const specItems = specTree.querySelectorAll('.spec-item');

                if (!query) {
                    // Clear search: remove all search-hidden classes, restore toggle behavior
                    specItems.forEach(function(specItem) {
                        specItem.classList.remove('search-hidden');
                        var assertions = specItem.querySelectorAll('.assertion-item');
                        assertions.forEach(function(a) { a.classList.remove('search-hidden'); });
                    });
                    // Restore completed specs toggle behavior
                    updateHiddenCount();
                    return;
                }

                // Active search: filter specs and assertions
                specItems.forEach(function(specItem) {
                    var specSearchText = specItem.dataset.searchText || '';
                    var specMatches = specSearchText.indexOf(query) !== -1;
                    var assertionItems = specItem.querySelectorAll('.assertion-item');
                    var anyAssertionMatches = false;

                    assertionItems.forEach(function(assertionItem) {
                        var assertionSearchText = assertionItem.dataset.searchText || '';
                        var assertionMatches = assertionSearchText.indexOf(query) !== -1;

                        if (assertionMatches) {
                            assertionItem.classList.remove('search-hidden');
                            anyAssertionMatches = true;
                        } else {
                            assertionItem.classList.add('search-hidden');
                        }
                    });

                    if (specMatches || anyAssertionMatches) {
                        // Show the spec (override completed toggle)
                        specItem.classList.remove('search-hidden');

                        // If an assertion matched, expand the spec to reveal it
                        if (anyAssertionMatches) {
                            var assertionsList = specItem.querySelector('.assertions-list');
                            var toggle = specItem.querySelector('.toggle-icon');
                            var header = specItem.querySelector('.spec-header');
                            if (assertionsList && !assertionsList.classList.contains('expanded')) {
                                assertionsList.classList.add('expanded');
                                if (toggle) { toggle.classList.add('expanded'); toggle.textContent = '\\u25BC'; }
                                if (header) { header.classList.add('expanded'); }
                            }
                        }

                        // If only spec name matches (no assertions matched), show all assertions
                        if (specMatches && !anyAssertionMatches) {
                            assertionItems.forEach(function(a) { a.classList.remove('search-hidden'); });
                        }
                    } else {
                        specItem.classList.add('search-hidden');
                    }
                });
            });
        }

        function initializeMetroMapCollapse() {
            const metroMapSection = document.getElementById('metro-map-section');
            const metroMapToggle = document.getElementById('metro-map-toggle');

            // Load saved height from localStorage (default: 300px)
            const savedHeight = localStorage.getItem('spekkMetroMapHeight');
            if (savedHeight) {
                const h = parseInt(savedHeight, 10);
                if (h >= 100 && h <= 600) {
                    metroMapSection.style.height = h + 'px';
                }
            }

            // Load saved collapse preference from localStorage (default: false = expanded)
            const isCollapsed = localStorage.getItem('spekkMetroMapCollapsed') === 'true';

            if (isCollapsed) {
                metroMapSection.classList.add('collapsed');
            }

            // Handle toggle clicks
            metroMapToggle.addEventListener('click', function() {
                metroMapSection.classList.toggle('collapsed');
                const collapsed = metroMapSection.classList.contains('collapsed');
                localStorage.setItem('spekkMetroMapCollapsed', collapsed.toString());
            });

            // Drag handle resize functionality
            initializeMetroMapResize();
        }

        function initializeMetroMapResize() {
            const metroMapSection = document.getElementById('metro-map-section');
            const resizeHandle = document.getElementById('metro-map-resize-handle');
            if (!resizeHandle || !metroMapSection) return;

            let isDragging = false;
            let startY = 0;
            let startHeight = 0;

            resizeHandle.addEventListener('mousedown', function(e) {
                // Don't resize when collapsed
                if (metroMapSection.classList.contains('collapsed')) return;

                isDragging = true;
                startY = e.clientY;
                startHeight = metroMapSection.getBoundingClientRect().height;

                // Disable transition during drag for responsive feel
                metroMapSection.classList.add('no-transition');

                // Prevent text selection during drag
                document.body.style.cursor = 'ns-resize';
                document.body.style.userSelect = 'none';

                e.preventDefault();
            });

            document.addEventListener('mousemove', function(e) {
                if (!isDragging) return;

                const deltaY = e.clientY - startY;
                let newHeight = startHeight + deltaY;

                // Constrain between 100px min and 600px max
                newHeight = Math.max(100, Math.min(600, newHeight));

                metroMapSection.style.height = newHeight + 'px';
            });

            document.addEventListener('mouseup', function() {
                if (!isDragging) return;

                isDragging = false;

                // Re-enable transition
                metroMapSection.classList.remove('no-transition');

                // Restore cursor and selection
                document.body.style.cursor = '';
                document.body.style.userSelect = '';

                // Save new height to localStorage
                const currentHeight = Math.round(metroMapSection.getBoundingClientRect().height);
                localStorage.setItem('spekkMetroMapHeight', currentHeight.toString());
            });
        }

        // Event delegation for all clicks
        document.addEventListener('click', function(event) {
            const action = event.target.closest('[data-action]')?.dataset.action;
            
            if (action === 'toggle-spec') {
                const element = event.target.closest('[data-spec-id]');
                if (element) {
                    const specId = element.dataset.specId;
                    toggleSpec(specId, event);
                    
                    // Also show detail if this is a spec header with show-detail attribute
                    if (element.dataset.showDetail === 'true') {
                        showDetail(specId, 'spec', event);
                    }
                }
            } else if (action === 'show-detail') {
                const element = event.target.closest('[data-assertion-id]');
                if (element) {
                    const assertionId = element.dataset.assertionId;
                    const type = element.dataset.type;
                    if (assertionId && type) {
                        showDetail(assertionId, type, event);
                    }
                }
            }
        });
        
        function toggleSpec(specId, event) {
            const toggle = document.getElementById('toggle-' + specId);
            const assertions = document.getElementById('assertions-' + specId);
            const header = toggle.parentElement;
            
            if (assertions.classList.contains('expanded')) {
                assertions.classList.remove('expanded');
                toggle.classList.remove('expanded');
                header.classList.remove('expanded');
                toggle.textContent = '▶';
            } else {
                assertions.classList.add('expanded');
                toggle.classList.add('expanded');
                header.classList.add('expanded');
                toggle.textContent = '▼';
            }
        }
        
        // Metro map state management
        const metroMapState = {
            currentBranch: null,
            svgCache: new Map(), // branch -> SVG HTML
            panStates: new Map() // branch -> { currentX, currentY }
        };

        function showDetail(id, type, event) {
            // Hide all detail content
            document.querySelectorAll('.detail-content').forEach(el => {
                el.classList.remove('active');
            });
            document.getElementById('empty-state').style.display = 'none';

            // Remove selection from all items
            document.querySelectorAll('.assertion-item').forEach(el => {
                el.classList.remove('selected');
            });

            // Show selected content
            const detailElement = document.getElementById('detail-' + type + '-' + id);
            if (detailElement) {
                detailElement.classList.add('active');
            }

            // Update metro map for assertions, hide for specs
            if (type === 'assertion' && detailElement) {
                const branch = detailElement.dataset.branch || 'main';
                updateMetroMap(id, branch);
            } else {
                hideMetroMap();
            }

            // Mark assertion as selected in tree view
            if (type === 'assertion') {
                // Find the assertion item in tree view
                const assertionItem = document.querySelector('.assertion-item[data-assertion-id="' + id + '"]');
                if (assertionItem) {
                    assertionItem.classList.add('selected');

                    // Expand parent spec if collapsed
                    const assertionsList = assertionItem.closest('.assertions-list');
                    if (assertionsList && !assertionsList.classList.contains('expanded')) {
                        const specId = assertionsList.id.replace('assertions-', '');
                        const toggle = document.getElementById('toggle-' + specId);
                        const header = toggle.parentElement;

                        assertionsList.classList.add('expanded');
                        toggle.classList.add('expanded');
                        header.classList.add('expanded');
                        toggle.textContent = '▼';
                    }
                }
            }

            // Mark spec header as selected
            if (type === 'spec' && event) {
                // Remove selection from all spec headers
                document.querySelectorAll('.spec-header').forEach(el => {
                    el.classList.remove('selected');
                });

                // Find and select the spec header
                const toggleElement = document.getElementById('toggle-' + id);
                if (toggleElement) {
                    const specHeader = toggleElement.parentElement;
                    specHeader.classList.add('selected');
                }
            }
        }

        function updateMetroMap(assertionId, branch) {
            const metroMapSection = document.getElementById('metro-map-section');
            const container = document.getElementById('metro-map-container');
            const branchNameEl = document.getElementById('metro-branch-name');

            // Show metro map section (for assertions)
            // Use 'flex' to preserve the flex-direction: column layout from CSS
            metroMapSection.style.display = 'flex';

            // Update branch name
            branchNameEl.textContent = branch;

            // Check if we need to switch branches
            if (metroMapState.currentBranch !== branch) {
                // Hide all branch maps
                const allBranchMaps = container.querySelectorAll('.metro-map-branch');
                allBranchMaps.forEach(map => {
                    map.style.display = 'none';
                });

                // Show the map for this branch
                const branchMap = document.getElementById('metro-map-' + branch);
                if (branchMap) {
                    branchMap.style.display = 'block';
                    metroMapState.currentBranch = branch;

                    // Restore pan state if exists
                    const panState = metroMapState.panStates.get(branch);
                    if (panState) {
                        const svg = branchMap.querySelector('.metro-map');
                        if (svg) {
                            svg.style.transform = \`translate(\${panState.currentX}px, \${panState.currentY}px)\`;
                        }
                    }
                }
            }

            // Update current station highlight by toggling .metro-station-current class (no SVG re-render)
            const visibleMap = document.getElementById('metro-map-' + branch);
            if (!visibleMap) return;

            const allStations = visibleMap.querySelectorAll('.metro-station');
            allStations.forEach(station => {
                const stationId = station.dataset.assertionId;
                const circle = station.querySelector('circle');
                if (stationId === assertionId) {
                    station.classList.add('metro-station-current');
                    // Set SVG attributes for radius/stroke (CSS cannot reliably set SVG geometry attributes)
                    if (circle) {
                        circle.setAttribute('r', '10');
                        circle.setAttribute('stroke-width', '4');
                    }
                } else {
                    station.classList.remove('metro-station-current');
                    if (circle) {
                        circle.setAttribute('r', '8');
                        circle.setAttribute('stroke-width', '3');
                    }
                }
            });
        }

        function hideMetroMap() {
            const metroMapSection = document.getElementById('metro-map-section');
            metroMapSection.style.display = 'none';
        }

        // Metro station tooltip handling
        (function() {
            let tooltip = null;

            // Create tooltip element
            function createTooltip() {
                if (!tooltip) {
                    tooltip = document.createElement('div');
                    tooltip.className = 'metro-station-tooltip';
                    document.body.appendChild(tooltip);
                }
                return tooltip;
            }

            // Position tooltip above the station
            function positionTooltip(element, tooltipEl) {
                const rect = element.getBoundingClientRect();
                const tooltipRect = tooltipEl.getBoundingClientRect();

                // Position above the station, centered
                const left = rect.left + (rect.width / 2) - (tooltipRect.width / 2);
                const top = rect.top - tooltipRect.height - 10;

                tooltipEl.style.left = left + 'px';
                tooltipEl.style.top = top + 'px';
            }

            // Handle mouse events on metro stations
            document.addEventListener('mouseover', function(event) {
                const station = event.target.closest('.metro-station');
                if (station && station.classList.contains('metro-station')) {
                    // Get the title from the <title> element inside the station
                    const titleElement = station.querySelector('title');
                    if (titleElement) {
                        const titleText = titleElement.textContent;
                        const tooltipEl = createTooltip();
                        tooltipEl.textContent = titleText;

                        // Position and show tooltip
                        setTimeout(() => {
                            positionTooltip(station, tooltipEl);
                            tooltipEl.classList.add('visible');
                        }, 0);
                    }
                }
            });

            document.addEventListener('mouseout', function(event) {
                const station = event.target.closest('.metro-station');
                if (station && station.classList.contains('metro-station')) {
                    if (tooltip) {
                        tooltip.classList.remove('visible');
                    }
                }
            });

            // Update tooltip position on scroll in metro map panel
            document.getElementById('metro-map-container')?.addEventListener('scroll', function() {
                if (tooltip && tooltip.classList.contains('visible')) {
                    tooltip.classList.remove('visible');
                }
            });
        })();

        // Metro map pan functionality
        (function() {
            let isPanning = false;
            let startX = 0;
            let startY = 0;

            const mapContainer = document.getElementById('metro-map-container');
            if (!mapContainer) return;

            // Mouse down - start panning
            mapContainer.addEventListener('mousedown', function(e) {
                // Don't pan if clicking a station
                if (e.target.closest('.metro-station')) return;

                const currentBranch = metroMapState.currentBranch;
                if (!currentBranch) return;

                const visibleMap = document.getElementById('metro-map-' + currentBranch);
                const svg = visibleMap?.querySelector('.metro-map');
                if (!svg) return;

                isPanning = true;

                // Get or initialize pan state for this branch
                let panState = metroMapState.panStates.get(currentBranch);
                if (!panState) {
                    panState = { currentX: 0, currentY: 0 };
                    metroMapState.panStates.set(currentBranch, panState);
                }

                startX = e.clientX - panState.currentX;
                startY = e.clientY - panState.currentY;
                mapContainer.classList.add('panning');
            });

            // Mouse move - perform panning
            document.addEventListener('mousemove', function(e) {
                if (!isPanning) return;

                const currentBranch = metroMapState.currentBranch;
                if (!currentBranch) return;

                const visibleMap = document.getElementById('metro-map-' + currentBranch);
                const svg = visibleMap?.querySelector('.metro-map');
                if (!svg) return;

                const panState = metroMapState.panStates.get(currentBranch);
                if (!panState) return;

                const newX = e.clientX - startX;
                const newY = e.clientY - startY;

                // Calculate bounds from actual SVG dimensions (width/height attributes) + 60px edge margin
                const containerRect = mapContainer.getBoundingClientRect();
                const svgWidth = parseFloat(svg.getAttribute('width')) || containerRect.width;
                const svgHeight = parseFloat(svg.getAttribute('height')) || containerRect.height;
                const EDGE_MARGIN = 60;

                const minX = Math.min(0, containerRect.width - svgWidth - EDGE_MARGIN);
                const maxX = EDGE_MARGIN;
                const minY = Math.min(0, containerRect.height - svgHeight - EDGE_MARGIN);
                const maxY = EDGE_MARGIN;

                // Constrain to bounds with breathing room on all edges
                panState.currentX = Math.max(minX, Math.min(maxX, newX));
                panState.currentY = Math.max(minY, Math.min(maxY, newY));

                svg.style.transform = \`translate(\${panState.currentX}px, \${panState.currentY}px)\`;
            });

            // Mouse up - stop panning
            document.addEventListener('mouseup', function() {
                if (isPanning) {
                    isPanning = false;
                    mapContainer.classList.remove('panning');
                }
            });

            // Mouse leave - stop panning
            document.addEventListener('mouseleave', function() {
                if (isPanning) {
                    isPanning = false;
                    mapContainer.classList.remove('panning');
                }
            });

            // Mouse wheel for vertical (and horizontal with shift) scrolling
            mapContainer.addEventListener('wheel', function(e) {
                const currentBranch = metroMapState.currentBranch;
                if (!currentBranch) return;

                const visibleMap = document.getElementById('metro-map-' + currentBranch);
                const svg = visibleMap?.querySelector('.metro-map');
                if (!svg) return;

                let panState = metroMapState.panStates.get(currentBranch);
                if (!panState) {
                    panState = { currentX: 0, currentY: 0 };
                    metroMapState.panStates.set(currentBranch, panState);
                }

                // Calculate bounds
                const containerRect = mapContainer.getBoundingClientRect();
                const svgWidth = parseFloat(svg.getAttribute('width')) || containerRect.width;
                const svgHeight = parseFloat(svg.getAttribute('height')) || containerRect.height;
                const EDGE_MARGIN = 60;

                const minX = Math.min(0, containerRect.width - svgWidth - EDGE_MARGIN);
                const maxX = EDGE_MARGIN;
                const minY = Math.min(0, containerRect.height - svgHeight - EDGE_MARGIN);
                const maxY = EDGE_MARGIN;

                // Apply scroll delta (shift+wheel = horizontal, wheel = vertical)
                if (e.shiftKey) {
                    panState.currentX = Math.max(minX, Math.min(maxX, panState.currentX - e.deltaY));
                } else {
                    panState.currentY = Math.max(minY, Math.min(maxY, panState.currentY - e.deltaY));
                    panState.currentX = Math.max(minX, Math.min(maxX, panState.currentX - e.deltaX));
                }

                svg.style.transform = \`translate(\${panState.currentX}px, \${panState.currentY}px)\`;
                e.preventDefault();
            }, { passive: false });
        })();
    </script>
</body>
</html>`;
}