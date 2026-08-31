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
| `~/.config/spekk/` | **Global** -- applies to all your projects |
| `.spekk/` (project root) | **Local** -- applies to this project only |

Local files take precedence over global files.

The global directory follows the XDG Base Directory spec: `$XDG_CONFIG_HOME/spekk` when `XDG_CONFIG_HOME` is set, `~/.config/spekk` otherwise.

### Example: extend the builder globally

Create `~/.config/spekk/builder.prompt.md`:

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
2. Global override (~/.config/spekk/<agent>.prompt.override.md)
   ↓ overridden by
3. Local override (.spekk/<agent>.prompt.override.md)
   ↓ extended by
4. Global extend (~/.config/spekk/<agent>.prompt.md)
   ↓ extended by
5. Local extend (.spekk/<agent>.prompt.md)
```

### Version control

The `.spekk/` directory can be:

- **Committed** to your repo so the whole team shares customizations
- **Gitignored** if you prefer individual configuration

Choose whichever fits your team.

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
├── .spekk/                 # Local customizations
│   ├── builder.prompt.md   # Extend builder prompt
│   ├── coach.prompt.md     # Extend coach prompt
│   └── skills/             # Custom skills
│       ├── coach/
│       │   └── my-skill.md
│       └── builder/
│           └── my-skill.md
├── TODOS.md                # Meeting-extracted action items
└── CONTEXT.md              # Architecture decisions
```

The spec directory structure is detected automatically -- no configuration needed.

---

## Environment variables

### Sandbox provisioning

These variables are used by `spekk sandbox create` and other provisioning commands, run on your local machine.

**Choose an auth mode first.** `spekk sandbox create --auth <mode>` decides how the sandbox authenticates Claude, and the mode decides which credentials you need:

- `bedrock` — the default. Claude usage bills through the AWS Bedrock API.
- `subscription` — the agent authenticates with a Claude subscription token instead.

| Variable | Mode | Description |
|----------|------|-------------|
| `DO_API_TOKEN` | both | DigitalOcean API token for provisioning droplets |
| `GITHUB_TOKEN` | both | GitHub token for agent access to repositories |
| `SPEKK_HOST` | both | Control host hostname for sandbox registration |
| `AWS_ACCESS_KEY_ID` | bedrock | AWS credentials for the Bedrock API |
| `AWS_SECRET_ACCESS_KEY` | bedrock | AWS credentials for the Bedrock API |
| `AWS_DEFAULT_REGION` | bedrock | AWS region (e.g. `us-east-1`) |
| `CLAUDE_CODE_OAUTH_TOKEN` | subscription | Claude subscription token. Mint it with `claude setup-token`, which needs a Claude subscription. |

**A model pin belongs to its mode.** `ANTHROPIC_MODEL` names a model for whichever API the sandbox authenticates against, and the names differ: a Bedrock sandbox pins an inference profile such as `us.anthropic.claude-sonnet-5`, which a subscription rejects outright. Moving a sandbox between modes therefore drops any pin it had, reports what it dropped, and writes a replacement only if you supply one for the mode you are moving to.

`spekk sandbox create` refuses to start when a variable its mode needs is missing, and it names every missing one at once. It checks before it creates anything billable.

#### Minting a subscription token

`claude setup-token` mints the long-lived token and needs a Claude subscription. It runs on any machine with the Claude Code CLI; the sandbox itself is never involved, and the token is the only thing that reaches it.

The command opens a browser. When it cannot — a headless box, an SSH session, a terminal you are driving from a phone — it prints a URL and waits:

```
Browser didn't open? Use the url below to sign in
https://claude.com/cai/oauth/authorize?...
Paste code here if prompted >
```

Open that URL on any device, authorize, and paste the code it returns back at the prompt. The URL carries a PKCE challenge rather than a secret, so relaying it is safe; the code is single-use and short-lived. The token it produces is neither, and is the value to protect.

Three things to know if you automate the prompt rather than type at it.

Send the code and the Enter as **separate** writes. A carriage return in the same write as the pasted code is treated as part of the paste and never submits.

Restarting the command generates a fresh challenge, which invalidates any code you were issued for the previous run.

**Do not scrape the token out of the terminal. Verify it before you deploy it.** The command prints the token through a redrawing interface, so the byte stream is not the text on screen: it contains cursor movements that overwrite what came before. Stripping escape sequences does not reconstruct the result, because a character that was drawn and then overwritten is still in the stream. Replaying the stream through a terminal emulator does; a regular expression over it does not, however careful the pattern.

The failure this produces is quiet. A token missing a single character is still 100-odd characters long, still carries the right prefix, and still looks exactly like a token — and it fails only at first use, with `401 OAuth access token is invalid`, far from the step that produced it. Widening the terminal helps but does not make scraping correct.

So verify the captured value by using it, before it reaches a sandbox:

```bash
CLAUDE_CODE_OAUTH_TOKEN="$(cat path/to/token)" claude -p 'reply with OK'
```

A token that authenticates locally will authenticate on the sandbox. One that does not has been captured wrong, and no amount of inspecting it will show that.

Keep the token off command lines and out of shell history. Write it to a file only you can read, and let the environment pick it up from there:

```bash
umask 077 && printf '%s' 'TOKEN' > ~/.config/spekk/oauth-token
export CLAUDE_CODE_OAUTH_TOKEN="$(cat ~/.config/spekk/oauth-token)"
```

An assignment on the local side of an `ssh` command is not forwarded, and a secret written inline is visible in both machines' process lists. To credential a remote sandbox unattended, put the value in a root-only file on that machine and source it there — `infrastructure/sandbox/setup-credentials.sh` documents the form.

> **A subscription's rate limit is shared, and it runs out for everyone at once.** Every session authenticated with the same subscription draws on one quota: each sandbox using that token, and the interactive sessions of the person whose subscription it is. Several busy sandboxes contend with each other, and when the quota is spent they all stall together until the window resets. Bedrock bills per token and has no such ceiling. Weigh that before you move a sandbox whose work has to finish on demand, and remember that a subscription is one person's seat rather than a team credential.

### Agent runtime

These variables are read by the agent binary on the sandbox VM (typically from `/etc/spekk/agent.env`). They are injected during provisioning — you don't set them manually.

| Variable | Required | Description |
|----------|----------|-------------|
| `SPEKK_AGENT_TOKEN` | Yes | Bearer token for authenticating the WebSocket connection to the control host |
| `SPEKK_HOST` | Yes | Control host hostname the agent connects to |
| `WORKSPACE` | No | Working directory for Claude sessions (default: `/opt/spekk/workspace`) |
| `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_DEFAULT_REGION` | bedrock sandboxes | AWS credentials Claude Code uses to reach the Bedrock API |
| `CLAUDE_CODE_USE_BEDROCK` | bedrock sandboxes | Set to `1`, which routes Claude Code at Bedrock |
| `CLAUDE_CODE_OAUTH_TOKEN` | subscription sandboxes | Claude subscription token |

The last three rows are written by whichever auth mode provisioned the sandbox, and only that mode's rows appear in the file. Claude Code reads whichever of these it finds, so a file carrying both sets would leave the choice to chance — which is why switching a sandbox's mode rewrites the file rather than adding to it.

For the full agent architecture, see [Sandbox Architecture](./advanced/sandbox-architecture.md).
