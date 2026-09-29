#!/usr/bin/env bash
# Smoke: one request per API case, with assertions. Run it before any load
# scenario — a load test against a broken endpoint only measures how fast it
# fails. Exits non-zero if any case fails.
#
# Usage: ./sh/smoke.sh
# Uses SMOKE_VEHICLE_ID / TRANSFER_CUSTOMER_A/B from seed/seed.sh for the
# transfer cases (skipped if not seeded).
set -uo pipefail
. "$(dirname "$0")/../lib/common.sh"

C="$CUSTOMER_URL/api/v1"
V="$VEHICLE_URL/api/v1"
RUN="$(date +%s)-$$"
pass=0 fail=0 skip=0

# req <method> <url> [json-body] — sets STATUS and BODY.
req() {
  local out
  if [ -n "${3:-}" ]; then
    out=$(curl -s -m 10 -w '\n%{http_code}' -X "$1" -H 'Content-Type: application/json' -d "$3" "$2")
  else
    out=$(curl -s -m 10 -w '\n%{http_code}' -X "$1" "$2")
  fi
  STATUS="${out##*$'\n'}"
  BODY="${out%$'\n'*}"
}

# expect <name> <status...> — asserts STATUS is one of the given codes.
expect() {
  local name="$1" code
  shift
  for code in "$@"; do
    if [ "$STATUS" = "$code" ]; then
      printf '  PASS  %-58s %s\n' "$name" "$STATUS"
      pass=$((pass + 1))
      return 0
    fi
  done
  printf '  FAIL  %-58s got %s, want %s  %s\n' "$name" "$STATUS" "$*" "$(printf '%s' "$BODY" | head -c 160)"
  fail=$((fail + 1))
  return 1
}

skip() { printf '  SKIP  %-58s %s\n' "$1" "$2"; skip=$((skip + 1)); }

# json_field <key> — scalar value of a top-level key in BODY (no jq needed).
json_field() {
  printf '%s' "$BODY" | sed -n "s/.*\"$1\"[[:space:]]*:[[:space:]]*\"\\{0,1\\}\\([^\",}]*\\).*/\\1/p"
}

echo "== infra"
req GET "$CUSTOMER_URL/metrics"; expect "customer-service GET /metrics" 200
req GET "$VEHICLE_URL/metrics";  expect "vehicle-service GET /metrics" 200
[ "$fail" -eq 0 ] || { echo "services not reachable — aborting"; exit 1; }

echo "== customer-service: create"
EMAIL="lt-smoke-$RUN@example.com"
req POST "$C/customer" "{\"name\":\"Smoke Test\",\"email\":\"$EMAIL\",\"phone\":\"+15550001111\",\"birth_day\":\"1990-01-01T00:00:00Z\"}"
expect "POST /customer valid -> 201" 201
ID=$(json_field id)
UPDATED=$(json_field updated_at)
req POST "$C/customer" "{\"name\":\"Smoke Test\",\"email\":\"$EMAIL\",\"birth_day\":\"1990-01-01T00:00:00Z\"}"
expect "POST /customer duplicate email -> 409" 409
req POST "$C/customer" "{\"email\":\"lt-smoke-$RUN-x@example.com\",\"birth_day\":\"1990-01-01T00:00:00Z\"}"
expect "POST /customer missing name -> 400" 400
req POST "$C/customer" "{\"name\":\"x\",\"email\":\"not-an-email\",\"birth_day\":\"1990-01-01T00:00:00Z\"}"
expect "POST /customer invalid email -> 400" 400
req POST "$C/customer" "{\"name\":\"x\",\"email\":\"lt-smoke-$RUN-y@example.com\"}"
expect "POST /customer missing birth_day -> 400" 400
req POST "$C/customer" "not json"
expect "POST /customer malformed json -> 400" 400

echo "== customer-service: read"
if [ -n "$ID" ]; then
  req GET "$C/customer/$ID"; expect "GET /customer/:id existing -> 200" 200
else
  skip "GET /customer/:id existing -> 200" "(create failed)"
fi
req GET "$C/customer/999999999"; expect "GET /customer/:id unknown -> 404" 404
req GET "$C/customer/abc";       expect "GET /customer/:id non-numeric -> 400" 400
req GET "$C/customer?limit=5"
if expect "GET /customer first page -> 200" 200; then
  NEXT=$(json_field next_cursor)
  if [ -n "$NEXT" ]; then
    req GET "$C/customer?cursor=$NEXT&limit=5"; expect "GET /customer?cursor=next_cursor -> 200" 200
  else
    skip "GET /customer?cursor=next_cursor -> 200" "(single page)"
  fi
fi
req GET "$C/customer?limit=1000"; expect "GET /customer limit over max (capped) -> 200" 200
req GET "$C/customer?cursor=-1";  expect "GET /customer negative cursor -> 400" 400
req GET "$C/customer?limit=abc";  expect "GET /customer non-numeric limit -> 400" 400

