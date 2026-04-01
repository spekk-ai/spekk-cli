import { test, describe } from 'node:test';
import assert from 'node:assert';
import { existsSync } from 'node:fs';
import { getTemplatePath, readTemplate, renderCloudInit } from '../templates.js';

describe('sandbox bundled templates', () => {
  test('getTemplatePath returns absolute path', () => {
    const p = getTemplatePath('some-file.yaml');
    assert.ok(p.startsWith('/'), 'path should be absolute');
  });

  test('cloud-init.yaml is not bundled', () => {
    const cloudInitPath = getTemplatePath('cloud-init.yaml');
    assert.ok(!existsSync(cloudInitPath), 'cloud-init.yaml should NOT be bundled');
  });
});
