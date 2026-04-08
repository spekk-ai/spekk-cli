---
id: redis-lock-backend
parent: external-lock-backend
created: 2026-04-08T16:00:00Z
priority: 2
status: not_started
depends-on: define-lock-store-interface
branch: feature/external-lock-backend
---

# Redis Lock Backend

## What Must Be True

A `redis` backend adapter implements `LockStore` using Redis `SET NX PX` for atomic acquisition and a Lua script for safe release. This enables teams running a self-hosted or managed Redis instance (on any cloud, on-prem, or a team member's laptop) to coordinate locks with real atomic primitives.

## Success Criteria

- ✅ New file `internal/locks/redis.go` contains `RedisBackend` struct implementing `LockStore`
- ✅ New dependency: `github.com/redis/go-redis/v9` (latest stable)
- ✅ Constructor `NewRedisBackend(url string, defaultTTL time.Duration) (*RedisBackend, error)` parses the Redis URL and verifies connectivity via `PING`
- ✅ Key scheme: `spekk:lock:{assertionID}` — prefixed to avoid collisions with other Redis users
- ✅ `Acquire`:
  - Serializes `LockInfo` to JSON
  - Runs `SET spekk:lock:{assertionID} {jsonLockInfo} NX PX {ttlMillis}`
  - On `OK` reply → success
  - On nil reply → fetches current holder via `GET` and returns `ErrLockHeld` with holder info
- ✅ `Release`:
  - Runs a Lua script that atomically checks `GET == lockID` before `DEL`
  - If value matches → delete, return success
  - If value missing → return `ErrLockNotFound`
  - If value mismatches (someone else holds it now) → return `ErrLockExpired`
- ✅ `Inspect`: `GET spekk:lock:{assertionID}`, parse JSON, return `LockInfo` or `ErrLockNotFound`
- ✅ `List`: scans `spekk:lock:*` using `SCAN` (not `KEYS`), reads each value, returns deserialized list
- ✅ `ForceUnlock`: `DEL spekk:lock:{assertionID}`
- ✅ TTL is enforced by Redis itself via `PX` — no client-side TTL math required
- ✅ Unit tests use `miniredis` (`github.com/alicebob/miniredis/v2`) so no real Redis required in CI
- ✅ Test coverage: acquire, double acquire (second fails), release, release after expiry, inspect missing, list multiple, force-unlock, concurrent acquire race
- ✅ Integration test (opt-in via `REDIS_TEST_URL` env var) against real Redis

## Release Lua Script

```lua
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("DEL", KEYS[1])
else
    return 0
end
```

Wrapped in a Go helper that interprets the return value against the contract.

## Connection Semantics

- Single shared `*redis.Client` per `RedisBackend` instance
- Respects `context.Context` deadlines and cancellations on all operations
- No connection pooling config exposed for MVP — rely on go-redis defaults

## Out of Scope

- Redis Cluster / Sentinel awareness (single instance only for MVP)
- Redis ACL / user-level auth beyond what's in the URL
- Pub/Sub-based live lock change notifications (can be added later for `spekk show --watch`)
- TLS configuration beyond `rediss://` URL scheme

## Notes

Redis was chosen over DynamoDB because (a) it runs anywhere — a laptop, a Docker container, a free-tier managed instance — and (b) it's already the default distributed lock store across most dev teams. Any team can stand up Redis in minutes; DynamoDB requires AWS commitment.
