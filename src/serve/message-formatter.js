/**
 * Message Formatter
 *
 * Transforms structured JSON messages from the browser extension into
 * human-readable text prompts that Claude can understand. Visual context
 * (element selections, screenshots, action recordings) is formatted
 * inline so the coach can reference it naturally.
 */

/**
 * Format an element_selection attachment into readable text.
 *
 * @param {object} att - The element_selection attachment
 * @returns {string} Formatted description
 */
function formatElementSelection(att) {
  const parts = [`Selected element: \`${att.selector}\``];

  // Build the tag description (e.g., "button.submit-btn#main")
  const tagParts = [att.tag];
  if (att.classes && att.classes.length > 0) {
    tagParts.push('.' + att.classes.join('.'));
  }
  if (att.id) {
    tagParts.push('#' + att.id);
  }
  parts.push(`(${tagParts.join('')})`);

  if (att.innerText) {
    // Truncate very long text
    const text = att.innerText.length > 100
      ? att.innerText.slice(0, 100) + '...'
      : att.innerText;
    parts.push(`text: "${text}"`);
  }

  if (att.boundingBox) {
    const bb = att.boundingBox;
    if (bb.width !== undefined && bb.height !== undefined) {
      parts.push(`dimensions: ${bb.width}x${bb.height}`);
    }
  }

  return parts.join(', ');
}

/**
 * Format a screenshot attachment into readable text.
 *
 * @param {object} att - The screenshot attachment
 * @returns {string} Formatted description
 */
function formatScreenshot(att) {
  if (att.description && att.elementSelector) {
    return `[Screenshot of \`${att.elementSelector}\`: ${att.description}]`;
  }
  if (att.elementSelector) {
    return `[Screenshot of \`${att.elementSelector}\`]`;
  }
  if (att.description) {
    return `[Screenshot: ${att.description}]`;
  }
  return '[Screenshot of current page]';
}

/**
 * Format an action_recording attachment into readable text.
 *
 * @param {object} att - The action_recording attachment
 * @returns {string} Formatted description
 */
function formatActionRecording(att) {
  if (!att.actions || att.actions.length === 0) {
    return 'Recorded actions: (none)';
  }

  const lines = ['Recorded actions:'];
  for (let i = 0; i < att.actions.length; i++) {
    const action = att.actions[i];
    lines.push(`  ${i + 1}. ${formatSingleAction(action)}`);
  }
  return lines.join('\n');
}

/**
 * Format a single recorded action into a readable sentence.
 *
 * @param {object} action - A single action object
 * @returns {string} Formatted action description
 */
function formatSingleAction(action) {
  const selector = action.selector ? `\`${action.selector}\`` : 'element';

  switch (action.type) {
    case 'click':
      return `Clicked on ${selector}`;
    case 'dblclick':
      return `Double-clicked on ${selector}`;
    case 'input':
    case 'type':
      return action.value !== undefined
        ? `Typed "${action.value}" in ${selector}`
        : `Typed in ${selector}`;
    case 'change':
      return action.value !== undefined
        ? `Changed ${selector} to "${action.value}"`
        : `Changed ${selector}`;
    case 'select':
      return action.value !== undefined
        ? `Selected "${action.value}" in ${selector}`
        : `Selected option in ${selector}`;
    case 'focus':
      return `Focused on ${selector}`;
    case 'blur':
      return `Left ${selector}`;
    case 'scroll':
      return `Scrolled ${action.direction || 'on'} ${selector}`;
    case 'hover':
      return `Hovered over ${selector}`;
    case 'keypress':
    case 'keydown':
      return action.key
        ? `Pressed "${action.key}" in ${selector}`
        : `Pressed key in ${selector}`;
    case 'navigate':
    case 'navigation':
      return action.url
        ? `Navigated to ${action.url}`
        : `Navigated to new page`;
    case 'submit':
      return `Submitted form ${selector}`;
    default:
      return action.value !== undefined
        ? `${action.type} on ${selector} (value: "${action.value}")`
        : `${action.type} on ${selector}`;
  }
}

/**
 * Format a single attachment into readable text.
 *
 * @param {object} att - An attachment object with a `type` field
 * @returns {string} Formatted attachment text
 */
function formatAttachment(att) {
  switch (att.type) {
    case 'element_selection':
      return formatElementSelection(att);
    case 'screenshot':
      return formatScreenshot(att);
    case 'action_recording':
      return formatActionRecording(att);
    default:
      return `[Attachment: ${att.type}]`;
  }
}

/**
 * Format a complete extension message into a human-readable prompt for Claude.
 *
 * Handles the following message types:
 * - chat: User text, optionally with visual context attachments
 * - element_selection: Standalone element selection (no chat text)
 * - action_recording: Standalone action recording (no chat text)
 * - ping: Returns null (not forwarded to Claude)
 *
 * @param {string} rawMessage - Raw JSON string from the WebSocket
 * @returns {string|null} Formatted text prompt, or null if message should not be forwarded
 */
export function formatMessageForClaude(rawMessage) {
  let data;
  try {
    data = typeof rawMessage === 'string' ? JSON.parse(rawMessage) : rawMessage;
  } catch {
    // If it's not JSON, pass it through as-is (plain text message)
    return rawMessage;
  }

  if (typeof data !== 'object' || data === null) {
    return String(rawMessage);
  }

  switch (data.type) {
    case 'chat': {
      return formatChatMessage(data);
    }

    case 'element_selection': {
      // Standalone element selection (not part of a chat attachment)
      const att = { type: 'element_selection', ...data };
      return `[User selected an element]\n${formatElementSelection(att)}`;
    }

    case 'action_recording': {
      // Standalone action recording
      const att = { type: 'action_recording', actions: data.actions };
      return `[User recorded browser actions]\n${formatActionRecording(att)}`;
    }

    case 'init': {
      const parts = ['[Session initialized]'];
      if (data.url) parts.push(`Current page: ${data.url}`);
      if (data.title) parts.push(`Page title: ${data.title}`);
      if (data.version) parts.push(`Extension version: ${data.version}`);
      return parts.join('\n');
    }

    case 'ping':
      return null; // Don't forward pings to Claude

    default:
      // Unknown message type — forward as-is for resilience
      return rawMessage;
  }
}

/**
 * Format a chat message (with optional attachments) into readable text.
 *
 * @param {object} data - Parsed chat message
 * @returns {string} Formatted message
 */
function formatChatMessage(data) {
  const parts = [];

  // Add the user's text message
  if (data.content) {
    parts.push(data.content);
  }

  // Format and append any attachments as visual context
  if (Array.isArray(data.attachments) && data.attachments.length > 0) {
    parts.push('');  // blank line before context
    parts.push('Visual context:');
    for (const att of data.attachments) {
      parts.push(formatAttachment(att));
    }
  }

  return parts.join('\n');
}

/**
 * Format an init message into a human-readable prompt for Claude.
 *
 * @param {object} data - Validated init data (camelCase)
 * @returns {string} Formatted init message
 */
function formatInitMessage(data) {
  const parts = ['[Session initialized]'];
  if (data.url) parts.push(`Current page: ${data.url}`);
  if (data.title) parts.push(`Page title: ${data.title}`);
  if (data.version) parts.push(`Extension version: ${data.version}`);
  return parts.join('\n');
}

// Export individual formatters for testing
export {
  formatElementSelection,
  formatScreenshot,
  formatActionRecording,
  formatSingleAction,
  formatAttachment,
  formatChatMessage,
  formatInitMessage,
};
