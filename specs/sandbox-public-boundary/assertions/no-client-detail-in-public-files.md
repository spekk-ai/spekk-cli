---
id: no-client-detail-in-public-files
parent: sandbox-public-boundary
created: 2026-08-08T21:00:00Z
priority: 1
status: done
---

# Every Agent Prompt Forbids Carrying Real Work Between Repositories

## Description

The control-host half of this boundary was already stated. The client half was not, and it leaked: a coach prompt taught through a real advisory conversation including the client's commercial position, a release note quoted an internal remark about whether a named person had signed off, and one project's spec vocabulary had spread into five example files across docs, specs and tests.

The cause was proximity, not carelessness. An agent writing a prompt, a release note or a test fixture reaches for the nearest example, and the nearest example is the real work it just finished somewhere else. So the control belongs in the prompt, at the moment the example is chosen.

## Success Criteria

- The coach, builder and observer prompts each carry the rule, before their first working section.
- The rule is about **provenance, not classification**. It says: when writing into a repository other than the one the work came from, invent the example. It does not ask the agent to identify confidential material — an agent cannot tell from the text alone which project name is a client and which is a sample, so a rule resting on that judgement fails exactly when it is needed.
- The rule names the surfaces that carry prose across repositories: prompt, spec, release note, test fixture, commit message, PR, chat message.
- The rule states that a repository is assumed public until checked.
