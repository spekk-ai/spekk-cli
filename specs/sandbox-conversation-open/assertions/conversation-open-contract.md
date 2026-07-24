---
id: conversation-open-contract
parent: sandbox-conversation-open
created: 2026-07-23T00:00:00Z
priority: 1
status: in_progress
locked-by: builder-home-wsl2-632627-1784853437
---

# Shared Package Holds the Request-File Contract

The `spekk conversation open` writer (`cmd/spekk`) and the worker drainer +
frame constructor (`cmd/sandbox`) are separate `main` packages that must agree
on the same request-file contract: the spool environment-variable name, the
JSON shape of a request file, and the set of allowed severities. Rather than
each package re-declaring these literals — where they can silently drift apart —
a single shared internal package is the source of truth both import.

## Success Criteria

- A shared internal package exists (e.g. `internal/conversation`) and is
  importable by both `cmd/spekk` and `cmd/sandbox`. It contains only the
  contract — no WebSocket, no CLI, no worker logic — so both binaries can depend
  on it without pulling in each other's concerns.
- It exports a named constant for the spool environment-variable name whose
  value is `SPEKK_CONVERSATION_SPOOL` (e.g. `conversation.SpoolEnvVar`).
- It exports the request-file struct with json tags **exactly** `title`,
  `body`, `severity` and **no** `session_id` field (the worker stamps the
  session id; the file never carries it — see `conversation-open-frame`).
  Marshalling a value produces a JSON object with only those three keys.
- It exports the severity values as named constants (`info`, `warning`,
  `critical`), the default (`info`), and a single validity check (e.g.
  `IsValidSeverity(string) bool` or an equivalent) so severity handling is
  single-sourced.
- `cmd/spekk` (the `conversation open` writer) imports this package and uses its
  env-var constant, request struct, and severity constants/validity check. It
  does **not** declare its own copy of the env-var name literal, the request
  struct, or the severity value set.
- `cmd/sandbox` (the drainer in `invoke.go` and the frame constructor in
  `message.go`) imports this package and uses its env-var constant, request
  struct, and severity constants. It does **not** re-declare them.
- **Single source of truth (checkable):** the string literal
  `"SPEKK_CONVERSATION_SPOOL"` appears in exactly one place — the shared package.
  A grep for that literal outside the shared package's directory returns
  nothing. Neither `cmd/spekk` nor `cmd/sandbox` defines its own request-file
  struct type or its own list of severity string literals.
- **Boundary rule (public repo):** the package name, its doc comment, and its
  symbol names are generic — they describe the conversation-open request
  contract and name nothing about the control host's implementation stack,
  private repo, or internal admin surface.
- A unit test in the package asserts: the request struct round-trips through
  JSON with exactly `{title, body, severity}` and no `session_id` key; an absent
  severity is representable and the default constant is `info`; and the validity
  check accepts the three severities and rejects anything else.
