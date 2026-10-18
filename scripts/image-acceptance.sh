#!/usr/bin/env bash
#
# Live acceptance of template images (specs/sandbox-template-images/) on
# DigitalOcean. Run it from a control host that has the spekk CLI and the
# credentials of `spekk sandbox create`. It costs a few droplet-hours and
# two snapshots for the time it runs.
#
# Usage:
#   scripts/image-acceptance.sh --project-setup <script> --project-repo <owner/name> \
#       [--project-check <script>] [--region nyc3] [--size s-2vcpu-4gb] [--provider digitalocean]
#
#   --project-setup  The image setup script of a project, for example one
#                    that installs a browser and pulls the stack's images.
#   --project-repo   The repository that the project check clones on the
#                    project sandbox, with the GITHUB_TOKEN of the sandbox.
#   --project-check  A script that runs as agent in that clone on the project
#                    sandbox. It must exit 0 when the image works for the
#                    project (for example: a browser smoke test, and a stack
#                    start that pulls no image). Default: no project check.
#   --provider       digitalocean (default) or gcp. With gcp, --region is the
#                    zone, and --size defaults to e2-medium.
#
# Needs: DIGITALOCEAN_TOKEN (or DO_API_TOKEN), GITHUB_TOKEN, and the model
# credential of the auth mode (CLAUDE_CODE_OAUTH_TOKEN for subscription, else
# the AWS variables for bedrock). The sandboxes are not registered with a
# control host: SPEKK_HOST is set to an address that does not resolve.
#
# The script removes each sandbox and image it made, also on failure or
# Ctrl-C, and prints what it removed. At the end it prints a timing table.

set -euo pipefail

setup="" repo="" check="" region="" size="" provider="digitalocean"
while [ $# -gt 0 ]; do
  case "$1" in
    --project-setup) setup=$2; shift 2 ;;
    --project-repo) repo=$2; shift 2 ;;
    --project-check) check=$2; shift 2 ;;
    --region) region=$2; shift 2 ;;
    --size) size=$2; shift 2 ;;
    --provider) provider=$2; shift 2 ;;
    -h|--help) sed -n '2,30p' "$0"; exit 0 ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
done
[ -f "$setup" ] || { echo "--project-setup <script> is required" >&2; exit 2; }
[ -n "$repo" ] || { echo "--project-repo <owner/name> is required" >&2; exit 2; }
[ -z "$check" ] || [ -f "$check" ] || { echo "--project-check $check: no such file" >&2; exit 2; }
command -v spekk >/dev/null || { echo "spekk is not on PATH" >&2; exit 2; }
case "$provider" in
  digitalocean)
    : "${region:=nyc3}" "${size:=s-2vcpu-4gb}"
    [ -n "${DIGITALOCEAN_TOKEN:-}${DO_API_TOKEN:-}" ] || { echo "set DIGITALOCEAN_TOKEN or DO_API_TOKEN" >&2; exit 2; } ;;
  gcp) : "${region:=us-central1-a}" "${size:=e2-medium}" ;;
  *) echo "--provider must be digitalocean or gcp" >&2; exit 2 ;;
esac
[ -n "${GITHUB_TOKEN:-}" ] || { echo "set GITHUB_TOKEN" >&2; exit 2; }

auth=bedrock
[ -z "${CLAUDE_CODE_OAUTH_TOKEN:-}" ] || auth=subscription
export SPEKK_HOST=acceptance.invalid
unset SPEKK_API_TOKEN

run=$(date -u +%m%d%H%M)
base="acc-base-$run"
proj="acc-proj-$run"
sandboxes=() images=()
declare -A secs

log() { printf '\n=== %s\n' "$*"; }

