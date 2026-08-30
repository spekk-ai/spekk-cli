#!/usr/bin/env bash
#
# setup-credentials.sh — Inject secrets into a freshly provisioned sandbox
# droplet, or move an existing one between auth modes.
#
# Copy the script to the droplet and run it there, so its prompts reach your
# terminal:
#
#   scp setup-credentials.sh root@DROPLET_IP:/tmp/
#   ssh -t root@DROPLET_IP 'SPEKK_AUTH_MODE=subscription bash /tmp/setup-credentials.sh'
#
# Any value can come from the environment instead of a prompt, which is how an
# unattended run avoids stopping for input. Set it on the remote side of the ssh
# command, inside the quotes, as above: an assignment on the local side applies
# to ssh itself, and ssh does not forward it.
#
# Pass NON-SECRETS that way -- SPEKK_AUTH_MODE, SPEKK_HOST, AWS_DEFAULT_REGION.
# A secret on a command line is in the local shell history and in both machines'
# process lists, which is the exposure the non-echoing prompts exist to avoid.
# For an unattended run, feed secrets through a root-only file on the droplet
# and source it:
#
#   ssh -t root@DROPLET_IP 'set -a; . /root/creds; set +a; SPEKK_AUTH_MODE=subscription bash /tmp/setup-credentials.sh; shred -u /root/creds'
#
# Switching a live sandbox rewrites the env file whole, so every value it does
# not prompt for is carried over from the file already there. The one thing
# that cannot be recovered if lost is SPEKK_AGENT_TOKEN: the control host holds
# only its hash, so a blanked one means re-registering the agent.
#
# SPEKK_AUTH_MODE selects which credential the agent authenticates Claude with:
#
#   bedrock       (default) bills through the AWS Bedrock API
#   subscription  uses a Claude subscription token from `claude setup-token`
#
# The credential prompts do not echo.
set -euo pipefail

AGENT_ENV_FILE="${AGENT_ENV_FILE:-/etc/spekk/agent.env}"
SPEKK_AUTH_MODE="${SPEKK_AUTH_MODE:-bedrock}"

# Reject an unknown mode before anything reads a credential or writes a file.
case "$SPEKK_AUTH_MODE" in
    bedrock | subscription) ;;
    *)
        echo "ERROR: invalid SPEKK_AUTH_MODE '${SPEKK_AUTH_MODE}': must be 'bedrock' or 'subscription'" >&2
        exit 1
        ;;
esac

# prompt_secret reads a value into the named variable without echoing it, and
# does nothing if the environment already supplied one. A token typed at a
# visible prompt stays in the operator's terminal scrollback.
prompt_secret() {
    local var="$1" label="$2"
    while [ -z "${!var:-}" ]; do
        read -rsp "${label}: " "${var?}"
        echo
        # An empty value here would be written as an empty credential, and on
        # a switch the file holding the old one has already been read. Ask
        # again rather than brick the sandbox.
        [ -n "${!var:-}" ] || echo "    (required)" >&2
    done
}

# prompt_plain reads a value that is not a secret.
prompt_plain() {
    local var="$1" label="$2"
    while [ -z "${!var:-}" ]; do
        read -rp "${label}: " "${var?}"
        [ -n "${!var:-}" ] || echo "    (required)" >&2
    done
}

# render_agent_env prints the agent env file.
#
# Only the chosen mode's model credential appears. CLAUDE_CODE_USE_BEDROCK and
# CLAUDE_CODE_OAUTH_TOKEN each select a credential path of their own, so a file
# carrying both would leave the choice to whichever Claude Code reads first --
# and a droplet switched to a subscription could keep billing through Bedrock
# with nothing to show for it.
render_agent_env() {
    if [ "$SPEKK_AUTH_MODE" = "subscription" ]; then
        echo "CLAUDE_CODE_OAUTH_TOKEN=${CLAUDE_CODE_OAUTH_TOKEN:-}"
    else
        echo "AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID:-}"
        echo "AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY:-}"
        echo "AWS_DEFAULT_REGION=${AWS_DEFAULT_REGION:-}"
        echo "CLAUDE_CODE_USE_BEDROCK=1"
    fi
    echo "GITHUB_TOKEN=${GITHUB_TOKEN:-}"
    echo "SPEKK_HOST=${SPEKK_HOST:-}"
    echo "SPEKK_AGENT_TOKEN=${SPEKK_AGENT_TOKEN:-}"
    # These two are not credentials, but the Go path writes them and the file
    # is rewritten whole. Omitting them would make a switch quietly change the
    # agent's workspace and its git identity.
    echo "WORKSPACE=${WORKSPACE:-/opt/spekk/workspace}"
    echo "SPEKK_AGENT_NAME=${SPEKK_AGENT_NAME:-$(hostname)}"
}

# load_existing seeds the variables from the env file already on the droplet,
# so a mode switch keeps everything the mode does not decide. Without it the
# whole-file rewrite that makes a switch a switch would also discard the agent
# token, the host, and the workspace settings.
#
# Only known keys are read, and only when the environment has not already
# supplied one. Values are taken literally: no eval, no sourcing, so a value
# containing shell metacharacters cannot run.
load_existing() {
    [ -r "$AGENT_ENV_FILE" ] || return 0
    local key value
    while IFS='=' read -r key value; do
        case "$key" in
            SPEKK_AGENT_TOKEN | SPEKK_HOST | GITHUB_TOKEN | WORKSPACE | SPEKK_AGENT_NAME | \
                AWS_ACCESS_KEY_ID | AWS_SECRET_ACCESS_KEY | AWS_DEFAULT_REGION | CLAUDE_CODE_OAUTH_TOKEN) ;;
            *) continue ;;
        esac
        [ -n "$value" ] || continue
        [ -n "${!key:-}" ] || printf -v "$key" '%s' "$value"
    done < "$AGENT_ENV_FILE"
}

