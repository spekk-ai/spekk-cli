import { test, describe } from 'node:test';
import assert from 'node:assert';
import { validateFields } from '../index.js';

describe('Timestamp Validation', () => {
  describe('Valid ISO 8601 UTC Timestamps', () => {
    test('accepts valid ISO 8601 UTC timestamp format', () => {
      const validTimestamps = [
        '2026-01-20T16:26:00Z',
        '2025-12-31T23:59:59Z',
        '2024-01-01T00:00:00Z',
        '2026-06-15T14:30:45Z'
      ];
      
      for (const timestamp of validTimestamps) {
        const validData = {
          id: 'test-spec',
          created: timestamp,
          priority: 1
        };
        
        assert.doesNotThrow(() => {
          validateFields(validData, 'test-file.md', false);
        }, `Should accept valid timestamp: ${timestamp}`);
      }
    });

    test('accepts valid timestamps in updated field', () => {
      const validData = {
        id: 'test-spec',
        created: '2026-01-20T16:26:00Z',
        updated: '2026-01-21T10:15:30Z',
        priority: 1
      };
      
      assert.doesNotThrow(() => {
        validateFields(validData, 'test-file.md', false);
      }, 'Should accept valid updated timestamp');
    });
  });

  describe('Invalid Timestamp Formats', () => {
    test('rejects timestamps missing time component', () => {
      const invalidTimestamps = [
        '2026-01-20',
        '2025-12-31',
        '2024-06-15'
      ];
      
      for (const timestamp of invalidTimestamps) {
        const invalidData = {
          id: 'test-spec',
          created: timestamp,
          priority: 1
        };
        
        assert.throws(() => {
          validateFields(invalidData, 'test-file.md', false);
        }, (error) => {
          return error.message.includes("Invalid ISO 8601 timestamp in 'created' field") &&
                 error.message.includes(timestamp);
        }, `Should reject timestamp missing time: ${timestamp}`);
      }
    });

    test('rejects timestamps missing timezone', () => {
      const invalidTimestamps = [
        '2026-01-20T16:26:00',
        '2025-12-31T23:59:59',
        '2024-01-01T00:00:00'
      ];
      
      for (const timestamp of invalidTimestamps) {
        const invalidData = {
          id: 'test-spec',
          created: timestamp,
          priority: 1
        };
        
        assert.throws(() => {
          validateFields(invalidData, 'test-file.md', false);
        }, (error) => {
          return error.message.includes("Invalid ISO 8601 timestamp in 'created' field");
        }, `Should reject timestamp missing timezone: ${timestamp}`);
      }
    });

    test('rejects timestamps with wrong timezone format', () => {
      const invalidTimestamps = [
        '2026-01-20T16:26:00+00:00',
        '2026-01-20T16:26:00-05:00',
        '2026-01-20T16:26:00+0000'
      ];
      
      for (const timestamp of invalidTimestamps) {
        const invalidData = {
          id: 'test-spec',
          created: timestamp,
          priority: 1
        };
        
        assert.throws(() => {
          validateFields(invalidData, 'test-file.md', false);
        }, (error) => {
          return error.message.includes("Invalid ISO 8601 timestamp in 'created' field");
        }, `Should reject timestamp with wrong timezone format: ${timestamp}`);
      }
    });

    test('rejects completely invalid date formats', () => {
      const invalidTimestamps = [
        'Jan 20, 2026',
        'invalid-date',
        '20/01/2026',
        '2026/01/20',
        'not-a-timestamp'
      ];
      
      for (const timestamp of invalidTimestamps) {
        const invalidData = {
          id: 'test-spec',
          created: timestamp,
          priority: 1
        };
        
        assert.throws(() => {
          validateFields(invalidData, 'test-file.md', false);
        }, (error) => {
          return error.message.includes("Invalid ISO 8601 timestamp in 'created' field");
        }, `Should reject invalid timestamp format: ${timestamp}`);
      }
    });
  });

  describe('Updated Field Validation', () => {
    test('validates updated field when present', () => {
      const invalidUpdatedTimestamps = [
        '2026-01-20',
        '2026-01-20T16:26:00',
        '2026-01-20T16:26:00+00:00',
        'invalid-date'
      ];
      
      for (const updatedTimestamp of invalidUpdatedTimestamps) {
        const invalidData = {
          id: 'test-spec',
          created: '2026-01-20T16:26:00Z',
          updated: updatedTimestamp,
          priority: 1
        };
        
        assert.throws(() => {
          validateFields(invalidData, 'test-file.md', false);
        }, (error) => {
          return error.message.includes("Invalid ISO 8601 timestamp in 'updated' field") &&
                 error.message.includes(updatedTimestamp);
        }, `Should reject invalid updated timestamp: ${updatedTimestamp}`);
      }
    });

    test('allows missing updated field', () => {
      const validData = {
        id: 'test-spec',
        created: '2026-01-20T16:26:00Z',
        priority: 1
      };
      
      assert.doesNotThrow(() => {
        validateFields(validData, 'test-file.md', false);
      }, 'Should accept spec without updated field');
    });
  });

  describe('Required Field Validation for Assertions', () => {
    test('validates timestamps in assertion context', () => {
      const validAssertionData = {
        id: 'test-assertion',
        parent: 'test-spec',
        created: '2026-01-20T16:26:00Z',
        priority: 1
      };
      
      assert.doesNotThrow(() => {
        validateFields(validAssertionData, 'test-assertion.md', true);
      }, 'Should accept valid assertion with timestamp');

      const invalidAssertionData = {
        id: 'test-assertion',
        parent: 'test-spec',
        created: '2026-01-20',
        priority: 1
      };
      
      assert.throws(() => {
        validateFields(invalidAssertionData, 'test-assertion.md', true);
      }, (error) => {
        return error.message.includes("Invalid ISO 8601 timestamp in 'created' field");
      }, 'Should reject invalid timestamp in assertion');
    });
  });

  describe('Error Message Quality', () => {
    test('provides clear error messages for created field', () => {
      const invalidData = {
        id: 'test-spec',
        created: '2026-01-20',
        priority: 1
      };
      
      assert.throws(() => {
        validateFields(invalidData, 'test-file.md', false);
      }, (error) => {
        return error.message === "Invalid ISO 8601 timestamp in 'created' field: '2026-01-20' in test-file.md";
      }, 'Should provide exact error message for invalid created field');
    });

    test('provides clear error messages for updated field', () => {
      const invalidData = {
        id: 'test-spec',
        created: '2026-01-20T16:26:00Z',
        updated: '2026-01-20T16:26:00',
        priority: 1
      };
      
      assert.throws(() => {
        validateFields(invalidData, 'test-file.md', false);
      }, (error) => {
        return error.message === "Invalid ISO 8601 timestamp in 'updated' field: '2026-01-20T16:26:00' in test-file.md";
      }, 'Should provide exact error message for invalid updated field');
    });
  });

  describe('Timestamp Pattern Validation', () => {
    test('validates exact ISO 8601 format YYYY-MM-DDTHH:MM:SSZ', () => {
      const testCases = [
        // Valid cases
        { timestamp: '2026-01-20T16:26:00Z', shouldPass: true },
        { timestamp: '2024-12-31T23:59:59Z', shouldPass: true },
        { timestamp: '2025-01-01T00:00:00Z', shouldPass: true },
        
        // Invalid cases - missing components
        { timestamp: '2026-01-20', shouldPass: false },
        { timestamp: '2026-01-20T16:26:00', shouldPass: false },
        
        // Invalid cases - wrong separators
        { timestamp: '2026/01/20T16:26:00Z', shouldPass: false },
        { timestamp: '2026-01-20 16:26:00Z', shouldPass: false },
        
        // Invalid cases - wrong timezone
        { timestamp: '2026-01-20T16:26:00+00:00', shouldPass: false },
        { timestamp: '2026-01-20T16:26:00-05:00', shouldPass: false },
        
        // Invalid cases - wrong format
        { timestamp: '26-01-20T16:26:00Z', shouldPass: false },
        { timestamp: '2026-1-20T16:26:00Z', shouldPass: false },
        { timestamp: '2026-01-1T16:26:00Z', shouldPass: false },
        { timestamp: '2026-01-20T1:26:00Z', shouldPass: false },
        { timestamp: '2026-01-20T16:6:00Z', shouldPass: false },
        { timestamp: '2026-01-20T16:26:0Z', shouldPass: false }
      ];
      
      for (const testCase of testCases) {
        const data = {
          id: 'test-spec',
          created: testCase.timestamp,
          priority: 1
        };
        
        if (testCase.shouldPass) {
          assert.doesNotThrow(() => {
            validateFields(data, 'test-file.md', false);
          }, `Should accept valid timestamp: ${testCase.timestamp}`);
        } else {
          assert.throws(() => {
            validateFields(data, 'test-file.md', false);
          }, (error) => {
            return error.message.includes("Invalid ISO 8601 timestamp in 'created' field");
          }, `Should reject invalid timestamp: ${testCase.timestamp}`);
        }
      }
    });
  });
});