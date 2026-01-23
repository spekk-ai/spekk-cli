import { promises as fs } from 'node:fs';
import path from 'node:path';
import { parseAllSpecs } from '../parser/index.js';

// Observer configuration
const DEFAULT_SCAN_INTERVAL = 30; // seconds
const OBSERVATIONS_DIR = 'observations';

export class ObserverAgent {
  constructor(options = {}) {
    this.scanInterval = options.scanInterval || DEFAULT_SCAN_INTERVAL;
    this.quiet = options.quiet || false;
    this.running = false;
    this.scanCount = 0;
  }

  log(message) {
    if (!this.quiet) {
      const timestamp = new Date().toISOString().replace('T', ' ').replace(/\.\d{3}Z/, '');
      console.log(`[${timestamp}] ${message}`);
    }
  }

  async ensureObservationsDirectory() {
    try {
      await fs.mkdir(OBSERVATIONS_DIR, { recursive: true });
    } catch (error) {
      // Directory might already exist, that's fine
    }
  }

  async scanForDrift() {
    this.log('Observer scan starting...');
    
    let issues = [];

    try {
      // Scan specs directory
      this.log('Scanning specs/ directory...');
      const { specs, assertions } = parseAllSpecs();
      this.log(`Found ${specs.length} specs and ${assertions.length} assertions`);

      // Type 1: Code-Spec Misalignment - Check for missing CLI commands
      await this.checkMissingCliCommands(specs, assertions, issues);

      // Type 1: Code-Spec Misalignment - Check for missing directories/files
      await this.checkMissingPaths(specs, assertions, issues);

      // Type 2: Outdated Specs - Check for done specs with changed files
      await this.checkOutdatedSpecs(specs, assertions, issues);

      // Type 4: Spec Conflicts - Check for contradictory requirements
      await this.checkSpecConflicts(specs, assertions, issues);

      this.log(`Found ${issues.length} potential issues`);

      // Create observations for any issues found
      for (const issue of issues) {
        await this.createObservation(issue);
      }

      this.log(`Scan complete. Next scan in ${this.scanInterval}s...`);
      
    } catch (error) {
      this.log(`❌ Error during scan: ${error.message}`);
    }
  }

  async checkMissingCliCommands(specs, assertions, issues) {
    // Look for assertions that require npm commands
    for (const assertion of assertions) {
      if (assertion.content.includes('npm run ') && assertion.status !== 'done') {
        const matches = assertion.content.match(/`npm run (\w+)`/g);
        if (matches) {
          for (const match of matches) {
            const command = match.replace(/`npm run (\w+)`/, '$1');
            if (await this.isCommandMissing(command)) {
              issues.push({
                type: 'code_spec_misalignment',
                severity: 'high',
                title: `Missing CLI command: npm run ${command}`,
                description: `Assertion ${assertion.id} requires 'npm run ${command}' but it's not defined in package.json`,
                affected_specs: [assertion.parent],
                affected_files: ['package.json', assertion.file]
              });
            }
          }
        }
      }
    }
  }

  async isCommandMissing(command) {
    try {
      const packageJson = JSON.parse(await fs.readFile('package.json', 'utf8'));
      return !packageJson.scripts || !packageJson.scripts[command];
    } catch (error) {
      return true; // If we can't read package.json, assume missing
    }
  }

