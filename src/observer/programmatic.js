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

      // Type 1: Code-Spec Misalignment - Check assertion success criteria against code
      await this.checkAssertionSuccessCriteria(specs, assertions, issues);

      // Type 1: Code-Spec Misalignment - Check function/feature requirements
      await this.checkFunctionRequirements(specs, assertions, issues);

      // Type 2: Outdated Specs - Check for done specs with changed files
      await this.checkOutdatedSpecs(specs, assertions, issues);

      // Type 3: Spec Compression Opportunities - Check for specs that could be consolidated  
      await this.checkSpecCompressionOpportunities(specs, assertions, issues);

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

  async checkAssertionSuccessCriteria(specs, assertions, issues) {
    for (const assertion of assertions) {
      if (assertion.status === 'done') continue; // Skip completed assertions

      // Extract success criteria from assertion content
      const criteria = this.extractSuccessCriteria(assertion.content);
      if (criteria.length === 0) continue;

      for (const criterion of criteria) {
        const misalignments = await this.validateCriterion(criterion, assertion);
        
        for (const misalignment of misalignments) {
          issues.push({
            type: 'code_spec_misalignment',
            severity: this.categorizeSeverity(criterion, assertion),
            title: `Assertion success criteria not met: ${criterion.text}`,
            description: `Assertion ${assertion.id} requires "${criterion.text}" but ${misalignment.issue}`,
            affected_specs: [assertion.parent],
            affected_files: [assertion.file, ...misalignment.files],
            criterion_type: criterion.type,
            criterion_details: criterion
          });
        }
      }
    }
  }

  extractSuccessCriteria(content) {
    const criteria = [];
    
    // Match checkbox items in Success Criteria sections
    const successSectionMatch = content.match(/## Success Criteria\s*\n\n([\s\S]*?)(?=\n##|\n#|$)/);
    if (!successSectionMatch) return criteria;
    
    const successSection = successSectionMatch[1];
    const checkboxMatches = successSection.match(/- \[ \] (.+)/g);
    
    if (checkboxMatches) {
      for (const match of checkboxMatches) {
        const text = match.replace(/^- \[ \] /, '').trim();
        const criterion = {
          text,
          type: this.classifyCriterion(text),
          raw: match
        };
        criteria.push(criterion);
      }
    }
    
    return criteria;
  }

  classifyCriterion(text) {
    const lowerText = text.toLowerCase();
    
    if (lowerText.includes('function') && lowerText.includes('exists')) {
      return 'function_exists';
    } else if (lowerText.includes('function') && lowerText.includes('parameter')) {
      return 'function_parameters';
    } else if (lowerText.includes('function') && lowerText.includes('return')) {
      return 'function_behavior';
    } else if (lowerText.includes('endpoint') || lowerText.includes('api')) {
      return 'api_endpoint';
    } else if (lowerText.includes('file') || lowerText.includes('directory')) {
      return 'file_structure';
    } else if (lowerText.includes('test') || lowerText.includes('validation')) {
      return 'validation';
    } else {
      return 'general';
    }
  }

  async validateCriterion(criterion, assertion) {
    const misalignments = [];
    
    try {
      switch (criterion.type) {
        case 'function_exists':
          await this.validateFunctionExists(criterion, assertion, misalignments);
          break;
        case 'function_parameters':
          await this.validateFunctionParameters(criterion, assertion, misalignments);
          break;
        case 'function_behavior':
          await this.validateFunctionBehavior(criterion, assertion, misalignments);
          break;
        case 'api_endpoint':
          await this.validateApiEndpoint(criterion, assertion, misalignments);
          break;
        case 'file_structure':
          await this.validateFileStructure(criterion, assertion, misalignments);
          break;
        default:
          // For general criteria, do basic text-based validation
          await this.validateGeneral(criterion, assertion, misalignments);
          break;
      }
    } catch (error) {
      // If validation throws an error, treat it as a potential misalignment
      misalignments.push({
        issue: `validation failed: ${error.message}`,
        files: []
      });
    }
    
    return misalignments;
  }

  async validateFunctionExists(criterion, assertion, misalignments) {
    // Extract function name and file path from criterion text
    const functionMatch = criterion.text.match(/function\s+`([^`]+)`.*?in\s+([^\s]+)/i);
    if (!functionMatch) return;
    
    const [, functionName, filePath] = functionMatch;
    
    if (!await this.pathExists(filePath)) {
      misalignments.push({
        issue: `file ${filePath} does not exist`,
        files: [filePath]
      });
      return;
    }
    
    try {
      const fileContent = await fs.readFile(filePath, 'utf8');
      
      // Look for function declaration patterns
      const functionPatterns = [
        new RegExp(`function\\s+${functionName}\\s*\\(`, 'i'),
        new RegExp(`const\\s+${functionName}\\s*=.*?function`, 'i'),
        new RegExp(`export\\s+function\\s+${functionName}\\s*\\(`, 'i'),
        new RegExp(`${functionName}\\s*:\\s*function`, 'i'),
        new RegExp(`${functionName}\\s*=\\s*\\(.*?\\)\\s*=>`, 'i')
      ];
      
      const functionExists = functionPatterns.some(pattern => pattern.test(fileContent));
      
      if (!functionExists) {
        misalignments.push({
          issue: `function ${functionName} not found in ${filePath}`,
          files: [filePath]
        });
      }
    } catch (error) {
      misalignments.push({
        issue: `could not read file ${filePath}: ${error.message}`,
        files: [filePath]
      });
    }
  }

  async validateFunctionParameters(criterion, assertion, misalignments) {
    // Extract function name and expected parameters - handle different patterns
    let expectedParams = [];
    let functionName = null;
    
    // Pattern 1: "Function accepts two parameters (a, b)"
    const paramMatch1 = criterion.text.match(/function.*?accepts\s+(.*?)\s+parameters?.*?\(([^)]+)\)/i);
    if (paramMatch1) {
      const [, countDesc, paramList] = paramMatch1;
      expectedParams = paramList.split(',').map(p => p.trim());
    }
    
    // Pattern 2: "Function accepts X parameters" (extract count)
    const paramMatch2 = criterion.text.match(/function.*?accepts\s+(\w+)\s+parameters?/i);
    if (paramMatch2 && !paramMatch1) {
      const countWord = paramMatch2[1].toLowerCase();
      const countMap = { 'one': 1, 'two': 2, 'three': 3, 'four': 4, 'five': 5 };
      const expectedCount = countMap[countWord] || parseInt(countWord) || 0;
      expectedParams = new Array(expectedCount).fill('param'); // Placeholder names
    }
    
    if (expectedParams.length === 0) {
      return;
    }
    
    // Find function name - try multiple patterns
    let functionMatch = criterion.text.match(/function\s+`([^`]+)`/i);
    if (functionMatch) {
      functionName = functionMatch[1];
    } else {
      // Look for function name in the assertion content more broadly
      const assertionContent = assertion.content;
      const globalFunctionMatch = assertionContent.match(/function\s+`([^`]+)`/i);
      if (globalFunctionMatch) {
        functionName = globalFunctionMatch[1];
      } else {
        // Extract function names from backticks in the assertion
        const allFunctionMatches = assertionContent.match(/`([a-zA-Z_][a-zA-Z0-9_]*)`/g);
        if (allFunctionMatches && allFunctionMatches.length > 0) {
          // Use the first function name found
          functionName = allFunctionMatches[0].replace(/`/g, '');
        }
      }
    }
    
    // Find files that might contain this function
    const jsFiles = await this.findJsFiles();
    
    for (const file of jsFiles) {
      try {
        const content = await fs.readFile(file, 'utf8');
        const functionRegex = new RegExp(`(?:function\\s+${functionName}|const\\s+${functionName}\\s*=.*?function|${functionName}\\s*=\\s*\\([^)]*\\)|export\\s+function\\s+${functionName})\\s*\\(([^)]*)\\)`, 'i');
        const match = content.match(functionRegex);
        
        if (match) {
          const actualParams = match[1] ? match[1].split(',').map(p => p.trim()).filter(p => p) : [];
          
          if (actualParams.length !== expectedParams.length) {
            misalignments.push({
              issue: `function ${functionName} has ${actualParams.length} parameters but should have ${expectedParams.length}`,
              files: [file]
            });
          }
        }
      } catch (error) {
        // Ignore file read errors for this validation
      }
    }
  }

  async validateFunctionBehavior(criterion, assertion, misalignments) {
    // This is a simplified behavioral validation
    // In a real implementation, you'd want to parse and analyze the function body
    const behaviorKeywords = ['return', 'throw', 'error handling', 'validation'];
    
    for (const keyword of behaviorKeywords) {
      if (criterion.text.toLowerCase().includes(keyword)) {
        // Find the function and check if it has the expected behavior pattern
        const functionMatch = criterion.text.match(/function\s+`([^`]+)`/i);
        if (functionMatch) {
          const functionName = functionMatch[1];
          const hasExpectedBehavior = await this.checkFunctionForBehavior(functionName, keyword);
          
          if (!hasExpectedBehavior) {
            misalignments.push({
              issue: `function ${functionName} does not implement expected ${keyword} behavior`,
              files: await this.findFilesContainingFunction(functionName)
            });
          }
        }
      }
    }
  }

  async validateApiEndpoint(criterion, assertion, misalignments) {
    // Extract endpoint information
    const endpointMatch = criterion.text.match(/(\w+)\s+([\/\w]+)\s+endpoint/i);
    if (!endpointMatch) return;
    
    const [, method, path] = endpointMatch;
    
    // Look for API route definitions in common files
    const routeFiles = await this.findRouteFiles();
    let found = false;
    
    for (const file of routeFiles) {
      try {
        const content = await fs.readFile(file, 'utf8');
        const routePatterns = [
          new RegExp(`app\\.${method.toLowerCase()}\\s*\\(\\s*['"\`]${path}['"\`]`, 'i'),
          new RegExp(`router\\.${method.toLowerCase()}\\s*\\(\\s*['"\`]${path}['"\`]`, 'i'),
          new RegExp(`@${method.toUpperCase()}\\s*\\(\\s*['"\`]${path}['"\`]`, 'i')
        ];
        
        if (routePatterns.some(pattern => pattern.test(content))) {
          found = true;
          break;
        }
      } catch (error) {
        // Ignore file read errors
      }
    }
    
    if (!found) {
      misalignments.push({
        issue: `${method.toUpperCase()} ${path} endpoint not found`,
        files: routeFiles
      });
    }
  }

  async validateFileStructure(criterion, assertion, misalignments) {
    // Extract file/directory paths from criterion
    const pathMatches = criterion.text.match(/`([^`]+\.[a-z]+)`/g);
    if (!pathMatches) return;
    
    for (const match of pathMatches) {
      const filePath = match.replace(/`/g, '');
      if (!await this.pathExists(filePath)) {
        misalignments.push({
          issue: `required file ${filePath} does not exist`,
          files: [filePath]
        });
      }
    }
  }

  async validateGeneral(criterion, assertion, misalignments) {
    // Basic validation for general criteria - look for keywords in codebase
    const keywords = this.extractKeywords(criterion.text);
    if (keywords.length === 0) return;
    
    const codebaseFiles = await this.getAllCodeFiles();
    let foundRelevantCode = false;
    
    for (const file of codebaseFiles.slice(0, 10)) { // Limit to avoid performance issues
      try {
        const content = await fs.readFile(file, 'utf8');
        if (keywords.some(keyword => content.toLowerCase().includes(keyword.toLowerCase()))) {
          foundRelevantCode = true;
          break;
        }
      } catch (error) {
        // Ignore file read errors
      }
    }
    
    // Only flag as misalignment if we can't find any relevant code
    if (!foundRelevantCode && keywords.length > 0) {
      misalignments.push({
        issue: `no code found implementing: ${keywords.join(', ')}`,
        files: codebaseFiles.slice(0, 5) // Show first few files as context
      });
    }
  }

  async checkFunctionRequirements(specs, assertions, issues) {
    // This method focuses on detecting when functions/features exist but don't meet requirements
    for (const assertion of assertions) {
      if (assertion.status === 'done') continue;
      
      // Look for specific functional requirements in assertions
      const requirements = this.extractFunctionRequirements(assertion.content);
      
      for (const requirement of requirements) {
        const violations = await this.validateFunctionRequirement(requirement, assertion);
        
        for (const violation of violations) {
          issues.push({
            type: 'code_spec_misalignment',
            severity: this.categorizeSeverity(requirement, assertion),
            title: `Function exists but doesn't meet requirements: ${requirement.description}`,
            description: `${violation.description} in ${violation.location}`,
            affected_specs: [assertion.parent],
            affected_files: [assertion.file, ...violation.files],
            requirement_type: requirement.type
          });
        }
      }
    }
  }

  extractFunctionRequirements(content) {
    const requirements = [];
    
    // Look for specific patterns that indicate functional requirements
    const patterns = [
      { regex: /(\w+)\s+endpoint.*?returns?\s+(.+)/gi, type: 'api_return' },
      { regex: /(\w+)\s+endpoint.*?with\s+(.+)/gi, type: 'api_feature' },
      { regex: /function.*?(\w+).*?includes?\s+(.+)/gi, type: 'function_includes' },
      { regex: /API responds? with\s+(.+)/gi, type: 'api_response' },
      { regex: /Each\s+(\w+).*?includes?\s+(.+)/gi, type: 'object_structure' }
    ];
    
    for (const pattern of patterns) {
      let match;
      while ((match = pattern.regex.exec(content)) !== null) {
        requirements.push({
          type: pattern.type,
          description: match[0],
          target: match[1] || 'unknown',
          details: match[2] || 'unknown',
          raw: match[0]
        });
      }
    }
    
    return requirements;
  }

  async validateFunctionRequirement(requirement, assertion) {
    const violations = [];
    
    try {
      switch (requirement.type) {
        case 'api_return':
          await this.validateApiReturn(requirement, violations);
          break;
        case 'api_feature':
          await this.validateApiFeature(requirement, violations);
          break;
        case 'api_response':
          await this.validateApiResponse(requirement, violations);
          break;
        case 'object_structure':
          await this.validateObjectStructure(requirement, violations);
          break;
        default:
          // Generic validation
          await this.validateGenericRequirement(requirement, violations);
          break;
      }
    } catch (error) {
      violations.push({
        description: `Validation error for requirement: ${error.message}`,
        location: 'validation system',
        files: []
      });
    }
    
    return violations;
  }

  async validateApiReturn(requirement, violations) {
    // Look for API endpoints and check if they return what's expected
    const routeFiles = await this.findRouteFiles();
    
    for (const file of routeFiles) {
      try {
        const content = await fs.readFile(file, 'utf8');
        
        // Basic pattern matching for common return value issues
        if (content.includes('res.json(users)') && requirement.details.includes('pagination')) {
          if (!content.includes('page') && !content.includes('limit') && !content.includes('offset')) {
            violations.push({
              description: `API returns user list but missing pagination as required by spec`,
              location: file,
              files: [file]
            });
          }
        }
        
        if (content.includes('GET') && content.includes('/users') && requirement.target.includes('user')) {
          if (!content.includes('POST') && requirement.description.includes('POST')) {
            violations.push({
              description: `Found GET /users endpoint but missing required POST endpoint`,
              location: file,
              files: [file]
            });
          }
        }
      } catch (error) {
        // Ignore file read errors
      }
    }
  }

  async validateApiFeature(requirement, violations) {
    // Check if API features are properly implemented
    const routeFiles = await this.findRouteFiles();
    
    for (const file of routeFiles) {
      try {
        const content = await fs.readFile(file, 'utf8');
        
        if (requirement.details.includes('validation')) {
          if (content.includes('POST') && !content.includes('validate') && !content.includes('schema')) {
            violations.push({
              description: `POST endpoint exists but missing validation as required`,
              location: file,
              files: [file]
            });
          }
        }
      } catch (error) {
        // Ignore file read errors
      }
    }
  }

  async validateApiResponse(requirement, violations) {
    // Check API response patterns
    if (requirement.details.includes('HTTP status codes')) {
      const routeFiles = await this.findRouteFiles();
      
      for (const file of routeFiles) {
        try {
          const content = await fs.readFile(file, 'utf8');
          
          if (content.includes('res.json') && !content.includes('res.status')) {
            violations.push({
              description: `API responses found but missing proper HTTP status codes`,
              location: file,
              files: [file]
            });
          }
        } catch (error) {
          // Ignore file read errors
        }
      }
    }
  }

  async validateObjectStructure(requirement, violations) {
    // Check if object structures match requirements
    if (requirement.details.includes('id, name, email')) {
      const files = await this.findJsFiles();
      
      for (const file of files) {
        try {
          const content = await fs.readFile(file, 'utf8');
          
          if (content.includes('user') || content.includes('User')) {
            const requiredFields = ['id', 'name', 'email'];
            const missingFields = requiredFields.filter(field => !content.includes(field));
            
            if (missingFields.length > 0) {
              violations.push({
                description: `User object structure missing required fields: ${missingFields.join(', ')}`,
                location: file,
                files: [file]
              });
            }
          }
        } catch (error) {
          // Ignore file read errors
        }
      }
    }
  }

  async validateGenericRequirement(requirement, violations) {
    // Basic validation for generic requirements
    const keywords = this.extractKeywords(requirement.description);
    const files = await this.getAllCodeFiles();
    
    let foundImplementation = false;
    
    for (const file of files.slice(0, 10)) {
      try {
        const content = await fs.readFile(file, 'utf8');
        if (keywords.some(keyword => content.includes(keyword))) {
          foundImplementation = true;
          break;
        }
      } catch (error) {
        // Ignore file read errors
      }
    }
    
    if (!foundImplementation) {
      violations.push({
        description: `No implementation found for requirement: ${requirement.description}`,
        location: 'codebase',
        files: []
      });
    }
  }

  // Helper methods
  categorizeSeverity(criterion, assertion) {
    // Categorize severity based on impact
    if (assertion.priority === 1) return 'high';
    if (criterion.type === 'function_exists' || criterion.type === 'api_endpoint') return 'high';
    if (criterion.type === 'validation' || criterion.type === 'function_behavior') return 'medium';
    return 'low';
  }

  extractKeywords(text) {
    // Extract meaningful keywords from text
    const words = text.toLowerCase().split(/\s+/);
    const stopWords = new Set(['the', 'a', 'an', 'and', 'or', 'but', 'in', 'on', 'at', 'to', 'for', 'of', 'with', 'by']);
    return words.filter(word => word.length > 2 && !stopWords.has(word) && /^[a-z]+$/.test(word));
  }

  async findJsFiles() {
    // Find JavaScript/TypeScript files in the project
    try {
      const { globSync } = await import('glob');
      return globSync('**/*.{js,ts,jsx,tsx}', { 
        ignore: ['node_modules/**', 'dist/**', 'build/**', '.git/**'],
        absolute: true 
      });
    } catch (error) {
      return ['src/**/*.js', 'lib/**/*.js']; // Fallback
    }
  }

  async findRouteFiles() {
    // Find files likely to contain API routes
    try {
      const { globSync } = await import('glob');
      return globSync('**/{routes,router,api,controllers}/**/*.{js,ts}', { 
        ignore: ['node_modules/**', 'dist/**', 'build/**'],
        absolute: true 
      });
    } catch (error) {
      return ['src/routes/**/*.js', 'src/api/**/*.js']; // Fallback
    }
  }

  async getAllCodeFiles() {
    // Get all code files for generic searching
    try {
      const { globSync } = await import('glob');
      return globSync('**/*.{js,ts,jsx,tsx,py,java,c,cpp,cs,php,rb,go}', { 
        ignore: ['node_modules/**', 'dist/**', 'build/**', '.git/**'],
        absolute: true 
      }).slice(0, 50); // Limit for performance
    } catch (error) {
      return []; // Fallback
    }
  }

  async checkFunctionForBehavior(functionName, behaviorKeyword) {
    const files = await this.findJsFiles();
    
    for (const file of files) {
      try {
        const content = await fs.readFile(file, 'utf8');
        
        // Find the function and check its body for the behavior
        const functionRegex = new RegExp(`(?:function\\s+${functionName}|${functionName}\\s*[=:].*?)\\s*\\{([^{}]*(?:\\{[^{}]*\\}[^{}]*)*)\\}`, 's');
        const match = content.match(functionRegex);
        
        if (match && match[1]) {
          const functionBody = match[1];
          const hasExpectedBehavior = functionBody.toLowerCase().includes(behaviorKeyword.toLowerCase());
          if (hasExpectedBehavior) return true;
        }
      } catch (error) {
        // Ignore file read errors
      }
    }
    
    return false;
  }

  async findFilesContainingFunction(functionName) {
    const files = await this.findJsFiles();
    const containingFiles = [];
    
    for (const file of files) {
      try {
        const content = await fs.readFile(file, 'utf8');
        if (content.includes(functionName)) {
          containingFiles.push(file);
        }
      } catch (error) {
        // Ignore file read errors
      }
    }
    
    return containingFiles;
  }

  async checkOutdatedSpecs(specs, assertions, issues) {
    // This method now delegates to the comprehensive identifyOutdatedSpecs
    const outdatedObservations = await this.identifyOutdatedSpecs({ 
      specs: specs.reduce((acc, spec) => ({ ...acc, [spec.id]: spec }), {}),
      assertions: assertions
    });

    for (const observation of outdatedObservations) {
      issues.push({
        type: observation.type,
        severity: observation.severity,
        title: observation.title,
        description: observation.description,
        affected_specs: observation.affected_specs,
        affected_files: observation.affected_files
      });
    }
  }

  async identifyOutdatedSpecs(specsData) {
    const observations = [];
    const specs = Array.isArray(specsData) ? specsData : Object.values(specsData.specs || specsData);
    const assertions = specsData.assertions || [];
    
    await this.ensureObservationsDirectory();
    
    for (const spec of specs) {
      // 1. Check specs marked done but code significantly changed
      await this.checkSpecCodeChanges(spec, observations);
      
      // 2. Check for deprecated/removed functionality references
      await this.checkDeprecatedReferences(spec, observations);
      
      // 3. Check for irrelevant success criteria
      await this.checkIrrelevantCriteria(spec, assertions, observations);
      
      // 4. Check for basic done/incomplete assertion mismatches (legacy functionality)
      await this.checkDoneSpecWithIncompleteAssertions(spec, assertions, observations);
    }
    
    // 5. Check for duplicate functionality across specs
    await this.checkDuplicateFunctionality(specs, observations);
    
    // 6. Check timestamp patterns for stale specs
    await this.checkTimestampPatterns(specs, observations);
    
    // 7. Check for outdated architectural patterns
    await this.checkOutdatedPatterns(specs, observations);
    
    // Create observation files for each finding
    for (const observation of observations) {
      await this.createOutdatedSpecObservation(observation);
    }
    
    return observations;
  }
  
  async checkSpecCodeChanges(spec, observations) {
    if (spec.status !== 'done') return;
    
    try {
      // Look for files referenced in the spec
      const fileReferences = this.extractFileReferences(spec.content);
      const specCompletionTime = new Date(spec.created || '2000-01-01');
      
      let hasSignificantChanges = false;
      const modifiedFiles = [];
      
      for (const filePath of fileReferences) {
        try {
          await fs.access(filePath);
          const stats = await fs.stat(filePath);
          
          // If file was modified significantly after spec completion
          if (stats.mtime > specCompletionTime) {
            const timeDiff = (stats.mtime - specCompletionTime) / (1000 * 60 * 60 * 24); // days
            if (timeDiff > 1) { // Modified more than 1 day after spec completion
              hasSignificantChanges = true;
              modifiedFiles.push(filePath);
            }
          }
        } catch (error) {
          // File doesn't exist anymore - also a significant change
          hasSignificantChanges = true;
          modifiedFiles.push(filePath + ' (removed)');
        }
      }
      
      if (hasSignificantChanges) {
        observations.push({
          type: 'outdated-spec-code-changed',
          severity: 'medium',
          title: `Spec marked done but code significantly changed: ${spec.id}`,
          description: `Spec ${spec.id} is marked as done but related files have been modified or removed since completion`,
          affected_specs: [spec.id],
          affected_files: [spec.file, ...modifiedFiles.filter(f => !f.includes('(removed)'))],
          evidence: `Modified files: ${modifiedFiles.join(', ')}`,
          recommendation: 'Review spec to ensure it still accurately reflects current implementation'
        });
      }
    } catch (error) {
      // Ignore errors in this detection method
    }
  }
  
  extractFileReferences(content) {
    const filePatterns = [
      /`([^`]+\.(js|ts|jsx|tsx|py|java|c|cpp|cs|php|rb|go|md|json|yaml|yml))`/g,
      /\b([a-zA-Z0-9_-]+\/[a-zA-Z0-9_/-]+\.(js|ts|jsx|tsx|py|java|c|cpp|cs|php|rb|go|md|json|yaml|yml))\b/g
    ];
    
    const files = new Set();
    
    for (const pattern of filePatterns) {
      let match;
      while ((match = pattern.exec(content)) !== null) {
        files.add(match[1]);
      }
    }
    
    return Array.from(files);
  }
  
  async checkDeprecatedReferences(spec, observations) {
    try {
      // Check for references to removed npm packages
      const packageJsonPath = 'package.json';
      let packageJson = {};
      
      try {
        const packageContent = await fs.readFile(packageJsonPath, 'utf8');
        packageJson = JSON.parse(packageContent);
      } catch (error) {
        return; // No package.json found
      }
      
      const allDependencies = {
        ...packageJson.dependencies,
        ...packageJson.devDependencies,
        ...packageJson.peerDependencies
      };
      
      // Look for package references in spec content
      const packageReferences = this.extractPackageReferences(spec.content);
      const removedPackages = packageReferences.filter(pkg => !allDependencies[pkg]);
      
      if (removedPackages.length > 0) {
        observations.push({
          type: 'outdated-spec-deprecated-reference',
          severity: 'medium',
          title: `Spec references removed packages: ${spec.id}`,
          description: `Spec ${spec.id} references packages that are no longer in package.json: ${removedPackages.join(', ')}`,
          affected_specs: [spec.id],
          affected_files: [spec.file, packageJsonPath],
          evidence: `Removed packages: ${removedPackages.join(', ')}`,
          recommendation: 'Update spec to remove references to deprecated packages or add packages back if still needed'
        });
      }
      
      // Check for missing file references
      const fileReferences = this.extractFileReferences(spec.content);
      const missingFiles = [];
      
      for (const filePath of fileReferences) {
        try {
          await fs.access(filePath);
        } catch (error) {
          missingFiles.push(filePath);
        }
      }
      
      if (missingFiles.length > 0) {
        observations.push({
          type: 'outdated-spec-missing-reference',
          severity: 'medium',
          title: `Spec references missing files: ${spec.id}`,
          description: `Spec ${spec.id} references files that no longer exist: ${missingFiles.join(', ')}`,
          affected_specs: [spec.id],
          affected_files: [spec.file],
          evidence: `Missing files: ${missingFiles.join(', ')}`,
          recommendation: 'Update spec to remove references to missing files or restore files if still needed'
        });
      }
    } catch (error) {
      // Ignore errors in this detection method
    }
  }
  
  extractPackageReferences(content) {
    const packagePatterns = [
      /`([a-zA-Z0-9_-]+)`/g, // Backtick-wrapped package names
      /import.*?from\s+['"]([^'"]+)['"]/g, // Import statements
      /require\(['"]([^'"]+)['"]\)/g // Require statements
    ];
    
    const packages = new Set();
    
    for (const pattern of packagePatterns) {
      let match;
      while ((match = pattern.exec(content)) !== null) {
        const packageName = match[1];
        // Filter for likely package names (no file extensions, no paths)
        if (!packageName.includes('/') && !packageName.includes('.') && 
            packageName.length > 1 && /^[a-zA-Z0-9_-]+$/.test(packageName)) {
          packages.add(packageName);
        }
      }
    }
    
    return Array.from(packages);
  }
  
  async checkIrrelevantCriteria(spec, assertions, observations) {
    const specAssertions = assertions.filter(a => a.parent === spec.id);
    
    for (const assertion of specAssertions) {
      const criteria = this.extractSuccessCriteria(assertion.content);
      const irrelevantCriteria = [];
      
      for (const criterion of criteria) {
        // Check if criterion references non-existent functionality
        if (await this.isCriterionIrrelevant(criterion)) {
          irrelevantCriteria.push(criterion.text);
        }
      }
      
      if (irrelevantCriteria.length > 0) {
        observations.push({
          type: 'outdated-spec-irrelevant-criteria',
          severity: 'low',
          title: `Spec has irrelevant success criteria: ${spec.id}`,
          description: `Spec ${spec.id} contains success criteria that may no longer be relevant: ${irrelevantCriteria.join('; ')}`,
          affected_specs: [spec.id],
          affected_files: [spec.file],
          evidence: `Irrelevant criteria: ${irrelevantCriteria.join(', ')}`,
          recommendation: 'Review and update success criteria to match current system needs'
        });
      }
    }
  }
  
  async isCriterionIrrelevant(criterion) {
    // Look for criteria that reference deprecated technologies or patterns
    const deprecatedKeywords = [
      'jQuery', 'bower', 'grunt', 'gulp', 'webpack 1', 'angular 1', 'angularjs',
      'coffeescript', 'jade', 'stylus', 'less', 'sass-node', 'node-sass'
    ];
    
    const text = criterion.text.toLowerCase();
    return deprecatedKeywords.some(keyword => text.includes(keyword.toLowerCase()));
  }
  
  async checkDoneSpecWithIncompleteAssertions(spec, assertions, observations) {
    if (spec.status === 'done') {
      const specAssertions = assertions.filter(a => a.parent === spec.id);
      const allDone = specAssertions.every(a => a.status === 'done');
      
      if (!allDone) {
        const incompleteCount = specAssertions.filter(a => a.status !== 'done').length;
        observations.push({
          type: 'outdated-spec-incomplete-assertions',
          severity: 'medium',
          title: `Spec marked done but has incomplete assertions: ${spec.id}`,
          description: `Spec ${spec.id} has status 'done' but ${incompleteCount} assertions are not completed`,
          affected_specs: [spec.id],
          affected_files: [spec.file],
          evidence: `${incompleteCount} incomplete assertions out of ${specAssertions.length} total`,
          recommendation: 'Either complete remaining assertions or update spec status to reflect actual completion state'
        });
      }
    }
  }
  
  async checkDuplicateFunctionality(specs, observations) {
    for (let i = 0; i < specs.length; i++) {
      for (let j = i + 1; j < specs.length; j++) {
        const specA = specs[i];
        const specB = specs[j];
        
        const similarity = this.calculateContentSimilarity(specA.content, specB.content);
        
        if (similarity > 0.7) { // 70% similarity threshold
          observations.push({
            type: 'outdated-spec-duplicate-functionality',
            severity: 'medium',
            title: `Specs have duplicate functionality: ${specA.id} and ${specB.id}`,
            description: `Specs ${specA.id} and ${specB.id} have very similar content and may represent duplicate functionality`,
            affected_specs: [specA.id, specB.id],
            affected_files: [specA.file, specB.file],
            evidence: `Content similarity: ${Math.round(similarity * 100)}%`,
            recommendation: 'Consider consolidating these specs or clarifying their distinct purposes'
          });
          
          // Only report each pair once
          break;
        }
      }
    }
  }
  
  calculateContentSimilarity(contentA, contentB) {
    const wordsA = this.extractSignificantWords(contentA);
    const wordsB = this.extractSignificantWords(contentB);
    
    if (wordsA.length === 0 || wordsB.length === 0) return 0;
    
    const setA = new Set(wordsA);
    const setB = new Set(wordsB);
    const intersection = new Set([...setA].filter(word => setB.has(word)));
    const union = new Set([...setA, ...setB]);
    
    return intersection.size / union.size; // Jaccard similarity
  }
  
  extractSignificantWords(content) {
    const stopWords = new Set(['the', 'a', 'an', 'and', 'or', 'but', 'in', 'on', 'at', 'to', 'for', 'of', 'with', 'by', 'this', 'that', 'these', 'those', 'is', 'are', 'was', 'were', 'be', 'been', 'have', 'has', 'had', 'do', 'does', 'did', 'will', 'would', 'could', 'should', 'may', 'might', 'must', 'can', 'shall']);
    
    return content.toLowerCase()
      .replace(/[^\w\s]/g, ' ')
      .split(/\s+/)
      .filter(word => word.length > 2 && !stopWords.has(word));
  }
  
  async checkTimestampPatterns(specs, observations) {
    const now = new Date();
    const oneYearAgo = new Date(now.getFullYear() - 1, now.getMonth(), now.getDate());
    
    const veryOldSpecs = specs.filter(spec => {
      const created = new Date(spec.created);
      return created < oneYearAgo && spec.status === 'done';
    });
    
    if (veryOldSpecs.length > 0) {
      // Only flag as stale if there's recent activity in other specs
      const recentSpecs = specs.filter(spec => {
        const created = new Date(spec.created);
        const threeMonthsAgo = new Date(now.getFullYear(), now.getMonth() - 3, now.getDate());
        return created > threeMonthsAgo;
      });
      
      if (recentSpecs.length > 0) {
        const staleSpecIds = veryOldSpecs.map(s => s.id);
        observations.push({
          type: 'outdated-spec-timestamp-stale',
          severity: 'low',
          title: `Old specs may be stale: ${staleSpecIds.join(', ')}`,
          description: `${veryOldSpecs.length} specs are over a year old while development appears to be active on newer specs`,
          affected_specs: staleSpecIds,
          affected_files: veryOldSpecs.map(s => s.file),
          evidence: `Oldest spec: ${Math.min(...veryOldSpecs.map(s => new Date(s.created)))}`,
          recommendation: 'Review old specs to ensure they still represent current system needs or archive if no longer relevant'
        });
      }
    }
  }
  
  async checkOutdatedPatterns(specs, observations) {
    const outdatedPatterns = [
      { pattern: /jquery|jQuery/i, modern: 'React/Vue/Angular', category: 'frontend' },
      { pattern: /callbacks?.*async/i, modern: 'async/await or Promises', category: 'async' },
      { pattern: /var\s+\w+/i, modern: 'const/let', category: 'variables' },
      { pattern: /function\s*\([^)]*\)\s*\{/i, modern: 'arrow functions', category: 'functions' },
      { pattern: /bower/i, modern: 'npm or yarn', category: 'package-manager' }
    ];
    
    for (const spec of specs) {
      const conflicts = [];
      
      for (const { pattern, modern, category } of outdatedPatterns) {
        if (pattern.test(spec.content)) {
          // Check if there are other specs using modern patterns
          const hasModernSpecs = specs.some(otherSpec => 
            otherSpec.id !== spec.id && 
            this.containsModernPattern(otherSpec.content, category)
          );
          
          if (hasModernSpecs) {
            conflicts.push({ pattern: pattern.source, modern, category });
          }
        }
      }
      
      if (conflicts.length > 0) {
        observations.push({
          type: 'outdated-spec-pattern-conflict',
          severity: 'low',
          title: `Spec uses outdated patterns: ${spec.id}`,
          description: `Spec ${spec.id} references outdated patterns while other specs use modern alternatives`,
          affected_specs: [spec.id],
          affected_files: [spec.file],
          evidence: conflicts.map(c => `${c.category}: suggests ${c.modern}`).join(', '),
          recommendation: 'Consider updating spec to align with modern development patterns used elsewhere in the project'
        });
      }
    }
  }
  
  containsModernPattern(content, category) {
    const modernPatterns = {
      'frontend': /react|vue|angular|svelte/i,
      'async': /async\/await|Promise\./i,
      'variables': /\b(const|let)\s+\w+/i,
      'functions': /=>\s*\{|=>\s*[^{]/i,
      'package-manager': /npm|yarn/i
    };
    
    const pattern = modernPatterns[category];
    return pattern && pattern.test(content);
  }
  
  async createOutdatedSpecObservation(observation) {
    const timestamp = new Date().toISOString();
    const observationId = `${observation.type}-${Date.now()}-${Math.random().toString(36).substr(2, 5)}`;
    
    // Find existing observations to get next sequence number
    let existingFiles = [];
    try {
      existingFiles = await fs.readdir('observations');
    } catch (error) {
      // Directory doesn't exist yet
    }
    
    const sequenceNum = String(existingFiles.length + 1).padStart(3, '0');
    const filename = `${timestamp.replace(/[:.]/g, '-').replace('T', 'T').replace('Z', 'Z')}-${sequenceNum}.md`;
    const filePath = path.join('observations', filename);
    
    const frontmatter = `---
