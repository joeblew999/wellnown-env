#!/bin/bash
# test-auth-lifecycle.sh - Regression test for auth lifecycle transitions
#
# Tests all 4 auth modes with fresh start between each:
#   none -> token -> nkey -> jwt -> none
#
# Usage:
#   ./scripts/test-auth-lifecycle.sh         # Run full test
#   ./scripts/test-auth-lifecycle.sh quick   # Quick test (no mesh, hub only)

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR/.."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

QUICK_MODE="${1:-full}"
PASSED=0
FAILED=0
START_TIME=$(date +%s)

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_ok() { echo -e "${GREEN}[PASS]${NC} $1"; ((PASSED++)); }
log_fail() { echo -e "${RED}[FAIL]${NC} $1"; ((FAILED++)); }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_section() { echo -e "\n${YELLOW}========================================${NC}"; echo -e "${YELLOW}$1${NC}"; echo -e "${YELLOW}========================================${NC}"; }

cleanup() {
    log_info "Cleaning up..."
    task mesh:down 2>/dev/null || true
    pkill -f "NATS_NAME=hub" 2>/dev/null || true
    sleep 1
}

trap cleanup EXIT

# Test connection with current auth mode
test_connection() {
    local mode=$1
    local auth_args
    auth_args=$(./scripts/nats-auth.sh)

    log_info "Testing connection with mode: $mode (args: $auth_args)"

    if nats -s nats://localhost:4222 $auth_args account info >/dev/null 2>&1; then
        log_ok "Connection successful with $mode auth"
        return 0
    else
        log_fail "Connection failed with $mode auth"
        return 1
    fi
}

# Test that connection fails without proper auth
test_connection_fails_without_auth() {
    local mode=$1

    if [ "$mode" = "none" ]; then
        log_info "Skipping negative test for none mode"
        return 0
    fi

    log_info "Testing that connection fails without auth (mode: $mode)"

    if nats -s nats://localhost:4222 account info >/dev/null 2>&1; then
        log_fail "Connection should have failed without auth in $mode mode"
        return 1
    else
        log_ok "Connection correctly rejected without auth in $mode mode"
        return 0
    fi
}

# Start hub and wait for it to be ready
start_hub() {
    log_info "Starting hub..."
    GOWORK=off NATS_NAME=hub NATS_PORT=4222 NATS_DATA=./.data/hub go run . &
    HUB_PID=$!

    # Wait for hub to be ready
    for i in {1..30}; do
        if nc -z localhost 4222 2>/dev/null; then
            log_ok "Hub started (PID: $HUB_PID)"
            sleep 1
            return 0
        fi
        sleep 0.5
    done

    log_fail "Hub failed to start"
    return 1
}

stop_hub() {
    log_info "Stopping hub..."
    kill $HUB_PID 2>/dev/null || true
    wait $HUB_PID 2>/dev/null || true
    sleep 1
}

# Test a single auth mode
test_auth_mode() {
    local mode=$1
    local setup_task=$2

    log_section "Testing $mode auth mode"

    # Clean start
    log_info "Fresh start: cleaning data and auth..."
    task clean >/dev/null 2>&1 || true
    task auth:clean >/dev/null 2>&1 || true

    # Set up auth mode
    if [ -n "$setup_task" ]; then
        log_info "Setting up $mode auth..."
        if ! task $setup_task; then
            log_fail "Failed to set up $mode auth"
            return 1
        fi
    fi

    # Verify mode
    local current_mode
    current_mode=$(task auth:status 2>&1 | grep "Auth mode:" | awk '{print $3}')
    if [ "$current_mode" = "$mode" ] || [ "$current_mode" = "none" -a "$mode" = "none" ]; then
        log_ok "Auth mode correctly set to $mode"
    else
        log_fail "Auth mode is $current_mode, expected $mode"
        return 1
    fi

    # Start hub and test
    if start_hub; then
        test_connection "$mode"
        test_connection_fails_without_auth "$mode"
        stop_hub
    else
        return 1
    fi

    return 0
}

# Main test sequence
main() {
    log_section "Auth Lifecycle Regression Test"
    log_info "Mode: $QUICK_MODE"
    log_info "Started at: $(date)"

    # Ensure clean state
    cleanup

    # Test each auth mode
    test_auth_mode "none" ""
    test_auth_mode "token" "auth:token"
    test_auth_mode "nkey" "auth:nkey"
    test_auth_mode "jwt" "auth:jwt"

    # Return to none
    log_section "Final cleanup - returning to dev mode"
    task clean >/dev/null 2>&1 || true
    task auth:clean >/dev/null 2>&1 || true

    # Summary
    END_TIME=$(date +%s)
    DURATION=$((END_TIME - START_TIME))

    log_section "Test Summary"
    echo -e "Duration: ${DURATION}s"
    echo -e "${GREEN}Passed: $PASSED${NC}"
    echo -e "${RED}Failed: $FAILED${NC}"

    if [ $FAILED -eq 0 ]; then
        echo -e "\n${GREEN}All tests passed!${NC}"
        exit 0
    else
        echo -e "\n${RED}Some tests failed!${NC}"
        exit 1
    fi
}

main "$@"