# cleanup runs on each exit. It removes what this run made, and the builder
# of an image whose build was killed before it could remove it.
cleanup() {
  local rc=$? s i
  trap - EXIT INT TERM
  log "Cleanup"
  for s in "${sandboxes[@]}" "image-$base" "image-$proj"; do
    if spekk sandbox list 2>/dev/null | awk '{print $1}' | grep -qx "$s"; then
      if spekk sandbox destroy "$s" --force >/dev/null 2>&1; then
        echo "removed sandbox $s"
      else
        echo "COULD NOT REMOVE sandbox $s: run spekk sandbox destroy $s --force" >&2
      fi
    fi
  done
  for i in "${images[@]}"; do
    if spekk sandbox image list 2>/dev/null | awk '{print $1}' | grep -qx "$i"; then
      if spekk sandbox image delete "$i" >/dev/null 2>&1; then
        echo "removed image $i"
      else
        echo "COULD NOT REMOVE image $i: run spekk sandbox image delete $i" >&2
      fi
    fi
  done
  exit "$rc"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

# timed runs a command and records its seconds under a label.
timed() {
  local label=$1 start
  shift
  start=$(date +%s)
  "$@"
  secs[$label]=$(( $(date +%s) - start ))
  echo "($label: ${secs[$label]}s)"
}

create() { # create <name> [--image <image>]
  sandboxes+=("$1")
  local name=$1
  shift
  spekk sandbox create --name "$name" --provider "$provider" --region "$region" --size "$size" --auth "$auth" --one-off --purpose "image acceptance $run" "$@"
}

on() { # on <sandbox> <command>: run a command as root on a sandbox
  printf '%s\n' "$2" | spekk sandbox ssh "$1" 'sudo bash -s'
}

identity() { # the host key fingerprint and the machine ID of a sandbox
  on "$1" 'ssh-keygen -lf /etc/ssh/ssh_host_ed25519_key.pub | cut -d" " -f2; cat /etc/machine-id'
}

log "Build the base image $base"
images+=("$base")
timed "image build (base)" spekk sandbox image build --name "$base" --provider "$provider" --regions "$region" --size "$size"

log "Build the project image $proj with $setup"
images+=("$proj")
timed "image build (project)" spekk sandbox image build --name "$proj" --setup "$setup" --provider "$provider" --regions "$region" --size "$size"
spekk sandbox image list

log "Fresh create (stock image, full cloud-init)"
timed "create: fresh" create "acc-fresh-$run"

log "Project setup on the fresh sandbox (what each task pays without an image)"
timed "project setup on fresh" bash -c "spekk sandbox ssh acc-fresh-$run 'sudo bash -s' < '$setup'"

log "Two creates from the base image"
timed "create: base image" create "acc-base-a-$run" --image "$base"
create "acc-base-b-$run" --image "$base"

log "Create from the project image"
timed "create: project image" create "acc-proj-$run" --image "$proj"

log "Check: two sandboxes from one image have their own host key and machine ID"
a=$(identity "acc-base-a-$run")
b=$(identity "acc-base-b-$run")
echo "acc-base-a: $(echo "$a" | tr '\n' ' ')"
echo "acc-base-b: $(echo "$b" | tr '\n' ' ')"
if [ "$(echo "$a" | sed -n 1p)" = "$(echo "$b" | sed -n 1p)" ]; then echo "FAIL: the SSH host keys are the same" >&2; exit 1; fi
if [ "$(echo "$a" | sed -n 2p)" = "$(echo "$b" | sed -n 2p)" ]; then echo "FAIL: /etc/machine-id is the same" >&2; exit 1; fi
echo "PASS: host keys and machine IDs differ"

log "Check: the sandbox record names the image"
if grep -q "\"image\": \"$proj\"" "${XDG_CONFIG_HOME:-$HOME/.config}/spekk/sandboxes.json"; then
  echo "PASS: the record of acc-proj-$run has image $proj"
else
  echo "FAIL: no record has image $proj" >&2; exit 1
fi

if [ -n "$check" ]; then
  log "Project check on the project sandbox ($check)"
  on "acc-proj-$run" "su - agent -c 'rm -rf ~/acceptance && git clone --quiet https://github.com/$repo.git ~/acceptance'"
  timed "project check" bash -c "spekk sandbox ssh acc-proj-$run \"sudo su - agent -c 'cd ~/acceptance && bash -s'\" < '$check'"
  echo "PASS: project check"
fi

log "Timing"
printf '%-28s %8s\n' "STEP" "SECONDS"
for k in "image build (base)" "image build (project)" "create: fresh" "project setup on fresh" "create: base image" "create: project image" "project check"; do
  [ -n "${secs[$k]:-}" ] && printf '%-28s %8s\n' "$k" "${secs[$k]}"
done
echo
echo "Baseline before images: a fresh create took about 4-5 minutes (4m13s for s-2vcpu-4gb on 2026-10-17)."
echo "Expected: a create from an image takes about 1 minute."
