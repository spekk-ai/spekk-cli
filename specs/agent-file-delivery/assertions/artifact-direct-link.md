---
id: artifact-direct-link
parent: agent-file-delivery
created: 2026-11-20T00:00:00Z
priority: 1
status: done
branch: feature/artifact-direct-link
depends-on: artifact-upload-command
---

# Artifact upload prints the direct link first

A person who gets the app link of an artifact must click through the app detail page to see the content. spekk-app now returns `view_url`, the direct link on the artifact host (`https://<artifact host>/a/<id>/`), next to `app_url`. The artifact host keeps its checks: a person with no access goes to the app to sign in.

## Success criteria

- When the upload answer has `view_url`, `spekk artifact upload` prints it as the first line, as a plain URL with no label, so that the agent shares it. The second line is `App page: <app_url>` when the answer has `app_url`.
- When the answer has no `view_url` (an older app), the command prints `app_url` alone, as before.
- When the answer has neither link, the command fails with "the app returned no artifact link", as before.
- The usage text and `docs/cli-reference.md` say which link to share.

**Tests:** A local HTTP server answers with both links (two lines, `view_url` first) and with `app_url` only (one line, as before).
