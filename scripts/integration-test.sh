#!/usr/bin/env bash
# Run mc/cmd integration tests (Test_FullSuite) against a throwaway libreFS container.
#
# Usage:
#   ./scripts/integration-test.sh [go-test-args...]
#
# Extra go-test args are appended to the test command, e.g.:
#   ./scripts/integration-test.sh -v -run Test_FullSuite/bucket
#
# Environment overrides:
#   SERVER_IMAGE   — libreFS server image  (default: ghcr.io/librefs/librefs:latest)
#   BUILD_IMAGE    — lc build/test image   (default: lc-dev:latest)
#   SERVER_PORT    — host port for S3 API  (default: 9000)
#   ACCESS_KEY     — root access key       (default: minioadmin)
#   SECRET_KEY     — root secret key       (default: minioadmin)
#   TIMEOUT        — go test timeout       (default: 300s)

set -euo pipefail

SERVER_IMAGE="${SERVER_IMAGE:-ghcr.io/librefs/librefs:latest}"
BUILD_IMAGE="${BUILD_IMAGE:-lc-dev:latest}"
SERVER_PORT="${SERVER_PORT:-9000}"
ACCESS_KEY="${ACCESS_KEY:-minioadmin}"
SECRET_KEY="${SECRET_KEY:-minioadmin}"
TIMEOUT="${TIMEOUT:-300s}"

CONTAINER_NAME="lc-inttest-server-$$"

# ── cleanup trap ─────────────────────────────────────────────────────────────
cleanup() {
    echo ">>> Stopping test server ($CONTAINER_NAME)..."
    docker stop "$CONTAINER_NAME" >/dev/null 2>&1 || true
    docker rm   "$CONTAINER_NAME" >/dev/null 2>&1 || true
}
trap cleanup EXIT

# ── start server ──────────────────────────────────────────────────────────────
echo ">>> Starting libreFS server ($SERVER_IMAGE) on port $SERVER_PORT..."
docker run -d \
    --name "$CONTAINER_NAME" \
    -e MINIO_ROOT_USER="$ACCESS_KEY" \
    -e MINIO_ROOT_PASSWORD="$SECRET_KEY" \
    -p "${SERVER_PORT}:9000" \
    "$SERVER_IMAGE" \
    server /data

# ── wait for health ───────────────────────────────────────────────────────────
echo ">>> Waiting for server to be ready..."
WAIT_MAX=60
WAIT=0
until curl -sf "http://127.0.0.1:${SERVER_PORT}/minio/health/live" >/dev/null 2>&1; do
    if [ "$WAIT" -ge "$WAIT_MAX" ]; then
        echo "ERROR: server did not become healthy after ${WAIT_MAX}s" >&2
        docker logs "$CONTAINER_NAME" >&2
        exit 1
    fi
    sleep 1
    WAIT=$((WAIT + 1))
done
echo ">>> Server ready after ${WAIT}s."

# ── build the lc binary ───────────────────────────────────────────────────────
echo ">>> Building lc binary..."
docker run --rm \
    -v "$(pwd)":/app \
    -w /app \
    "$BUILD_IMAGE" \
    go build -o lc .

# ── run tests ────────────────────────────────────────────────────────────────
echo ">>> Running integration tests..."
docker run --rm \
    --network host \
    -v "$(pwd)":/app \
    -w /app \
    -e MC_TEST_RUN_FULL_SUITE=true \
    -e MC_TEST_SERVER_ENDPOINT="127.0.0.1:${SERVER_PORT}" \
    -e MC_TEST_ACCESS_KEY="$ACCESS_KEY" \
    -e MC_TEST_SECRET_KEY="$SECRET_KEY" \
    -e MC_TEST_SKIP_BUILD=true \
    -e MC_TEST_BINARY_PATH=/app/lc \
    "$BUILD_IMAGE" \
    go test -timeout "$TIMEOUT" -v -run "Test_FullSuite" ./cmd/ "$@"
