import { parseAllSpecs, findNextAssertion } from '../parser/index.js';

// Status icons mapping
const STATUS_ICONS = {
  'done': '✅',
  'in_progress': '🚧',
  'not_started': '📋',
  'blocked': '⏸️',
  'draft': '📝'
};

// Get status icon for a given status
function getStatusIcon(status) {
  return STATUS_ICONS[status] || STATUS_ICONS['not_started'];
}

// Calculate completion statistics for a spec
function getCompletionStats(assertions) {
  const total = assertions.length;
  const done = assertions.filter(a => a.status === 'done').length;
  return { done, total };
}

// Format a spec with its assertions
function formatSpec(spec, assertions) {
  const specAssertions = assertions.filter(a => a.parent === spec.id);
  const { done, total } = getCompletionStats(specAssertions);
  const specIcon = getStatusIcon(spec.status);
  
  let output = [];
  
  // Spec header with completion ratio
  output.push(`${specIcon} ${spec.title} (${done}/${total} assertions complete)`);
  
  // Sort assertions by priority, then by creation date
  const sortedAssertions = specAssertions.sort((a, b) => {
    if (a.priority !== b.priority) {
      return a.priority - b.priority;
    }
    if (a.created !== b.created) {
      return a.created.localeCompare(b.created);
    }
    return a.id.localeCompare(b.id);
  });
  
  // Add assertions with indentation
  for (const assertion of sortedAssertions) {
    const icon = getStatusIcon(assertion.status);
    output.push(`  ${icon} ${assertion.title}`);
  }
  
  return output;
}

// Main status display function
export async function showStatus() {
  try {
    const { specs, assertions } = parseAllSpecs();
    
    // Handle empty specs directory
    if (specs.length === 0 && assertions.length === 0) {
      console.log('📋 No specifications found in specs/ directory');
      console.log('');
      console.log('To get started, create a spec file following the pattern:');
      console.log('  specs/{spec-name}/{spec-name}.md');
      console.log('  specs/{spec-name}/assertions/');
      return;
    }
    
    console.log('📊 Spekk Status Overview');
    console.log('========================');
    console.log('');
    
    // Sort specs by priority, then by id
    const sortedSpecs = specs.sort((a, b) => {
      if (a.priority !== b.priority) {
        return a.priority - b.priority;
      }
      return a.id.localeCompare(b.id);
    });
    
    // Display each spec with its assertions
    for (const spec of sortedSpecs) {
      const specOutput = formatSpec(spec, assertions);
      for (const line of specOutput) {
        console.log(line);
      }
      console.log(''); // Empty line between specs
    }
    
    // Overall statistics
    const totalAssertions = assertions.length;
    const doneAssertions = assertions.filter(a => a.status === 'done').length;
    const inProgressAssertions = assertions.filter(a => a.status === 'in_progress').length;
    const notStartedAssertions = assertions.filter(a => a.status === 'not_started').length;
    const blockedAssertions = assertions.filter(a => a.status === 'blocked').length;
    
    console.log('📈 Overall Progress');
    console.log('-------------------');
    console.log(`Total: ${totalAssertions} assertions`);
    console.log(`✅ Done: ${doneAssertions}`);
    console.log(`🚧 In Progress: ${inProgressAssertions}`);
    console.log(`📋 Not Started: ${notStartedAssertions}`);
    if (blockedAssertions > 0) {
      console.log(`⏸️ Blocked: ${blockedAssertions}`);
    }
    
    const completionPercentage = totalAssertions > 0 ? Math.round((doneAssertions / totalAssertions) * 100) : 0;
    console.log(`📊 Completion: ${completionPercentage}%`);
    console.log('');
    
    // Show next priority item
    const nextAssertion = findNextAssertion(assertions);
    if (nextAssertion) {
      const parentSpec = specs.find(s => s.id === nextAssertion.parent);
      console.log('🎯 Next Priority Item');
      console.log('---------------------');
      console.log(`→ ${nextAssertion.title}`);
      console.log(`  Spec: ${parentSpec?.title || nextAssertion.parent}`);
      console.log(`  Priority: ${nextAssertion.priority}`);
      console.log(`  Status: ${getStatusIcon(nextAssertion.status)} ${nextAssertion.status}`);
      console.log(`  File: ${nextAssertion.file}`);
    } else {
      console.log('🎉 All specifications complete!');
      console.log('No remaining assertions to work on.');
    }
    
  } catch (error) {
    console.error('❌ Error displaying status:', error.message);
    process.exit(1);
  }
}

// Direct invocation support
if (import.meta.url === `file://${process.argv[1]}`) {
  showStatus();
}