# Spekk CLI 1.29.0 - The Spec Explorer on a Phone, and Two New Skills

`spekk show` wrote one page for one width of screen. On a phone the two panels stayed two panels, and the tree was too small to read or to tap. This release gives the page a second layout. It also adds two built-in skills: one that decides whether a promise in the specs deserves a property-based test, and one that reviews what the builder just built.

## The spec explorer on a narrow screen

At 768 pixels wide or less, `spekk show` shows each spec as a full-screen card in a vertical deck, in place of the tree and the detail panel. The deck snaps, so one card comes to rest in view at a time and a fast swipe cannot skip past one. A swipe up on a card raises a sheet that lists that spec's assertions, and a tap on an assertion opens its markdown in a full-screen reader. A sticky search bar filters which cards appear, and the hide-completed toggle and the branch filter apply to the deck as they do to the tree.

One HTML file serves both layouts, and a CSS media query separates them. A rotation or a resize changes the layout live, with no reload and no second run of `spekk show`. Above 768 pixels the page is what it was. The desktop layout is unchanged, and the cross-branch metro map stays desktop-only.

## `spekk coach property-tests`

A new coach skill decides whether a promise deserves a property-based test, then writes the test at the layer that fits it: a browser explorer for the surface a person sees, or a property-based library for the rules below it.

The value gate comes before any code, because search budget and review time are finite. A property must restate an assertion that is `done`, need search that a fixed-input test cannot supply, guard a failure that would matter, have evidence behind it, cost less than it is worth, and keep the portfolio balanced across risk areas. The skill refuses two anti-patterns: exhaustive enumeration of a trivial finite space, and a property that repeats a fixture test.

What it does after the gate: it studies the code through fixed lenses for both layers, writes the catalog entry, chooses the form, and implements it in the project's house pattern with the API of the installed tool version only. It then runs the property clean against seeded data, proves that the run reached the state the property guards, and files one issue for each violation that survives, with a strict expected failure that names it.

The method is adapted from the public Antithesis agent skills, Apache 2.0, for projects that run their own tools without the Antithesis platform.

## `spekk builder review`

The verify phase of the dev loop was the only phase that fetched no instructions, and its one paragraph pointed at a skill that does not ship with spekk. The review is now a procedure that ships in the binary.

Its scope is the assertions marked `done` on the current branch since the branch left its base, plus the diff from that base to `HEAD`. On the base branch itself, you name the commit range. It reads the code through six lenses, and each lens has a remedy:

1. Every success criterion of every assertion in scope is checked against the real code. One that is not met is fixed, or the assertion is set to `failed`.
2. Every test earns its place. A test that passes when its behavior is broken, that repeats the implementation, that duplicates another test, or that exercises a mock is deleted.
3. Nothing goes beyond what the assertions ask for. Generality, configuration, and abstraction that nobody asked for are removed, and a hunk that no assertion accounts for is reverted or explained.
4. Errors are loud. An error that is dropped, defaulted, or caught too broadly is fixed.
5. The spec tree is sound. `spekk validate` and `spekk next` succeed, no lock is stale, and every `**Tests:**` link resolves.
6. The diff is fit to publish. No secret, no private name, and no reference to another repository.

This skill fixes what it finds, which is the difference from an observer skill: the builder role may write code. It writes no observation file. Its output is the fixes in the working tree, and a short report that gives a verdict for each assertion, what was fixed, what was deleted, and what stays open.

One file serves two entry points. `spekk skill show builder review` adopts the skill in the session that built the code, so the review keeps that context. `spekk builder review` runs it in a fresh session, which is what a large or high-stakes change needs, because independence is what a self-review loses. The `spekk-dev-loop` skill loads it in the verify phase. There is no new flag and no new command.

## The docs match the code again

Every page in `docs/` and the README were read against the binary's `--help` output, the flag tables in `cmd/spekk/main.go`, and every environment variable the code reads. The facts were corrected first, then the prose.

The CLI reference had no section for `spekk list`, `spekk conversation open`, `spekk observer digest`, `scan-check`, `announce`, `spekk skills list`, `spekk install <agent> <skill>`, `spekk uninstall`, `spekk update`, or `spekk version`. All of them are there now. The configuration page gained the eight environment variables it never listed, and `.spekk/dont-flag.yaml`. The `spekk sandbox create` flag table gained `--provider`, `--ip`, `--ssh-key`, `--ssh-user`, and `--auth`, and the page no longer says that a sandbox must be a DigitalOcean droplet.

Some of what the pages said was wrong. `spekk init` refreshes or creates the managed block in `specs/README.md`; it does not stop when `specs/` exists. The default for `--specs-dir` is `specs/` at the git root, not `./specs`. The selection rules for `spekk next` omitted two steps: it skips every assertion of a `draft` spec, and an `in_progress` assertion with a fresh `locked-by`. In assertion frontmatter, `parent` is required and was not listed, and `status` is optional with the default `not_started`, not required. The global skills directory has been `~/.config/spekk/skills/<agent>/` since 1.7.0, and two pages still gave the old path. The install page's PowerShell snippet chose `amd64` on every 64-bit machine, ARM64 included; it now reads `PROCESSOR_ARCHITECTURE`.
