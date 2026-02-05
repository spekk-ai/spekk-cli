## Installing `@spekk/cli` from Gemfury

### 1. Get a Gemfury token

Ask a team lead for a Gemfury **read** (or full access) token for the `thinknimble` org.

### 2. Add the token to your shell

**zsh** — add to `~/.zshrc`:

```bash
export GEMFURY_SPEKK_TOKEN=your_token_here
```

**bash** — add to `~/.bashrc`:

```bash
export GEMFURY_SPEKK_TOKEN=your_token_here
```

Then reload your shell:

```bash
# zsh
source ~/.zshrc

# bash
source ~/.bashrc
```

### 3. Configure npm

Add these two lines to your **global** `~/.npmrc`:

```
@spekk:registry=https://npm.fury.io/thinknimble/
//npm.fury.io/thinknimble/:_authToken=${GEMFURY_SPEKK_TOKEN}
```

You can do this manually or run:

```bash
npm config set @spekk:registry https://npm.fury.io/thinknimble/
npm config set //npm.fury.io/thinknimble/:_authToken "$GEMFURY_SPEKK_TOKEN"
```

### 4. Install

```bash
npm install -g @spekk/cli
```

### 5. Verify

```bash
spekk --help
```

### Updating

```bash
npm update -g @spekk/cli
```
