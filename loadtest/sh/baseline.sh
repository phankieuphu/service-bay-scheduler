#!/usr/bin/env bash
# Baseline: normal traffic at a constant rate (issue #38 §2 — 10 min).
# The numbers every later scenario is compared against.
#
# Usage: TIER=1k|100k|1m MIX=read|mixed|write ./sh/baseline.sh
# Overrides: RATE (ops/s), DURATION (e.g. 600, 10m)
set -uo pipefail
. "$(dirname "$0")/../lib/common.sh"
. "$(dirname "$0")/../lib/engine.sh"

RATE="${RATE:-$(tier_rate baseline)}"
DURATION="$(to_secs "${DURATION:-10m}")"

engine_init baseline
run_stages "$RATE" "$DURATION:$RATE:steady"
report
