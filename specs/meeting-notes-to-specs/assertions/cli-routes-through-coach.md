---
id: cli-routes-through-coach
parent: meeting-notes-to-specs
created: 2026-02-24T23:59:00Z
priority: 1
status: done
---

# CLI Routes Through Coach

Meeting processing is accessed via `spekk coach meeting`, not as a top-level `spekk meeting` command.

## Success Criteria

- `spekk coach meeting` launches the coach with meeting-processing skill active
- No top-level `spekk meeting` command exists
- Meeting processing is a subcommand of coach, keeping the three-agent architecture (coach, builder, observer)
- `spekk coach meeting <transcript-file>` accepts a transcript file argument
- Without a file argument, coach prompts the user to paste or provide a transcript
