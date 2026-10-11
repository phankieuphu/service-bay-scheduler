#!/bin/sh
# Loads the dev seed data (init-customer.sql, init-vehicle.sql) into
# customer_db and vehicle_db. The tables belong to the services' migrations,
# so this waits until customer-service and vehicle-service have created
# them, and skips a table that already has rows, so it's safe to re-run.
#
# docker-compose runs it as the db-seed service. Elsewhere:
#   PGHOST=localhost PGUSER=postgres PGPASSWORD=postgres sh postgres/seed/seed.sh
#   kubectl -n service-bay exec postgres-0 -- sh /seed/seed.sh
set -eu

DIR=$(cd "$(dirname "$0")" && pwd)
PSQL="psql -X -q -v ON_ERROR_STOP=1 -U ${PGUSER:-postgres}"
export PGOPTIONS="${PGOPTIONS:--c client_min_messages=warning}"
TIMEOUT_SEC="${SEED_TIMEOUT_SEC:-180}"

# wait_for <db> <table>
wait_for() {
  waited=0
  until [ "$($PSQL -d "$1" -Atc "SELECT to_regclass('$2') IS NOT NULL" 2>/dev/null)" = t ]; do
    if [ "$waited" -ge "$TIMEOUT_SEC" ]; then
      echo "seed: $1.$2 still doesn't exist after ${TIMEOUT_SEC}s; is the service running its migrations?" >&2
      exit 1
    fi
    sleep 2
    waited=$((waited + 2))
  done
}

# seed <db> <table> <file>
seed() {
  wait_for "$1" "$2"
  rows=$($PSQL -d "$1" -Atc "SELECT count(*) FROM $2")
  if [ "$rows" -gt 0 ]; then
    echo "seed: $1.$2 already has $rows rows; skipping $(basename "$3")"
    return
  fi
  echo "seed: loading $(basename "$3") into $1"
  $PSQL -d "$1" -f "$3" >/dev/null
}

seed customer_db customer "$DIR/init-customer.sql"
seed vehicle_db vehicle "$DIR/init-vehicle.sql"
echo "seed: done"
