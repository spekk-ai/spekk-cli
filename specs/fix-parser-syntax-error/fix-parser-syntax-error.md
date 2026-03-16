---
id: fix-parser-syntax-error
created: 2026-01-28T21:15:00Z
priority: 1
status: not_started
---

# Fix Parser Syntax Error

## Overview

The spekk CLI parser has a syntax error that prevents it from running. There's an illegal `continue` statement outside of a loop context.

## Error Details

```
file:///Users/william/thinknimble/spekk-cli/src/parser/index.js:78
          continue;
          ^^^^^^^^

SyntaxError: Illegal continue statement: no surrounding iteration statement
```

## Required Fix

The parser code at line 78 in `src/parser/index.js` has a `continue` statement that's not inside a loop. This is invalid JavaScript and must be fixed for the CLI to function.