id: ${observationId}
created: ${timestamp}
type: ${observation.type}
severity: ${observation.severity}
affected_specs: [${observation.affected_specs.map(s => s).join(', ')}]
affected_files: [${observation.affected_files.map(f => f).join(', ')}]
---`;
    
    const content = `${frontmatter}

# ${observation.title}

## Issue Description
${observation.description}

## Evidence
${observation.evidence || 'Automatically detected during observer scan'}

## Impact
${this.getImpactMessage(observation.severity)}

## Recommendation
${observation.recommendation || 'Review affected specs and update as needed'}
`;
    
    await fs.writeFile(filePath, content, 'utf8');
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

  async checkSpecCompressionOpportunities(specs, assertions, issues) {
    // Skip if too few specs to compress
    if (specs.length < 2) return;

    // Find specs with overlapping functionality, scope, or targeting same components
    const compressionCandidates = [];

    for (let i = 0; i < specs.length; i++) {
      for (let j = i + 1; j < specs.length; j++) {
        const specA = specs[i];
        const specB = specs[j];

        const overlapAnalysis = this.analyzeSpecOverlap(specA, specB, assertions);
        
        if (overlapAnalysis.shouldCompress) {
          compressionCandidates.push({
            specs: [specA, specB],
            overlapType: overlapAnalysis.overlapType,
            confidence: overlapAnalysis.confidence,
            reason: overlapAnalysis.reason,
            evidence: overlapAnalysis.evidence
          });
        }
      }
    }

    // Create compression opportunity observations
    for (const candidate of compressionCandidates) {
      // Consider priority and status when suggesting merges
      const highPrioritySpecs = candidate.specs.filter(s => s.priority === 1);
      const completedSpecs = candidate.specs.filter(s => s.status === 'done');
      
      let compressionAdvice = '';
      let severity = 'medium';

      if (completedSpecs.length > 0) {
        severity = 'low';
        compressionAdvice = 'Consider consolidation carefully as some specs are already completed.';
      } else if (highPrioritySpecs.length > 1) {
        severity = 'high';
        compressionAdvice = 'High-priority specs with overlapping scope may benefit from consolidation to avoid duplication of effort.';
      } else {
        compressionAdvice = 'These specs could be consolidated to reduce complexity and improve clarity.';
      }

      issues.push({
        type: 'compression_opportunity',
        severity: severity,
        title: `Spec compression opportunity: ${candidate.specs.map(s => s.id).join(' + ')}`,
        description: `${candidate.reason}. ${compressionAdvice}`,
        affected_specs: candidate.specs.map(s => s.id),
        affected_files: candidate.specs.map(s => s.file),
        overlap_type: candidate.overlapType,
        confidence: candidate.confidence,
        evidence: candidate.evidence,
        original_specs: candidate.specs.map(s => ({ id: s.id, file: s.file, title: s.title }))
      });
    }
  }

  analyzeSpecOverlap(specA, specB, assertions) {
    let overlapScore = 0;
    let evidence = [];
    let overlapType = [];

    // 1. Check for overlapping functionality or scope
    const functionalOverlap = this.checkFunctionalOverlap(specA, specB);
    if (functionalOverlap.overlap) {
      overlapScore += functionalOverlap.score;
      evidence.push(functionalOverlap.evidence);
      overlapType.push('functional');
    }

    // 2. Check if specs target the same system component
    const componentOverlap = this.checkComponentOverlap(specA, specB);
    if (componentOverlap.overlap) {
      overlapScore += componentOverlap.score;
      evidence.push(componentOverlap.evidence);
      overlapType.push('component');
    }

    // 3. Check for similar success criteria patterns in assertions
    const criteriaOverlap = this.checkSuccessCriteriaOverlap(specA, specB, assertions);
    if (criteriaOverlap.overlap) {
      overlapScore += criteriaOverlap.score;
      evidence.push(criteriaOverlap.evidence);
      overlapType.push('criteria');
    }

    // 4. Check for similar domains/keywords
    const domainOverlap = this.checkDomainOverlap(specA, specB);
    if (domainOverlap.overlap) {
      overlapScore += domainOverlap.score;
      evidence.push(domainOverlap.evidence);
      overlapType.push('domain');
    }

    const shouldCompress = overlapScore >= 2; // Threshold for suggesting compression
    const confidence = Math.min(overlapScore / 4, 1.0); // Confidence as percentage

    let reason = '';
    if (overlapType.includes('functional')) {
      reason = 'Specs have overlapping functionality that could be consolidated';
    } else if (overlapType.includes('component')) {
      reason = 'Specs target the same system components';
    } else if (overlapType.includes('criteria')) {
      reason = 'Specs have similar success criteria patterns that suggest they could be merged';
    } else if (overlapType.includes('domain')) {
      reason = 'Specs operate in the same domain and could benefit from consolidation';
    }

    return {
      shouldCompress,
      overlapType: overlapType.join(', '),
      confidence,
      reason,
      evidence
    };
  }

  checkFunctionalOverlap(specA, specB) {
    const functionalKeywords = [
      'login', 'authentication', 'auth', 'user management', 'session',
      'database', 'api', 'endpoint', 'crud', 'create', 'read', 'update', 'delete',
      'validation', 'error handling', 'logging', 'configuration', 'config'
    ];

    const contentA = specA.content.toLowerCase();
    const contentB = specB.content.toLowerCase();
    
    let matchCount = 0;
    const matches = [];

    for (const keyword of functionalKeywords) {
      if (contentA.includes(keyword) && contentB.includes(keyword)) {
        matchCount++;
        matches.push(keyword);
      }
    }

    const overlap = matchCount >= 2;
    const score = overlap ? Math.min(matchCount, 3) : 0;
    const evidence = overlap ? `Both specs mention: ${matches.join(', ')}` : '';

    return { overlap, score, evidence };
  }

  checkComponentOverlap(specA, specB) {
    // Look for file paths, directories, or component names
    const pathPatterns = [
      /\b[\w-]+\/[\w-]+(?:\/[\w-]+)*\.\w+\b/g, // File paths
      /\b[\w-]+\/[\w-]+(?:\/[\w-]+)*\b/g, // Directory paths
      /\bsrc\/[\w-]+\b/g, // Src directories
      /\bcomponents\/[\w-]+\b/g // Components
    ];

    const pathsA = new Set();
    const pathsB = new Set();

    for (const pattern of pathPatterns) {
      const matchesA = specA.content.match(pattern) || [];
      const matchesB = specB.content.match(pattern) || [];
      
      matchesA.forEach(match => pathsA.add(match.toLowerCase()));
      matchesB.forEach(match => pathsB.add(match.toLowerCase()));
    }

    const commonPaths = [...pathsA].filter(path => pathsB.has(path));
    const overlap = commonPaths.length > 0;
    const score = overlap ? Math.min(commonPaths.length, 2) : 0;
    const evidence = overlap ? `Both specs reference: ${commonPaths.join(', ')}` : '';

    return { overlap, score, evidence };
  }

  checkSuccessCriteriaOverlap(specA, specB, assertions) {
    const assertionsA = assertions.filter(a => a.parent === specA.id);
    const assertionsB = assertions.filter(a => a.parent === specB.id);

    const criteriaA = [];
    const criteriaB = [];

    assertionsA.forEach(assertion => {
      const criteria = this.extractSuccessCriteria(assertion.content);
      criteriaA.push(...criteria.map(c => c.text.toLowerCase()));
    });

    assertionsB.forEach(assertion => {
      const criteria = this.extractSuccessCriteria(assertion.content);
      criteriaB.push(...criteria.map(c => c.text.toLowerCase()));
    });

    // Look for similar criteria patterns
    let similarityScore = 0;
    const similarities = [];

    for (const criteriaTextA of criteriaA) {
      for (const criteriaTextB of criteriaB) {
        const similarity = this.calculateTextSimilarity(criteriaTextA, criteriaTextB);
        if (similarity > 0.6) { // 60% similarity threshold
          similarityScore++;
          similarities.push(`"${criteriaTextA}" ≈ "${criteriaTextB}"`);
        }
      }
    }

    const overlap = similarityScore > 0;
    const score = overlap ? Math.min(similarityScore, 2) : 0;
    const evidence = overlap ? `Similar criteria: ${similarities.slice(0, 3).join(', ')}` : '';

    return { overlap, score, evidence };
  }

  checkDomainOverlap(specA, specB) {
    const domainsA = this.extractDomains(specA.content);
    const domainsB = this.extractDomains(specB.content);

    const commonDomains = domainsA.filter(domain => domainsB.includes(domain));
    const overlap = commonDomains.length > 0;
    const score = overlap ? Math.min(commonDomains.length, 2) : 0;
    const evidence = overlap ? `Shared domains: ${commonDomains.join(', ')}` : '';

    return { overlap, score, evidence };
  }

  calculateTextSimilarity(textA, textB) {
    const wordsA = textA.split(/\s+/).filter(w => w.length > 2);
    const wordsB = textB.split(/\s+/).filter(w => w.length > 2);
    
    if (wordsA.length === 0 || wordsB.length === 0) return 0;

    const commonWords = wordsA.filter(word => wordsB.includes(word));
    const similarity = commonWords.length / Math.max(wordsA.length, wordsB.length);
    
    return similarity;
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

    // Add type-specific metadata
    if (issue.type === 'spec_conflicts') {
      frontmatter += `
