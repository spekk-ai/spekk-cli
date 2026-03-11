---
id: watch-server-binds-to-ipv4
parent: fix-watch-server-ipv4
created: 2026-03-11T00:25:00Z
priority: 1
status: in_progress
locked-by: builder-Williams-MBP.local-54279-1773188871
branch: fix/watch-server-ipv4
---

# Watch server binds to 127.0.0.1 (IPv4) explicitly

`src/show/server.js` binds to `'127.0.0.1'` instead of `'localhost'` so tests pass on Linux.

## Success Criteria

- `listenWithRetry` in `src/show/server.js` calls `server.listen(port, '127.0.0.1', ...)` (not `'localhost'`)
- All tests in `src/__tests__/watch-server-module.test.js` pass on Linux
- All tests in `src/__tests__/watch-mode-integration.test.js` pass on Linux
- Test in `watch-server-module.test.js` for binding address still passes:
  `addr.address === '127.0.0.1'` (the `|| addr.address === '::1'` branch becomes dead code but assertion still holds)
- `npm test` exits 0
