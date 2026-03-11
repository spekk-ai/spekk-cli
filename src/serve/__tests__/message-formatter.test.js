import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { formatMessageForClaude } from '../message-formatter.js';

describe('formatMessageForClaude', () => {

  it('formats a chat message with no attachments', () => {
    const raw = JSON.stringify({ type: 'chat', content: 'Hello coach' });
    assert.equal(formatMessageForClaude(raw), 'Hello coach');
  });

  it('formats chat with element_selection attachment', () => {
    const raw = JSON.stringify({
      type: 'chat',
      content: 'What is this button?',
      attachments: [{
        type: 'element_selection',
        selector: 'button.submit-btn',
        tag: 'button',
        classes: ['submit-btn', 'primary'],
        id: 'main-submit',
        innerText: 'Submit Form',
        boundingBox: { x: 100, y: 200, width: 120, height: 40 },
      }],
    });
    const result = formatMessageForClaude(raw);
    assert.ok(result.includes('What is this button?'));
    assert.ok(result.includes('Visual context:'));
    assert.ok(result.includes('`button.submit-btn`'));
    assert.ok(result.includes('"Submit Form"'));
    assert.ok(result.includes('120x40'));
  });

  it('formats chat with screenshot attachment', () => {
    const raw = JSON.stringify({
      type: 'chat',
      content: 'The page looks broken here',
      attachments: [{
        type: 'screenshot',
        dataUrl: 'data:image/png;base64,abc123',
        elementSelector: '.main-content',
        description: 'Error state',
      }],
    });
    const result = formatMessageForClaude(raw);
    assert.ok(result.includes('[Screenshot of `.main-content`: Error state]'));
  });

  it('formats chat with action_recording attachment', () => {
    const raw = JSON.stringify({
      type: 'chat',
      content: 'I found a bug doing this',
      attachments: [{
        type: 'action_recording',
        actions: [
          { type: 'click', selector: '#login-btn', timestamp: 1000 },
          { type: 'input', selector: '#email', value: 'user@test.com', timestamp: 2000 },
        ],
      }],
    });
    const result = formatMessageForClaude(raw);
    assert.ok(result.includes('Recorded actions:'));
    assert.ok(result.includes('Clicked on `#login-btn`'));
    assert.ok(result.includes('Typed "user@test.com" in `#email`'));
  });

  it('formats standalone element_selection message', () => {
    const raw = JSON.stringify({
      type: 'element_selection',
      selector: 'div.header',
      tag: 'div',
      classes: ['header'],
    });
    const result = formatMessageForClaude(raw);
    assert.ok(result.includes('[User selected an element]'));
    assert.ok(result.includes('`div.header`'));
  });

  it('formats standalone action_recording message', () => {
    const raw = JSON.stringify({
      type: 'action_recording',
      actions: [{ type: 'click', selector: '#btn' }],
    });
    const result = formatMessageForClaude(raw);
    assert.ok(result.includes('[User recorded browser actions]'));
    assert.ok(result.includes('Clicked on `#btn`'));
  });

  it('formats init message', () => {
    const raw = JSON.stringify({
      type: 'init',
      url: 'https://example.com/dashboard',
      title: 'Dashboard - MyApp',
      version: '1.2.0',
    });
    const result = formatMessageForClaude(raw);
    assert.ok(result.includes('[Session initialized]'));
    assert.ok(result.includes('https://example.com/dashboard'));
    assert.ok(result.includes('Dashboard - MyApp'));
  });

  it('returns null for ping messages', () => {
    assert.equal(formatMessageForClaude(JSON.stringify({ type: 'ping' })), null);
  });

  it('passes through non-JSON strings as-is', () => {
    assert.equal(formatMessageForClaude('just plain text'), 'just plain text');
  });

  it('passes through unknown message types as-is', () => {
    const raw = JSON.stringify({ type: 'future_type', data: 'something' });
    assert.equal(formatMessageForClaude(raw), raw);
  });

  it('accepts pre-parsed objects', () => {
    assert.equal(formatMessageForClaude({ type: 'chat', content: 'Hello' }), 'Hello');
  });
});
