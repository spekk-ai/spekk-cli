---
id: spec-explorer-web-interface
created: 2026-01-22T21:00:00Z
priority: 2
---

# Spec Explorer Web Interface

## Overview

The `spekk show` command generates a web-based interface for exploring specs as an interactive tree with detailed views. This provides a visual alternative to the text-based `spekk status` command.

## Core Functionality

The command creates a temporary web page that displays the spec hierarchy as an expandable tree interface, similar to a file explorer. Users can navigate the spec structure and view detailed information about each spec and assertion.

## Interface Design

**Tree Navigation:**
- Specs appear as top-level items in an expandable dropdown tree
- Assertions appear as sub-items under their parent specs
- Visual indicators show status and priority for each item

**Detail Panel:**
- Right-side panel shows details when clicking any spec or assertion
- Emphasizes status (not_started, in_progress, done) and priority (1, 2, 3)
- Displays spec/assertion content and metadata

## File Generation

**Location:** Generated HTML file is placed at `.spekk/index.html`
**Directory Creation:** `.spekk/` directory is created automatically if it doesn't exist
**Browser Launch:** Generated file opens in the system's default browser

## Success Criteria

Users can run `spekk show` in any spec-driven project directory and immediately see a visual representation of their spec hierarchy with detailed status information.