# Spekk CLI 1.25.0 — Four Silent Failures

Every fix in this release is for something that failed without saying so. A turn died and reported nothing. A sandbox stopped accepting work while its heartbeats kept flowing. Drift that had been fixed once could never be reported again. A comment in frontmatter quietly changed a value, or deleted a list.

**Read the upgrading section before you take this.** One change rewrites values your index already holds.

## A turn survives its connection

The agent client derived a per-connection context, handed it to the worker, and signalled the `claude` process when it ended. A WebSocket drop mid-turn therefore sent SIGTERM to the work, and the client then reconnected onto a turn that was already dead — the control host saw a dispatch that never replied at all. The longer the job, the likelier it failed.

The worker now takes the process lifetime, so only a real shutdown ends a turn. A turn no longer holds the connection it started on either: a sender resolves whichever connection is live at the moment it sends, and a frame that ends a turn waits up to 90 seconds for one, which carries the report across a reconnect. Stream frames do not wait, because they drive a live display and blocking on them would stall the read of the child's output.

The reconnect backoff is fixed in the same pass. It only ever doubled, so a process that had dropped a few times waited the full 60 seconds for every later reconnect, even after hours of health. It now resets after a connection that lived, and a host that accepts and closes at once no longer resets it at all.

## A follow-up message no longer wedges the sandbox

A dispatch started a runner every time, including for a session already draining. Two runners over one worker released the same slot twice; the second release blocked on a queue at its cap while holding the pool mutex, and every later dispatch blocked behind it — permanently. The connection stayed up and the heartbeats kept flowing, so the sandbox looked healthy while accepting no work at all.

A session's queue is now filled without blocking, a runner is started only for a worker just claimed, and a slot returns exactly once per claim. A full queue is refused with `capacity_exceeded` rather than stalling the pool, and that frame now names its session.

## Resolved drift can be found again

An observation suppressed a new finding from any ref that was not main. Every observer branch is cut from `origin/main` and carries a copy of every observation already merged, so one open observer branch was enough to make all of history suppress new findings — and the more drift a team fixed in a file, the more permanently that file was closed.

An observation now suppresses a finding only while it is a **live claim**: read from the branch named after it, and only until its slug reaches main. Dedup keys on the type and the slug rather than an overlapping evidence path, so two unrelated findings in one file no longer collide, and a dated recurrence is covered by its own claim. `--slug` is checked at the gate, because a slug the observation format rejects would file a record the index skips forever.

## A comment in frontmatter is a comment

The frontmatter scanner never treated a trailing `# ...` as a comment. A comment on a key that opens a list made the key read as a scalar and discarded every item under it. Any quote character was treated as opening a quoted scalar, so an apostrophe in plain text protected the rest of a line while a backslash escape truncated a value. And an indented line still set a top-level key, so a `priority:` written inside a prose block became the assertion's priority.

`specs/frontmatter-grammar/` now records the rules the scanner implements, and the limits it does not.

## Upgrading

**A value with an unquoted ` #` loses its tail, and this rewrites rows your index already holds.** The reading is correct YAML and it is the point of the change, but it happens with no warning:

```yaml
desc: fixes #204     # indexes as "fixes"
ref: #123            # loses its value entirely; the row disappears
note: "a # b"        # unchanged, the quote protects it
link: x.com/a#frag   # unchanged, no space before the hash
```

Quote any frontmatter value whose `#` is data. Measured against real spec trees this is rare — 619 files carried no such value — but check before you upgrade a project that uses issue references in frontmatter.

Nothing else needs action. The index schema is unchanged since 1.24.0, and the sandbox agent's wire format is unchanged, so a sandbox can be updated on its own.

```bash
spekk update
```