echo "== customer-service: update (optimistic lock on updated_at)"
if [ -n "$ID" ] && [ -n "$UPDATED" ]; then
  req PUT "$C/customer/$ID" "{\"name\":\"Smoke Test Renamed\",\"updated_at\":\"$UPDATED\"}"
  expect "PUT /customer/:id current updated_at -> 204" 204
  req PUT "$C/customer/$ID" "{\"name\":\"Smoke Test Stale\",\"updated_at\":\"$UPDATED\"}"
  expect "PUT /customer/:id stale updated_at -> 409" 409
  req GET "$C/customer/$ID"; UPDATED=$(json_field updated_at)
  req PUT "$C/customer/$ID" "{\"updated_at\":\"$UPDATED\"}"
  expect "PUT /customer/:id nothing to update -> 400" 400
  req PUT "$C/customer/$ID" "{\"name\":\"x\"}"
  expect "PUT /customer/:id missing updated_at -> 400" 400
  req PUT "$C/customer/999999999" "{\"name\":\"x\",\"updated_at\":\"$UPDATED\"}"
  expect "PUT /customer/:id unknown -> 404" 404
  req PUT "$C/customer/abc" "{\"name\":\"x\",\"updated_at\":\"$UPDATED\"}"
  expect "PUT /customer/:id non-numeric -> 400" 400

  # Two concurrent PUTs with the same updated_at: exactly one may win.
  req GET "$C/customer/$ID"; UPDATED=$(json_field updated_at)
  tmp=$(mktemp -d)
  for n in 1 2 3; do
    curl -s -o /dev/null -w '%{http_code}' -X PUT -H 'Content-Type: application/json' \
      -d "{\"name\":\"Race $n\",\"updated_at\":\"$UPDATED\"}" "$C/customer/$ID" >"$tmp/$n" &
  done
  wait
  wins=$(cat "$tmp"/* | grep -o 204 | wc -l | tr -d ' ')
  rm -rf "$tmp"
  STATUS="$wins winner(s)"; BODY=""
  [ "$wins" = "1" ] && STATUS=1
  expect "PUT /customer/:id x3 concurrent -> exactly 1 winner" 1
else
  skip "PUT /customer/:id cases" "(create failed)"
fi

echo "== customer-service: soft delete"
if [ -n "$ID" ]; then
  req DELETE "$C/customer/$ID"; expect "DELETE /customer/:id -> 204" 204
  req GET "$C/customer/$ID";    expect "GET /customer/:id after delete -> 404" 404
  req DELETE "$C/customer/$ID"; expect "DELETE /customer/:id already deleted -> 404" 404
else
  skip "DELETE /customer/:id cases" "(create failed)"
fi
req DELETE "$C/customer/abc"; expect "DELETE /customer/:id non-numeric -> 400" 400

echo "== vehicle-service: read"
req GET "$V/vehicle/$VEHICLE_ID_MIN"; expect "GET /vehicle/:id existing -> 200" 200
req GET "$V/vehicle/999999999";       expect "GET /vehicle/:id unknown -> 404" 404
req GET "$V/vehicle/abc";             expect "GET /vehicle/:id non-numeric -> 400" 400

echo "== vehicle-service: transfer"
req POST "$V/transfer" "{}";          expect "POST /transfer empty body -> 400" 400
req POST "$V/transfer" "not json";    expect "POST /transfer malformed json -> 400" 400
req POST "$V/transfer" '{"vehicle_id":999999999,"from":1,"to":2}'
expect "POST /transfer unknown vehicle -> 404" 404

if [ "$SMOKE_VEHICLE_ID" -gt 0 ] && transfers_enabled; then
  VID="$SMOKE_VEHICLE_ID" A="$TRANSFER_CUSTOMER_A" B="$TRANSFER_CUSTOMER_B"
  # Normalise: if an earlier run died with B as owner, move it back to A.
  req POST "$V/transfer" "{\"vehicle_id\":$VID,\"from\":$B,\"to\":$A}"

  req POST "$V/transfer" "{\"vehicle_id\":$VID,\"from\":$B,\"to\":$A}"
  expect "POST /transfer from non-owner -> 409" 409
  req POST "$V/transfer" "{\"vehicle_id\":$VID,\"from\":$A,\"to\":$B,\"date\":\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}"
  expect "POST /transfer A->B (with date) -> 204" 204
  req POST "$V/transfer" "{\"vehicle_id\":$VID,\"from\":$B,\"to\":$A}"
  expect "POST /transfer B->A (no date) -> 204" 204

  # Row lock in UnassignVehicleFromCustomer: 3 concurrent transfers of the
  # same vehicle from its owner — exactly one may win.
  tmp=$(mktemp -d)
  for to in "$B" $((B + 1)) $((B + 2)); do
    curl -s -o /dev/null -w '%{http_code}' -X POST -H 'Content-Type: application/json' \
      -d "{\"vehicle_id\":$VID,\"from\":$A,\"to\":$to}" "$V/transfer" >"$tmp/$to" &
  done
  wait
  wins=$(cat "$tmp"/* | grep -o 204 | wc -l | tr -d ' ')
  for to in "$B" $((B + 1)) $((B + 2)); do
    [ "$(cat "$tmp/$to")" = "204" ] && winner="$to"
  done
  rm -rf "$tmp"
  STATUS="$wins winner(s)"; BODY=""
  [ "$wins" = "1" ] && STATUS=1
  expect "POST /transfer x3 concurrent same vehicle -> exactly 1 winner" 1
  # restore ownership to A for the next run
  [ -n "${winner:-}" ] && req POST "$V/transfer" "{\"vehicle_id\":$VID,\"from\":$winner,\"to\":$A}"
else
  skip "POST /transfer ownership cases" "(run seed/seed.sh first)"
fi

echo
echo "passed=$pass failed=$fail skipped=$skip"
[ "$fail" -eq 0 ]
