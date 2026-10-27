# Spekk CLI 1.49.0 - The Machine Downloads the Release Binaries

This minor release changes how the deploy step puts the agent-client and the spekk CLI on a machine. The machine now downloads both binaries from the GitHub release itself. When it cannot, the CLI uses the old path: it downloads the binaries to your computer and copies them with `scp`. There are no new commands or flags, and the sandbox protocol does not change.

## Why this changed

Before this release, the deploy step downloaded both binaries (approximately 40 MB) to the operator's computer and copied them to the machine with one `scp`. On some operator networks, that large transfer failed, and `spekk sandbox create` stopped at `copying binary` each time, while each small step before it passed. Now only commands cross the operator's connection when the machine can download the release.

## The machine downloads and checks the binaries

The deploy step that `spekk sandbox create`, `spekk sandbox provision`, `spekk sandbox deploy`, and machine mode (`spekk sandbox create --machine`) share does these steps:

1. The CLI reads the expected sha256 of each binary from the GitHub release API (`GET /repos/spekk-ai/spekk-cli/releases/tags/<tag>`, field `digest`), as before. An asset with no digest stops the deploy.
2. Over SSH, as the login user, the machine downloads `sandbox-linux-<arch>` and `spekk-linux-<arch>` from `https://github.com/spekk-ai/spekk-cli/releases/download/<tag>/` into the staged paths in the login home. It uses `curl`. When `curl` is not installed, it uses `wget`.
3. The machine checks each file with `sha256sum -c` against the expected sha256. A failed download or a mismatch removes both staged files.
4. The install step checks the staged files against the release sha256 again before it replaces anything, as before.

## Time limits

- `curl` runs with `--connect-timeout 10 --speed-limit 1024 --speed-time 30 --max-time 300 --retry 2`. It stops an attempt when the transfer is below 1024 bytes per second for 30 seconds, or when the attempt runs for more than 300 seconds. These limits apply to each attempt, and `--retry 2` gives 3 attempts. Thus a slow transfer can take approximately 15 minutes for each file before the fallback starts, and approximately 30 minutes for the two files.
- `wget` runs as `timeout 300 wget -q -T 10 --tries=2`. The `timeout 300` limit applies to each file, for all tries.

## The fallback

When the download on the machine fails for any cause, the CLI prints this message and continues:

```text
spekk: the machine could not download release <tag> (<cause>); copying it from this computer
```

The cause is the last line of the output of the download command, for example `neither curl nor wget is installed` or `sandbox-linux-amd64 does not match the release sha256`. Then the CLI downloads both binaries to your computer and copies them to the machine with `scp`, as in 1.48.1. When that `scp` fails, the deploy stops with `copying binary`, as before.

## What you must do

Update spekk to 1.49.0. You do not change a configuration value. The next `create`, `provision`, or `deploy` uses the new download.

## Compatibility

There is no protocol change. The protocol version stays `1.6`, and the protocol major stays `1`. An agent at 1.49.0 and a control host at any `1.x` version connect as before.

A machine where the download fails uses the fallback. Examples are a machine with no `curl` and no `wget`, a machine with `wget` but no `timeout` command, and a BusyBox `wget` that cannot complete the download. In a test on Ubuntu, BusyBox 1.36.1 `wget` accepted the options but failed the TLS connection to the release download, so the fallback ran.
