#!/usr/bin/env bash
# Fires a mix of requests at the vehicle-service endpoints to generate
# Prometheus metrics data for local testing.
#
# Usage: ./scripts/fake-traffic.sh [base_url] [rounds]
set -euo pipefail

BASE_URL="${1:-http://localhost:8081}/api/v1"
ROUNDS="${2:-20000}"

for i in $(seq 1 "$ROUNDS"); do
  # GET /customers/:id/vehicles - vary the customer id
  curl -s -o /dev/null -w "GET /customers/$i/vehicles -> %{http_code}\n" \
    "$BASE_URL/customers/$i/vehicles"

  # GET /vehicle (list, first page)
  curl -s -o /dev/null -w "GET /vehicle -> %{http_code}\n" \
    "$BASE_URL/vehicle?limit=20"

  # GET /vehicle/:id - vary the id (404 past the last seeded vehicle)
  curl -s -o /dev/null -w "GET /vehicle/$i -> %{http_code}\n" \
    "$BASE_URL/vehicle/$i"

  # POST /transfer - alternate between a well-formed body (hits the real
  # service/DB, likely 404/409/500 for made-up ids) and a bad body (400)
  if (( i % 3 == 0 )); then
    curl -s -o /dev/null -w "POST /transfer (bad body) -> %{http_code}\n" \
      -X POST -H "Content-Type: application/json" \
      -d '{}' \
      "$BASE_URL/transfer"
  else
    curl -s -o /dev/null -w "POST /transfer -> %{http_code}\n" \
      -X POST -H "Content-Type: application/json" \
      -d "{\"vehicle_id\": $i, \"from\": 1, \"to\": 2}" \
      "$BASE_URL/transfer"
  fi

  sleep 0.2
done

echo
echo "Done. Scrape metrics with:"
echo "  curl ${1:-http://localhost:8081}/metrics | grep http_requests_total"
