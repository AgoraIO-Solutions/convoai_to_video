#!/usr/bin/env bash
set -euo pipefail

# E2E integration test: publisher + subscriber with token auth
#
# Required environment variables:
#   APP_ID   - Agora App ID
#   APP_CERT - Agora App Certificate

if [ -z "${APP_ID:-}" ] || [ -z "${APP_CERT:-}" ]; then
    echo "Error: APP_ID and APP_CERT environment variables are required"
    echo "Usage: APP_ID=xxx APP_CERT=yyy ./test_e2e.sh"
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

PUBLISHER_PID=""
RESULT=0

cleanup() {
    if [ -n "$PUBLISHER_PID" ] && kill -0 "$PUBLISHER_PID" 2>/dev/null; then
        echo "Cleaning up publisher (PID $PUBLISHER_PID)..."
        kill "$PUBLISHER_PID" 2>/dev/null || true
        wait "$PUBLISHER_PID" 2>/dev/null || true
    fi
}
trap cleanup EXIT

run_test() {
    local test_name="$1"
    local channel_name="$2"
    local enable_string_uid="$3"
    local publisher_uid="100"

    echo ""
    echo "=== $test_name ==="
    echo ""

    # Generate publisher token using tokengen
    echo "Generating publisher token..."
    local pub_token
    pub_token=$(./tokengen -appID "$APP_ID" -appCert "$APP_CERT" \
        -channelName "$channel_name" -uid "$publisher_uid" -role publisher)
    echo "[test] Generated publisher token for uid=$publisher_uid"

    # Start publisher in background
    echo "Starting publisher on channel=$channel_name enableStringUID=$enable_string_uid..."
    ./parent \
        -appID "$APP_ID" \
        -token "$pub_token" \
        -channelName "$channel_name" \
        -userID "$publisher_uid" \
        -enableStringUID="$enable_string_uid" &
    PUBLISHER_PID=$!
    echo "[test] Publisher started (PID $PUBLISHER_PID)"

    # Wait for publisher to connect and start streaming
    echo "[test] Waiting 5s for publisher to connect..."
    sleep 5

    # Check publisher is still running
    if ! kill -0 "$PUBLISHER_PID" 2>/dev/null; then
        echo "[test] FAIL: Publisher exited prematurely"
        PUBLISHER_PID=""
        return 1
    fi

    # Run subscriber (blocking — exits 0 on success, 1 on failure)
    echo "[test] Starting subscriber..."
    local sub_result=0
    ./subscriber \
        -appID "$APP_ID" \
        -appCert "$APP_CERT" \
        -channelName "$channel_name" \
        -publisherUID "$publisher_uid" \
        -enableStringUID="$enable_string_uid" || sub_result=$?

    # Kill publisher
    echo "[test] Stopping publisher..."
    kill "$PUBLISHER_PID" 2>/dev/null || true
    wait "$PUBLISHER_PID" 2>/dev/null || true
    PUBLISHER_PID=""

    if [ "$sub_result" -eq 0 ]; then
        echo ""
        echo "$test_name: PASSED"
        return 0
    else
        echo ""
        echo "$test_name: FAILED (subscriber exit code: $sub_result)"
        return 1
    fi
}

echo "======================================="
echo "E2E Integration Test: Publisher + Subscriber"
echo "======================================="

# Test 1: String UID mode
if run_test "Test 1: String UID mode" "e2e-str-$$" "true"; then
    echo ""
else
    RESULT=1
fi

# Test 2: Numeric UID mode
if run_test "Test 2: Numeric UID mode" "e2e-num-$$" "false"; then
    echo ""
else
    RESULT=1
fi

echo "======================================="
if [ "$RESULT" -eq 0 ]; then
    echo "ALL TESTS PASSED"
else
    echo "SOME TESTS FAILED"
fi
echo "======================================="

exit $RESULT
