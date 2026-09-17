#!/usr/bin/env bash
# Fires 3 concurrent POST /api/v1/transfer requests for the SAME vehicle to
# 3 different target customers, to verify TransferVehicle's row-locking
# correctly lets exactly one win and rejects the other two.
#
# Usage: ./transfer-race-test.sh
# Env overrides: BASE_URL, VEHICLE_ID, FROM_CUSTOMER, TO_CUSTOMERS

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8081}"

# Edit these for each test run.
# FROM_CUSTOMER must be the vehicle's *current* owner (status='CURRENT' in
# customer_vehicle) or every request will 404. This drifts as you re-run the
# script (ownership moves to whichever TO_CUSTOMERS entry wins), so check:
#   docker exec postgres psql -U postgres -d vehicle_db -c \
#     "select customer_id from customer_vehicle where vehicle_id=$VEHICLE_ID and status='CURRENT'"
VEHICLE_ID="${VEHICLE_ID:-50}"
FROM_CUSTOMER="${FROM_CUSTOMER:-101}"
TO_CUSTOMERS=(${TO_CUSTOMERS:-104 102 103})
DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

OUT_DIR="$(mktemp -d)"
echo "Vehicle $VEHICLE_ID: from $FROM_CUSTOMER -> {${TO_CUSTOMERS[*]}} (concurrent)"
echo "Results dir: $OUT_DIR"
echo

pids=()
for to in "${TO_CUSTOMERS[@]}"; do
  (
    body=$(curl -s -o "$OUT_DIR/$to.body" -w '%{http_code}' \
      -X POST "$BASE_URL/api/v1/transfer" \
      -H 'Content-Type: application/json' \
      -d "{\"vehicle_id\":$VEHICLE_ID,\"from\":$FROM_CUSTOMER,\"to\":$to,\"date\":\"$DATE\"}")
    echo "$body" > "$OUT_DIR/$to.status"
  ) &
  pids+=($!)
done

for pid in "${pids[@]}"; do
  wait "$pid"
done

echo "----- results -----"
winners=0
for to in "${TO_CUSTOMERS[@]}"; do
  status="$(cat "$OUT_DIR/$to.status")"
  bodyfile="$OUT_DIR/$to.body"
  body="$(cat "$bodyfile" 2>/dev/null || true)"
  printf 'to=%-6s status=%-3s body=%s\n' "$to" "$status" "$body"
  if [ "$status" = "204" ]; then
    winners=$((winners + 1))
  fi
done

echo
echo "winners (204 No Content): $winners"
if [ "$winners" -eq 1 ]; then
  echo "PASS: exactly one transfer won the race."
else
  echo "FAIL: expected exactly 1 winner, got $winners."
fi
