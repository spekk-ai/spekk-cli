#!/usr/bin/env bash
#
# Fail if a private term appears anywhere in the tracked tree.
#
# This repository is public. Its prompts, specs, release notes and test
# fixtures are all written from real work, and example content is where client
# detail leaks: a project name in a release note, a real scenario in a prompt,
# a client's spec vocabulary reused as sample data. None of that is caught by
# a compiler or a test.
#
# The list of terms is NOT stored here. Writing "do not mention X" in a public
# repository publishes X, which is the thing being prevented. The terms come
# from the environment instead:
#
#   SPEKK_PRIVATE_TERMS  newline- or comma-separated, case-insensitive
#
# In CI that is a repository secret. Locally, put one term per line in
# `.private-terms` (gitignored) and this script reads it.
#
# Matching is plain, case-insensitive substring. That over-matches by design:
# a false positive costs one line in the allowlist below, and a false negative
# costs a disclosure.

set -uo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."

TERMS="${SPEKK_PRIVATE_TERMS:-}"
if [ -z "$TERMS" ] && [ -f .private-terms ]; then
    TERMS="$(cat .private-terms)"
fi

# No terms is a failure, not a pass. A check that silently does nothing when
# its configuration is missing reports "clean" for a tree it never read, which
# is worse than having no check at all: it is a check everyone believes in.
if [ -z "${TERMS//[[:space:],]/}" ]; then
    echo "check-private-terms: no terms supplied." >&2
    echo "  Set the SPEKK_PRIVATE_TERMS secret, or create a local .private-terms file." >&2
    exit 2
fi

# Files whose content is generated, vendored, or otherwise not ours to edit.
EXCLUDES=(
    ':!internal/show/template.html'   # minified third-party bundle
    ':!go.sum'
    ':!LICENSE'
)

status=0
while IFS= read -r term; do
    term="$(echo "$term" | tr -d '\r' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')"
    [ -z "$term" ] && continue
    # -I skips binary files; -F is a literal match, so a term containing a dot
    # or a dash is not read as a pattern.
    if hits=$(git grep -nIiF -- "$term" -- . "${EXCLUDES[@]}" 2>/dev/null); then
        echo "check-private-terms: '$term' appears in tracked files:" >&2
        echo "$hits" | head -20 | sed 's/^/  /' >&2
        status=1
    fi
done < <(echo "$TERMS" | tr ',' '\n')

if [ "$status" -eq 0 ]; then
    echo "check-private-terms: clean."
fi
exit "$status"
