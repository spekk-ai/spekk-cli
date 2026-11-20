---
id: text-artifact-command
parent: agent-file-delivery
created: 2026-11-20T00:00:00Z
priority: 1
status: done
branch: fix/text-artifact-shoulds
---

# The artifact command publishes text and code files

## Success criteria

- `.md` and `.markdown` remain markdown; the existing image and video extensions keep their kinds.
- `.txt`, `.log`, `.csv`, `.tsv`, `.json`, `.jsonl`, `.yaml`, `.yml`, `.toml`, `.ini`, `.sql`, `.py`, `.go`, `.js`, `.ts`, `.tsx`, `.jsx`, `.sh`, `.rb`, `.java`, `.kt`, `.rs`, `.c`, `.h`, `.cpp`, `.css`, `.graphql`, `.proto`, `.env.example`, `.diff`, and `.patch` infer `text`.
- An unknown extension infers `text` only when its bytes are valid UTF-8 without NUL; other unknown files keep the current inference error.
- An unknown file over 1 MB gives the inference error and asks for `--kind`.
- `--kind text` sends the text kind explicitly. A server that refuses this kind gives a clear error that identifies an older app as the possible cause.
- The command keeps the 1 MB limit for text and prints the direct link first after success.

**Tests:** cmd/spekk/artifact_test.go
