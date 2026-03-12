import { describe, it, beforeEach, afterEach, mock } from 'node:test';
import assert from 'node:assert';

// We need to mock fetch globally before importing the module
let originalFetch;
let mockFetchFn;

describe('DigitalOcean API client', () => {
  beforeEach(() => {
    originalFetch = globalThis.fetch;
    process.env.DO_API_TOKEN = 'test-token-123';
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    delete process.env.DO_API_TOKEN;
  });

  function setupFetch(responseBody, status = 200) {
    globalThis.fetch = mock.fn(async () => ({
      ok: status >= 200 && status < 300,
      status,
      statusText: status === 200 ? 'OK' : 'Error',
      json: async () => responseBody,
    }));
    return globalThis.fetch;
  }

  it('throws if DO_API_TOKEN is not set', async () => {
    delete process.env.DO_API_TOKEN;
    const { createDroplet } = await import('../do-api.js?v=1');
    await assert.rejects(
      () => createDroplet({ name: 'test', region: 'nyc1', size: 's-1vcpu-1gb' }),
      /DO_API_TOKEN/
    );
  });

  it('createDroplet POSTs to /v2/droplets and returns the droplet', async () => {
    const mockDroplet = { id: 12345, name: 'spekk-test', status: 'new' };
    const fetchMock = setupFetch({ droplet: mockDroplet });

    const { createDroplet } = await import('../do-api.js?v=2');
    const result = await createDroplet({
      name: 'spekk-test',
      region: 'nyc1',
      size: 's-2vcpu-4gb',
      userData: '#cloud-config',
      sshKeyIds: [123],
    });

    assert.deepStrictEqual(result, mockDroplet);
    const [url, opts] = fetchMock.mock.calls[0].arguments;
    assert.ok(url.includes('/v2/droplets'));
    assert.strictEqual(opts.method, 'POST');
    const body = JSON.parse(opts.body);
    assert.strictEqual(body.name, 'spekk-test');
    assert.strictEqual(body.user_data, '#cloud-config');
    assert.deepStrictEqual(body.ssh_keys, [123]);
    assert.ok(body.tags.includes('spekk-sandbox'));
  });

  it('getDroplet GETs /v2/droplets/{id}', async () => {
    const mockDroplet = { id: 99, status: 'active', networks: {} };
    setupFetch({ droplet: mockDroplet });

    const { getDroplet } = await import('../do-api.js?v=3');
    const result = await getDroplet(99);
    assert.deepStrictEqual(result, mockDroplet);
  });

  it('listDroplets filters by tag', async () => {
    const mockDroplets = [{ id: 1 }, { id: 2 }];
    const fetchMock = setupFetch({ droplets: mockDroplets });

    const { listDroplets } = await import('../do-api.js?v=4');
    const result = await listDroplets('spekk-sandbox');
    assert.deepStrictEqual(result, mockDroplets);
    const [url] = fetchMock.mock.calls[0].arguments;
    assert.ok(url.includes('tag_name=spekk-sandbox'));
  });

  it('deleteDroplet DELETEs /v2/droplets/{id}', async () => {
    globalThis.fetch = mock.fn(async () => ({
      ok: true,
      status: 204,
      statusText: 'No Content',
      json: async () => { throw new Error('no body'); },
    }));

    const { deleteDroplet } = await import('../do-api.js?v=5');
    const result = await deleteDroplet(42);
    assert.deepStrictEqual(result, { success: true });
  });

  it('listSSHKeys GETs /v2/account/keys', async () => {
    const mockKeys = [{ id: 1, fingerprint: 'aa:bb' }];
    setupFetch({ ssh_keys: mockKeys });

    const { listSSHKeys } = await import('../do-api.js?v=6');
    const result = await listSSHKeys();
    assert.deepStrictEqual(result, mockKeys);
  });

  it('throws with DO error message on API errors', async () => {
    setupFetch({ id: 'not_found', message: 'Droplet not found' }, 404);

    const { getDroplet } = await import('../do-api.js?v=7');
    await assert.rejects(
      () => getDroplet(99999),
      (err) => {
        assert.ok(err.message.includes('Droplet not found'));
        assert.ok(err.message.includes('404'));
        return true;
      }
    );
  });
});
