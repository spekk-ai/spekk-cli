---
id: test-helper-scripts-are-portable
parent: tests-pass-on-any-machine
created: 2026-11-19T15:58:31Z
priority: 2
status: not_started
branch: feature/tests-pass-on-any-machine
---

# Shell Scripts That Tests Write Run Under BSD and GNU Userland

## Success Criteria

- `TestBuilderLoopExitWhenDone` passes on macOS with the system `sed`, and on Linux with GNU `sed`.
- No fake binary or helper script that a test writes uses a flag whose meaning differs between BSD and GNU tools. One example is `sed -i` without a suffix argument: GNU reads the next word as the expression, and BSD reads it as the backup suffix.
- A fake binary that is meant to succeed exits 0 on both platforms. When it fails, the test reports the script's own stderr, not only the loop's timeout (`signal: killed`).

**Note:** "Portable" here means POSIX `sh` plus tool flags that BSD and GNU both accept. A script may avoid the tool altogether, for example by rewriting the file with a test-built helper binary.
