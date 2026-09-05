---
icon: lucide/settings
---

# Configuration

Spekk has no configuration file of its own. You change its behavior with files in two directories, and with environment variables.

| What | Where |
|------|-------|
| Prompt extensions and overrides | `.spekk/` in the project, and `~/.config/spekk/` |
| Skills | `.spekk/skills/<agent>/` and `~/.config/spekk/skills/<agent>/` |
| Observation suppressions | `.spekk/dont-flag.yaml`, committed on `main` |
| Sandbox records, SSH keys, and known hosts | `~/.config/spekk/sandboxes.json`, `~/.config/spekk/keys/`, `~/.config/spekk/known_hosts/` |
| Derived files spekk writes | `.spekk/index.db`, `.spekk/index.html`, the observer logs and lock files under `.spekk/` |

The global directory follows the XDG rule: `$XDG_CONFIG_HOME/spekk` when `XDG_CONFIG_HOME` is set, `~/.config/spekk` otherwise. When spekk finds the old `~/.spekk` directory and no new one, it moves the old directory to the new path. In a terminal it asks you to press Enter first. In a pipe or a script it moves the directory and prints a notice on stderr.

## Prompt customization

Spekk layers the agent prompts (coach, builder, observer). You can extend or override a prompt at two levels.

### Extend or override

The file name says what the file does:

| Pattern | Behavior |
|---------|----------|
| `<agent>.prompt.md` | **Extends** the base prompt. Spekk appends your content after the built-in prompt. |
| `<agent>.prompt.override.md` | **Overrides** the base prompt. Your content replaces the built-in prompt. |

Extend files still apply after an override.

### Where the files go

| Location | Scope |
|----------|-------|
| `~/.config/spekk/` | **Global**: every project |
| `.spekk/` at the project root | **Local**: this project only |

A local file wins over a global file.

### Example: extend the builder for every project

Create `~/.config/spekk/builder.prompt.md`:

```markdown
## Company Standards

- Use TypeScript strict mode
- All functions must have JSDoc comments
- No console.log in production code
```

Spekk appends this to the base builder prompt in every project.

### Example: override the coach for one project

Create `.spekk/coach.prompt.override.md`:

```markdown
# Custom Coach Agent

You are a coach agent for a Django/HTMX project.
When creating specs, follow Django conventions and
reference the project's existing app structure.
```

This replaces the base coach prompt in this project. Extend files still apply after it.

### Resolution order

```
1. Base prompt (built into the binary)
   overridden by
2. Global override (~/.config/spekk/<agent>.prompt.override.md)
   overridden by
3. Local override (.spekk/<agent>.prompt.override.md)
   extended by
4. Global extend (~/.config/spekk/<agent>.prompt.md)
   extended by
5. Local extend (.spekk/<agent>.prompt.md)
```

`spekk prompt <agent>` prints the result.

### Version control

Commit `.spekk/` when the team shares the same customizations. Add it to `.gitignore` when each person keeps their own. `.spekk/index.db` and `.spekk/index.html` are derived files; spekk adds the index to `.gitignore` for you. `.spekk/dont-flag.yaml` must be committed, because a suppression takes effect only from `main`.

## Skills

