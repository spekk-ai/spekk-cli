#!/usr/bin/env python3
"""Set the agent's commit and pull request attribution policy."""

import json
import os
import pathlib
import subprocess
import sys
import tempfile


def replace_file(path: pathlib.Path, content: bytes, mode: int) -> None:
    with tempfile.NamedTemporaryFile(dir=path.parent, delete=False) as staged:
        staged.write(content)
        staged_path = pathlib.Path(staged.name)
    try:
        staged_path.chmod(mode)
        staged_path.replace(path)
    finally:
        staged_path.unlink(missing_ok=True)


def configure(home: pathlib.Path, assisted_by: pathlib.Path) -> None:
    settings_path = home / ".claude" / "settings.json"
    settings_path.parent.mkdir(mode=0o700, exist_ok=True)
    if settings_path.exists():
        settings = json.loads(settings_path.read_text(encoding="utf-8"))
        if not isinstance(settings, dict):
            raise ValueError("Claude settings must be a JSON object")
    else:
        settings = {}

    attribution = settings.setdefault("attribution", {})
    if not isinstance(attribution, dict):
        raise ValueError("Claude attribution settings must be a JSON object")
    attribution.update({"commit": "", "pr": ""})
    replace_file(settings_path, (json.dumps(settings, indent=2) + "\n").encode(), 0o600)

    bin_dir = home / ".local" / "bin"
    bin_dir.mkdir(mode=0o755, parents=True, exist_ok=True)
    replace_file(bin_dir / "assisted-by", assisted_by.read_bytes(), 0o755)
    subprocess.run(
        ["git", "config", "--global", "trailer.assisted-by.ifExists", "addIfDifferent"],
        check=True,
        env={**os.environ, "HOME": str(home)},
    )


if __name__ == "__main__":
    if len(sys.argv) != 2:
        sys.exit("usage: configure-agent-attribution.py <assisted-by-file>")
    configure(pathlib.Path.home(), pathlib.Path(sys.argv[1]))
