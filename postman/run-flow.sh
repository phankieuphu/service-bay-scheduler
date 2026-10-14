#!/usr/bin/env bash
# Runs the "Full flow" folder of service-bay.postman_collection.json with newman
# against a running stack (docker compose up -d), and fails on any failed assertion.
#
#   postman/run-flow.sh                       # every flow
#   FLOW="4. Checkout validation" postman/run-flow.sh   # one flow (exact folder name)
#   BILLING_URL=http://localhost:58084 postman/run-flow.sh   # point at other ports
#
# Card steps need Stripe test keys on payment-service (STRIPE_SECRET_KEY=sk_test_...);
# without them they are reported as skipped and the cash/void steps still run.
# Extra arguments go straight to newman, e.g. --reporters cli,junit.
set -euo pipefail
cd "$(dirname "$0")"

NEWMAN_VERSION=6.2.2

if ! command -v npx >/dev/null 2>&1; then
  echo "npx not found: install Node.js 18+ (newman runs through npx)." >&2
  exit 1
fi

exec npx --yes "newman@${NEWMAN_VERSION}" run service-bay.postman_collection.json \
  --folder "${FLOW:-Full flow}" \
  --env-var "identityUrl=${IDENTITY_URL:-http://localhost:8083}" \
  --env-var "customerUrl=${CUSTOMER_URL:-http://localhost:8080}" \
  --env-var "vehicleUrl=${VEHICLE_URL:-http://localhost:8081}" \
  --env-var "billingUrl=${BILLING_URL:-http://localhost:8084}" \
  --env-var "paymentUrl=${PAYMENT_URL:-http://localhost:8085}" \
  --color on \
  "$@"
