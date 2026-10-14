#!/usr/bin/env bash
# Runs the "Full flow" folder of service-bay.postman_collection.json with newman
# against a running stack (docker compose up -d), and fails on any failed
# assertion. Card steps need Stripe test keys on billing-service; without them
# they're reported as skipped.
#
#   postman/run-flow.sh                                   # the whole flow
#   BILLING_URL=http://localhost:18084 postman/run-flow.sh # other ports
#   postman/run-flow.sh --reporters cli,junit             # extra args go to newman
set -euo pipefail
cd "$(dirname "$0")"

NEWMAN_VERSION=6.2.2

if ! command -v npx >/dev/null 2>&1; then
  echo "npx not found: install Node.js 18+ (newman runs through npx)." >&2
  exit 1
fi

exec npx --yes "newman@${NEWMAN_VERSION}" run service-bay.postman_collection.json \
  --folder "Full flow" \
  --env-var "identityUrl=${IDENTITY_URL:-http://localhost:8083}" \
  --env-var "customerUrl=${CUSTOMER_URL:-http://localhost:8080}" \
  --env-var "vehicleUrl=${VEHICLE_URL:-http://localhost:8081}" \
  --env-var "schedulerUrl=${SCHEDULER_URL:-http://localhost:8085}" \
  --env-var "billingUrl=${BILLING_URL:-http://localhost:8084}" \
  "$@"
