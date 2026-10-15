---
id: control-host-client
parent: sandbox-self-register
created: 2026-10-15T00:00:00Z
priority: 1
status: not_started
branch: feat/sandbox-self-register
---

# The CLI Calls the Control Host With the User's API Token

One client in `internal/sandbox` makes every call from the CLI to the control host. It reads its two settings from the environment, and it keeps every credential out of logs and stdout.

**Tests:** internal/sandbox/controlhost_test.go

## Success Criteria

- `SPEKK_API_TOKEN` is the credential for every call. `SPEKK_HOST` is the address.
- A bare host such as `spekk.example.com` becomes `https://spekk.example.com`. A host with `https://` keeps it. A trailing slash is removed.
- `http://` is accepted only for a loopback host (`localhost`, `127.0.0.1`, `::1`), and a bare loopback host becomes `http://`. Any other `http://` host is an error, because the API token would cross the network in clear text.
- A command that needs the client and finds `SPEKK_API_TOKEN` or `SPEKK_HOST` empty fails with an error that names the missing variable.
- One function sets the `Authorization: Token <key>` header. No other code sets an `Authorization` header on a control host request.
- The client does not follow a redirect. A `3xx` answer is an error like any other status, so the `Authorization` header never goes to a second address.
- A response that is not 2xx becomes an error with the method, the path, the status code, and the `detail` text of the body when it has one. The error never contains the API token or an agent token.
- No command writes the API token or an agent token that the control host returned to stdout or to stderr.

## Verification

- `go test ./internal/sandbox` covers the host rules, the missing-variable errors, the single auth header, an error from a 4xx response, and a redirect that is not followed.
