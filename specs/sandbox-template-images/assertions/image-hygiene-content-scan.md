---
id: image-hygiene-content-scan
parent: sandbox-template-images
created: 2026-10-18T04:00:00Z
priority: 1
status: done
depends-on: image-hygiene-broad-patterns
branch: feature/sandbox-template-images
---

# The Hygiene Check Finds a Secret by Its Shape, in Each Home and Data Directory

**Tests:** internal/sandbox/image_test.go

## Success Criteria

- "Each home" is `/root`, each directory in `/home`, and the home (field 6) of each user in `/etc/passwd` that is a directory, without duplicates. `/`, `/dev`, `/proc`, `/sys`, and `/run` are not homes. The clean step and the file checks of `image-hygiene-broad-patterns` use this list.
- The `.ssh` check also searches a `.ssh` that is a symlink to a directory.
- The file checks also fail on `.config/doctl/config.yaml` (the DigitalOcean CLI token) in each home.
- "The content roots" are each home, `/opt`, `/srv`, and each directory in `/var/lib` except `/var/lib/docker` and `/var/lib/containerd` (they keep the pulled container images). The check reads only text files (`grep -rlIs`).
- The check fails the build on each text file in the content roots that has a private key header, `-----BEGIN ([A-Z0-9]+ )?PRIVATE KEY-----`, at the end of a line or followed by an escaped newline `\n` and the key (a key in a JSON file, for example a GCP service account key). Code that only names the header (`b"-----BEGIN ... PRIVATE KEY-----"`) is not a finding. The key check does not search `node_modules`, `site-packages`, or `dist-packages`, because installed packages often ship test keys.
- The check fails the build on each text file in the content roots or in `/etc` that has a token in a known format: `gh[pousr]_[A-Za-z0-9]{36}`, `github_pat_[A-Za-z0-9_]{22,}`, `sk-ant-[A-Za-z0-9_-]{20,}`, `AKIA[0-9A-Z]{16}`, `dop_v1_[a-f0-9]{64}`, `glpat-[A-Za-z0-9_-]{20}`, or `xox[abprs]-[A-Za-z0-9-]{10,}`. An AWS key that ends in `EXAMPLE` (the key in the AWS documentation) is not a finding.
- The remote URL check reads each git config: each `*.git/config` file in the content roots (a clone or a bare repository), `.gitconfig` and `.config/git/config` in each home, and `/etc/gitconfig`. In the same files, any `extraheader` line fails the build.
- The secret variable check also reads `.bash_profile`, `.bash_login`, `.bash_aliases`, `.zshrc`, `.zprofile`, and `.zshenv` in each home, `/etc/bash.bashrc`, each `.env` file in the content roots, and the `Environment=` lines of the files in `/etc/systemd/system`.
- The content scans add at most about 30 seconds to the build of a normal image.

## Verification

- `TestBuildImage` has one row for each piece of logic, not one row for each file name: a home from `/etc/passwd` outside `/home`, a `.ssh` that is a symlink, one token-format match, a private key in a JSON file, and an `extraheader`. Each row fails on the code before this assertion. The clean builder has code that names a key header, a test key in `node_modules`, and the AWS documentation key, and still becomes an image.
- A walkthrough builds one image for each planted secret of the last review with the fake cloud: each plant fails the build or is not in the image, and the normal setup still makes an image.
