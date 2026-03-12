---
id: sandbox-use-uv-for-python
parent: sandbox-command
created: 2026-03-12T20:00:00Z
priority: 1
status: not_started
branch: feature/sandbox-command
---

# Sandbox Uses uv for Python Package Management

## Requirement

Ubuntu 24.04 marks its Python as externally managed (PEP 668), so `pip install` fails without `--break-system-packages`. Instead of working around pip, use `uv` — it's faster, avoids the managed-environment issue, and is a standard tool on this project.

## Success Criteria

- `cloud-init.yaml` installs `uv` system-wide during provisioning (via the official installer: `curl -LsSf https://astral.sh/uv/install.sh | sh`)
- `cloud-init.yaml` uses `uv pip install --system websockets` instead of `pip3 install websockets`
- `cloud-init.yaml` no longer lists `python3-pip` in the `packages` section (replace with `python3-venv` which uv needs)
- `src/sandbox/create.js` `deployAgentClient` function uses `uv pip install --system websockets` instead of `pip3 install websockets`
- `src/sandbox/deploy.js` uses `uv pip install --system --upgrade websockets` instead of `pip3 install --upgrade websockets`
- No `pip` or `pip3` commands remain in any sandbox-related files
