---
id: csv-flag
parent: list-output-format
created: 2026-07-12T22:00:00Z
priority: 1
status: done
branch: feat/list-filter-by-status
depends-on: default-table-format
---

# `--csv` Outputs RFC 4180 CSV with Header Row

## Description

`spekk list --csv` prints comma-separated values conforming to RFC 4180.
A header row appears on the first line. Fields containing commas, double
quotes, or newlines are enclosed in double quotes; embedded double quotes are
escaped as `""`.

## Success Criteria

- `spekk list --csv` produces a header row `id,status,pri,title`.
- Fields that contain a comma (e.g. a title like "Auth, Permissions & Roles")
  are double-quoted: `"Auth, Permissions & Roles"`.
- Fields that contain a double quote are double-quoted and the embedded quote
  is doubled: `"She said ""hello"""`.
- Fields that contain neither comma nor double quote are not quoted.
- Each row ends with CRLF (`\r\n`) as specified by RFC 4180.
- A unit test in `internal/formatter/formatter_test.go` exercises at least one
  row with a comma in the title to verify correct quoting.
