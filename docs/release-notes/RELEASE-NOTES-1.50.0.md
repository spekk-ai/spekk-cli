# Spekk CLI 1.50.0 - A Default Model Pin for New Sandboxes

This minor release adds `--model` to `spekk sandbox create`. The command writes `ANTHROPIC_MODEL` to the sandbox env file. The pin names the model that Claude Code uses on that sandbox.

## The model pin

Without `--model`, `create` writes the default for its auth mode: `ANTHROPIC_MODEL=us.anthropic.claude-opus-5-5` for `bedrock`, or `ANTHROPIC_MODEL=claude-opus-5-5` for `subscription`. `--model <name>` writes the name that you give. `--model ""` writes no `ANTHROPIC_MODEL` line, so Claude Code uses its own default. The caller's `ANTHROPIC_MODEL` does not set the sandbox pin.

## What you must do

Update the CLI to 1.50.0 to use `--model` on a new sandbox. If the Bedrock default is not valid in your AWS region, give `create` a model name for that region with `--model`.

An existing sandbox keeps its env file when you update the CLI or run `spekk sandbox deploy`. To write the mode's default pin to an existing sandbox, run `spekk sandbox provision <name> --force` with the credentials for its auth mode in your environment. This rewrites the env file and replaces any pin you set on the machine. `provision` has no `--model` flag.

## Compatibility

There is no protocol change. The protocol version stays `1.6`, and the protocol major stays `1`.
