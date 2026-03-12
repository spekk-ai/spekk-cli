/**
 * WebSocket message shapes and typed API factory.
 *
 * Uses createWSApi from @thinknimble/tn-models to provide Zod-validated,
 * camelCase <-> snake_case converted, event-based WebSocket messaging.
 *
 * Wire protocol: `{ event: 'coach:<operation>', data: { ... } }`
 */

import { z } from 'zod';
import { createWSApi } from '@thinknimble/tn-models';

// -- Shared sub-schemas -------------------------------------------------------

const attachmentZod = z.object({
  type: z.string(),
  selector: z.string().optional(),
  tag: z.string().optional(),
  classes: z.array(z.string()).optional(),
  id: z.string().optional(),
  innerText: z.string().optional(),
  boundingBox: z.object({
    x: z.number().optional(),
    y: z.number().optional(),
    width: z.number().optional(),
    height: z.number().optional(),
  }).optional(),
  dataUrl: z.string().optional(),
  elementSelector: z.string().optional(),
  description: z.string().optional(),
  actions: z.array(z.record(z.unknown())).optional(),
});

const actionZod = z.object({
  type: z.string(),
  selector: z.string().optional(),
  value: z.string().optional(),
  key: z.string().optional(),
  url: z.string().optional(),
  direction: z.string().optional(),
  timestamp: z.number().optional(),
});

// -- Server -> Browser (send) -------------------------------------------------

const serverSendOps = {
  status: {
    inputShape: {
      state: z.string(),
      detail: z.string().optional(),
    },
  },
  assistant: {
    inputShape: {
      content: z.string(),
      sessionId: z.string(),
    },
  },
  result: {
    inputShape: {
      content: z.string(),
      isError: z.boolean(),
      sessionId: z.string(),
    },
  },
  error: {
    inputShape: {
      message: z.string(),
    },
  },
  agentExited: {
    inputShape: {
      code: z.number(),
    },
  },
};

// -- Browser -> Server (receive) ----------------------------------------------

const serverReceiveOps = {
  chat: {
    outputShape: {
      content: z.string().optional(),
      attachments: z.array(attachmentZod).optional(),
    },
  },
  elementSelection: {
    outputShape: {
      selector: z.string(),
      tag: z.string(),
      classes: z.array(z.string()).optional(),
      id: z.string().optional(),
      innerText: z.string().optional(),
      boundingBox: z.object({
        width: z.number(),
        height: z.number(),
      }).optional(),
    },
  },
  actionRecording: {
    outputShape: {
      actions: z.array(actionZod),
    },
  },
  init: {
    outputShape: {
      url: z.string().optional(),
      title: z.string().optional(),
      version: z.string().optional(),
    },
  },
};

// -- Factory ------------------------------------------------------------------

/**
 * Create a typed WebSocket API for the serve bridge.
 *
 * @param {import('./ws-adapter.js').createWSAdapter} wsAdapter - WSClientLike adapter
 * @returns Typed API with send/on/off methods
 */
export function createServeApi(wsAdapter) {
  return createWSApi({
    channel: 'coach',
    client: wsAdapter,
    operations: {
      send: serverSendOps,
      receive: serverReceiveOps,
    },
  });
}
