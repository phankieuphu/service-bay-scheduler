#!/usr/bin/env bash
# Spike: jump from baseline to 10x baseline in < 30s, hold, drop back, and
# check the system recovers (issue #38 §2). The SLO verdict is taken over
# the warm-up and recovery phases; the peak phase is reported but only has
# to not fall over.
#
# Usage: TIER=1k|100k|1m MIX=read|mixed|write ./sh/spike.sh
# Overrides: BASE_RATE, SPIKE_RATE (ops/s), PEAK_DURATION (3m),
#            RECOVERY_DURATION (5m)
set -uo pipefail
. "$(dirname "$0")/../lib/common.sh"
. "$(dirname "$0")/../lib/engine.sh"

BASE_RATE="${BASE_RATE:-$(tier_rate baseline)}"
SPIKE_RATE="${SPIKE_RATE:-$(tier_rate spike)}"
PEAK_DURATION="$(to_secs "${PEAK_DURATION:-3m}")"
RECOVERY_DURATION="$(to_secs "${RECOVERY_DURATION:-5m}")"

engine_init spike
run_stages "$BASE_RATE" \
  "60:$BASE_RATE:warmup" \
  "20:$SPIKE_RATE:spike_up" \
  "$PEAK_DURATION:$SPIKE_RATE:peak" \
  "20:$BASE_RATE:spike_down" \
  "$RECOVERY_DURATION:$BASE_RATE:recovery"
report "warmup recovery"
