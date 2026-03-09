import { describe, it, before, after } from 'node:test';
import assert from 'node:assert/strict';
import { startServe } from '../index.js';
import WebSocket from 'ws';

const TEST_PORT = 19118 + Math.floor(Math.random() * 1000);

describe('spekk serve', () => {
  let server;

  before(() => {
    server = startServe({ port: TEST_PORT, host: 'localhost' });
  });

  after((_, done) => {
    server.wss.close(() => {
      server.httpServer.close(() => done());
    });
    process.removeAllListeners('SIGINT');
    process.removeAllListeners('SIGTERM');
  });

  it('responds to health check on HTTP', async () => {
    const res = await fetch(`http://localhost:${TEST_PORT}/health`);
    assert.equal(res.status, 200);
    const body = await res.json();
    assert.deepEqual(body, { status: 'ok' });
  });

  it('returns 404 for unknown HTTP routes', async () => {
    const res = await fetch(`http://localhost:${TEST_PORT}/unknown`);
    assert.equal(res.status, 404);
  });

  it('accepts WebSocket connections and responds to ping/pong', async () => {
    const ws = new WebSocket(`ws://localhost:${TEST_PORT}`);

    await new Promise((resolve, reject) => {
      ws.on('open', resolve);
      ws.on('error', reject);
      setTimeout(() => reject(new Error('Connection timeout')), 3000);
    });

    const pongReceived = await new Promise((resolve, reject) => {
      ws.on('pong', () => resolve(true));
      ws.ping();
      setTimeout(() => reject(new Error('Pong timeout')), 3000);
    });
    assert.equal(pongReceived, true);

    ws.close();
    await new Promise(resolve => ws.on('close', resolve));
  });

  it('spawns a Claude subprocess per connection that produces output', async () => {
    const ws = new WebSocket(`ws://localhost:${TEST_PORT}`);

    await new Promise((resolve, reject) => {
      ws.on('open', resolve);
      ws.on('error', reject);
      setTimeout(() => reject(new Error('Connection timeout')), 3000);
    });

    // Send a message in the new wire format - expect an { event, data } response
    const message = await new Promise((resolve, reject) => {
      ws.on('message', (data) => {
        resolve(JSON.parse(data.toString()));
      });
      ws.send(JSON.stringify({ event: 'coach:chat', data: { content: 'Say hello' } }));
      setTimeout(() => reject(new Error('No response within timeout')), 15000);
    });

    assert.ok(message, 'Should receive a JSON message from Claude subprocess');
    assert.equal(typeof message, 'object');
    assert.ok(message.event, 'Response should have an event field');
    assert.ok(message.event.startsWith('coach:'), 'Event should be namespaced under coach channel');

    ws.close();
    await new Promise(resolve => ws.on('close', resolve));
  });
});
