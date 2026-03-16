---
id: gemfury-package-management
created: 2026-02-05T12:00:00Z
priority: 1
status: not_started
---

# GemFury Package Management

## What Must Be True

The @spekk/cli package is distributed via GemFury (npm.fury.io/thinknimble). Documentation exists for both:
- **Users**: How to install the package from GemFury
- **Maintainers**: How to publish new versions to GemFury

## Context

GemFury is used as a private npm registry for the thinknimble organization. This allows controlled distribution of the CLI tool.

## Assertions

1. `readme-contains-installation-instructions` - README explains how users install from GemFury
2. `contributing-documents-publish-process` - CONTRIBUTING.md explains how maintainers publish releases
