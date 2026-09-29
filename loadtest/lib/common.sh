#!/usr/bin/env bash
# Shared config for loadtest/ sh runners. Sourced, not executed.
# Bash 3.2 compatible (macOS /bin/bash): no associative arrays, no mapfile.

LOADTEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Seeded id ranges (see seed/seed.sh). Values already in the environment win.
if [ -f "$LOADTEST_DIR/.env.seed" ]; then
  # shellcheck source=/dev/null
  . "$LOADTEST_DIR/.env.seed"
fi

CUSTOMER_URL="${CUSTOMER_URL:-http://localhost:8080}"
VEHICLE_URL="${VEHICLE_URL:-http://localhost:8081}"

# Fallbacks when seed/seed.sh hasn't run: ranges of postgres/seed/*.sql, and
# transfers disabled (they need a pool of vehicles with known ownership).
CUSTOMER_ID_MIN="${CUSTOMER_ID_MIN:-1}"
CUSTOMER_ID_MAX="${CUSTOMER_ID_MAX:-1000}"
LT_CUSTOMER_MIN="${LT_CUSTOMER_MIN:-$CUSTOMER_ID_MIN}"
LT_CUSTOMER_MAX="${LT_CUSTOMER_MAX:-$CUSTOMER_ID_MAX}"
VEHICLE_ID_MIN="${VEHICLE_ID_MIN:-1}"
VEHICLE_ID_MAX="${VEHICLE_ID_MAX:-5000}"
TRANSFER_VEHICLE_START="${TRANSFER_VEHICLE_START:-0}"
TRANSFER_POOL="${TRANSFER_POOL:-0}"
TRANSFER_CUSTOMER_A="${TRANSFER_CUSTOMER_A:-0}"
TRANSFER_CUSTOMER_B="${TRANSFER_CUSTOMER_B:-0}"
SMOKE_VEHICLE_ID="${SMOKE_VEHICLE_ID:-0}"

# SLO from issue #38's acceptance criteria.
SLO_P95_MS="${SLO_P95_MS:-200}"
SLO_P99_MS="${SLO_P99_MS:-500}"
SLO_ERR_PCT="${SLO_ERR_PCT:-0.1}"

TIER="$(printf '%s' "${TIER:-1k}" | tr 'A-Z' 'a-z')"
MIX="${MIX:-mixed}"

# tier_rate <scenario> — target operations/s for the current TIER, from the
# traffic model in issue #38 (1k / 100k / 1M registered users). "stress" is
# the ceiling the ramp climbs to (2x the tier's stress target), so the break
# point is found rather than assumed. Keep in sync with k6/lib/config.js.
tier_rate() {
  case "$TIER:$1" in
    1k:baseline) echo 5 ;;     1k:load) echo 10 ;;     1k:stress) echo 50 ;;
    1k:spike) echo 50 ;;       1k:soak) echo 5 ;;
    100k:baseline) echo 30 ;;  100k:load) echo 100 ;;  100k:stress) echo 500 ;;
    100k:spike) echo 300 ;;    100k:soak) echo 50 ;;
    1m:baseline) echo 300 ;;   1m:load) echo 1000 ;;   1m:stress) echo 5000 ;;
    1m:spike) echo 3000 ;;     1m:soak) echo 500 ;;
    *) die "unknown TIER/scenario: $TIER/$1 (TIER must be 1k, 100k or 1m)" ;;
  esac
}

# Operation mix, in percent. Order matches OPS below. "mixed" is the 90/10
# read/write split assumed in issue #38. Keep in sync with k6/lib/mix.js.
OPS="cust_get cust_list veh_get cust_create cust_update cust_delete transfer"
mix_weights() {
  case "$MIX" in
    read)  echo "40 15 45  0  0 0  0" ;;
    mixed) echo "36 13 41  3  3 1  3" ;;
    write) echo " 8  3  9 25 25 5 25" ;;
    *) die "unknown MIX: $MIX (read|mixed|write)" ;;
  esac
}

transfers_enabled() {
  [ "$TRANSFER_POOL" -gt 0 ] && [ "$TRANSFER_CUSTOMER_A" -gt 0 ] && [ "$TRANSFER_CUSTOMER_B" -gt 0 ]
}

die() { echo "ERROR: $*" >&2; exit 1; }
log() { printf '[%s] %s\n' "$(date +%H:%M:%S)" "$*" >&2; }

# now_ms — wall clock in milliseconds (bash 3.2 has no EPOCHREALTIME and
# BSD date has no %N, so fall back to perl, which macOS ships).
if [ -n "${EPOCHREALTIME:-}" ]; then
  now_ms() { local t="${EPOCHREALTIME/[.,]/}"; echo $((t / 1000)); }
elif date +%s%3N 2>/dev/null | grep -q '^[0-9]*$'; then
  now_ms() { date +%s%3N; }
elif command -v perl >/dev/null 2>&1; then
  now_ms() { perl -MTime::HiRes=time -e 'printf "%d\n", time * 1000'; }
else
  now_ms() { echo $(($(date +%s) * 1000)); }
fi

# sleep_ms <ms> — no-op for ms <= 0.
sleep_ms() {
  [ "$1" -gt 0 ] || return 0
  sleep "$(printf '%d.%03d' $(($1 / 1000)) $(($1 % 1000)))"
}

# require_up — fail fast if either service isn't answering.
require_up() {
  local url
  for url in "$CUSTOMER_URL" "$VEHICLE_URL"; do
    curl -s -o /dev/null -m 5 "$url/metrics" || die "$url is not reachable"
  done
}

# reset_transfer_pool — put every transfer-pool vehicle back on owner A so
# the deterministic A<->B schedule below starts in sync with the DB. Set
# RESET_OWNERSHIP=0 to skip (e.g. no DB access from where the test runs).
reset_transfer_pool() {
  transfers_enabled || return 0
  [ "${RESET_OWNERSHIP:-1}" = "1" ] || return 0
  if ! "$LOADTEST_DIR/seed/seed.sh" reset >/dev/null 2>&1; then
    log "WARN: could not reset transfer-pool ownership (seed/seed.sh reset failed);"
    log "      transfers may return 409 until the A<->B schedule re-syncs."
  fi
}
