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

function getPriorityIcon(priority) {
  return '';
}

function generateDetailStatusBadge(status) {
  const icon = getStatusIcon(status);
  return `<span class="detail-status-badge">${icon}</span>`;
}

function generateDetailPriorityBadge(priority) {
  return `<span class="detail-priority-badge">${priority}</span>`;
}

function escapeForJS(str) {
  // Use JSON.stringify for proper JavaScript string escaping
  return JSON.stringify(str);
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

function calculateDependencyDepth(assertion, assertions) {
  if (!assertion.dependsOn) {
    return 0;
  }

  const parent = assertions.find(a => a.id === assertion.dependsOn);
  if (!parent) {
    return 0;
  }

  return 1 + calculateDependencyDepth(parent, assertions);
}

function generateMetroMapSVG(currentAssertion, allAssertions) {
  // Filter assertions in the same branch
  const assertionBranch = currentAssertion.branch || 'main';
  const branchAssertions = allAssertions
    .filter(a => (a.branch || 'main') === assertionBranch)
    .map(a => ({
      ...a,
      depth: calculateDependencyDepth(a, allAssertions)
    }))
    .sort((a, b) => {
      // Sort by depth first, then by created date
      if (a.depth !== b.depth) return a.depth - b.depth;
      return a.created.localeCompare(b.created);
    });

  if (branchAssertions.length === 0) {
    return '';
  }

  const stationSpacing = 110;
  const startX = 60;
  const trackY = 80;
  const stationRadius = 8;
  const currentStationRadius = 10;
  const verticalSpacing = 50;

  // Calculate positions - handle multiple assertions at same depth
  const positions = new Map();
  const depthCounts = new Map();

  branchAssertions.forEach((assertion) => {
    const depth = assertion.depth;
    const countAtDepth = depthCounts.get(depth) || 0;
    depthCounts.set(depth, countAtDepth + 1);

    const x = startX + (depth * stationSpacing);
    const y = trackY + (countAtDepth * verticalSpacing);

    positions.set(assertion.id, { x, y });
  });

  const maxX = Math.max(...Array.from(positions.values()).map(p => p.x)) + 100;
  const maxY = Math.max(...Array.from(positions.values()).map(p => p.y)) + 50;
  const svgWidth = Math.max(maxX, 800);
  const svgHeight = Math.max(maxY, 200);

  // Generate SVG
  let svg = `
<svg class="metro-map" width="${svgWidth}" height="${svgHeight}" viewBox="0 0 ${svgWidth} ${svgHeight}">
  <!-- Main track -->
  <line class="metro-track" x1="40" y1="${trackY}" x2="${maxX}" y2="${trackY}"/>
  `;

  // Add dependency lines
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
        fill="none" stroke="${getStatusColor(assertion.status)}" stroke-width="3" opacity="0.4"/>`;
        } else {
          svg += `
  <!-- Dependency: ${escapeHTML(assertion.dependsOn)} → ${escapeHTML(assertion.id)} -->
  <line class="metro-dependency" x1="${parentPos.x}" y1="${parentPos.y}" x2="${childPos.x}" y2="${childPos.y}"
        stroke="${getStatusColor(assertion.status)}" stroke-width="4" opacity="0.3"/>`;
        }
      }
    }
  });

  // Add stations
  branchAssertions.forEach(assertion => {
    const pos = positions.get(assertion.id);
    if (!pos) return;

    const isCurrent = assertion.id === currentAssertion.id;
    const radius = isCurrent ? currentStationRadius : stationRadius;
    const color = getStatusColor(assertion.status);
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
    <circle r="${radius}" fill="${color}" stroke="#fff" stroke-width="${strokeWidth}"${glowFilter}/>
    <text class="metro-label" y="28" style="font-size: 10px; fill: #1e293b; text-anchor: middle; font-weight: ${isCurrent ? '700' : '400'};">${escapeHTML(displayTitle)}</text>
  </g>`;
  });

  svg += `
</svg>`;

  return svg;
}

export function generateSpecExplorerHTML(specs, assertions) {
  // Get project name from current working directory
  const projectName = basename(process.cwd());
  
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

        .metro-map-section {
            margin: 20px 0;
            padding: 16px;
            background: #f8fafc;
            border-radius: 8px;
            border: 1px solid #e2e8f0;
            overflow-x: auto;
        }

        .metro-map-title {
            font-size: 13px;
            font-weight: 600;
            color: #475569;
            margin-bottom: 12px;
            text-transform: uppercase;
            letter-spacing: 0.5px;
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
            transition: transform 0.15s;
        }

        .metro-station:hover {
            transform: scale(1.15);
        }

        .metro-label {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            pointer-events: none;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="tree-panel">
            <div class="header">
                <h1>Spec Tree - ${projectName}</h1>
                <p>${specs.length} specs, ${assertions.length} assertions</p>
            </div>
            
            <ul class="spec-tree">
                ${specHierarchy.map(spec => `
                    <li class="spec-item">
                        <div class="spec-header" data-action="toggle-spec" data-spec-id="${escapeHTML(spec.id)}" data-show-detail="true" data-type="spec">
                            <span class="toggle-icon" id="toggle-${spec.id}">▶</span>
                            <span class="priority-badge priority-${spec.priority}">${spec.priority}</span>
                            <span class="status-badge status-${spec.status}"></span>
                            <span class="spec-title">${escapeHTML(spec.title)}</span>
                        </div>
                        
                        <ul class="assertions-list" id="assertions-${spec.id}">
                            ${spec.assertions.map(assertion => `
                                <li class="assertion-item" data-action="show-detail" data-assertion-id="${escapeHTML(assertion.id)}" data-type="assertion">
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
                <div class="detail-content" id="detail-assertion-${assertion.id}">
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
                    <div class="metro-map-section">
                        <h3 class="metro-map-title">Branch Dependencies</h3>
                        ${generateMetroMapSVG(assertion, assertions)}
                    </div>
                    <div class="detail-body">
                        <pre style="white-space: pre-wrap; font-family: inherit;">${escapeHTML(assertion.content)}</pre>
                    </div>
                </div>
            `).join('')}
        </div>
    </div>
    
    <script>
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
    </script>
</body>
</html>`;
}