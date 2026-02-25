---
id: coach-skills-system
created: 2026-01-23T22:14:00Z
priority: 1
status: not_started
---

# Coach Skills System

Extends the coach agent with specialized skills that can be automatically suggested and applied during user conversations.

## Overview

The coach agent should be able to:
- Automatically detect when specialized skills would be helpful
- Suggest and apply relevant skills during conversations
- Maintain a lean, extensible framework for adding new skills
- Provide structured outputs when using skills

## Skills Available

### Business Model Validator
Helps validate startup/business models through structured questioning and scoring across:
- Industry background & expertise
- Product validation
- Market demand
- Business goals & strategy
- Traction & momentum
- Hypothesis prioritization

Produces quantitative health score and actionable recommendations.

### Meeting Notes to Specs
Processes meeting transcripts and extracts actionable outcomes. Activated via `spekk coach meeting`. Categorizes content into:
- **Todos** - action items and follow-ups → TODOS.md
- **Specs** - features and product changes → spec files in specs/
- **Context** - architectural decisions → CONTEXT.md

See `specs/meeting-notes-to-specs/` for full spec.

## Success Criteria

- Coach automatically suggests skills when appropriate contexts arise
- Business model validation skill works end-to-end
- Framework supports adding future skills without major changes
- All skill interactions produce structured, useful outputs
