#!/usr/bin/env bash
# Runs a k6 scenario with the seeded id ranges and settings from the
# environment passed through as -e flags.
#
# Usage: ./k6/run.sh <smoke|baseline|load|stress|spike|soak> [extra k6 args]
#   TIER=100k MIX=write ./k6/run.sh load
#   RATE=200 DURATION=5m ./k6/run.sh baseline
#   ./k6/run.sh stress --out experimental-prometheus-rw   # push to Prometheus
set -uo pipefail
. "$(dirname "$0")/../lib/common.sh"

SCENARIO="${1:-}"
[ -f "$LOADTEST_DIR/k6/$SCENARIO.js" ] || die "usage: $0 <smoke|baseline|load|stress|spike|soak> [k6 args]"
shift
command -v k6 >/dev/null || die "k6 not installed — https://grafana.com/docs/k6/latest/set-up/install-k6/"

require_up
[ "$SCENARIO" = smoke ] || reset_transfer_pool

RESULTS_DIR="${RESULTS_DIR:-$LOADTEST_DIR/results}"
SLO_ERR_RATE="${SLO_ERR_RATE:-$(awk -v p="$SLO_ERR_PCT" 'BEGIN { print p / 100 }')}"
mkdir -p "$RESULTS_DIR"

args=()
for var in CUSTOMER_URL VEHICLE_URL TIER MIX \
  CUSTOMER_ID_MIN CUSTOMER_ID_MAX LT_CUSTOMER_MIN LT_CUSTOMER_MAX VEHICLE_ID_MIN VEHICLE_ID_MAX \
  TRANSFER_VEHICLE_START TRANSFER_POOL TRANSFER_CUSTOMER_A TRANSFER_CUSTOMER_B SMOKE_VEHICLE_ID \
  RATE DURATION RAMP MAX_RATE STEPS STEP_DURATION BASE_RATE SPIKE_RATE PEAK_DURATION RECOVERY_DURATION \
  SLO_P95_MS SLO_P99_MS SLO_ERR_RATE PROBE_WARMUP_SAMPLES RESULTS_DIR; do
  eval "val=\${$var:-}"
  # shellcheck disable=SC2154
  [ -n "$val" ] && args+=(-e "$var=$val")
done
args+=(-e "RUN_ID=$(date +%Y%m%d%H%M%S)")

log "k6 run $SCENARIO (tier=$TIER mix=$MIX) -> $RESULTS_DIR"
exec k6 run "${args[@]}" "$@" "$LOADTEST_DIR/k6/$SCENARIO.js"
