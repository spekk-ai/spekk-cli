---
id: bootstrap-system-recovery
created: 2026-01-28T21:30:00Z
priority: 1
status: not_started
---

# Bootstrap System Recovery

## Overview

**CRITICAL BOOTSTRAPPING PROBLEM:** The builder cannot run due to parser syntax errors, which means the builder cannot fix itself. This requires manual intervention to restore basic functionality.

## Current Failures

1. `spekk builder` fails with parser syntax error
2. Parser has illegal `continue` statement at line 78
3. Builder uses internal parser instead of global CLI
4. CLI reads from wrong directory context

## Bootstrap Strategy

Since the builder cannot run to fix itself, these critical fixes must be implemented manually or with direct code intervention to restore the system.

**Immediate Goal:** Get `spekk next` working so the builder can resume automated implementation.