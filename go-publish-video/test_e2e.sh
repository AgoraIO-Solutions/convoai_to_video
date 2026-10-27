#!/usr/bin/env bash
set -euo pipefail

# Audio smoke check: publisher + subscriber with token auth.
# Video reception and A/V synchronization require manual verification.
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

for binary in parent subscriber tokengen; do
    if [ ! -x "./$binary" ]; then
        echo "Error: ./$binary is missing or not executable; run make build subscriber tokengen"
        exit 1
    fi
done

PUBLISHER_PID=""
RESULT=0

cleanup() {
    if [ -n "$PUBLISHER_PID" ]; then
        if kill -0 "$PUBLISHER_PID" 2>/dev/null; then
            echo "Cleaning up publisher (PID $PUBLISHER_PID)..."
            kill "$PUBLISHER_PID" 2>/dev/null || true
            for ((attempt = 0; attempt < 50; attempt++)); do
                kill -0 "$PUBLISHER_PID" 2>/dev/null || break
                sleep 0.1
            done
            if kill -0 "$PUBLISHER_PID" 2>/dev/null; then
                kill -KILL "$PUBLISHER_PID" 2>/dev/null || true
            fi
        fi
        wait "$PUBLISHER_PID" 2>/dev/null || true
        PUBLISHER_PID=""
    fi
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

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
    # This function runs in an if condition, so check failures explicitly:
    # Bash disables errexit inside the function in that context.
    if ! pub_token=$(./tokengen -appID "$APP_ID" -appCert "$APP_CERT" \
        -channelName "$channel_name" -uid "$publisher_uid" -role publisher \
        -enableStringUID="$enable_string_uid"); then
        echo "[test] FAIL: Publisher token generation failed"
        return 1
    fi
    if [ -z "$pub_token" ]; then
        echo "[test] FAIL: Publisher token is empty"
        return 1
    fi
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

    # The subscriber waits for connection and audio with its own timeouts.
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

    if ! kill -0 "$PUBLISHER_PID" 2>/dev/null; then
        echo "[test] FAIL: Publisher exited during the audio check"
        sub_result=1
    fi

    echo "[test] Stopping publisher..."
    cleanup

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
echo "Audio Smoke Check: Publisher + Subscriber (video not checked)"
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
    echo "AUDIO SMOKE CHECKS PASSED (video not checked)"
else
    echo "AUDIO SMOKE CHECKS FAILED"
fi
echo "======================================="

exit $RESULT
