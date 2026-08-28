---
id: agents-query-the-index
created: 2026-08-21T18:20:00Z
priority: 2
---

# An agent searches the spec tree with `spekk query`

## Problem

The coach prompt tells an agent to find assertions by keyword like this:

```bash
spekk list --json | grep "keyword"  # find assertions by keyword without loading every file
```

That is the wrong tool for the job, and it fails in a way the agent cannot see.

**It returns a fragment, not a record.** `spekk list --json` prints indented JSON, so `grep` matches one line and prints that line alone. A hit on a title gives the agent `"title": "..."` with no `id` and no `file`. The agent has a match it cannot act on, and no way to know what it matched.

**It matches the wrong fields.** `grep` reads every line of the record, so a keyword that appears in a `file` path or a `parent` id looks exactly like a keyword in a title.

**It re-parses the whole tree to answer.** `spekk list` walks and parses every spec file. `spekk query` reads `.spekk/index.db`, which the command refreshes first, so the answer is as current and costs a SQL statement.

The two commands see the same information. Neither `spekk list --json` nor the index holds an assertion body: the index tables carry `id`, `parent_id`, `title`, `status`, `priority`, `branch`, and `file`, and the list JSON carries the same set plus `depends_on`. So `grep` buys nothing that SQL cannot do better, and it loses the record structure on the way.

## Solution

Recommend `spekk query` first in every agent prompt that tells an agent how to find an assertion. It answers from the index, it returns whole rows, it filters with `WHERE ... LIKE`, it joins `depends_on`, and it counts and groups.

Keep `grep -rl "keyword" specs/` for a search of the spec **prose**. No table holds a body, so that is the one search `spekk query` cannot serve, and the prompts must keep saying so.

## Scope

- In scope: the search guidance in `coach.prompt.md` and `builder.prompt.md`.
- Out of scope: the observer prompt, which recommends no search at all; a full-text index of spec bodies, which would make `grep` unnecessary but is a far larger change with its own staleness problem; and any change to `spekk list`, which stays the right tool for a human at a terminal who wants a table.
