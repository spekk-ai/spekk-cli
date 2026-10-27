---
id: sandbox-attachment-downloads
created: 2026-10-21T20:09:18Z
priority: 1
---

# Sandbox attachment downloads

The agent receives attachment IDs, filenames, and MIME types through its existing message protocol. It downloads the exact bytes through the authenticated attachment route and gives the saved paths to the model through standard input. It does not start the model when required file input fails.

The CLI enforces a separate download safety ceiling of 10 files, 20 MiB for one file, and 40 MiB for one turn. The control host can impose lower upload limits. Downloads require HTTP 200, reject redirects, use a timeout and the turn context, and stop at the byte ceiling. Each turn has a private directory, safe filenames, and private files. A failed batch removes its partial output.
