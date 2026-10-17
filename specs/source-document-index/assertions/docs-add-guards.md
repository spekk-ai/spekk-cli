---
id: docs-add-guards
parent: source-document-index
created: 2026-10-17T16:00:00Z
priority: 2
status: not_started
branch: feature/source-document-index
depends-on: docs-add-text-formats
---

# `spekk docs add` Refuses Secrets and Keeps Large Binaries Out of Git

## Success criteria

- Before it writes anything, `docs add` scans the plaintext of each file for: PEM private key headers (`-----BEGIN ... PRIVATE KEY-----`), AWS access key ids (`AKIA` or `ASIA` and 16 uppercase letters or digits), GitHub tokens (`ghp_`, `gho_`, `ghu_`, `ghs_`, `ghr_`, `github_pat_`), Slack tokens (`xox[abprs]-`), Anthropic keys (`sk-ant-`), and Google API keys (`AIza` and 35 characters).
- On a match, that file fails: the message names the file, the kind of secret, and the line number, and never prints the secret. It says that git history is permanent and that `--allow-secrets` adds the file anyway after a person checks it.
- A converted file (anything that is not `.md`, `.markdown`, `.txt`) larger than 5 MB (5,242,880 bytes) is a pointer: its path under `raw/` is added once to `raw/.gitignore`, the stub keeps `sha256`, `size`, and `source`, and the derived text is committed as usual. The command says that the raw file stays local.
- On a clone without the raw file, `spekk docs status` shows the document as `pointer` (not missing), and search and show work from the derived text.

**Tests:** the secret scan (each pattern found with its line number, the secret not in the message, a near miss not flagged) and the size rule (a file one byte above the limit is a pointer, one at the limit is not).
