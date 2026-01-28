---
id: fix-builder-test-validation
created: 2026-01-28T21:40:00Z
priority: 1
status: not_started
---

# Fix Builder Test Validation

## Overview

The builder is incorrectly marking assertions as `done` without properly validating that tests pass. This leads to CI failures while the builder thinks everything is working.

## Current Problems

1. **Builder prompt references wrong test commands** - Says `just test` but project uses `npm test`
2. **Builder not actually checking test results** - Marks assertions `done` despite failing tests
3. **37 tests currently failing** - But builder completed assertions anyway

## Root Cause

The builder prompt instructs to use `just` commands that don't exist in this project, and doesn't properly verify test success before marking assertions complete.