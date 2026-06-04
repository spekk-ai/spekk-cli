#!/usr/bin/env bash
set -eo pipefail

# Tests for scripts/release.sh
# Validates token checking, binary detection, and versioned naming

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RELEASE_SCRIPT="$SCRIPT_DIR/release.sh"
passed=0
failed=0

assert_eq() {
    local desc="$1" expected="$2" actual="$3"
    if [ "$expected" = "$actual" ]; then
        echo "  PASS: $desc"
        passed=$((passed + 1))
    else
        echo "  FAIL: $desc"
        echo "    expected: $expected"
        echo "    actual:   $actual"
        failed=$((failed + 1))
    fi
}

assert_contains() {
    local desc="$1" needle="$2" haystack="$3"
    if echo "$haystack" | grep -qF -- "$needle"; then
        echo "  PASS: $desc"
        passed=$((passed + 1))
    else
        echo "  FAIL: $desc"
        echo "    expected to contain: $needle"
        failed=$((failed + 1))
    fi
}

echo "=== Release Script Tests ==="

# --- Test 1: Missing GEMFURY_USER ---
echo ""
echo "Test 1: Fails when GEMFURY_USER is not set"
exit_code=0
output=$(env -u GEMFURY_USER GEMFURY_TOKEN=fake bash "$RELEASE_SCRIPT" 2>&1) || exit_code=$?

assert_eq "exits with non-zero" "1" "$exit_code"
assert_contains "error mentions GEMFURY_USER" "GEMFURY_USER" "$output"

# --- Test 2: Missing GEMFURY_TOKEN ---
echo ""
echo "Test 2: Fails when GEMFURY_TOKEN is not set"
exit_code=0
output=$(GEMFURY_USER=fake env -u GEMFURY_TOKEN bash "$RELEASE_SCRIPT" 2>&1) || exit_code=$?

assert_eq "exits with non-zero" "1" "$exit_code"
assert_contains "error mentions GEMFURY_TOKEN" "GEMFURY_TOKEN" "$output"

# --- Test 3: Script is executable ---
echo ""
echo "Test 3: Script is executable"
if [ -x "$RELEASE_SCRIPT" ]; then
    echo "  PASS: release.sh is executable"
    passed=$((passed + 1))
else
    echo "  FAIL: release.sh is not executable"
    failed=$((failed + 1))
fi

# --- Test 4: Script has proper shebang ---
echo ""
echo "Test 4: Script has bash shebang"
first_line=$(head -1 "$RELEASE_SCRIPT")
assert_eq "has bash shebang" "#!/usr/bin/env bash" "$first_line"

# --- Test 5: Script references all 6 platforms ---
echo ""
echo "Test 5: Script references all 6 platforms"
script_content=$(cat "$RELEASE_SCRIPT")
for platform in darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64; do
    if echo "$script_content" | grep -q "$platform"; then
        echo "  PASS: contains $platform"
        passed=$((passed + 1))
    else
        echo "  FAIL: missing $platform"
        failed=$((failed + 1))
    fi
done

# --- Test 6: Script creates versioned artifact names ---
echo ""
echo "Test 6: Script creates versioned artifact names"
assert_contains "versioned naming pattern" '-v${VERSION}' "$script_content"

# --- Test 7: Script uploads to Gemfury push endpoint ---
echo ""
echo "Test 7: Script uses Gemfury push endpoint"
assert_contains "push.fury.io endpoint" "push.fury.io" "$script_content"

# --- Test 8: Script uses GEMFURY_TOKEN for auth ---
echo ""
echo "Test 8: Script authenticates with GEMFURY_TOKEN"
assert_contains "token in upload URL" 'GEMFURY_TOKEN' "$script_content"

# --- Summary ---
echo ""
echo "=== Results: $passed passed, $failed failed ==="

if [ "$failed" -gt 0 ]; then
    exit 1
fi
