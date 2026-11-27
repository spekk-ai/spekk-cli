---
id: serve-stop-status
parent: sandbox-remote-serve
created: 2026-11-08T00:00:00Z
priority: 2
status: not_started
depends-on: serve-start
branch: feature/sandbox-remote-serve
---

# Remote serve can be inspected and stopped

## Success Criteria

- `spekk sandbox serve status <name>` reports `running` or `stopped`. When running it shows the port, the uptime, whether a client is connected, and the sandbox's auth mode. `--json` prints the same fields as one object, with no nonce.
- `spekk sandbox serve stop <name>` stops serve and every `claude` it spawned, and removes the nonce file. Status reads `stopped` afterwards.
- `stop` on a sandbox where serve is not running exits 0 and says it was not running.
- After the sandbox reboots, `status` reads `stopped` and `connect` gives its not-running message.
- Stopping serve never stops or restarts the worker service.
