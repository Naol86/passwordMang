#!/bin/sh
# Smoke check for a running server: used by `make start` after the Docker stack is up.
# Usage: scripts/smoke-check.sh [api-url] [metrics-url]
set -eu

API="${1:-http://localhost:8080}"
METRICS="${2:-http://localhost:9090}"
WAIT_SECONDS=60

fail() {
	echo "smoke check FAILED: $1" >&2
	exit 1
}

printf 'waiting for %s/readyz ' "$API"
i=0
until curl -sf "$API/readyz" >/dev/null 2>&1; do
	i=$((i + 1))
	[ "$i" -ge "$WAIT_SECONDS" ] && { echo; fail "server not ready after ${WAIT_SECONDS}s"; }
	printf '.'
	sleep 1
done
echo ' ready'

check() {
	name="$1" url="$2" want_status="$3" want_body="$4"
	body_file=$(mktemp)
	status=$(curl -s -o "$body_file" -w '%{http_code}' "$url")
	body=$(cat "$body_file")
	rm -f "$body_file"
	[ "$status" = "$want_status" ] || fail "$name: HTTP $status, want $want_status"
	case "$body" in
	*"$want_body"*) echo "  ok  $name" ;;
	*) fail "$name: body does not contain $want_body: $body" ;;
	esac
}

check "liveness"            "$API/healthz"        200 '"status":"ok"'
check "readiness (database)" "$API/readyz"        200 '"status":"ok"'
check "version"             "$API/api/v1/meta"    200 '"apiVersion":"v1"'
check "error shape"         "$API/api/v1/missing" 404 '"code":"route_not_found"'
check "metrics"             "$METRICS/metrics"    200 'pm_db_pool_max_conns'
echo "smoke check passed"
