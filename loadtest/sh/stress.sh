#!/usr/bin/env bash
# Stress: step the rate up until the SLO breaks, and record the max rate that
# still met it (issue #38 §2). Each step ramps for 15s then holds; every 10s
# the last completed 10s window is checked, and after BREAK_CHECKS
# consecutive breaches (or dropped ticks) the run stops.
#
# Usage: TIER=1k|100k|1m MIX=read|mixed|write ./sh/stress.sh
# Overrides: MAX_RATE (ops/s ceiling), STEPS (10), STEP_DURATION (2m),
#            BREAK_CHECKS (3)
#
# For a per-pod RPS/vCPU number (the input to the HPA sizing in issue #38),
# scale the service to 1 replica first.
set -uo pipefail
. "$(dirname "$0")/../lib/common.sh"
. "$(dirname "$0")/../lib/engine.sh"

MAX_RATE="${MAX_RATE:-$(tier_rate stress)}"
STEPS="${STEPS:-10}"
STEP_DURATION="$(to_secs "${STEP_DURATION:-2m}")"
BREAK_CHECKS="${BREAK_CHECKS:-3}"

breaches=0
last_good_rate=0

# check_slo <tick> — called after every tick by run_stages.
check_slo() {
  local tick="$1" from to n p95 err rate drops
  [ $((tick % 10)) -eq 0 ] && [ "$tick" -ge 20 ] || return 0
  # Leave 5s for the window's slowest requests to land in the log.
  from=$((tick - 15)); to=$((tick - 5))
  read -r n p95 err <<<"$(window_stats "$from" "$to")"
  rate=$(awk -v t="$from" '$1 == t { print $3 }' "$RESULTS/schedule.log")
  drops=$(awk -v a="$from" -v b="$to" '$1 >= a && $1 < b' "$RESULTS/dropped.log" | wc -l | tr -d ' ')
  if [ "$drops" -gt 0 ] || awk -v e="$err" -v p="$p95" -v se="$SLO_ERR_PCT" -v sp="$SLO_P95_MS" \
       'BEGIN { exit !(e >= se || p >= sp) }'; then
    breaches=$((breaches + 1))
    log "window@${rate}ops/s: p95=${p95}ms err=${err}% dropped=${drops} -> BREACH $breaches/$BREAK_CHECKS"
    if [ "$breaches" -ge "$BREAK_CHECKS" ]; then
      echo "breaking_rate=$rate last_good_rate=$last_good_rate p95_ms=$p95 err_pct=$err dropped_ticks=$drops" \
        >"$RESULTS/break.txt"
      touch "$STOP_FILE"
    fi
  else
    breaches=0
    last_good_rate="$rate"
    log "window@${rate}ops/s: p95=${p95}ms err=${err}% n=$n -> ok"
  fi
}

engine_init stress
stages=()
i=1
while [ "$i" -le "$STEPS" ]; do
  target=$((MAX_RATE * i / STEPS))
  stages+=("15:$target:step$i" "$((STEP_DURATION - 15)):$target:step$i")
  i=$((i + 1))
done

ON_TICK=check_slo
run_stages 0 "${stages[@]}"

SLO_ENFORCE=0 report
echo
if [ -f "$RESULTS/break.txt" ]; then
  echo "BREAK POINT: $(cat "$RESULTS/break.txt")" | tee -a "$RESULTS/report.txt"
else
  echo "NO BREAK up to $MAX_RATE ops/s (last good window: $last_good_rate ops/s) — raise MAX_RATE." |
    tee -a "$RESULTS/report.txt"
fi
