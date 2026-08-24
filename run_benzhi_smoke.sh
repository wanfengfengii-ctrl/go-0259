#!/usr/bin/env bash
#
# run_benzhi_smoke.sh — deterministic smoke test for the MycoCycle grow-bag
# transfer gate. It builds the binary, starts the service against a temporary
# SQLite database, probes the health and task-lock API, then cleans up every
# process and temporary file. No external network access is required.
set -euo pipefail

PORT="${SMOKE_PORT:-18080}"
ADDR="127.0.0.1:${PORT}"
BASE="http://${ADDR}"

WORK_DIR="$(mktemp -d)"
BIN="${WORK_DIR}/mycocycle"
DB_PATH="${WORK_DIR}/mycocycle.db"
SERVER_PID=""

cleanup() {
  if [[ -n "${SERVER_PID}" ]] && kill -0 "${SERVER_PID}" 2>/dev/null; then
    kill "${SERVER_PID}" 2>/dev/null || true
    wait "${SERVER_PID}" 2>/dev/null || true
  fi
  rm -rf "${WORK_DIR}"
}
trap cleanup EXIT

echo "==> building binary"
go build -o "${BIN}" ./cmd/mycocycle

echo "==> starting service on ${ADDR}"
DB_PATH="${DB_PATH}" ADDR="${ADDR}" LOG_REQUESTS=false "${BIN}" &
SERVER_PID=$!

echo "==> waiting for health endpoint"
ready=""
for _ in $(seq 1 100); do
  if health_tmp="$(curl -s "${BASE}/healthz" 2>/dev/null)"; then
    ready="${health_tmp}"
    break
  fi
  sleep 0.1
done
if [[ -z "${ready}" ]]; then
  echo "service did not become ready" >&2
  exit 1
fi
if [[ "${ready}" != *'"ok"'* ]]; then
  echo "unexpected health response: ${ready}" >&2
  exit 1
fi
echo "health: ${ready}"

echo "==> locking a transfer inspection task"
LOCK_BODY='{
  "strain_revision":"strain-po-2024.03",
  "substrate_revision":"sub-hw-01",
  "substrate_summary":"hardwood sawdust + wheat bran 78:20",
  "sterilizer_summary":"run-A-2026-08-21",
  "inoculation_line":"line-1",
  "bag_batch":"B-SMOKE-001",
  "bag_positions":["P1","P2"],
  "rack_id":"R1",
  "probe_window":{"probe_id":"probe-1","start":1,"end":100},
  "schedule":{"day_ages":[1,2]},
  "thresholds":{
    "contamination":{"value":0,"scale":0},
    "maturity_min":{"value":700,"scale":1},
    "maturity_max":{"value":1000,"scale":1},
    "moisture_min":{"value":600,"scale":1},
    "moisture_max":{"value":700,"scale":1},
    "ph_min":{"value":550,"scale":2},
    "ph_max":{"value":650,"scale":2}
  },
  "reviewers":["alice","bob","carol","dave"]
}'

lock_resp="$(curl -s -X POST "${BASE}/v1/tasks/lock" \
  -H 'Content-Type: application/json' \
  -d "${LOCK_BODY}")"
if [[ "${lock_resp}" != *'"task_id"'* ]]; then
  echo "lock failed: ${lock_resp}" >&2
  exit 1
fi
echo "lock: ${lock_resp}"

echo "==> checking diagnostics"
diag="$(curl -s "${BASE}/v1/diagnostics")"
if [[ "${diag}" != *'"tasks_by_state"'* ]]; then
  echo "diagnostics failed: ${diag}" >&2
  exit 1
fi
echo "diagnostics: ${diag}"

echo "==> checking catalog"
catalog="$(curl -s "${BASE}/v1/catalog")"
if [[ "${catalog}" != *'"strains"'* ]]; then
  echo "catalog failed: ${catalog}" >&2
  exit 1
fi

echo "SMOKE OK"
