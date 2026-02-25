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
- The activation message includes the full content of the meeting skill markdown file (not just a vague "activate skill" instruction)
- No top-level `spekk meeting` command exists
- Meeting processing is a subcommand of coach, keeping the three-agent architecture (coach, builder, observer)
- `spekk coach meeting <transcript-file>` accepts a transcript file argument
- Without a file argument, coach prompts the user to paste or provide a transcript

## Bug (2026-02-25)

Currently the CLI appends a 3-line activation instruction ("Activate your meeting-notes-to-specs skill immediately") but never reads or inlines `specs/coach-skills-system/meeting-notes-to-specs-skill.md`. The coach agent doesn't know what workflow to follow and falls back to regular coach behavior. See `cli-subcommands-inline-skill-content` assertion in coach-skills-system for the generic fix.
