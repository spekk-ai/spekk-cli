---
id: builder-review-skill
created: 2026-09-03T19:40:00Z
priority: 1
---

# Builder Review Skill

## Problem

The `spekk-dev-loop` skill runs three phases in one session. The scope phase fetches the full coach role with `spekk prompt coach`. The build phase fetches the full builder role with `spekk prompt builder`. The verify phase fetches nothing. It is one paragraph: re-read the code with an adversarial eye, apply the test bar, flag or fix. A reader gets a mood, not a procedure, so the quality of the verify phase depends on what the session remembers to check.

The paragraph also points at `code-quality-principles` for the full test bar. That skill is a maintainer's personal skill. It does not ship with spekk, so a user who installs spekk cannot follow the reference.

The builder already has the skill machinery that would carry a procedure. `spekk builder <skill>` launches a fresh session with the skill inlined, `spekk skill show builder <name>` prints a skill for a running session to adopt, and `spekk builder --help` lists what is available. The package directory `specs/builder-skills/` is empty, and `legacyAliases["builder"]` is `{}`. The builder is the only role with no built-in skill.

## Change

Ship one built-in builder skill, `review`, at `specs/builder-skills/review-skill.md`. It is the procedure for the verify phase. It reviews what was just built against the assertions that were marked `done`, and it applies a fixed set of lenses, each with a named remedy.

The verify phase of the dev loop loads it with `spekk skill show builder review` and follows it in the same session. This keeps the pilot's result (issue #154): the review reuses the context that the build made, and it does not pay for a fresh context load.

The same file serves the case where independence matters more than context. Issue #154 states it: independence is the one thing a self-review loses. For a high-stakes or large change, `spekk builder review` launches the skill in a fresh session. The dev loop names this path as the exception, in the same way it names the escalation to sub-agents.

## The review procedure

**Scope.** The assertions marked `done` on the current branch since it left its base branch, and the diff between that base and `HEAD`. When the session runs on the base branch itself, the user names the commit range.

**Lenses, in this order, each with its remedy:**

1. **Each criterion, against the real code.** Re-read every success criterion of every in-scope assertion, and check it against the code as it is, not the summary in memory. For a criterion with non-obvious behavior, trace one concrete input through the code. A criterion that is not met is fixed now. One that cannot be fixed now sets the assertion to `failed`, which is the status the model already has for a confirmed gap.
2. **Each test earns its place.** A test that would still pass if the behavior it pins were broken tests nothing. A test that restates the implementation, that duplicates another test, or that exercises a mock instead of the real path is deleted. The bar is lean and high value.
3. **Nothing more than the assertion asked for.** Generality, configuration, and abstraction that no assertion requires are removed. A hunk in the diff that no assertion accounts for is reverted, or the reason it stays is stated.
4. **Errors are loud.** An error that is dropped, replaced with a default, or caught broadly is a defect.
5. **The spec tree is sound.** `spekk validate` exits 0, `spekk next` succeeds, no assertion carries a stale lock, and each `**Tests:**` link points to a file that exists.
6. **The diff is fit to publish.** Nothing in it that the project's own rules forbid: a secret, a private name, a reference to a repository that is not this one.

**Output.** Fixes in the working tree, plus a short report in the session: each in-scope assertion with a verdict, what was fixed, what was deleted, and what stays open. The review writes no observation file. That is the observer's contract, and the builder is not the observer.

**Gate.** The push waits for the review. The old multi-agent loop had this rule, and the single-session loop keeps it.

## Decisions

- **One skill, two entry points.** `spekk skill show builder review` for the in-session review, `spekk builder review` for the fresh-context review. One file, no fork of the content.
- **`review`, not `verify`.** The dev loop's phase is called verify. The skill is called review because it is also the name a user reaches for on its own (`spekk builder review`), and because the observer already owns the word "scan".
- **The remedy is a fix, not a report.** The builder role may write code, so the review fixes what it finds. This is the difference from an observer skill, which may only recommend.
- **`failed` is the escape hatch.** A criterion the review cannot meet in the session sets the assertion to `failed`. No new status, no new frontmatter field.
- **The test bar moves into spekk.** The dev-loop skill stops pointing at a private skill. It states the bar in one line and points at the review skill for the procedure.

## Temptations rejected

- **No new CLI flag or command.** Discovery, help, `spekk skill show`, and `spekk builder <skill>` already exist. The skill lands through them.
- **No scope arguments.** No `--base` flag, no assertion filter. The skill states how to find its scope from git and the spec tree.
- **No observation output.** The review does not write to `observations/`. It is not a scan, and its findings are fixed on the spot.
- **No review-specific frontmatter.** No `reviewed:` field, no review log in the assertion body.
- **No second copy of the lenses in the dev-loop skill.** The dev loop loads the skill. It does not restate it.

## Assertions

See `assertions/` for what must be true.
