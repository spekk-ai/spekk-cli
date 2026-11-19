---
id: app-tests-never-touch-the-real-config-dir
parent: tests-pass-on-any-machine
created: 2026-11-19T15:58:31Z
priority: 1
status: not_started
branch: feature/tests-pass-on-any-machine
---

# App-Client Tests Use a Per-Test Request Store on Every OS

On macOS, `os.UserConfigDir()` ignores `XDG_CONFIG_HOME`. The app tests set that variable, so on macOS they write to the developer's real request store. This is priority 1 because the tests change the developer's real data, not only their own results.

## Success Criteria

- On Linux and macOS, each test in `cmd/spekk` and `internal/appclient` that reaches the request store uses a store directory inside `t.TempDir()`.
- No test creates, reads, or modifies a file under the real user config directory (`os.UserConfigDir()` as resolved before the test changes any variable). A test asserts this: it records that directory's request-store files before and after the app tests run, and expects no change.
- A request store that already holds entries with the same request IDs the tests use (for example `manual-1`) does not change any test result.
- Running the app tests twice in a row with `-count=1` passes both times on macOS.
- Outside tests, `spekk` still keeps its request store in the user config directory, at the same path as today, so existing stores are still found.

**Note:** The test fix must not depend on `XDG_CONFIG_HOME`, because `os.UserConfigDir()` honors it only on Unix other than macOS. The store location needs a seam that works the same on every OS, for example a path the client takes as input or an environment variable spekk itself reads.