Skills follow the same layers as prompts. See [Skills](coach-skills.md) for the file format, and [`spekk install <agent> <skill>`](cli-reference.md#spekk-install-agent-skill) for the registry.

| Layer | Path |
|-------|------|
| Local | `.spekk/skills/<agent>/*.md` |
| Global | `~/.config/spekk/skills/<agent>/*.md` |
| Package | The skills in the binary |

### Skill registry

`spekk install <agent> <skill>` fetches from [`github.com/spekk-ai/spekk-skills`](https://github.com/spekk-ai/spekk-skills). Two variables point it at a mirror:

| Variable | Default | Description |
|----------|---------|-------------|
| `SPEKK_SKILLS_RAW_BASE` | `https://raw.githubusercontent.com/spekk-ai/spekk-skills/main` | Base URL for the raw skill files. The skill is fetched from `<base>/<agent>/<skill>.md` |
| `SPEKK_SKILLS_API_BASE` | `https://api.github.com/repos/spekk-ai/spekk-skills/contents` | Base URL for the directory listing that `--list <agent>` reads |

## Project structure

Spekk expects the specs in a `specs/` directory at the project root:

```
your-project/
├── specs/
│   ├── README.md               # Written by spekk init
│   ├── feature-name/
│   │   ├── feature-name.md
│   │   └── assertions/
│   │       └── assertion-name.md
│   └── ...
├── .spekk/                     # Local customizations and derived files
│   ├── builder.prompt.md       # Extend the builder prompt
│   ├── coach.prompt.md         # Extend the coach prompt
│   ├── dont-flag.yaml          # Observation suppressions
│   ├── index.db                # The SQLite index (gitignored)
│   └── skills/                 # Custom skills
│       ├── coach/
│       │   └── my-skill.md
│       └── builder/
│           └── my-skill.md
├── observations/               # Observer findings, on observer/<slug> branches
├── TODOS.md                    # Action items, from the meeting skill
└── CONTEXT.md                  # Decisions, from the meeting skill
```

Every command finds `specs/` at the git root, from any directory in the repository. Outside a repository it uses the working directory. `--specs-dir` on `next`, `list`, `validate`, and `index` reads another directory.

## Suppressing observations

`.spekk/dont-flag.yaml` lists drift that the observer must not report. The observer runs `spekk observer scan-check` before it files an observation, and a match there means nothing is filed: no observation, no branch, no index row, no announcement. The file is read from `main` (or `master`) through git, not from the working tree, so a suppression takes effect when it is committed on the main branch. That is on purpose: every suppression goes through a reviewed change, and each one carries a reason and an author.

```yaml
# .spekk/dont-flag.yaml
- match: docs/generated/**
  reason: Generated files. The generator is the source of truth.
  by: alice
- match: legacy-report-endpoint
  reason: Scheduled for removal in Q4. Do not spec it.
  by: bob
  until: 2026-12-31
```

Each entry has these fields, and no others:

| Field | Required | Description |
|-------|----------|-------------|
| `match` | Yes | A glob, matched against each evidence path of the would-be observation, and against its slug |
| `reason` | Yes | Why the drift is not a finding |
| `by` | Yes | Who added the entry |
| `until` | No | `YYYY-MM-DD`. The entry suppresses through that day (UTC) and then expires. Absent means permanent |

A path matches as written, and also after normalization, so `./parser.go` and `parser.go:42` do not defeat a pattern for `parser.go`. The pattern itself is never rewritten. An unknown field, a missing required field, a bad date, or a malformed glob is an error, and `scan-check` fails with a message that names the entry. A broken suppression file is never read as empty, because that would report every suppressed finding again.

There is no flag, environment variable, or prompt instruction that suppresses drift another way.

## Environment variables

### Every command

| Variable | Description |
|----------|-------------|
| `XDG_CONFIG_HOME` | When set, the global directory is `$XDG_CONFIG_HOME/spekk` instead of `~/.config/spekk` |
| `CI` | When set, `spekk show --watch` does not open a browser |

### The install script

| Variable | Default | Description |
|----------|---------|-------------|
| `SPEKK_INSTALL_DIR` | `~/.local/bin` | Where `install.sh` puts the binary |
| `SPEKK_VERSION` | `latest` | The release tag to install, for example `v1.28.0` |

### Skill registry

`SPEKK_SKILLS_RAW_BASE` and `SPEKK_SKILLS_API_BASE`. See [Skill registry](#skill-registry).

### Inside a sandbox session

| Variable | Description |
|----------|-------------|
| `SPEKK_CONVERSATION_SPOOL` | The spool directory for `spekk conversation open` and `spekk observer announce`. The agent client sets it for each Claude session. Outside a sandbox session it is not set, and both commands fail with a message that says so |

### Sandbox provisioning

`spekk sandbox create`, `spekk sandbox provision`, and `spekk sandbox deploy` read these variables on your machine.

**Choose an auth mode first.** `spekk sandbox create --auth <mode>` decides how the sandbox authenticates Claude, and the mode decides which credentials you need:

- `bedrock`, the default. Claude usage bills through the AWS Bedrock API.
- `subscription`. The agent authenticates with a Claude subscription token.

| Variable | Needed for | Description |
|----------|------------|-------------|
| `GITHUB_TOKEN` | every sandbox | Downloads the agent binary and the cloud-init template from the spekk release. On the sandbox, it gives the agent access to your repositories |
| `SPEKK_HOST` | every sandbox | The control host the agent connects to. A scheme and a trailing slash are removed |
| `DO_API_TOKEN` | `--provider digitalocean` | DigitalOcean API token. `DIGITALOCEAN_TOKEN` is accepted too. Not needed for a machine you already have |
| `AWS_ACCESS_KEY_ID` | `bedrock` | AWS credential for the Bedrock API |
| `AWS_SECRET_ACCESS_KEY` | `bedrock` | AWS credential for the Bedrock API |
| `AWS_DEFAULT_REGION` | `bedrock` | AWS region, for example `us-east-1` |
| `CLAUDE_CODE_OAUTH_TOKEN` | `subscription` | Claude subscription token. Mint it with `claude setup-token`, which needs a Claude subscription |

`spekk sandbox create` refuses to start when a variable its mode needs is missing, and it names every missing variable at once, before it creates anything billable. `spekk sandbox provision` makes the same check, for the mode the sandbox was created with, before it touches the machine. `spekk sandbox destroy` for a DigitalOcean sandbox needs `DO_API_TOKEN`. `spekk sandbox status` prints the stored fields without it.

**A model pin belongs to its mode.** `ANTHROPIC_MODEL` names a model for the API the sandbox authenticates against, and the names differ. A Bedrock sandbox pins an inference profile such as `us.anthropic.claude-sonnet-5`, which a subscription rejects. When `infrastructure/sandbox/setup-credentials.sh` moves a sandbox between modes, it drops the pin, reports what it dropped, and writes a new pin only when you supply one for the new mode.

#### Minting a subscription token

`claude setup-token` mints the long-lived token. It needs a Claude subscription. It runs on any machine with the Claude Code CLI. The sandbox is not involved, and the token is the only thing that reaches it.

The command opens a browser. When it cannot, on a headless machine, in an SSH session, or in a terminal you drive from a phone, it prints a URL and waits:

```
Browser didn't open? Use the url below to sign in
https://claude.com/cai/oauth/authorize?...
Paste code here if prompted >
```

Open the URL on any device, authorize, and paste the code it returns at the prompt. The URL carries a PKCE challenge, not a secret, so you can relay it. The code is single-use and short-lived. The token it produces is neither, and the token is the value to protect.

Three things to know when you automate the prompt instead of typing at it:

- Send the code and the Enter as two separate writes. A carriage return in the same write as the pasted code is treated as part of the paste, and never submits.
- A restart of the command generates a new challenge, which invalidates the code from the previous run.
- **Do not scrape the token out of the terminal. Verify it before you deploy it.** The command prints the token through an interface that redraws, so the byte stream is not the text on the screen: it contains cursor movements that overwrite what came before. Removing the escape sequences does not rebuild the result, because a character that was drawn and then overwritten is still in the stream. A terminal emulator that replays the stream rebuilds it; a regular expression does not.

The failure this produces is silent. A token with one character missing is still about 100 characters long, still carries the right prefix, and still looks like a token. It fails at first use, with `401 OAuth access token is invalid`, far from the step that produced it. A wider terminal helps but does not make scraping correct.

So verify the captured value by using it, before it reaches a sandbox:

```bash
CLAUDE_CODE_OAUTH_TOKEN="$(cat path/to/token)" claude -p 'reply with OK'
```

A token that authenticates on your machine authenticates on the sandbox. One that does not was captured wrong, and no inspection of it shows that.

Keep the token off command lines and out of shell history. Write it to a file only you can read, and let the environment pick it up from there:

```bash
umask 077 && printf '%s' 'TOKEN' > ~/.config/spekk/oauth-token
export CLAUDE_CODE_OAUTH_TOKEN="$(cat ~/.config/spekk/oauth-token)"
```

An assignment on the local side of an `ssh` command is not forwarded, and a secret written inline is visible in both machines' process lists. To credential a remote sandbox unattended, put the value in a root-only file on that machine and source it there. `infrastructure/sandbox/setup-credentials.sh` documents the form.

> **A subscription's rate limit is shared, and it runs out for everyone at once.** Every session authenticated with the same subscription draws on one quota: each sandbox that uses the token, and the interactive sessions of the person whose subscription it is. Several busy sandboxes contend with each other, and when the quota is spent they all stall until the window resets. Bedrock bills per token and has no such ceiling. Weigh that before you move a sandbox whose work has to finish on demand, and remember that a subscription is one person's seat, not a team credential.

### Agent runtime

The agent binary on the sandbox reads these variables from `/etc/spekk/agent.env`. `spekk sandbox create` and `spekk sandbox provision` write that file. You do not set them by hand.

| Variable | Required | Description |
|----------|----------|-------------|
| `SPEKK_AGENT_TOKEN` | Yes | Bearer token for the WebSocket connection to the control host. `create` and `provision` generate it and print it for you to register |
| `SPEKK_HOST` | Yes | Control host the agent connects to |
| `WORKSPACE` | No | Working directory for Claude sessions (default: `/opt/spekk/workspace`) |
| `SPEKK_AGENT_NAME` | No | `spekk-<name>`. The agent binary does not read it; `setup-credentials.sh` uses it as the git author name |
| `GITHUB_TOKEN` | Yes | Access to your repositories, for Claude Code on the sandbox |
| `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_DEFAULT_REGION` | `bedrock` sandboxes | AWS credentials Claude Code uses to reach the Bedrock API |
| `CLAUDE_CODE_USE_BEDROCK` | `bedrock` sandboxes | Set to `1`, which routes Claude Code to Bedrock |
| `CLAUDE_CODE_OAUTH_TOKEN` | `subscription` sandboxes | Claude subscription token |

The file carries the variables of one mode only. Claude Code reads whichever of these it finds, so a file with both sets would leave the choice to chance. That is why a change of mode rewrites the file instead of adding to it.

The agent passes its environment to the `claude` child process and reads only `SPEKK_AGENT_TOKEN`, `SPEKK_HOST`, and `WORKSPACE` itself. For the full agent architecture, see [Sandbox Architecture](./advanced/sandbox-architecture.md).
