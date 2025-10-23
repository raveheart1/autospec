#!/bin/bash
# T089-T090: Test runner script for all validation tests
# Runs all bats tests and reports results

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test directories
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$TEST_DIR/.." && pwd)"

echo "SpecKit Validation Test Runner"
echo "==============================="
echo ""

# Check if bats is installed
if ! command -v bats &> /dev/null; then
    echo -e "${RED}ERROR: bats-core is not installed${NC}"
    echo ""
    echo "Please install bats-core to run tests:"
    echo "  npm install -g bats"
    echo "  OR"
    echo "  brew install bats-core"
    echo ""
    echo "See tests/README.md for detailed installation instructions"
    exit 1
fi

# Display bats version
BATS_VERSION=$(bats --version)
echo "Using: $BATS_VERSION"
echo ""

# Test suites to run
TEST_SUITES=(
    "lib/validation-lib.bats"
    "scripts/workflow-validate.bats"
    "scripts/implement-validate.bats"
    "hooks/stop-speckit-specify.bats"
    "hooks/stop-speckit-plan.bats"
    "hooks/stop-speckit-tasks.bats"
    "hooks/stop-speckit-implement.bats"
    "hooks/stop-speckit-clarify.bats"
    "quickstart-validation.bats"
    "integration.bats"
)

# Track results
TOTAL_SUITES=0
PASSED_SUITES=0
FAILED_SUITES=0
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Temp file for detailed results
RESULTS_FILE=$(mktemp)

echo "Running test suites..."
echo ""

# Run each test suite
for suite in "${TEST_SUITES[@]}"; do
    TOTAL_SUITES=$((TOTAL_SUITES + 1))
    suite_path="$TEST_DIR/$suite"

    if [ ! -f "$suite_path" ]; then
        echo -e "${YELLOW}SKIP${NC} $suite (file not found)"
        continue
    fi

    echo -n "Running $suite... "

    # Run bats and capture output
    if bats "$suite_path" > "$RESULTS_FILE" 2>&1; then
        echo -e "${GREEN}PASS${NC}"
        PASSED_SUITES=$((PASSED_SUITES + 1))

        # Count tests in this suite
        suite_tests=$(grep -c "^ok " "$RESULTS_FILE" || echo "0")
        TOTAL_TESTS=$((TOTAL_TESTS + suite_tests))
        PASSED_TESTS=$((PASSED_TESTS + suite_tests))
    else
        echo -e "${RED}FAIL${NC}"
        FAILED_SUITES=$((FAILED_SUITES + 1))

        # Show failures
        echo "  Failures:"
        grep "^not ok" "$RESULTS_FILE" | sed 's/^/    /' || true

        # Count tests
        suite_passed=$(grep -c "^ok " "$RESULTS_FILE" || echo "0")
        suite_failed=$(grep -c "^not ok " "$RESULTS_FILE" || echo "0")
        TOTAL_TESTS=$((TOTAL_TESTS + suite_passed + suite_failed))
        PASSED_TESTS=$((PASSED_TESTS + suite_passed))
        FAILED_TESTS=$((FAILED_TESTS + suite_failed))
    fi
done

# Clean up temp file
rm -f "$RESULTS_FILE"

echo ""
echo "==============================="
echo "Test Results Summary"
echo "==============================="
echo ""
echo "Test Suites:"
echo "  Total:  $TOTAL_SUITES"
echo -e "  ${GREEN}Passed: $PASSED_SUITES${NC}"
if [ "$FAILED_SUITES" -gt 0 ]; then
    echo -e "  ${RED}Failed: $FAILED_SUITES${NC}"
else
    echo "  Failed: 0"
fi
echo ""
echo "Individual Tests:"
echo "  Total:  $TOTAL_TESTS"
echo -e "  ${GREEN}Passed: $PASSED_TESTS${NC}"
if [ "$FAILED_TESTS" -gt 0 ]; then
    echo -e "  ${RED}Failed: $FAILED_TESTS${NC}"
else
    echo "  Failed: 0"
fi
echo ""

# Calculate pass rate
if [ "$TOTAL_TESTS" -gt 0 ]; then
    PASS_RATE=$(( (PASSED_TESTS * 100) / TOTAL_TESTS ))
    echo "Pass Rate: ${PASS_RATE}%"
    echo ""
fi

# Exit with appropriate code
if [ "$FAILED_SUITES" -eq 0 ] && [ "$FAILED_TESTS" -eq 0 ]; then
    echo -e "${GREEN}✓ All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}✗ Some tests failed${NC}"
    echo ""
    echo "To run individual test suites:"
    echo "  bats tests/lib/validation-lib.bats"
    echo "  bats tests/scripts/workflow-validate.bats"
    echo "  # etc..."
    exit 1
fi
