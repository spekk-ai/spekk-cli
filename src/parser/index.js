import fs from 'fs';
import path from 'path';

// Simple YAML frontmatter parser (since we don't have gray-matter)
function parseFrontmatter(content) {
  const lines = content.split('\n');
  
  if (lines[0] !== '---') {
    throw new Error('File must start with --- YAML frontmatter delimiter');
  }
  
  let frontmatterEnd = -1;
  for (let i = 1; i < lines.length; i++) {
    if (lines[i] === '---') {
      frontmatterEnd = i;
      break;
    }
  }
  
  if (frontmatterEnd === -1) {
    throw new Error('Missing closing --- delimiter for YAML frontmatter');
  }
  
  const yamlContent = lines.slice(1, frontmatterEnd).join('\n');
  const markdownContent = lines.slice(frontmatterEnd + 1).join('\n');
  
  // Simple YAML parser for our specific needs
  const frontmatter = {};
  yamlContent.split('\n').forEach(line => {
    const match = line.match(/^([^:]+):\s*(.+)$/);
    if (match) {
      const key = match[1].trim();
      let value = match[2].trim();
      
      // Handle different value types
      if (value === 'true') value = true;
      else if (value === 'false') value = false;
      else if (/^\d+$/.test(value)) value = parseInt(value);
      else if (value.startsWith('"') && value.endsWith('"')) {
        value = value.slice(1, -1);
      }
      
      frontmatter[key] = value;
    }
  });
  
  return { data: frontmatter, content: markdownContent };
}

