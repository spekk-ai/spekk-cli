---
id: publish-workflow-fails-on-error
parent: gemfury-package-management
created: 2026-02-05T21:15:00Z
priority: 1
status: done
---

# Publish Workflow Fails on Error

## What Must Be True

The GitHub Action publish workflow fails (non-zero exit) when GemFury returns an error response, even if the HTTP status is 200.

## Context

GemFury returns HTTP 200 with error messages in the response body (e.g., "account access denied"). The workflow must check the response body and fail explicitly when the upload doesn't succeed.

## Success Criteria

- Workflow captures curl response output
- Workflow checks response for success indicator ("ok")
- Workflow exits with non-zero code if response doesn't indicate success
- Workflow outputs clear error message on failure
- GitHub Actions marks the run as failed when publish fails

## Validation

- Simulate a publish failure (e.g., invalid token) and verify the workflow run shows as failed in GitHub Actions
