#!/usr/bin/env bash
#
# Fail if a release note exists but the docs navigation does not list it.
#
# The nav in zensical.toml is hand-maintained, so it drifts silently: a
# release adds docs/release-notes/RELEASE-NOTES-X.Y.Z.md, nothing links it,
# and the page is unreachable on the docs site while still being present in
# the repository. Three releases had accumulated that way before anyone
# noticed, and nothing failed -- an unlisted page looks exactly like a page
# that was never written.
#
# Both directions are checked. A nav entry pointing at a file that no longer
# exists is the same class of defect seen from the other side.

set -uo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."

NAV="zensical.toml"
NOTES_DIR="docs/release-notes"

# -printf is a GNU extension. BSD find, which is the find on macOS, refuses
# it, and the script then read an empty list and reported every nav entry as
# orphaned. Strip the directory with sed instead, which every find supports.
on_disk=$(find "$NOTES_DIR" -name 'RELEASE-NOTES-*.md' | sed 's|.*/||' | sort)
in_nav=$(grep -oE 'release-notes/RELEASE-NOTES-[^"]+\.md' "$NAV" | sed 's|release-notes/||' | sort -u)

missing=$(comm -23 <(echo "$on_disk") <(echo "$in_nav"))
orphaned=$(comm -13 <(echo "$on_disk") <(echo "$in_nav"))

status=0
if [ -n "$missing" ]; then
    echo "check-docs-nav: release notes not listed in $NAV:" >&2
    echo "$missing" | sed 's/^/  /' >&2
    status=1
fi
if [ -n "$orphaned" ]; then
    echo "check-docs-nav: $NAV lists files that do not exist:" >&2
    echo "$orphaned" | sed 's/^/  /' >&2
    status=1
fi

if [ "$status" -eq 0 ]; then
    echo "check-docs-nav: clean ($(echo "$on_disk" | wc -l | tr -d ' ') release notes, all listed)."
fi
exit "$status"
