const { ObserverAgent } = require('../programmatic');
const fs = require('fs');
const path = require('path');

describe('Observer Outdated Specs Detection', () => {
  let mockFs, mockParseAllSpecs, originalConsoleLog;
  
  beforeEach(() => {
    jest.clearAllMocks();
    originalConsoleLog = console.log;
    console.log = jest.fn();
    
    mockFs = {
      existsSync: jest.fn(),
      readFileSync: jest.fn(),
      writeFileSync: jest.fn(),
      mkdirSync: jest.fn(),
      readdirSync: jest.fn(() => [])
    };
    
    Object.keys(mockFs).forEach(method => {
      fs[method] = mockFs[method];
    });
  });

  afterEach(() => {
    console.log = originalConsoleLog;
  });

  describe('identifies specs marked done but code significantly changed', () => {
    test('detects done spec with file modifications after spec completion', async () => {
      const mockSpecs = {
        'test-feature': {
          id: 'test-feature',
          status: 'done',
          assertions: [
            {
              id: 'test-assertion',
              parent: 'test-feature',
              status: 'done',
              created: '2026-01-20T10:00:00Z'
            }
          ]
        }
      };

      mockFs.existsSync.mockImplementation(filePath => {
        return filePath.includes('src/feature.js') || filePath.includes('observations');
      });
      
      mockFs.readFileSync.mockImplementation(filePath => {
        if (filePath.includes('src/feature.js')) {
          return 'function newFeature() { return "significantly changed code"; }';
        }
        return '';
      });

      const statsMock = jest.fn().mockReturnValue({
        mtime: new Date('2026-01-21T10:00:00Z') // Modified after spec completion
      });
      fs.statSync = statsMock;

      const observer = new ObserverAgent();
      const observations = await observer.identifyOutdatedSpecs(mockSpecs);

      expect(observations).toHaveLength(1);
      expect(observations[0].type).toBe('outdated-spec-code-changed');
      expect(observations[0].affected_specs).toContain('test-feature');
    });
  });

  describe('detects specs referencing deprecated functionality', () => {
    test('identifies specs mentioning removed npm packages', async () => {
      const mockSpecs = {
        'legacy-feature': {
          id: 'legacy-feature',
          status: 'done',
          content: 'This feature uses the deprecated-package for functionality',
          assertions: []
        }
      };

      mockFs.readFileSync.mockImplementation(filePath => {
        if (filePath.includes('package.json')) {
          return JSON.stringify({
            dependencies: {
              'modern-package': '^1.0.0'
            },
            devDependencies: {}
          });
        }
        return '';
      });
      
      mockFs.existsSync.mockImplementation(filePath => 
        filePath.includes('package.json') || filePath.includes('observations')
      );

      const observer = new ObserverAgent();
      const observations = await observer.identifyOutdatedSpecs(mockSpecs);

      expect(observations).toHaveLength(1);
      expect(observations[0].type).toBe('outdated-spec-deprecated-reference');
      expect(observations[0].severity).toBe('medium');
    });

    test('identifies specs referencing removed files', async () => {
      const mockSpecs = {
        'file-reference-spec': {
          id: 'file-reference-spec',
          status: 'done',
          content: 'See implementation in src/removed-file.js',
          assertions: []
        }
      };

      mockFs.existsSync.mockImplementation(filePath => 
        filePath.includes('observations') && !filePath.includes('src/removed-file.js')
      );

      const observer = new ObserverAgent();
      const observations = await observer.identifyOutdatedSpecs(mockSpecs);

      expect(observations).toHaveLength(1);
      expect(observations[0].type).toBe('outdated-spec-missing-reference');
    });
  });

  describe('finds specs with irrelevant success criteria', () => {
    test('detects success criteria for non-existent functionality', async () => {
      const mockSpecs = {
        'outdated-criteria-spec': {
          id: 'outdated-criteria-spec',
          status: 'done',
          assertions: [
            {
              id: 'outdated-assertion',
              parent: 'outdated-criteria-spec',
              status: 'done',
              content: 'Success criteria: Feature X must integrate with removed service Y'
            }
          ]
        }
      };

      mockFs.existsSync.mockReturnValue(true);
      
      const observer = new ObserverAgent();
      const observations = await observer.identifyOutdatedSpecs(mockSpecs);

      expect(observations).toHaveLength(1);
      expect(observations[0].type).toBe('outdated-spec-irrelevant-criteria');
    });
  });

  describe('identifies duplicate functionality', () => {
    test('detects specs that duplicate existing functionality', async () => {
      const mockSpecs = {
        'duplicate-spec-1': {
          id: 'duplicate-spec-1',
          status: 'done',
          content: 'Implement user authentication system',
          assertions: []
        },
        'duplicate-spec-2': {
          id: 'duplicate-spec-2',
          status: 'done',
          content: 'Create user login and authentication',
          assertions: []
        }
      };

      mockFs.existsSync.mockReturnValue(true);

      const observer = new ObserverAgent();
      const observations = await observer.identifyOutdatedSpecs(mockSpecs);

      const duplicateObs = observations.find(obs => obs.type === 'outdated-spec-duplicate-functionality');
      expect(duplicateObs).toBeDefined();
      expect(duplicateObs.affected_specs).toEqual(expect.arrayContaining(['duplicate-spec-1', 'duplicate-spec-2']));
    });
  });

  describe('considers timestamp patterns for stale detection', () => {
    test('identifies specs that are very old relative to active development', async () => {
      const mockSpecs = {
        'ancient-spec': {
          id: 'ancient-spec',
          status: 'done',
          created: '2024-01-01T10:00:00Z', // Very old
          assertions: []
        },
        'recent-spec': {
          id: 'recent-spec',
          status: 'done',
          created: '2026-01-27T10:00:00Z', // Recent
          assertions: []
        }
      };

      mockFs.existsSync.mockReturnValue(true);

      const observer = new ObserverAgent();
      const observations = await observer.identifyOutdatedSpecs(mockSpecs);

      const staleObs = observations.find(obs => obs.type === 'outdated-spec-timestamp-stale');
      expect(staleObs).toBeDefined();
      expect(staleObs.affected_specs).toContain('ancient-spec');
      expect(staleObs.affected_specs).not.toContain('recent-spec');
    });
  });

  describe('flags specs conflicting with newer patterns', () => {
    test('detects specs using outdated architectural patterns', async () => {
      const mockSpecs = {
        'old-pattern-spec': {
          id: 'old-pattern-spec',
          status: 'done',
          content: 'Use jQuery for DOM manipulation and callbacks for async operations',
          assertions: []
        },
        'modern-pattern-spec': {
          id: 'modern-pattern-spec',
          status: 'done', 
          content: 'Use React hooks and async/await for modern patterns',
          assertions: []
        }
      };

      mockFs.existsSync.mockReturnValue(true);

      const observer = new ObserverAgent();
      const observations = await observer.identifyOutdatedSpecs(mockSpecs);

      const patternObs = observations.find(obs => obs.type === 'outdated-spec-pattern-conflict');
      expect(patternObs).toBeDefined();
      expect(patternObs.affected_specs).toContain('old-pattern-spec');
    });
  });

  describe('creates observations suggesting retirement or updates', () => {
    test('generates appropriate observation format for outdated spec', async () => {
      const mockSpecs = {
        'retirement-candidate': {
          id: 'retirement-candidate',
          status: 'done',
          content: 'Feature implemented with deprecated library',
          created: '2024-06-01T10:00:00Z',
          assertions: []
        }
      };

      mockFs.existsSync.mockReturnValue(true);
      mockFs.readdirSync.mockReturnValue(['2026-01-28T20-00-00-001Z.md']);

      const observer = new ObserverAgent();
      await observer.identifyOutdatedSpecs(mockSpecs);

      expect(mockFs.writeFileSync).toHaveBeenCalledWith(
        expect.stringContaining('observations/'),
        expect.stringContaining('type: outdated-spec'),
        'utf8'
      );

      const writtenContent = mockFs.writeFileSync.mock.calls[0][1];
      expect(writtenContent).toContain('affected_specs: [retirement-candidate]');
      expect(writtenContent).toContain('## Recommendation');
    });
  });

  describe('integration with observer loop', () => {
    test('outdated spec detection is called during observer scan', async () => {
      mockFs.existsSync.mockReturnValue(true);
      mockFs.readdirSync.mockReturnValue([]);
      
      const mockSpecs = {
        'test-spec': {
          id: 'test-spec',
          status: 'done',
          assertions: []
        }
      };

      const parseAllSpecs = jest.fn().mockResolvedValue(mockSpecs);
      
      const observer = new ObserverAgent();
      observer.parseAllSpecs = parseAllSpecs;
      observer.identifyOutdatedSpecs = jest.fn().mockResolvedValue([]);
      
      await observer.scanForDrift();
      
      expect(observer.identifyOutdatedSpecs).toHaveBeenCalledWith(mockSpecs);
    });
  });
});