conflict_type: ${issue.conflict_type || 'unknown'}
blocking: ${issue.blocking || false}`;
    } else if (issue.type === 'compression_opportunity') {
      frontmatter += `
overlap_type: ${issue.overlap_type || 'unknown'}
confidence: ${issue.confidence || 0}
original_specs:
${issue.original_specs.map(spec => `  - id: ${spec.id}\n    file: ${spec.file}\n    title: ${spec.title || spec.id}`).join('\n')}`;
    }

    frontmatter += `
---`;

    let recommendation = 'Review the affected specs and files to determine if updates are needed.';
    
    // Add type-specific recommendations
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
    } else if (issue.type === 'compression_opportunity') {
      recommendation = `**SPEC COMPRESSION OPPORTUNITY**: These specs have overlapping scope and could be consolidated:

1. Review the overlapping specs: ${issue.affected_specs.join(', ')}
2. Analyze the overlap: ${issue.evidence.join('; ')}
3. Consider merging into a single comprehensive spec
4. Maintain traceability by referencing original specs in the consolidated version
5. Update any existing assertions to point to the new consolidated spec

**Confidence Level**: ${Math.round(issue.confidence * 100)}%
**Overlap Type**: ${issue.overlap_type}`;
    }
    
    let contentBody = '';
    
    if (issue.type === 'compression_opportunity') {
      contentBody = `# ${issue.title}

## Issue Description
${issue.description}

## Overlapping Specifications
${issue.affected_specs.map(spec => `- **${spec}**: Referenced in specs/${spec}/${spec}.md`).join('\n')}

## Overlap Analysis
**Type**: ${issue.overlap_type}
**Confidence**: ${Math.round(issue.confidence * 100)}%

**Evidence**:
${issue.evidence.map(evidence => `- ${evidence}`).join('\n')}

## Original Specification Details
${issue.original_specs.map(spec => `### ${spec.title || spec.id}
- **File**: ${spec.file}
- **ID**: ${spec.id}`).join('\n\n')}

## Impact
${this.getImpactMessage(issue.severity)}

## Recommendation
${recommendation}
`;
    } else {
      contentBody = `# ${issue.title}

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
    }

    const content = frontmatter + '\n\n' + contentBody;

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