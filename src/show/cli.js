import { mkdirSync, existsSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
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
}

function generateSpecExplorerHTML(specs, assertions) {
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
    <title>Spec Explorer - Spekk</title>
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
            padding: 2px 6px;
            border-radius: 4px;
            font-size: 11px;
            font-weight: 500;
            text-transform: uppercase;
            margin-right: 8px;
        }
        
        .status-not_started {
            background: #fef3c7;
            color: #92400e;
        }
        
        .status-in_progress {
            background: #dbeafe;
            color: #1e40af;
        }
        
        .status-done {
            background: #d1fae5;
            color: #065f46;
        }
        
        .priority-badge {
            padding: 2px 6px;
            border-radius: 4px;
            font-size: 10px;
            font-weight: 600;
            color: white;
            margin-right: 8px;
        }
        
        .priority-1 {
            background: #ef4444;
        }
        
        .priority-2 {
            background: #f59e0b;
        }
        
        .priority-3 {
            background: #10b981;
        }
        
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
            gap: 12px;
            margin-bottom: 16px;
        }
        
        .meta-item {
            font-size: 12px;
            color: #64748b;
        }
        
        .detail-body {
            prose: true;
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
    </style>
</head>
<body>
    <div class="container">
        <div class="tree-panel">
            <div class="header">
                <h1>Spec Tree</h1>
                <p>${specs.length} specs, ${assertions.length} assertions</p>
            </div>
            
            <ul class="spec-tree">
                ${specHierarchy.map(spec => `
                    <li class="spec-item">
                        <div class="spec-header" onclick="toggleSpec('${spec.id}')">
                            <span class="toggle-icon" id="toggle-${spec.id}">▶</span>
                            <span class="status-badge status-${spec.status}">${spec.status.replace('_', ' ')}</span>
                            <span class="priority-badge priority-${spec.priority}">${spec.priority}</span>
                            <span class="spec-title">${spec.title}</span>
                        </div>
                        
                        <ul class="assertions-list" id="assertions-${spec.id}">
                            ${spec.assertions.map(assertion => `
                                <li class="assertion-item" onclick="showDetail('${assertion.id}', 'assertion', event)">
                                    <div style="display: flex; align-items: center;">
                                        <span class="status-badge status-${assertion.status}">${assertion.status.replace('_', ' ')}</span>
                                        <span class="priority-badge priority-${assertion.priority}">${assertion.priority}</span>
                                        <span>${assertion.title}</span>
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
                        <div class="detail-title">${spec.title}</div>
                        <div class="detail-meta">
                            <span class="meta-item">Status: <strong>${spec.status}</strong></span>
                            <span class="meta-item">Priority: <strong>${spec.priority}</strong></span>
                            <span class="meta-item">File: <strong>${spec.file}</strong></span>
                        </div>
                    </div>
                    <div class="detail-body">
                        <pre style="white-space: pre-wrap; font-family: inherit;">${spec.content}</pre>
                    </div>
                </div>
            `).join('')}
            
            ${assertions.map(assertion => `
                <div class="detail-content" id="detail-assertion-${assertion.id}">
                    <div class="detail-header">
                        <div class="detail-title">${assertion.title}</div>
                        <div class="detail-meta">
                            <span class="meta-item">Status: <strong>${assertion.status}</strong></span>
                            <span class="meta-item">Priority: <strong>${assertion.priority}</strong></span>
                            <span class="meta-item">Parent: <strong>${assertion.parent}</strong></span>
                            <span class="meta-item">File: <strong>${assertion.file}</strong></span>
                        </div>
                    </div>
                    <div class="detail-body">
                        <pre style="white-space: pre-wrap; font-family: inherit;">${assertion.content}</pre>
                    </div>
                </div>
            `).join('')}
        </div>
    </div>
    
    <script>
        function toggleSpec(specId) {
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
                showDetail(specId, 'spec', event);
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
            
            // Mark assertion as selected
            if (type === 'assertion' && event) {
                event.currentTarget.classList.add('selected');
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