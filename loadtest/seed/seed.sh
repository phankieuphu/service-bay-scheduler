#!/usr/bin/env bash
# Seeds dedicated load-test data into customer_db / vehicle_db and writes the
# resulting id ranges to loadtest/.env.seed, which every sh/k6 runner sources.
#
# All load-test rows are namespaced so they never touch the hand-written seed
# data in postgres/seed/:
#   customers  email LIKE 'lt-%@example.com'   ('lt-seed-*' = seeded here,
#                                               'lt-<run>-*' = created by tests)
#   vehicles   vin   LIKE 'LT%'
#
# Usage:
#   ./seed/seed.sh          # seed (idempotent) + reset transfer-pool ownership
#   ./seed/seed.sh reset    # only reset transfer-pool ownership back to owner A
#   ./seed/seed.sh clean    # delete every load-test row
#
# Connection (pick one):
#   PGHOST/PGPORT/PGUSER/PGPASSWORD         default localhost:5432 postgres/postgres
#   PSQL="docker exec -i postgres psql -U postgres"
#   PSQL="kubectl -n service-bay exec -i postgres-0 -- psql -U postgres"
#
# Sizes: SEED_CUSTOMERS (10000), SEED_VEHICLES (10000), TRANSFER_POOL (5000).
# TRANSFER_POOL must be >= the max VUs of any k6 run (each VU owns one vehicle).
set -euo pipefail

LOADTEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="$LOADTEST_DIR/.env.seed"

export PGHOST="${PGHOST:-localhost}" PGPORT="${PGPORT:-5432}"
export PGUSER="${PGUSER:-postgres}" PGPASSWORD="${PGPASSWORD:-postgres}"
PSQL="${PSQL:-psql}"

SEED_CUSTOMERS="${SEED_CUSTOMERS:-10000}"
SEED_VEHICLES="${SEED_VEHICLES:-10000}"
TRANSFER_POOL="${TRANSFER_POOL:-5000}"

if [ "$SEED_VEHICLES" -le "$TRANSFER_POOL" ]; then
  echo "SEED_VEHICLES ($SEED_VEHICLES) must be > TRANSFER_POOL ($TRANSFER_POOL)" >&2
  exit 1
fi

# q <db> <sql> — run SQL, print unaligned tuples only.
q() { $PSQL -d "$1" -v ON_ERROR_STOP=1 -qAt -c "$2"; }