// Extract title from markdown content (first H1 heading)
function extractTitle(content) {
  const match = content.match(/^# (.+)$/m);
  return match ? match[1] : 'Untitled';
}

// Validate required fields
function validateFields(data, filePath, isAssertion = false) {
  const requiredFields = isAssertion 
    ? ['id', 'parent', 'created', 'priority']
    : ['id', 'created', 'priority'];
    
  for (const field of requiredFields) {
    if (data[field] === undefined || data[field] === null) {
      throw new Error(`Missing required field '${field}' in ${filePath}`);
    }
  }
  
  // Validate ID format (kebab-case)
  const kebabCasePattern = /^[a-z][a-z0-9]*(-[a-z0-9]+)*$/;
  if (!kebabCasePattern.test(data.id)) {
    throw new Error(`Invalid id format '${data.id}' (must be kebab-case: lowercase with hyphens, no spaces/underscores/special chars) in ${filePath}`);
  }
  
  // Validate priority
  if (![1, 2, 3].includes(data.priority)) {
    throw new Error(`Invalid priority value '${data.priority}' (must be: 1, 2, or 3) in ${filePath}`);
  }
  
  // Validate status if present
  if (data.status && !['not_started', 'in_progress', 'done', 'draft', 'failed'].includes(data.status)) {
    throw new Error(`Invalid status value '${data.status}' (must be: not_started, in_progress, done, draft, failed) in ${filePath}`);
  }
  
  // Validate timestamp format
  const timestampPattern = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$/;
  if (!timestampPattern.test(data.created)) {
    throw new Error(`Invalid ISO 8601 timestamp in 'created' field: '${data.created}' in ${filePath}`);
  }
  
  if (data.updated && !timestampPattern.test(data.updated)) {
    throw new Error(`Invalid ISO 8601 timestamp in 'updated' field: '${data.updated}' in ${filePath}`);
  }
}

// Validate folder structure requirements
function validateFolderStructure(specsDir) {
  // Check for flat .md files at specs/ level (not allowed)
  const specsDirContents = fs.readdirSync(specsDir);
  const flatMdFiles = specsDirContents.filter(item => {
    const itemPath = path.join(specsDir, item);
    return fs.statSync(itemPath).isFile() && item.endsWith('.md');
  });
  
  if (flatMdFiles.length > 0) {
    throw new Error(`Invalid folder structure: Found flat .md files in specs/: ${flatMdFiles.join(', ')}. All specs must be in folders following the pattern specs/{spec-id}/{spec-id}.md`);
  }
  
  // Check each spec directory has required structure
  const specDirs = specsDirContents.filter(item => {
    const itemPath = path.join(specsDir, item);
    return fs.statSync(itemPath).isDirectory();
  });
  
  for (const specDir of specDirs) {
    const specDirPath = path.join(specsDir, specDir);
    const expectedSpecFile = path.join(specDirPath, `${specDir}.md`);
    const assertionsDir = path.join(specDirPath, 'assertions');
    
    // Check main spec file exists with matching name
    if (!fs.existsSync(expectedSpecFile)) {
      throw new Error(`Invalid folder structure: Missing main spec file specs/${specDir}/${specDir}.md`);
    }
    
    // Check assertions directory exists
    if (!fs.existsSync(assertionsDir)) {
      throw new Error(`Invalid folder structure: Missing assertions directory specs/${specDir}/assertions/`);
    }
    
    // Verify assertions directory is actually a directory
    if (!fs.statSync(assertionsDir).isDirectory()) {
      throw new Error(`Invalid folder structure: specs/${specDir}/assertions must be a directory`);
    }
  }
}

// Read and parse all specs and assertions from current working directory
function parseAllSpecs() {
  const specsDir = path.join(process.cwd(), 'specs');
  
  if (!fs.existsSync(specsDir)) {
    return { specs: [], assertions: [] };
  }
  
  // Validate folder structure before parsing
  validateFolderStructure(specsDir);
  
  const specs = [];
  const assertions = [];
  const specIds = new Map(); // Changed to Map to track file names
  
  const specDirs = fs.readdirSync(specsDir).filter(dir => {
    const dirPath = path.join(specsDir, dir);
    return fs.statSync(dirPath).isDirectory();
  });
  
  for (const specDir of specDirs) {
    const specDirPath = path.join(specsDir, specDir);
    const specFilePath = path.join(specDirPath, `${specDir}.md`);
    
    // Parse spec file
    if (fs.existsSync(specFilePath)) {
      const content = fs.readFileSync(specFilePath, 'utf8');
      const { data, content: markdownContent } = parseFrontmatter(content);
      
      validateFields(data, `specs/${specDir}/${specDir}.md`, false);
      
      // Check for duplicate spec IDs
      const currentSpecFile = `specs/${specDir}/${specDir}.md`;
      if (specIds.has(data.id)) {
        const existingFile = specIds.get(data.id);
        throw new Error(`Duplicate spec id '${data.id}' found in files: ${existingFile}, ${currentSpecFile}`);
      }
      specIds.set(data.id, currentSpecFile);
      
      specs.push({
        ...data,
        status: data.status || 'not_started',
        file: `specs/${specDir}/${specDir}.md`,
        title: extractTitle(markdownContent),
        content: content
      });
    }
    
    // Parse assertions
    const assertionsDir = path.join(specDirPath, 'assertions');
    if (fs.existsSync(assertionsDir)) {
      const assertionIds = new Map(); // Changed to Map to track file names
      const assertionFiles = fs.readdirSync(assertionsDir).filter(file => file.endsWith('.md'));
      
      for (const assertionFile of assertionFiles) {
        const assertionPath = path.join(assertionsDir, assertionFile);
        const content = fs.readFileSync(assertionPath, 'utf8');
        const { data, content: markdownContent } = parseFrontmatter(content);
        
        validateFields(data, `specs/${specDir}/assertions/${assertionFile}`, true);
        
        // Check for duplicate assertion IDs within this spec
        if (assertionIds.has(data.id)) {
          const existingFile = assertionIds.get(data.id);
          throw new Error(`Duplicate assertion id '${data.id}' in spec '${specDir}' found in files: ${existingFile}, ${assertionFile}`);
        }
        assertionIds.set(data.id, assertionFile);
        
        assertions.push({
          ...data,
          status: data.status || 'not_started',
          file: `specs/${specDir}/assertions/${assertionFile}`,
          title: extractTitle(markdownContent),
          content: content
        });
      }
    }
  }
  
  // Validate that assertion parents exist
  for (const assertion of assertions) {
    if (!specIds.has(assertion.parent)) {
      throw new Error(`Parent spec '${assertion.parent}' not found for assertion '${assertion.id}'`);
    }
  }
  
  // Update parent spec statuses based on child assertions
  for (const spec of specs) {
    const computedStatus = computeParentStatus(spec.id, assertions);
    spec.status = computedStatus;
  }
  
  return { specs, assertions };
}

// Compute parent spec status based on child assertions
function computeParentStatus(parentId, assertions) {
  const childAssertions = assertions.filter(a => a.parent === parentId);
  
  if (childAssertions.length === 0) {
    return 'not_started';
  }
  
  // Filter out draft children for status computation
  const activeChildren = childAssertions.filter(a => a.status !== 'draft');
  
  if (activeChildren.length === 0) {
    return 'not_started';
  }
  
  // If any children are failed, parent is failed
  if (activeChildren.some(a => a.status === 'failed')) {
    return 'failed';
  }
  
  // If all active children are done, parent is done
  if (activeChildren.every(a => a.status === 'done')) {
    return 'done';
  }
  
  // If any children are in_progress or not_started, parent is in_progress
  if (activeChildren.some(a => ['in_progress', 'not_started'].includes(a.status))) {
    return 'in_progress';
  }
  
  // Default to not_started
  return 'not_started';
}

// Find next priority assertion
function findNextAssertion(assertions) {
  // Filter to incomplete items (exclude done and draft, but include failed for retry)
  const incomplete = assertions.filter(a => !['done', 'draft'].includes(a.status));
  
  if (incomplete.length === 0) {
    return null;
  }
  
  // Sort by priority (1 highest), then by created date (oldest first), then by id
  incomplete.sort((a, b) => {
    if (a.priority !== b.priority) {
      return a.priority - b.priority;
    }
    if (a.created !== b.created) {
      return a.created.localeCompare(b.created);
    }
    return a.id.localeCompare(b.id);
  });
  
  return incomplete[0];
}

// Main function
export function run(options = {}) {
  try {
    const { specs, assertions } = parseAllSpecs();
    
    if (specs.length === 0 && assertions.length === 0) {
      console.log(JSON.stringify({
        status: 'empty',
        message: 'No specifications found in specs/ directory'
      }, null, 2));
      return;
    }
    
    // If --all flag is specified, return complete hierarchy
    if (options.all) {
      const specsWithAssertions = specs.map(spec => ({
        id: spec.id,
        title: spec.title,
        status: spec.status,
        priority: spec.priority,
        file: spec.file,
        assertions: assertions
          .filter(assertion => assertion.parent === spec.id)
          .map(assertion => ({
            id: assertion.id,
            title: assertion.title,
            status: assertion.status,
            priority: assertion.priority,
            file: assertion.file
          }))
          .sort((a, b) => {
            // Sort by priority, then by creation date
            if (a.priority !== b.priority) {
              return a.priority - b.priority;
            }
            return a.id.localeCompare(b.id);
          })
      })).sort((a, b) => {
        // Sort specs by priority, then by id
        if (a.priority !== b.priority) {
          return a.priority - b.priority;
        }
        return a.id.localeCompare(b.id);
      });
      
      console.log(JSON.stringify({
        type: 'hierarchy',
        specs: specsWithAssertions
      }, null, 2));
      return;
    }
    
    const nextAssertion = findNextAssertion(assertions);
    
    if (!nextAssertion) {
      console.log(JSON.stringify({
        type: 'complete',
        status: 'complete',
        message: 'All specifications are complete'
      }, null, 2));
      return;
    }
    
    // Find the parent spec
    const parentSpec = specs.find(s => s.id === nextAssertion.parent);
    
    console.log(JSON.stringify({
      type: 'assertion',
      id: nextAssertion.id,
      parent: nextAssertion.parent,
      file: nextAssertion.file,
      priority: nextAssertion.priority,
      status: nextAssertion.status,
      title: nextAssertion.title,
      content: nextAssertion.content,
      spec: parentSpec ? {
        id: parentSpec.id,
        file: parentSpec.file,
        title: parentSpec.title
      } : null
    }, null, 2));
    
  } catch (error) {
    console.log(JSON.stringify({
      error: true,
      message: error.message
    }, null, 2));
    process.exit(1);
  }
}

// Export the parser functions for testing
export { parseAllSpecs, findNextAssertion, parseFrontmatter, validateFields, extractTitle, validateFolderStructure, computeParentStatus };