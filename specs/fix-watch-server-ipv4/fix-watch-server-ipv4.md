---
id: fix-watch-server-ipv4
created: 2026-03-11T00:25:00Z
priority: 1
---

# Fix: Watch server binds to 127.0.0.1 (IPv4) explicitly

The watch server currently binds to `'localhost'` which resolves to `::1` (IPv6) on Linux.
Test requests using `http.get` or `fetch` connect to `127.0.0.1` (IPv4), causing ECONNREFUSED.

## Root Cause

`server.listen(port, 'localhost', ...)` in `src/show/server.js` uses `'localhost'` which resolves
to `::1` on Linux (Node 18+), while HTTP clients resolve `localhost` to `127.0.0.1` (IPv4).

## Fix

Bind explicitly to `'127.0.0.1'` instead of `'localhost'` — still localhost-only (secure),
but deterministically IPv4.

## Done When

Watch server and integration tests pass on Linux CI.
