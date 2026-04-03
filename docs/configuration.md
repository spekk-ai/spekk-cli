---
icon: lucide/settings
---

# Configuration

Customize Spekk's agent behavior without modifying the package.

## Prompt customization

Spekk uses a layered prompt system for its agents (coach, builder, observer). You can extend or override prompts at two levels.

### Extend vs override

The file naming convention determines how customization is applied:

| Pattern | Behavior |
|---------|----------|
| `<agent>.prompt.md` | **Extends** the base prompt. Your content is appended after the built-in prompt. |
| `<agent>.prompt.override.md` | **Overrides** the base prompt entirely. Your content replaces the built-in prompt. |

Extend files still layer on top of an override.

### Where to put customization files

| Location | Scope |
|----------|-------|
| `~/.spekk/` | **Global** -- applies to all your projects |
| `.spekk/` (project root) | **Local** -- applies to this project only |

Local files take precedence over global files.

### Example: extend the builder globally

Create `~/.spekk/builder.prompt.md`:

```markdown
## Company Standards

- Use TypeScript strict mode
- All functions must have JSDoc comments
- No console.log in production code
```

This is appended to the base builder prompt for every project.

### Example: override the coach for a project

Create `.spekk/coach.prompt.override.md`:

```markdown
# Custom Coach Agent

You are a coach agent for a Django/HTMX project.
When creating specs, follow Django conventions and
reference the project's existing app structure.
```

This completely replaces the base coach prompt for this project. Any extend files are still appended after the override.

### Resolution order

```
1. Base prompt (built into package)
   ↓ overridden by
2. Global override (~/.spekk/<agent>.prompt.override.md)
   ↓ overridden by
3. Local override (.spekk/<agent>.prompt.override.md)
   ↓ extended by
4. Global extend (~/.spekk/<agent>.prompt.md)
   ↓ extended by
5. Local extend (.spekk/<agent>.prompt.md)
```

### Version control

The `.spekk/` directory can be:

- **Committed** to your repo so the whole team shares customizations
- **Gitignored** if you prefer individual configuration

Choose whichever fits your team.

---

## Environment variables

### `GEMFURY_SPEKK_TOKEN`

Authentication token for the private npm registry.

```bash
export GEMFURY_SPEKK_TOKEN=your_token_here
```

Required for `npm install -g @spekk/cli`.

---

## Project structure

Spekk expects specs in a `specs/` directory at the project root:

```
your-project/
├── specs/
│   ├── feature-name/
│   │   ├── feature-name.md
│   │   └── assertions/
│   │       └── assertion-name.md
│   └── ...
├── .spekk/                 # Local prompt customizations
│   ├── builder.prompt.md
│   └── coach.prompt.md
├── TODOS.md                # Meeting-extracted action items
└── CONTEXT.md              # Architecture decisions
```

The spec directory structure is detected automatically -- no configuration needed.