check_customer_schema() {
  # GORM's soft-delete field (models.Customer.DeleteAt) needs this column;
  # postgres/init/02-customer-service.sql's ALTER has a typo ("AFTER TABLE"),
  # so a fresh volume may not have it and every customer query would 500.
  local has
  has=$(q customer_db "SELECT count(*) FROM information_schema.columns
                       WHERE table_name = 'customer' AND column_name = 'delete_at'")
  if [ "$has" != "1" ]; then
    echo "WARNING: customer.delete_at column is missing — customer-service reads will fail." >&2
    echo "         Fix with: ALTER TABLE customer ADD COLUMN delete_at timestamptz;" >&2
  fi
}

seed_customers() {
  echo "Seeding $SEED_CUSTOMERS customers..."
  q customer_db "
    INSERT INTO customer (name, email, phone, birthday, status)
    SELECT 'Load Test ' || g,
           'lt-seed-' || g || '@example.com',
           '+1555' || lpad(g::text, 7, '0'),
           date '1950-01-01' + (g * 37 % 20000),
           'ACTIVE'
    FROM generate_series(1, $SEED_CUSTOMERS) g
    ON CONFLICT (email) DO NOTHING" >/dev/null
}

seed_vehicles() {
  echo "Seeding $SEED_VEHICLES vehicles..."
  q vehicle_db "
    INSERT INTO vehicle_model (make, model, year) VALUES ('LoadTest', 'LT-1', 2024)
    ON CONFLICT (make, model, year) DO NOTHING;
    INSERT INTO vehicle (vin, license_plate, vehicle_model_id, warranty_end_date, status)
    SELECT 'LT' || lpad(g::text, 15, '0'),
           'LT-' || g,
           (SELECT id FROM vehicle_model WHERE make = 'LoadTest' AND model = 'LT-1' AND year = 2024),
           current_date + (g % 1500),
           'ACTIVE'
    FROM generate_series(1, $SEED_VEHICLES) g
    ON CONFLICT (vin) DO NOTHING" >/dev/null
}

# Ownership layout of LT vehicles (ordered by id):
#   rows 1..TRANSFER_POOL   -> customer A   (transfer pool, ping-ponged A<->B by tests)
#   row  TRANSFER_POOL+1    -> customer A   (reserved for smoke tests / race test)
#   remaining rows          -> spread over the other lt-seed customers
reset_ownership() {
  local a="$1" lt_min="$2" lt_count="$3"
  echo "Resetting ownership of LT vehicles (pool of $TRANSFER_POOL -> customer $a)..."
  q vehicle_db "
    BEGIN;
    DELETE FROM customer_vehicle
    WHERE vehicle_id IN (SELECT id FROM vehicle WHERE vin LIKE 'LT%');
    INSERT INTO customer_vehicle (customer_id, vehicle_id, owned_from, status)
    SELECT CASE WHEN rn <= $TRANSFER_POOL + 1 THEN $a
                ELSE $lt_min + 2 + (rn % GREATEST($lt_count - 2, 1)) END,
           id, current_date - 365, 'CURRENT'
    FROM (SELECT id, row_number() OVER (ORDER BY id) rn
          FROM vehicle WHERE vin LIKE 'LT%') v;
    COMMIT" >/dev/null
}

clean() {
  echo "Deleting load-test rows..."
  q vehicle_db "
    DELETE FROM customer_vehicle WHERE vehicle_id IN (SELECT id FROM vehicle WHERE vin LIKE 'LT%');
    DELETE FROM vehicle WHERE vin LIKE 'LT%';
    DELETE FROM vehicle_model WHERE make = 'LoadTest'" >/dev/null
  q customer_db "DELETE FROM customer WHERE email LIKE 'lt-%@example.com'" >/dev/null
  rm -f "$ENV_FILE"
  echo "Done. (outbox_message rows produced by tests are left in place.)"
}

write_env() {
  local cust_min cust_max lt_min lt_max lt_count veh_min veh_max pool_start pool_end smoke_vid
  cust_min=$(q customer_db "SELECT min(id) FROM customer")
  cust_max=$(q customer_db "SELECT max(id) FROM customer WHERE email LIKE 'lt-seed-%'")
  lt_min=$(q customer_db "SELECT min(id) FROM customer WHERE email LIKE 'lt-seed-%'")
  lt_max="$cust_max"
  lt_count=$(q customer_db "SELECT count(*) FROM customer WHERE email LIKE 'lt-seed-%'")

  veh_min=$(q vehicle_db "SELECT min(id) FROM vehicle")
  veh_max=$(q vehicle_db "SELECT max(id) FROM vehicle")
  pool_start=$(q vehicle_db "SELECT min(id) FROM vehicle WHERE vin LIKE 'LT%'")
  pool_end=$(q vehicle_db "SELECT id FROM vehicle WHERE vin LIKE 'LT%' ORDER BY id OFFSET $((TRANSFER_POOL - 1)) LIMIT 1")
  smoke_vid=$(q vehicle_db "SELECT id FROM vehicle WHERE vin LIKE 'LT%' ORDER BY id OFFSET $TRANSFER_POOL LIMIT 1")

  # Runners compute vehicle ids as start + offset, so the pool must be contiguous.
  if [ $((pool_end - pool_start + 1)) -ne "$TRANSFER_POOL" ]; then
    echo "ERROR: LT vehicle ids are not contiguous ($pool_start..$pool_end for a pool of $TRANSFER_POOL)." >&2
    echo "       Run '$0 clean' then '$0' to re-seed in one batch." >&2
    exit 1
  fi

  local a=$lt_min b=$((lt_min + 1))
  reset_ownership "$a" "$lt_min" "$lt_count"

  # ': ${VAR:=value}' so values already exported by the caller win.
  cat >"$ENV_FILE" <<EOF
# Generated by seed/seed.sh on $(date -u +%Y-%m-%dT%H:%M:%SZ) — do not edit.
: "\${CUSTOMER_ID_MIN:=$cust_min}"
: "\${CUSTOMER_ID_MAX:=$cust_max}"
: "\${LT_CUSTOMER_MIN:=$lt_min}"
: "\${LT_CUSTOMER_MAX:=$lt_max}"
: "\${VEHICLE_ID_MIN:=$veh_min}"
: "\${VEHICLE_ID_MAX:=$veh_max}"
: "\${TRANSFER_VEHICLE_START:=$pool_start}"
: "\${TRANSFER_POOL:=$TRANSFER_POOL}"
: "\${TRANSFER_CUSTOMER_A:=$a}"
: "\${TRANSFER_CUSTOMER_B:=$b}"
: "\${SMOKE_VEHICLE_ID:=$smoke_vid}"
EOF
  echo "Wrote $ENV_FILE:"
  grep -v '^#' "$ENV_FILE" | sed 's/^: "\${\([A-Z_]*\):=\(.*\)}"$/  \1=\2/'
}

case "${1:-seed}" in
  seed)
    check_customer_schema
    seed_customers
    seed_vehicles
    write_env
    ;;
  reset)
    [ -f "$ENV_FILE" ] || { echo "No $ENV_FILE — run '$0' first." >&2; exit 1; }
    # The seeded layout is the truth here, not this run's defaults/env.
    unset TRANSFER_POOL TRANSFER_CUSTOMER_A LT_CUSTOMER_MIN LT_CUSTOMER_MAX
    # shellcheck source=/dev/null
    . "$ENV_FILE"
    lt_count=$((LT_CUSTOMER_MAX - LT_CUSTOMER_MIN + 1))
    reset_ownership "$TRANSFER_CUSTOMER_A" "$LT_CUSTOMER_MIN" "$lt_count"
    ;;
  clean)
    clean
    ;;
  *)
    echo "usage: $0 [seed|reset|clean]" >&2
    exit 2
    ;;
esac
