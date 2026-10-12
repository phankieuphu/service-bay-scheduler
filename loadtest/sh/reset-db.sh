#!/usr/bin/env bash
# Rebuilds Postgres in the k8s cluster from scratch and gets the stack back to
# a state a load test can run against:
#
#   1. rebuild service-bay-postgres:local from postgres/ and load it into the
#      kind node (the CREATE DATABASE init script is baked into the image)
#   2. WIPE the postgres PVC so /docker-entrypoint-initdb.d runs again
#      (Postgres skips init whenever PGDATA already exists)
#   3. restart customer/vehicle/identity-service: they reconnect and apply
#      their migrations to the empty databases
#   4. verify the schema actually landed (customer.delete_at, vehicle table)
#   5. restart the localhost:8080/8081 port-forwards (they die with the pods)
#   6. seed load-test data (seed/seed.sh) and run sh/smoke.sh
#
# DESTRUCTIVE: every database in the postgres pod is dropped (incl. identity_db).
#
# Usage: ./sh/reset-db.sh            # asks for confirmation
#        YES=1 ./sh/reset-db.sh      # no prompt
# Env:   NS (service-bay), KIND_NODE (desktop-control-plane),
#        SKIP_BUILD=1, SKIP_SEED=1, SKIP_SMOKE=1,
#        SKIP_RESET=1 (skip steps 1-4: only port-forwards, seed and smoke)
set -euo pipefail
. "$(dirname "$0")/../lib/common.sh"

NS="${NS:-service-bay}"
KIND_NODE="${KIND_NODE:-desktop-control-plane}"
REPO_DIR="$(cd "$LOADTEST_DIR/.." && pwd)"
IMAGE="service-bay-postgres:local"
K="kubectl -n $NS"
PSQL_POD="$K exec -i postgres-0 -- psql -U postgres -v ON_ERROR_STOP=1"

command -v kubectl >/dev/null || die "kubectl not installed"
$K get sts postgres >/dev/null || die "statefulset postgres not found in namespace $NS"

if [ "${SKIP_RESET:-0}" != "1" ]; then
if [ "${YES:-0}" != "1" ]; then
  printf 'This WIPES all Postgres data in namespace %s (PVC postgres-data-postgres-0). Continue? [y/N] ' "$NS"
  read -r ans
  [ "$ans" = "y" ] || [ "$ans" = "Y" ] || die "aborted"
fi

# 1. image ------------------------------------------------------------------
if [ "${SKIP_BUILD:-0}" != "1" ]; then
  log "building $IMAGE"
  docker build -q -t "$IMAGE" "$REPO_DIR/postgres" >/dev/null
  log "loading $IMAGE into the cluster"
  if command -v kind >/dev/null; then
    kind load docker-image "$IMAGE" --name "${KIND_CLUSTER:-desktop}"
  else
    docker save "$IMAGE" | docker exec -i "$KIND_NODE" ctr -n k8s.io images import - >/dev/null
  fi
fi

# 2. wipe + recreate postgres -------------------------------------------------
log "scaling postgres to 0"
$K scale sts postgres --replicas=0
$K wait --for=delete pod/postgres-0 --timeout=180s 2>/dev/null || true
log "deleting PVC postgres-data-postgres-0"
$K delete pvc postgres-data-postgres-0 --ignore-not-found --wait=true
log "scaling postgres to 1 (the init script creates the databases)"
$K scale sts postgres --replicas=1
$K wait --for=condition=Ready pod/postgres-0 --timeout=300s

# 3. services: they apply their migrations on startup --------------------------
# Delete the pods rather than `rollout restart`: a rolling update surges a new
# pod before removing an old one, and on a memory-tight single-node kind
# cluster that pod stays Pending (Insufficient memory) forever.
log "restarting services"
for d in customer-service vehicle-service identity-service; do
  $K delete pod -l "app=$d" --wait=true
done
for d in customer-service vehicle-service identity-service; do
  $K rollout status "deploy/$d" --timeout=300s
done
# 4. verify: the tables come from the services' migrations, so check them
# directly rather than trusting pod readiness.
log "waiting for the services' migrations"
ok=0
for _ in $(seq 1 60); do
  if $PSQL_POD -d vehicle_db -Atc "select 1 from vehicle limit 1" >/dev/null 2>&1 &&
     $PSQL_POD -d customer_db -Atc "select delete_at from customer limit 1" >/dev/null 2>&1; then
    ok=1; break
  fi
  sleep 5
done
if [ "$ok" != "1" ]; then
  for d in customer-service vehicle-service; do
    $K logs "deploy/$d" | grep -iE 'migrat|error|fatal' | head -10 >&2
  done
  die "schema not complete after the services started (see errors above)"
fi
# The dev seed data used to be baked into init; it now loads after migrating.
log "loading dev seed data (postgres/seed)"
$K exec postgres-0 -- sh /seed/seed.sh
log "schema ok: customers=$($PSQL_POD -d customer_db -Atc 'select count(*) from customer')" \
  "vehicles=$($PSQL_POD -d vehicle_db -Atc 'select count(*) from vehicle')"

fi # SKIP_RESET

# 5. port-forwards -------------------------------------------------------------
log "restarting port-forwards (8080 customer, 8081 vehicle)"
pkill -f "port-forward svc/customer-service 8080:8080" 2>/dev/null || true
pkill -f "port-forward svc/vehicle-service 8081:8081" 2>/dev/null || true
PF_LOG="$LOADTEST_DIR/results/port-forward.log"
mkdir -p "$(dirname "$PF_LOG")"
nohup kubectl -n "$NS" port-forward svc/customer-service 8080:8080 >>"$PF_LOG" 2>&1 &
nohup kubectl -n "$NS" port-forward svc/vehicle-service 8081:8081 >>"$PF_LOG" 2>&1 &
for _ in $(seq 1 30); do
  curl -s -o /dev/null -m 2 "$CUSTOMER_URL/metrics" && curl -s -o /dev/null -m 2 "$VEHICLE_URL/metrics" && break
  sleep 1
done
require_up

# 6. seed + smoke ---------------------------------------------------------------
if [ "${SKIP_SEED:-0}" != "1" ]; then
  log "seeding load-test data"
  PSQL="$K exec -i postgres-0 -- psql -U postgres" "$LOADTEST_DIR/seed/seed.sh"
fi
if [ "${SKIP_SMOKE:-0}" != "1" ]; then
  log "running smoke test"
  exec "$LOADTEST_DIR/sh/smoke.sh"
fi
log "done"