# write_agent_env carries forward what the current file holds, asks for what is
# still missing, and replaces the file.
#
# The order is the point, so it lives in one function that a test can drive:
# reading before writing is what stops the whole-file rewrite from discarding
# the agent token, and prompting after reading is what stops it asking for
# values the droplet already has.
write_agent_env() {
    load_existing
    collect_credentials

    # The file is rewritten whole, never appended to. That is what makes a mode
    # switch a switch: the previous mode's variables are gone, not shadowed.
    echo "==> Writing ${AGENT_ENV_FILE}"
    render_agent_env > "$AGENT_ENV_FILE"
    chmod 600 "$AGENT_ENV_FILE"
    echo "    Done."
}

collect_credentials() {
    prompt_secret SPEKK_AGENT_TOKEN "Agent auth token (SPEKK_AGENT_TOKEN)"
    prompt_plain SPEKK_HOST "Control host (e.g. your-control-host.example)"

    if [ "$SPEKK_AUTH_MODE" = "subscription" ]; then
        prompt_secret CLAUDE_CODE_OAUTH_TOKEN "Claude subscription token (from 'claude setup-token')"
    else
        prompt_secret AWS_ACCESS_KEY_ID "AWS access key ID"
        prompt_secret AWS_SECRET_ACCESS_KEY "AWS secret access key"
        prompt_plain AWS_DEFAULT_REGION "AWS region (e.g. us-east-1)"
    fi

    prompt_secret GITHUB_TOKEN "GitHub PAT for agent"
}

main() {
    # Every file this writes holds a credential. Set the mask before creating
    # any of them, so none exists even briefly at the default 0644.
    umask 077

    echo "==> Spekk Sandbox Credential Setup (auth mode: ${SPEKK_AUTH_MODE})"

    # Wait for cloud-init to finish if it's still running
    if [ -f /run/cloud-init/result.json ]; then
        echo "    Cloud-init already complete."
    else
        echo "    Waiting for cloud-init to finish..."
        cloud-init status --wait || true
    fi

    # Check provisioning marker
    if [ ! -f /opt/spekk/.provisioned ]; then
        echo "ERROR: Droplet not fully provisioned. Check /var/log/cloud-init-output.log"
        exit 1
    fi

    write_agent_env

    # Configure git credentials for agent user
    echo "==> Configuring git credentials for agent user"
    su - agent -c "git config --global credential.helper store"

    # A git identity, so anything that commits on this box has an author.
    #
    # Without this, git falls back to <user>@<hostname>, which it will not accept
    # for a commit -- and every sandbox checked on 2026-08-08 had no identity at
    # all. The agent papers over it by passing one inline per command, so its own
    # commits succeed and nothing looks wrong, while any other process that
    # commits fails. `spekk observer announce` did exactly that: it delivered a
    # finding to chat, could not write the commit marking it announced, and so
    # announced the same finding again on every run after it.
    #
    # SPEKK_AGENT_NAME names the author. It falls back to the hostname, which is
    # already the sandbox name, so an operator who does not set it still gets an
    # attributable identity rather than none.
    AGENT_GIT_NAME="${SPEKK_AGENT_NAME:-$(hostname)}"
    echo "==> Setting git identity for agent user (${AGENT_GIT_NAME})"
    su - agent -c "git config --global user.name '${AGENT_GIT_NAME}'"
    su - agent -c "git config --global user.email '${AGENT_GIT_NAME}@spekk.local'"
    su - agent -c "echo 'https://agent:${GITHUB_TOKEN}@github.com' > ~/.git-credentials"
    su - agent -c "chmod 600 ~/.git-credentials"

    # Configure gh CLI for agent user
    echo "==> Configuring gh CLI"
    su - agent -c "echo '${GITHUB_TOKEN}' | gh auth login --with-token" 2>/dev/null || true

    # An older version of this script exported ANTHROPIC_API_KEY into the
    # agent's login shell. It stopped writing that file, but a droplet
    # credentialed before then still carries it -- a live key in every agent
    # shell, outside the env file, and outside what `spekk sandbox destroy`
    # knows to remove. Switching auth mode is exactly when it must go.
    if [ -e /home/agent/.bashrc.d/spekk.sh ]; then
        echo "==> Removing the legacy shell-profile credential"
        rm -f /home/agent/.bashrc.d/spekk.sh
    fi

    # systemd reads EnvironmentFile only when the unit starts, and a running
    # agent hands its own environment to every claude child. Without this, a
    # switch writes the new credential and the sandbox keeps billing the old
    # one, with nothing to show that it did not take.
    #
    # try-restart is a no-op on a droplet where the unit is not running yet,
    # which is the first-time-setup case.
    echo "==> Restarting the agent so the new credential takes effect"
    systemctl try-restart spekk-agent || true

    echo "==> Credential setup complete."
    echo ""
    echo "Next steps:"
    echo "  1. Run deploy-agent.sh to copy the Go binary to /opt/spekk/agent-client"
    echo "  2. systemctl enable --now spekk-agent"
    echo "  3. Check status: systemctl status spekk-agent"
    echo "  4. Check logs: journalctl -u spekk-agent -f"
}

# Sourcing the script defines its functions without running it, so
# render_agent_env can be checked without a droplet.
if [ "${BASH_SOURCE[0]}" = "$0" ]; then
    main "$@"
fi
