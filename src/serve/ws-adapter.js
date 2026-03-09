/**
 * WSClientLike adapter for raw `ws` WebSocket connections.
 *
 * The `ws` library only has generic `send(data)` and `on('message', handler)`.
 * This adapter adds event-based routing so it satisfies the WSClientLike
 * interface expected by `createWSApi` from @thinknimble/tn-models:
 *
 *   { send(event, data), on(event, handler), off(event, handler?) }
 */

/**
 * Create a WSClientLike adapter around a raw `ws` WebSocket.
 *
 * @param {import('ws').WebSocket} ws - A connected ws WebSocket instance
 * @returns {{ send: Function, on: Function, off: Function }}
 */
export function createWSAdapter(ws) {
  /** @type {Map<string, Set<Function>>} */
  const handlers = new Map();

  // Single listener on the raw ws that routes to registered event handlers
  ws.on('message', (raw) => {
    let parsed;
    try {
      parsed = JSON.parse(raw.toString());
    } catch {
      return; // non-JSON message, ignore
    }

    const { event, data } = parsed;
    if (!event) return;

    const set = handlers.get(event);
    if (set) {
      for (const fn of set) {
        fn(data);
      }
    }
  });

  return {
    /**
     * Send a named event with data over the WebSocket.
     * Serializes to `{ event, data }` JSON on the wire.
     */
    send(event, data) {
      if (ws.readyState === ws.OPEN) {
        ws.send(JSON.stringify({ event, data }));
      }
    },

    /**
     * Register a handler for a named event.
     */
    on(event, handler) {
      let set = handlers.get(event);
      if (!set) {
        set = new Set();
        handlers.set(event, set);
      }
      set.add(handler);
    },

    /**
     * Remove a handler (or all handlers) for a named event.
     */
    off(event, handler) {
      if (!handler) {
        handlers.delete(event);
        return;
      }
      const set = handlers.get(event);
      if (set) {
        set.delete(handler);
        if (set.size === 0) handlers.delete(event);
      }
    },
  };
}
