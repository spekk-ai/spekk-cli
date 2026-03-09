import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import {
  formatMessageForClaude,
  formatElementSelection,
  formatScreenshot,
  formatActionRecording,
  formatSingleAction,
  formatAttachment,
  formatChatMessage,
  formatInitMessage,
} from '../message-formatter.js';

describe('message-formatter', () => {

  // -- formatMessageForClaude (top-level) -----------------------------------

  describe('formatMessageForClaude', () => {

    it('formats a chat message with no attachments', () => {
      const raw = JSON.stringify({ type: 'chat', content: 'Hello coach' });
      const result = formatMessageForClaude(raw);
      assert.equal(result, 'Hello coach');
    });

    it('formats a chat message with element_selection attachment', () => {
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
      assert.ok(result.includes('button.submit-btn.primary#main-submit'));
      assert.ok(result.includes('"Submit Form"'));
      assert.ok(result.includes('120x40'));
    });

    it('formats a chat message with screenshot attachment', () => {
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
      assert.ok(result.includes('The page looks broken here'));
      assert.ok(result.includes('[Screenshot of `.main-content`: Error state]'));
    });

    it('formats a chat message with action_recording attachment', () => {
      const raw = JSON.stringify({
        type: 'chat',
        content: 'I found a bug doing this',
        attachments: [{
          type: 'action_recording',
          actions: [
            { type: 'click', selector: '#login-btn', timestamp: 1000 },
            { type: 'input', selector: '#email', value: 'user@test.com', timestamp: 2000 },
            { type: 'click', selector: '.submit-btn', timestamp: 3000 },
          ],
        }],
      });
      const result = formatMessageForClaude(raw);
      assert.ok(result.includes('I found a bug doing this'));
      assert.ok(result.includes('Recorded actions:'));
      assert.ok(result.includes('1. Clicked on `#login-btn`'));
      assert.ok(result.includes('2. Typed "user@test.com" in `#email`'));
      assert.ok(result.includes('3. Clicked on `.submit-btn`'));
    });

    it('formats a chat message with multiple mixed attachments', () => {
      const raw = JSON.stringify({
        type: 'chat',
        content: 'This button breaks when I click it',
        attachments: [
          {
            type: 'element_selection',
            selector: '.submit-btn',
            tag: 'button',
            classes: ['submit-btn'],
            innerText: 'Submit',
          },
          {
            type: 'screenshot',
            dataUrl: 'data:image/png;base64,abc',
            description: 'Error state',
          },
          {
            type: 'action_recording',
            actions: [
              { type: 'click', selector: '.submit-btn', timestamp: 1000 },
            ],
          },
        ],
      });
      const result = formatMessageForClaude(raw);
      assert.ok(result.includes('This button breaks when I click it'));
      assert.ok(result.includes('Visual context:'));
      assert.ok(result.includes('Selected element: `.submit-btn`'));
      assert.ok(result.includes('[Screenshot: Error state]'));
      assert.ok(result.includes('Recorded actions:'));
      assert.ok(result.includes('1. Clicked on `.submit-btn`'));
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
        actions: [
          { type: 'click', selector: '#btn', timestamp: 100 },
        ],
      });
      const result = formatMessageForClaude(raw);
      assert.ok(result.includes('[User recorded browser actions]'));
      assert.ok(result.includes('Clicked on `#btn`'));
    });

    it('returns null for ping messages', () => {
      const raw = JSON.stringify({ type: 'ping' });
      const result = formatMessageForClaude(raw);
      assert.equal(result, null);
    });

    it('formats init message with all fields', () => {
      const raw = JSON.stringify({
        type: 'init',
        url: 'https://example.com/dashboard',
        title: 'Dashboard - MyApp',
        version: '1.2.0',
      });
      const result = formatMessageForClaude(raw);
      assert.equal(result, [
        '[Session initialized]',
        'Current page: https://example.com/dashboard',
        'Page title: Dashboard - MyApp',
        'Extension version: 1.2.0',
      ].join('\n'));
    });

    it('formats minimal init message', () => {
      const raw = JSON.stringify({ type: 'init' });
      const result = formatMessageForClaude(raw);
      assert.equal(result, '[Session initialized]');
    });

    it('init messages never return null', () => {
      const raw = JSON.stringify({ type: 'init' });
      const result = formatMessageForClaude(raw);
      assert.notEqual(result, null);
    });

    it('passes through non-JSON strings as-is', () => {
      const result = formatMessageForClaude('just plain text');
      assert.equal(result, 'just plain text');
    });

    it('passes through unknown message types as-is', () => {
      const raw = JSON.stringify({ type: 'future_type', data: 'something' });
      const result = formatMessageForClaude(raw);
      assert.equal(result, raw);
    });

    it('accepts pre-parsed objects as well as strings', () => {
      const result = formatMessageForClaude({ type: 'chat', content: 'Hello' });
      assert.equal(result, 'Hello');
    });
  });

  // -- formatElementSelection -----------------------------------------------

  describe('formatElementSelection', () => {

    it('formats with all fields', () => {
      const result = formatElementSelection({
        type: 'element_selection',
        selector: 'button#save.btn.primary',
        tag: 'button',
        classes: ['btn', 'primary'],
        id: 'save',
        innerText: 'Save Changes',
        boundingBox: { x: 10, y: 20, width: 100, height: 40 },
      });
      assert.ok(result.includes('`button#save.btn.primary`'));
      assert.ok(result.includes('button.btn.primary#save'));
      assert.ok(result.includes('"Save Changes"'));
      assert.ok(result.includes('100x40'));
    });

    it('formats with only required fields', () => {
      const result = formatElementSelection({
        type: 'element_selection',
        selector: 'div.container',
        tag: 'div',
      });
      assert.ok(result.includes('`div.container`'));
      assert.ok(result.includes('(div)'));
      assert.ok(!result.includes('text:'));
      assert.ok(!result.includes('dimensions:'));
    });

    it('truncates long innerText', () => {
      const longText = 'a'.repeat(150);
      const result = formatElementSelection({
        type: 'element_selection',
        selector: 'p',
        tag: 'p',
        innerText: longText,
      });
      assert.ok(result.includes('...'));
      assert.ok(!result.includes(longText));
    });

    it('does not truncate innerText at exactly 100 chars', () => {
      const exactText = 'a'.repeat(100);
      const result = formatElementSelection({
        type: 'element_selection',
        selector: 'p',
        tag: 'p',
        innerText: exactText,
      });
      assert.ok(result.includes(`"${exactText}"`));
      assert.ok(!result.includes('...'));
    });
  });

  // -- formatScreenshot -----------------------------------------------------

  describe('formatScreenshot', () => {

    it('formats with elementSelector and description', () => {
      const result = formatScreenshot({
        type: 'screenshot',
        dataUrl: 'data:image/png;base64,abc',
        elementSelector: '#main',
        description: 'Error page',
      });
      assert.equal(result, '[Screenshot of `#main`: Error page]');
    });

    it('formats with only elementSelector', () => {
      const result = formatScreenshot({
        type: 'screenshot',
        dataUrl: 'data:image/png;base64,abc',
        elementSelector: '.sidebar',
      });
      assert.equal(result, '[Screenshot of `.sidebar`]');
    });

    it('formats with only description', () => {
      const result = formatScreenshot({
        type: 'screenshot',
        dataUrl: 'data:image/png;base64,abc',
        description: 'Login page',
      });
      assert.equal(result, '[Screenshot: Login page]');
    });

    it('formats with no optional fields', () => {
      const result = formatScreenshot({
        type: 'screenshot',
        dataUrl: 'data:image/png;base64,abc',
      });
      assert.equal(result, '[Screenshot of current page]');
    });
  });

  // -- formatActionRecording ------------------------------------------------

  describe('formatActionRecording', () => {

    it('formats multiple actions as numbered steps', () => {
      const result = formatActionRecording({
        type: 'action_recording',
        actions: [
          { type: 'click', selector: '#btn' },
          { type: 'input', selector: '#field', value: 'hello' },
          { type: 'click', selector: '.submit' },
        ],
      });
      const lines = result.split('\n');
      assert.equal(lines[0], 'Recorded actions:');
      assert.ok(lines[1].includes('1. Clicked on `#btn`'));
      assert.ok(lines[2].includes('2. Typed "hello" in `#field`'));
      assert.ok(lines[3].includes('3. Clicked on `.submit`'));
    });

    it('handles empty actions array', () => {
      const result = formatActionRecording({
        type: 'action_recording',
        actions: [],
      });
      assert.equal(result, 'Recorded actions: (none)');
    });

    it('handles missing actions', () => {
      const result = formatActionRecording({
        type: 'action_recording',
      });
      assert.equal(result, 'Recorded actions: (none)');
    });
  });

  // -- formatSingleAction --------------------------------------------------

  describe('formatSingleAction', () => {

    it('formats click action', () => {
      assert.equal(formatSingleAction({ type: 'click', selector: '#btn' }), 'Clicked on `#btn`');
    });

    it('formats dblclick action', () => {
      assert.equal(formatSingleAction({ type: 'dblclick', selector: '.item' }), 'Double-clicked on `.item`');
    });

    it('formats input action with value', () => {
      assert.equal(
        formatSingleAction({ type: 'input', selector: '#email', value: 'a@b.com' }),
        'Typed "a@b.com" in `#email`',
      );
    });

    it('formats input action without value', () => {
      assert.equal(formatSingleAction({ type: 'input', selector: '#field' }), 'Typed in `#field`');
    });

    it('formats type action (alias for input)', () => {
      assert.equal(
        formatSingleAction({ type: 'type', selector: '#name', value: 'John' }),
        'Typed "John" in `#name`',
      );
    });

    it('formats change action', () => {
      assert.equal(
        formatSingleAction({ type: 'change', selector: '#dropdown', value: 'option-2' }),
        'Changed `#dropdown` to "option-2"',
      );
    });

    it('formats select action', () => {
      assert.equal(
        formatSingleAction({ type: 'select', selector: '#country', value: 'US' }),
        'Selected "US" in `#country`',
      );
    });

    it('formats focus action', () => {
      assert.equal(formatSingleAction({ type: 'focus', selector: '#input' }), 'Focused on `#input`');
    });

    it('formats blur action', () => {
      assert.equal(formatSingleAction({ type: 'blur', selector: '#input' }), 'Left `#input`');
    });

    it('formats scroll action', () => {
      assert.equal(formatSingleAction({ type: 'scroll', selector: '.list', direction: 'down' }), 'Scrolled down `.list`');
    });

    it('formats hover action', () => {
      assert.equal(formatSingleAction({ type: 'hover', selector: '.menu-item' }), 'Hovered over `.menu-item`');
    });

    it('formats keypress action with key', () => {
      assert.equal(
        formatSingleAction({ type: 'keypress', selector: '#input', key: 'Enter' }),
        'Pressed "Enter" in `#input`',
      );
    });

    it('formats navigate action with url', () => {
      assert.equal(
        formatSingleAction({ type: 'navigate', url: '/dashboard' }),
        'Navigated to /dashboard',
      );
    });

    it('formats navigation action (alias for navigate) with url', () => {
      assert.equal(
        formatSingleAction({ type: 'navigation', url: 'https://example.com/settings' }),
        'Navigated to https://example.com/settings',
      );
    });

    it('formats navigation action without url', () => {
      assert.equal(
        formatSingleAction({ type: 'navigation' }),
        'Navigated to new page',
      );
    });

    it('formats submit action', () => {
      assert.equal(formatSingleAction({ type: 'submit', selector: '#login-form' }), 'Submitted form `#login-form`');
    });

    it('formats unknown action type with value', () => {
      assert.equal(
        formatSingleAction({ type: 'custom', selector: '#el', value: 'data' }),
        'custom on `#el` (value: "data")',
      );
    });

    it('formats unknown action type without value', () => {
      assert.equal(formatSingleAction({ type: 'custom', selector: '#el' }), 'custom on `#el`');
    });

    it('handles action without selector', () => {
      const result = formatSingleAction({ type: 'click' });
      assert.equal(result, 'Clicked on element');
    });
  });

  // -- formatAttachment -----------------------------------------------------

  describe('formatAttachment', () => {

    it('delegates element_selection to formatElementSelection', () => {
      const result = formatAttachment({
        type: 'element_selection',
        selector: 'div.test',
        tag: 'div',
      });
      assert.ok(result.includes('`div.test`'));
    });

    it('delegates screenshot to formatScreenshot', () => {
      const result = formatAttachment({
        type: 'screenshot',
        dataUrl: 'data:image/png;base64,abc',
        description: 'Test',
      });
      assert.equal(result, '[Screenshot: Test]');
    });

    it('delegates action_recording to formatActionRecording', () => {
      const result = formatAttachment({
        type: 'action_recording',
        actions: [{ type: 'click', selector: '#x' }],
      });
      assert.ok(result.includes('Recorded actions:'));
    });

    it('formats unknown attachment type with label', () => {
      const result = formatAttachment({ type: 'custom_widget', data: {} });
      assert.equal(result, '[Attachment: custom_widget]');
    });
  });

  // -- formatChatMessage ----------------------------------------------------

  describe('formatChatMessage', () => {

    it('returns just content when no attachments', () => {
      const result = formatChatMessage({ content: 'Hello' });
      assert.equal(result, 'Hello');
    });

    it('returns just content when attachments is empty array', () => {
      const result = formatChatMessage({ content: 'Hello', attachments: [] });
      assert.equal(result, 'Hello');
    });

    it('includes visual context header when attachments present', () => {
      const result = formatChatMessage({
        content: 'Check this',
        attachments: [{ type: 'element_selection', selector: 'div', tag: 'div' }],
      });
      assert.ok(result.includes('Visual context:'));
    });

    it('handles empty content with attachments', () => {
      const result = formatChatMessage({
        content: '',
        attachments: [{ type: 'screenshot', dataUrl: 'data:image/png;base64,abc' }],
      });
      assert.ok(result.includes('Visual context:'));
      assert.ok(result.includes('[Screenshot of current page]'));
    });
  });

  // -- formatInitMessage ------------------------------------------------------

  describe('formatInitMessage', () => {

    it('formats with all fields', () => {
      const result = formatInitMessage({
        url: 'https://example.com/dashboard',
        title: 'Dashboard - MyApp',
        version: '1.2.0',
      });
      assert.equal(result, [
        '[Session initialized]',
        'Current page: https://example.com/dashboard',
        'Page title: Dashboard - MyApp',
        'Extension version: 1.2.0',
      ].join('\n'));
    });

    it('formats with no optional fields', () => {
      const result = formatInitMessage({});
      assert.equal(result, '[Session initialized]');
    });

    it('formats with only url', () => {
      const result = formatInitMessage({ url: 'https://example.com' });
      assert.equal(result, '[Session initialized]\nCurrent page: https://example.com');
    });
  });
});
