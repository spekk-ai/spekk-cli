# Coach Skills Guide

The coach has specialized skills for common development tasks. Skills are markdown files that define triggers, workflows, and validation criteria.

## How Skills Work

Skills are **workflow instructions** that the coach follows:

1. **Trigger Detection** - Coach detects keywords in your input
2. **Skill Activation** - Coach reads the skill markdown file
3. **Workflow Execution** - Coach follows workflow steps using its intelligence
4. **Validation** - Coach validates output against success criteria

**Skills location:** `specs/coach-skills-system/`

## Available Skills

### 1. Meeting Notes to Specs

**Purpose:** Extract structured outputs from meeting transcripts.

**Triggers:**
- "meeting notes"
- "meeting transcript"
- "process meeting"
- "standup notes"
- "retro notes"

**CLI:** `spekk coach meeting [file]`

#### What It Extracts

**Todos** → `TODOS.md`
```markdown
## From Meeting: Sprint Planning (2024-01-15)

- [ ] Update API documentation for auth endpoints
- [ ] Review PRs for user profile feature
- [ ] Schedule security audit
```

**Specs** → `specs/`
```markdown
---
id: api-rate-limiting
created: 2024-01-15T10:30:00Z
priority: 1
status: draft
---

# API Rate Limiting

Prevent API abuse by implementing rate limits...
```

**Context** → `CONTEXT.md`
```markdown
## Architecture Decision: Use Redis for Rate Limiting (2024-01-15)

We decided to use Redis instead of in-memory storage because...
```

#### Workflow

1. **Receive transcript** (paste or file)
2. **Extract categories:**
   - Action items → todos
   - Feature requests → specs
   - Architectural decisions → context
3. **Show proposed outputs** for review
4. **Get user approval**
5. **Create/append files**
6. **Commit all changes together**

#### Example Usage

```bash
$ spekk coach meeting

Coach: I can process meeting notes. Paste your transcript:

User: [pastes meeting notes]

Coach: 
I found:
- 3 todos for TODOS.md
- 2 new specs to create
- 1 architecture decision for CONTEXT.md

Proceed? [y/N]

User: y

Coach: ✅ Created files and committed
```

---

### 2. Coordinator

**Purpose:** Create dependency-aware work plan with branch assignments.

**Triggers:**
- "plan the work"
- "dependency graph"
- "coordinate development"
- "organize branches"
- "what can we build in parallel"

**CLI:** `spekk coach coordinate`

#### What It Does

1. **Reads assertions** (draft/not_started)
2. **Analyzes dependencies** (what needs to be built first)
3. **Groups related work** into feature branches
4. **Shows dependency tree** for approval
5. **Updates YAML frontmatter** (depends-on, branch fields)
6. **Validates with parser** (catches errors)
7. **Commits changes** with summary

#### Key Concepts

**Single-Parent Dependencies**

Each assertion can depend on **at most one** other assertion:

```yaml
# Good:
depends-on: websocket-connection

# Bad (not supported):
depends-on:
  - websocket-connection
  - authentication
```

**Why?** Forces clearer thinking. If you have multiple prerequisites:
- **Sequence them:** A → B → C
- **Create junction:** "prerequisites-ready" assertion

**Branch Isolation**

Assertions on different branches should be **independent**:

```
feature/chat-system:
  - websocket-connection
  - chat-session-model
  - chat-message-input

feature/authentication:
  - password-hashing
  - session-tokens
```

These can be built **in parallel** by different developers.

#### Workflow Example

```bash
$ spekk coach coordinate

Coach: Analyzing 12 draft assertions...

Dependency tree:

feature/chat-system (4 assertions):
  websocket-connection (no dependencies)
    ↓
  chat-session-model (depends-on: websocket-connection)
    ↓
  chat-message-input (depends-on: chat-session-model)
  user-presence-tracking (no dependencies)

feature/authentication (2 assertions):
  password-hashing (no dependencies)
  session-tokens (no dependencies)

main (isolated work):
  update-button-styles (no dependencies)
  fix-header-typo (no dependencies)

Proceed with these updates? [y/N]

User: y

Coach: ✅ Updated 12 assertion files
      ✅ Committed changes

Next steps:
1. Create branches:
   git checkout -b feature/chat-system
   git checkout -b feature/authentication

2. Start building:
   git checkout feature/chat-system
   spekk builder --once
```

