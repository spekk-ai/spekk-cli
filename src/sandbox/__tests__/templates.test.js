import { test, describe } from 'node:test';
import assert from 'node:assert';
import { existsSync } from 'node:fs';
import { getTemplatePath, readTemplate, renderCloudInit } from '../templates.js';

describe('sandbox bundled templates', () => {
  test('getTemplatePath returns absolute paths to existing template files', () => {
    const cloudInitPath = getTemplatePath('cloud-init.yaml');
    const agentClientPath = getTemplatePath('agent-client.py');

    assert.ok(cloudInitPath.startsWith('/'), 'path should be absolute');
    assert.ok(existsSync(cloudInitPath), 'cloud-init.yaml should exist');
    assert.ok(existsSync(agentClientPath), 'agent-client.py should exist');
  });

  test('readTemplate returns file contents as string', async () => {
    const content = await readTemplate('cloud-init.yaml');
    assert.strictEqual(typeof content, 'string');
    assert.ok(content.includes('#cloud-config'), 'should contain cloud-config header');
  });

  test('cloud-init.yaml contains required provisioning components', async () => {
    const content = await readTemplate('cloud-init.yaml');
    assert.ok(content.includes('docker'), 'should install Docker');
    assert.ok(content.includes('nodejs') || content.includes('node'), 'should install Node.js');
    assert.ok(content.includes('git'), 'should install git');
    assert.ok(content.includes('gh'), 'should install GitHub CLI');
    assert.ok(content.includes('claude'), 'should install Claude Code CLI');
    assert.ok(content.includes('spekk-agent'), 'should set up systemd unit');
    assert.ok(content.includes('agent'), 'should reference agent user');
    assert.ok(content.includes('/opt/spekk/.provisioned'), 'should create provisioned marker');
    assert.ok(content.includes('ssh-ed25519 AAAA... your-key-here'), 'should have SSH key placeholder');
  });

  test('agent-client.py is a WebSocket agent client', async () => {
    const content = await readTemplate('agent-client.py');
    assert.ok(content.includes('websockets'), 'should use websockets library');
    assert.ok(content.includes('async'), 'should be async');
  });

  test('renderCloudInit substitutes SSH public key', async () => {
    const testKey = 'ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestKey user@host';
    const rendered = await renderCloudInit(testKey);
    assert.ok(rendered.includes(testKey), 'should contain the substituted key');
    assert.ok(!rendered.includes('your-key-here'), 'should not contain the placeholder');
  });
});
