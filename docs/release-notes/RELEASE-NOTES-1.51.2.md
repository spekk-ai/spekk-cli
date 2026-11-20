# Spekk CLI 1.51.2 - Text Artifact Uploads

`spekk artifact upload` now infers kind `text` for known text and code extensions, such as `.txt`, `.json`, `.sql`, `.py`, and `.go`. It also infers `text` for a file with an unknown extension when the file is at most 1 MiB, is valid UTF-8, and has no NUL bytes. Use `--kind text` to select this kind explicitly. Markdown, image, and video extensions keep their kinds.

The CLI refuses to infer a kind for `.html`, `.htm`, `.svg`, or `.xhtml` and prints `Error: cannot infer the artifact kind; active content is not a text artifact`. For a binary file with an unknown extension, or an unknown file over 1 MiB, it prints `Error: cannot infer the artifact kind; use --kind`. An explicit kind does not bypass the upload size limit.

On success, the CLI prints the direct content link first and `App page: <app_url>` second when the app returns both links. If the app returns only `app_url`, it prints that URL alone. When an app rejects kind `text` with HTTP 400 and `Choose the kind` or `Unknown kind` in its response, the CLI prints `Error: this app does not support text artifacts yet: <app response>`. HTTP 404 still prints `Error: artifacts are off on this app`.

## What you must do

Update the CLI to 1.51.2 to upload text and code artifacts. Kind `text` needs the Spekk app release with spekk-app #720, deployed 2026-11-20.

## Compatibility

The WebSocket protocol version stays `1.6`, and its major stays `1`.
