# Spekk CLI 1.51.1 - Direct Artifact Link First

This patch changes the links printed by `spekk artifact upload`. When the app returns `view_url`, the CLI prints that direct link on the artifact host as the first line, with no label. It prints `App page: <app_url>` on the second line. Share the first link to open the artifact content.

An app that does not return `view_url` causes the CLI to print `app_url` alone, with no label, as in 1.51.0. If the app returns neither link, the command fails with `the app returned no artifact link`.

## What you must do

Update the CLI to 1.51.1 to get the direct link first. The direct link needs the Spekk app release with spekk-app #718, deployed 2026-11-20. With an older app, the CLI continues to print the app page URL alone.

## Compatibility

The WebSocket protocol version stays `1.6`, and its major stays `1`.