#### YAML Updates

**Before:**
```yaml
---
id: chat-message-input
parent: chat-system
created: 2024-01-15T10:00:00Z
priority: 2
status: not_started
---
```

**After:**
```yaml
---
id: chat-message-input
parent: chat-system
created: 2024-01-15T10:00:00Z
priority: 2
status: not_started
depends-on: chat-session-model
branch: feature/chat-system
---
```

#### Parse Don't Validate

After updating YAML, coordinator runs `parseAllSpecs()` to validate:
- ✅ Dependency IDs exist
- ✅ No circular dependencies
- ✅ Valid branch names
- ✅ Well-formed YAML

If errors are found, changes are aborted and errors shown to user.

---

### 3. Business Model Validator

**Purpose:** Assess startup/business ideas with structured questions.

**Triggers:**
- "validate business model"
- "startup validation"
- "is this viable"
- "assess this startup"

**CLI:** `spekk coach` (interactive, trigger by asking)

#### What It Does

1. **Asks structured questions** about:
   - Problem being solved
   - Target market
   - Solution approach
   - Competitive landscape
   - Business model
2. **Scores responses** across dimensions
3. **Provides quantitative health score**
4. **Identifies risks and opportunities**

#### Example Usage

```bash
$ spekk coach

User: Can you validate my business model?

Coach: I'll assess your idea through structured questions.

What problem are you solving?

User: [explains problem]

Coach: Who is your target customer?

User: [describes customer]

... [more questions]

Coach: 
Business Model Health Score: 72/100

Strengths:
- Clear problem definition (9/10)
- Well-defined target market (8/10)

Risks:
- Competitive landscape is crowded (4/10)
- Unclear path to profitability (5/10)

Recommendations:
1. Focus on differentiation...
2. Validate pricing model...
```

---

## Creating Custom Skills

Skills are markdown files in `specs/coach-skills-system/`.

### Skill Format

```markdown
---
id: my-skill
created: 2024-01-15T00:00:00Z
---

# Skill Name

Brief description.

## Triggers

- "keyword 1"
- "keyword 2"

## Workflow

1. Step one
2. Step two
3. Step three

## Validation

- Success criterion 1
- Success criterion 2

## Notes

Additional context, examples, tips.
```

### Example: Code Review Skill

```markdown
---
id: code-review-skill
created: 2024-01-15T00:00:00Z
---

# Code Review Skill

Systematic code review checklist for pull requests.

## Triggers

- "review this PR"
- "code review"
- "review this code"

## Workflow

1. Read code changes (user provides diff or file)
2. Check code quality:
   - Readability
   - Test coverage
   - Error handling
   - Performance concerns
3. Check spec alignment:
   - Does it satisfy the assertion?
   - Are tests updated?
4. Provide structured feedback:
   - Required changes
   - Suggested improvements
   - Positive observations
5. Score review (approve/request changes)

## Validation

- All code changes reviewed
- Feedback is actionable
- Spec alignment verified

## Notes

Focus on constructive feedback. Highlight what's done well.
```

Save to `specs/coach-skills-system/code-review-skill.md` and the coach will automatically detect and use it!

---

## Skill Development Tips

### Good Skills

✅ **Clear triggers** - Easy to remember keywords  
✅ **Step-by-step workflow** - Coach knows exactly what to do  
✅ **Validation criteria** - Clear success definition  
✅ **Examples** - Help users understand what to expect  

### Avoid

❌ **Vague workflows** - "Figure out what to do"  
❌ **Too many steps** - Keep it focused  
❌ **Implementation details** - Describe WHAT, not HOW  
❌ **Overlapping triggers** - Each skill should be distinct  

---

## See Also

- [Getting Started Guide](./GETTING-STARTED.md)
- [CLI Reference](./CLI-REFERENCE.md)
- [Spec Format Guide](../README.md)
