#!/usr/bin/env bash
# Load: expected peak traffic (issue #38 §2 — 30 min). Ramp up, hold, ramp
# down; the SLO verdict is taken over the hold phase only.
#
# Usage: TIER=1k|100k|1m MIX=read|mixed|write ./sh/load.sh
# Overrides: RATE (ops/s), DURATION (hold, e.g. 30m), RAMP (e.g. 2m)
set -uo pipefail
. "$(dirname "$0")/../lib/common.sh"
. "$(dirname "$0")/../lib/engine.sh"

RATE="${RATE:-$(tier_rate load)}"
DURATION="$(to_secs "${DURATION:-30m}")"
RAMP="$(to_secs "${RAMP:-2m}")"

engine_init load
run_stages 0 \
  "$RAMP:$RATE:ramp_up" \
  "$DURATION:$RATE:hold" \
  "60:0:ramp_down"
report "hold"
