#!/usr/bin/env bash
set -euo pipefail

BIN=${1:?usage: scripts/smoke.sh <path-to-rockets-binary> [seconds]}
DURATION=${2:-60}
API=http://localhost:8088

curl -fsS "$API/health" >/dev/null || { echo "service is not running (make up)"; exit 1; }

"$BIN" launch "$API/messages" --message-delay=500ms --concurrency-level=1 >/tmp/rockets-smoke.log 2>&1 &
PID=$!
sleep "$DURATION"
kill "$PID" 2>/dev/null || true
wait "$PID" 2>/dev/null || true
sleep 2

curl -fsS "$API/rockets?sort=speed&order=desc&limit=20"
echo
