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

  async createObservation(issue) {
    const timestamp = new Date().toISOString();
    const filename = `${timestamp.replace(/[:.]/g, '-')}.md`;
    const filePath = path.join(OBSERVATIONS_DIR, filename);

    const observationId = `${issue.type}-${Date.now()}`;
    
    const content = `---
id: ${observationId}
created: ${timestamp}
type: ${issue.type}
severity: ${issue.severity}
affected_specs:
${issue.affected_specs.map(spec => `  - ${spec}`).join('\n')}
affected_files:
${issue.affected_files.map(file => `  - ${file}`).join('\n')}
---

# ${issue.title}

## Issue Description
${issue.description}

## Evidence
This observation was automatically generated during system scan.

## Impact
${this.getImpactMessage(issue.severity)}

## Recommendation
Review the affected specs and files to determine if updates are needed.
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