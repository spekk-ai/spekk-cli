#!/bin/bash

# Test script for prompt-exists assertion
# Validates the builder agent prompt file meets all requirements

PROMPT_FILE="specs/builder-agent/builder-agent.prompt.md"
EXIT_CODE=0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "Testing builder agent prompt file..."

# Test 1: File exists
echo -n "✓ File exists at correct path... "
if [ -f "$PROMPT_FILE" ]; then
    echo -e "${GREEN}PASS${NC}"
else
    echo -e "${RED}FAIL${NC}: File does not exist at $PROMPT_FILE"
    EXIT_CODE=1
fi

# Test 2: Contains role definition
echo -n "✓ Contains role definition... "
if grep -q "Builder Agent" "$PROMPT_FILE" && grep -q "Your Role" "$PROMPT_FILE"; then
    echo -e "${GREEN}PASS${NC}"
else
    echo -e "${RED}FAIL${NC}: Missing role definition section"
    EXIT_CODE=1
fi

# Test 3: Contains workflow steps
echo -n "✓ Contains complete workflow... "
if grep -q "Get Next Task" "$PROMPT_FILE" && grep -q "Update Status" "$PROMPT_FILE"; then
    echo -e "${GREEN}PASS${NC}"
else
    echo -e "${RED}FAIL${NC}: Missing workflow steps"
    EXIT_CODE=1
fi

# Test 4: Contains explicit STOP instruction
echo -n "✓ Contains explicit STOP instruction... "
if grep -q "ONE assertion at a time, then STOP" "$PROMPT_FILE" && grep -q "Stop" "$PROMPT_FILE"; then
    echo -e "${GREEN}PASS${NC}"
else
    echo -e "${RED}FAIL${NC}: Missing explicit STOP instruction"
    EXIT_CODE=1
fi

# Test 5: Uses global CLI commands (spekk next)
echo -n "✓ Uses global CLI commands... "
if grep -q "spekk next" "$PROMPT_FILE"; then
    echo -e "${GREEN}PASS${NC}"
else
    echo -e "${RED}FAIL${NC}: Should use 'spekk next' instead of 'npm run next'"
    EXIT_CODE=1
fi

# Test 6: References app/ directory structure
echo -n "✓ References app/ directory structure... "
if grep -q "app/" "$PROMPT_FILE"; then
    echo -e "${GREEN}PASS${NC}"
else
    echo -e "${RED}FAIL${NC}: Should reference app/ directory structure"
    EXIT_CODE=1
fi

# Test 7: No references to deprecated directories
echo -n "✓ No deprecated directory references... "
if ! grep -q "scripts/" "$PROMPT_FILE" && ! grep -q "tests/" "$PROMPT_FILE"; then
    echo -e "${GREEN}PASS${NC}"
else
    echo -e "${RED}FAIL${NC}: Contains references to deprecated scripts/ or tests/ directories"
    EXIT_CODE=1
fi

# Test 8: No local npm scripts for parsing
echo -n "✓ No local npm scripts for parsing... "
if ! grep -q "npm run next" "$PROMPT_FILE"; then
    echo -e "${GREEN}PASS${NC}"
else
    echo -e "${RED}FAIL${NC}: Should not use 'npm run next' (use global 'spekk next' instead)"
    EXIT_CODE=1
fi

# Test 9: Contains status values explanation
echo -n "✓ Contains status values explanation... "
if grep -q "not_started" "$PROMPT_FILE" && grep -q "in_progress" "$PROMPT_FILE" && grep -q "done" "$PROMPT_FILE"; then
    echo -e "${GREEN}PASS${NC}"
else
    echo -e "${RED}FAIL${NC}: Missing status values explanation"
    EXIT_CODE=1
fi

# Test 10: Contains priority levels
echo -n "✓ Contains priority levels explanation... "
if grep -q "priority" "$PROMPT_FILE" && (grep -q "1.*2.*3" "$PROMPT_FILE" || grep -q "Three priority levels" "$PROMPT_FILE"); then
    echo -e "${GREEN}PASS${NC}"
else
    echo -e "${RED}FAIL${NC}: Missing priority levels explanation"
    EXIT_CODE=1
fi

echo ""
if [ $EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}All tests PASSED!${NC}"
else
    echo -e "${RED}Some tests FAILED!${NC}"
fi

exit $EXIT_CODE