  async checkMissingPaths(specs, assertions, issues) {
    // Check for paths mentioned in specs that don't exist
    for (const assertion of assertions) {
      const pathMatches = assertion.content.match(/`([^`]+\.(js|md|json|ts))`/g);
      if (pathMatches) {
        for (const match of pathMatches) {
          const filePath = match.replace(/`([^`]+)`/, '$1');
          if (filePath.includes('/') && !await this.pathExists(filePath)) {
            issues.push({
              type: 'code_spec_misalignment',
              severity: 'medium',
              title: `Missing file referenced in spec: ${filePath}`,
              description: `Assertion ${assertion.id} references ${filePath} but it doesn't exist`,
              affected_specs: [assertion.parent],
              affected_files: [assertion.file]
            });
          }
        }
      }
    }
  }

  async pathExists(filePath) {
    try {
      await fs.access(filePath);
      return true;
    } catch {
      return false;
    }
  }

  async checkOutdatedSpecs(specs, assertions, issues) {
    // Check for specs marked as done but with recent file changes
    for (const spec of specs) {
      if (spec.status === 'done') {
        const specAssertions = assertions.filter(a => a.parent === spec.id);
        const allDone = specAssertions.every(a => a.status === 'done');
        
        if (!allDone) {
          issues.push({
            type: 'outdated_specs',
            severity: 'medium',
            title: `Spec marked done but has incomplete assertions: ${spec.id}`,
            description: `Spec ${spec.id} has status 'done' but contains assertions that are not done`,
            affected_specs: [spec.id],
            affected_files: [spec.file]
          });
        }
      }
    }
  }

  async checkSpecConflicts(specs, assertions, issues) {
    // Check for mutually exclusive technology requirements
    await this.checkTechnologyConflicts(specs, issues);
    
    // Check for conflicting file/directory structure requirements
    await this.checkFileStructureConflicts(specs, issues);
    
    // Check for contradictory behavior specifications
    await this.checkBehaviorConflicts(specs, issues);
    
    // Check for priority conflicts (high-priority specs blocking each other)
    await this.checkPriorityConflicts(specs, issues);
  }

  async checkTechnologyConflicts(specs, issues) {
    const technologies = {
      database: ['sqlite', 'postgresql', 'mysql', 'mongodb'],
      framework: ['react', 'vue', 'angular', 'svelte'],
      language: ['javascript', 'typescript', 'python', 'java']
    };

    for (const [category, techList] of Object.entries(technologies)) {
      const specsByTech = {};
      
      for (const spec of specs) {
        const content = spec.content.toLowerCase();
        const requiredTechs = techList.filter(tech => content.includes(tech));
        
        for (const tech of requiredTechs) {
          if (!specsByTech[tech]) {
            specsByTech[tech] = [];
          }
          specsByTech[tech].push(spec);
        }
      }

      // Find conflicts within the same category
      const usedTechs = Object.keys(specsByTech);
      if (usedTechs.length > 1) {
        const conflictingSpecs = [];
        const conflictingFiles = [];
        
        for (const tech of usedTechs) {
          conflictingSpecs.push(...specsByTech[tech].map(s => s.id));
          conflictingFiles.push(...specsByTech[tech].map(s => s.file));
        }

        issues.push({
          type: 'spec_conflicts',
          severity: 'high',
          title: `Conflicting ${category} requirements: ${usedTechs.join(' vs ')}`,
          description: `Multiple specs require different ${category} technologies that cannot be used together`,
          affected_specs: [...new Set(conflictingSpecs)],
          affected_files: [...new Set(conflictingFiles)],
          conflict_type: 'technology_exclusive',
          blocking: true
        });
      }
    }
  }

  async checkFileStructureConflicts(specs, issues) {
    const fileRequirements = {};
    
    for (const spec of specs) {
      // Look for file path patterns like src/config/database.js or src/config/database.ts
      const fileMatches = spec.content.match(/\b[\w-]+\/[\w-]+\/[\w-]+\.\w+\b/g);
      if (fileMatches) {
        for (const filePath of fileMatches) {
          const basePath = filePath.replace(/\.\w+$/, ''); // Remove extension
          const extension = filePath.match(/\.(\w+)$/)?.[1];
          
          if (!fileRequirements[basePath]) {
            fileRequirements[basePath] = [];
          }
          
          fileRequirements[basePath].push({
            spec: spec,
            fullPath: filePath,
            extension: extension
          });
        }
      }
    }

    // Check for conflicts (same base path, different extensions)
    for (const [basePath, requirements] of Object.entries(fileRequirements)) {
      const uniqueExtensions = [...new Set(requirements.map(r => r.extension))];
      
      if (uniqueExtensions.length > 1) {
        const conflictingSpecs = requirements.map(r => r.spec.id);
        const conflictingFiles = requirements.map(r => r.spec.file);
        const conflictingPaths = requirements.map(r => r.fullPath);

        issues.push({
          type: 'spec_conflicts',
          severity: 'medium',
          title: `Conflicting file structure requirements: ${conflictingPaths.join(' vs ')}`,
          description: `Multiple specs require different file types for the same logical path: ${basePath}`,
          affected_specs: conflictingSpecs,
          affected_files: conflictingFiles,
          conflict_type: 'file_structure',
          blocking: false
        });
      }
    }
  }

  async checkBehaviorConflicts(specs, issues) {
    // Look for contradictory behavior keywords
    const behaviorPatterns = {
      authentication: [
        { keyword: 'require.*auth', behavior: 'required' },
        { keyword: 'no.*auth|disable.*auth|without.*auth', behavior: 'disabled' }
      ],
      caching: [
        { keyword: 'enable.*cach|use.*cach', behavior: 'enabled' },
        { keyword: 'disable.*cach|no.*cach', behavior: 'disabled' }
      ],
      logging: [
        { keyword: 'enable.*log|use.*log', behavior: 'enabled' },
        { keyword: 'disable.*log|no.*log', behavior: 'disabled' }
      ]
    };

    for (const [feature, patterns] of Object.entries(behaviorPatterns)) {
      const specsByBehavior = {};
      
      for (const spec of specs) {
        const content = spec.content.toLowerCase();
        
        for (const pattern of patterns) {
          const regex = new RegExp(pattern.keyword, 'i');
          if (regex.test(content)) {
            if (!specsByBehavior[pattern.behavior]) {
              specsByBehavior[pattern.behavior] = [];
            }
            specsByBehavior[pattern.behavior].push(spec);
          }
        }
      }

      // Find conflicts
      const behaviors = Object.keys(specsByBehavior);
      if (behaviors.length > 1 && behaviors.includes('enabled') && behaviors.includes('disabled')) {
        const conflictingSpecs = [];
        const conflictingFiles = [];
        
        for (const behavior of behaviors) {
          conflictingSpecs.push(...specsByBehavior[behavior].map(s => s.id));
          conflictingFiles.push(...specsByBehavior[behavior].map(s => s.file));
        }

        issues.push({
          type: 'spec_conflicts',
          severity: 'high',
          title: `Contradictory ${feature} behavior requirements`,
          description: `Some specs require ${feature} to be enabled while others require it to be disabled`,
          affected_specs: [...new Set(conflictingSpecs)],
          affected_files: [...new Set(conflictingFiles)],
          conflict_type: 'behavior_contradiction',
          blocking: true
        });
      }
    }
  }

  async checkPriorityConflicts(specs, issues) {
    const highPrioritySpecs = specs.filter(s => s.priority === 1);
    
    // Check for dependency-like conflicts between high priority specs
    for (let i = 0; i < highPrioritySpecs.length; i++) {
      for (let j = i + 1; j < highPrioritySpecs.length; j++) {
        const specA = highPrioritySpecs[i];
        const specB = highPrioritySpecs[j];
        
        // Look for blocking keywords
        const blockingPatterns = [
          'must be completed before',
          'requires.*to be done first',
          'depends on.*completion',
          'blocks.*implementation',
          'prevents.*from'
        ];

        let hasConflict = false;
        let conflictDescription = '';

        for (const pattern of blockingPatterns) {
          const regex = new RegExp(pattern, 'i');
          
          if (regex.test(specA.content) || regex.test(specB.content)) {
            // Check if they reference each other's domains
            const domainsA = this.extractDomains(specA.content);
            const domainsB = this.extractDomains(specB.content);
            
            const hasOverlap = domainsA.some(domain => domainsB.includes(domain));
            
            if (hasOverlap) {
              hasConflict = true;
              conflictDescription = `High-priority specs have conflicting dependencies in overlapping domains: ${domainsA.filter(d => domainsB.includes(d)).join(', ')}`;
              break;
            }
          }
        }

        if (hasConflict) {
          issues.push({
            type: 'spec_conflicts',
            severity: 'high',
            title: `Priority conflict between high-priority specs: ${specA.id} vs ${specB.id}`,
            description: conflictDescription,
            affected_specs: [specA.id, specB.id],
            affected_files: [specA.file, specB.file],
            conflict_type: 'priority_blocking',
            blocking: true
          });
        }
      }
    }
  }

  extractDomains(content) {
    const domains = [];
    const domainKeywords = [
      'database', 'auth', 'authentication', 'ui', 'api', 'frontend', 'backend',
      'config', 'configuration', 'test', 'testing', 'deploy', 'deployment'
    ];

    const lowercaseContent = content.toLowerCase();
    
    for (const keyword of domainKeywords) {
      if (lowercaseContent.includes(keyword)) {
        domains.push(keyword);
      }
    }

    return domains;
  }

  async createObservation(issue) {
    const timestamp = new Date().toISOString();
    const filename = `${timestamp.replace(/[:.]/g, '-')}.md`;
    const filePath = path.join(OBSERVATIONS_DIR, filename);

    const observationId = `${issue.type}-${Date.now()}`;
    
    let frontmatter = `---
id: ${observationId}
created: ${timestamp}
type: ${issue.type}
severity: ${issue.severity}
affected_specs:
${issue.affected_specs.map(spec => `  - ${spec}`).join('\n')}
affected_files:
${issue.affected_files.map(file => `  - ${file}`).join('\n')}`;

    // Add conflict-specific metadata for spec conflicts
    if (issue.type === 'spec_conflicts') {
      frontmatter += `
conflict_type: ${issue.conflict_type || 'unknown'}
blocking: ${issue.blocking || false}`;
    }

    frontmatter += `
---`;

    let recommendation = 'Review the affected specs and files to determine if updates are needed.';
    
    // Add conflict-specific recommendations
    if (issue.type === 'spec_conflicts') {
      if (issue.blocking) {
        recommendation = `**BLOCKING CONFLICT**: This conflict prevents implementation progress. Immediate resolution required:

1. Review conflicting specs: ${issue.affected_specs.join(', ')}
2. Determine which approach is correct for the project requirements
3. Update or remove conflicting specifications
4. Consider if specs can be modified to be complementary rather than contradictory`;
      } else {
        recommendation = `**INFORMATIONAL CONFLICT**: This conflict may cause confusion but doesn't block implementation:

1. Review conflicting specs: ${issue.affected_specs.join(', ')}
2. Clarify intent and scope of each specification
3. Consider consolidating or clearly separating concerns
4. Document any intentional differences`;
      }
    }
    
    const content = `${frontmatter}

# ${issue.title}

## Issue Description
${issue.description}

## Conflicting Specifications
${issue.affected_specs.map(spec => `- **${spec}**: Referenced in specs/${spec}/${spec}.md`).join('\n')}

## Evidence
This observation was automatically generated during system scan.

## Impact
${this.getImpactMessage(issue.severity)}

## Recommendation
${recommendation}
`;

    await fs.writeFile(filePath, content, 'utf8');
    this.log(`Created observation: ${observationId}`);
  }

  getImpactMessage(severity) {
    switch (severity) {
      case 'high':
        return 'Critical functionality may be broken or inaccessible.';
      case 'medium':
        return 'Important system behavior may not match specifications.';
      case 'low':
        return 'Minor inconsistency that could cause confusion.';
      default:
        return 'Unknown impact level.';
    }
  }

  async start() {
    this.running = true;
    this.log('🔍 Observer Agent starting...');
    
    // Ensure observations directory exists
    await this.ensureObservationsDirectory();
    
    // Handle graceful shutdown
    process.on('SIGINT', () => {
      this.log('\n🛑 Stopping Observer Agent...');
      this.running = false;
      process.exit(0);
    });

    process.on('SIGTERM', () => {
      this.log('\n🛑 Stopping Observer Agent...');
      this.running = false;
      process.exit(0);
    });

    // Start continuous monitoring loop
    while (this.running) {
      try {
        this.scanCount++;
        await this.scanForDrift();
        
        // Sleep for the configured interval
        if (this.running) {
          await new Promise(resolve => setTimeout(resolve, this.scanInterval * 1000));
        }
      } catch (error) {
        this.log(`❌ Observer error: ${error.message}`);
        // Continue running even if there's an error
        await new Promise(resolve => setTimeout(resolve, this.scanInterval * 1000));
      }
    }
  }

  stop() {
    this.running = false;
  }
}

export async function startObserver(options = {}) {
  const observer = new ObserverAgent(options);
  await observer.start();
}