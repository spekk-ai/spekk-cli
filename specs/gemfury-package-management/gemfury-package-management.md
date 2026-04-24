---
id: gemfury-package-management
created: 2026-02-05T12:00:00Z
priority: 1
status: done
---

# GemFury Package Management (SUPERSEDED)

> **Superseded:** This spec is no longer applicable. The project has migrated from Node.js/npm to Go. Distribution now uses GitHub Releases with cross-compiled Go binaries instead of GemFury/npm. See the `golang-cleanup` spec for details.

## What Must Be True

~~The @spekk/cli package is distributed via GemFury (npm.fury.io/thinknimble).~~ Distribution now uses GitHub Releases with Go binaries.

## Assertions

1. `readme-contains-installation-instructions` - README explains how users install from GemFury
2. `contributing-documents-publish-process` - CONTRIBUTING.md explains how maintainers publish releases
