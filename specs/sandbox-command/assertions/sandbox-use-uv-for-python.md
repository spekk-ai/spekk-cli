---
id: sandbox-use-uv-for-python
parent: sandbox-command
created: 2026-03-12T20:00:00Z
priority: 1
status: done
branch: feature/sandbox-command
---

# Sandbox Uses uv for Python Package Management

## Requirement

Ubuntu 24.04 marks its Python as externally managed (PEP 668), so both `pip install` and `uv pip install --system` fail. Use `uv` with a venv at `/opt/spekk/.venv` instead.

## Success Criteria

- `cloud-init.yaml` installs `uv` system-wide: `curl -LsSf https://astral.sh/uv/install.sh | env UV_INSTALL_DIR=/usr/local/bin sh` (installs to `/usr/local/bin` so it's in PATH for all users)
- `cloud-init.yaml` creates a venv: `uv venv /opt/spekk/.venv`
- `cloud-init.yaml` installs websockets into the venv: `uv pip install --python /opt/spekk/.venv/bin/python websockets`
- `cloud-init.yaml` no longer lists `python3-pip` in the `packages` section (replace with `python3-venv`)
- `cloud-init.yaml` systemd unit uses `ExecStart=/opt/spekk/.venv/bin/python /opt/spekk/agent-client.py` instead of `/usr/bin/python3`
- `src/sandbox/create.js` `deployAgentClient` function uses `uv pip install --python /opt/spekk/.venv/bin/python websockets`
- `src/sandbox/deploy.js` uses `uv pip install --python /opt/spekk/.venv/bin/python --upgrade websockets`
- No `pip` or `pip3` commands remain in any sandbox-related files
- No `--system` flag is used with `uv pip install`
