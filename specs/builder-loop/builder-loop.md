---
id: builder-loop
created: 2026-07-09T16:00:00Z
priority: 1
---

# Builder Loop

## Overview

The builder loop (`spekk loop builder` and the continuous mode of `spekk builder`) runs assertions end-to-end without human intervention. It must know when it's done, report what it accomplished, handle stuck builds, and support post-build skill pipelines.

Currently the Go loop polls forever when complete, has no assertion tracking, no idle timeout for stuck builders, and no post-build skill support. The shell-based `spekk-loop` script handles all of these — this spec brings the Go implementation to parity.

## Context

- Go loop: `internal/agent/loop.go` (`RunBuilderLoop`)
- Go builder continuous mode: `internal/agent/builder.go` (`RunBuilder` without `--once`)
- Shell loop: `~/.claude/bin/spekk-loop` (reference implementation)
- Parser completion signal: `spekk next` returns `{"type": "complete"}` when all assertions are done

## Design

The parser is the source of truth for completion. Each loop iteration calls `spekk next` — when it returns `type: "complete"`, the loop knows all work is done. No output monitoring or marker detection needed.

```
Iteration 1:  spekk next → assertion → build → commit → count++
Iteration 2:  spekk next → assertion → build → commit → count++
Iteration 3:  spekk next → "complete" → DONE (count=2, run skills, exit)
```

## Assertions

See `assertions/` for what must be true about loop behavior.
