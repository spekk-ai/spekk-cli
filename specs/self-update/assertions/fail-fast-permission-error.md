---
id: fail-fast-permission-error
parent: self-update
created: 2026-06-12T00:00:00Z
priority: 1
status: done
branch: fix/update-permission-message
---

# Permission Problems Fail Fast With an Actionable Message

## Description

When spekk was installed into a root-owned directory (e.g. `/usr/local/bin`
via a sudo install), `spekk update` cannot write the new binary. This must
fail *before* downloading anything, and the error must tell the user exactly
what to do.

Implemented in PR #114.

## Success Criteria

- The temp file in the target directory is created **before** the asset
  download starts, so a non-writable install dir fails immediately rather
  than after a wasted download
- On a permission error (`os.IsPermission`) creating the temp file, the error
  is exactly:
  `no write permission for <dir> (spekk was likely installed with sudo) — run: sudo spekk update, or reinstall to user-owned ~/.local/bin for sudo-free updates: https://github.com/spekk-ai/spekk-cli#install`
  where `<dir>` is the directory containing the binary — it must offer both
  the immediate workaround (sudo) and the convention-aligned fix (reinstall
  to a user-owned directory)
- Other temp-file creation failures still surface as
  "cannot create temp file: <cause>"
- Atomicity guarantees from `atomic-binary-replace` are unchanged: temp file
  in the same directory, rename to swap, temp file removed on every error
  path

## Notes

Auto-escalating to sudo from inside the binary was considered and rejected;
the long-term fix is installing to a user-owned directory by default (see
`specs/install-script/`, assertion `default-install-dir-user-owned`).
