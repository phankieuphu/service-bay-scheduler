# Load tests

Load tests for `customer-service` and `vehicle-service` ([issue #38](https://github.com/phankieuphu/service-bay-scheduler/issues/38)).
Every scenario can run with two tools:

- **k6** (`k6/*.js`): the real load generator. Use it for anything above a few hundred ops/s and for the numbers in the report.
- **sh + curl** (`sh/*.sh`): needs no extra install. Use it for quick local runs, CI smoke checks, and the lower tiers.

Both tools use the same tiers, traffic mix, seeded data and SLO, so their results can be compared.

```
loadtest/
├── seed/seed.sh        seed / reset / clean the load-test data (psql)
├── lib/common.sh       tiers, mix, SLO, shared helpers
├── lib/engine.sh       curl arrival-rate engine + report (sh runners)
├── lib/k8s.sh          cluster preflight (Pending pods, metrics-server)
├── sh/                 smoke, baseline, load, stress, spike, soak
├── k6/                 the same scenarios in k6 + run.sh / in-cluster.sh wrappers
│   └── lib/            config, API calls, mix, phase tagging, /metrics probe, summary
└── results/            one folder / file per run (git-ignored)
```

## Quick start

```bash
cd loadtest
./seed/seed.sh              # once per database; writes .env.seed
./sh/smoke.sh               # every API case, asserted — run before any load
./k6/run.sh smoke           # same cases in k6

TIER=1k   ./sh/baseline.sh
TIER=100k ./k6/run.sh load
TIER=100k MIX=write ./k6/run.sh stress
```

The same commands through make: `make seed`, `make smoke`, `make k6-load TIER=100k`, `make sh-spike`.
From the repo root: `make loadtest SCENARIO=load TIER=100k [TOOL=sh]`.

Requirements: `curl`, `psql` (seeding only), and [k6](https://grafana.com/docs/k6/latest/set-up/install-k6/) for the k6 runners. The sh runners work with macOS's built-in bash 3.2.

## Scenarios

Rates are **operations per second**. One operation is one k6 iteration. A read is 1 request; an update (GET, then PUT) and a delete (POST, then DELETE) are 2 requests each. The tier numbers come from the traffic model in the issue comments: 1k / 100k / 1M registered users.

| Scenario | Shape | 1k | 100k | 1m | SLO judged on |
|---|---|---|---|---|---|
| `smoke` | 1 request per API case, with assertions | – | – | – | every check passes |
| `baseline` | constant rate, 10 min | 5 | 30 | 300 | whole run |
| `load` | 2 min ramp → hold 30 min → 1 min ramp-down | 10 | 100 | 1,000 | `hold` phase |
| `stress` | 10 steps × 2 min up to the ceiling | → 50 | → 500 | → 5,000 | per step; the report gives the break point |
| `spike` | baseline 1 min → 10× in 20 s → hold 3 min → back → 5 min recovery | 5 → 50 | 30 → 300 | 300 → 3,000 | `warmup` + `recovery` (peak: errors < 1%) |
| `soak` | 1 min ramp → constant 2 h | 5 | 50 | 500 | `soak` phase + goroutine growth < 1.5× |

**SLO** (the issue's acceptance criteria): p95 < 200 ms, p99 < 500 ms, error rate < 0.1%. Change it with `SLO_P95_MS`, `SLO_P99_MS` and `SLO_ERR_PCT`. For k6 you can also set `SLO_ERR_RATE` directly.

Common overrides: `RATE`, `DURATION` (`90s`, `10m`, `2h`), `RAMP`, `MAX_RATE`, `STEPS`, `STEP_DURATION`, `BASE_RATE`, `SPIKE_RATE`, `PEAK_DURATION`, `RECOVERY_DURATION`, `CUSTOMER_URL`, `VEHICLE_URL`.

### Traffic mix (`MIX=`)

| op | endpoint | read | mixed (default) | write |
|---|---|---|---|---|
| cust_get | `GET /api/v1/customer/:id` | 40 | 36 | 8 |
| cust_list | `GET /api/v1/customer?cursor=&limit=20` (half first page, half deep) | 15 | 13 | 3 |
| veh_get | `GET /api/v1/vehicle/:id` | 45 | 41 | 9 |
| cust_create | `POST /api/v1/customer` | – | 3 | 25 |
| cust_update | `GET` then `PUT /api/v1/customer/:id` with `updated_at` (optimistic lock) | – | 3 | 25 |
| cust_delete | `POST` then `DELETE /api/v1/customer/:id` (only customers the test created) | – | 1 | 5 |
| transfer | `POST /api/v1/transfer` | – | 3 | 25 |

`mixed` is the 90/10 read/write split from the issue. `write` puts load on the transactional-outbox write path, where every write also inserts an `outbox_message` row.

`GET /api/v1/customer-vehicle` is not tested: its handler is still an empty stub.

## Test data

`seed/seed.sh` inserts its own rows, so the hand-written seed data is never touched:

- `SEED_CUSTOMERS` (default 10,000) customers with emails `lt-seed-N@example.com`. Tests create more as `lt-<run>-…@example.com`, and those are the only customers the tests delete.
- `SEED_VEHICLES` (default 10,000) vehicles with VIN `LT…`. The first `TRANSFER_POOL` (default 5,000) belong to customer A and form the **transfer pool**. The next one is reserved for the smoke and race tests. The rest are spread over the other seeded customers.

**How transfers avoid lock contention:** requests never compete for the same `SELECT … FOR UPDATE` row lock, so the test measures the cost of a transfer, not queueing on a lock.
- In **k6**, each VU owns one pool vehicle and moves it back and forth between customers A and B. On a 409 the VU flips its view of the owner, so it recovers from an interrupted run.
- In **sh**, the k-th transfer moves vehicle `start + k mod pool` (A→B on even passes over the pool, B→A on odd passes).

Contention itself is covered by the "exactly 1 winner" checks in `smoke` and by `vehicle-service/scripts/transfer-race-test.sh`. `TRANSFER_POOL` must be at least the largest `maxVUs` k6 uses, which is about the peak rate; k6 warns if it isn't.

Before each load run, the runners call `seed/seed.sh reset` to give the whole pool back to A. Set `RESET_OWNERSHIP=0` if the machine running the test has no DB access.

`./seed/seed.sh clean` removes all load-test rows. Outbox rows are left in place.

## Running against Kubernetes

1. **Size the cluster for the tier.** Each overlay's "Suggested nodes" in [k8s/README.md](../k8s/README.md#load-test-tiers-issue-38) is the minimum. tier-100k needs 3 × (4 vCPU, 8 GiB); on Docker Desktop that means at least 12 GiB of memory under Settings → Resources. Otherwise some pods stay Pending and the run measures a smaller deployment than the tier describes. On a single small node, stick to tier-1k.
2. **Install metrics-server once per cluster:** `kubectl apply -k k8s/addons/metrics-server`. Without it the HPAs show `<unknown>` and never scale.
3. **Deploy the tier into a fresh namespace**, so disk sizes and StatefulSet settings come from the overlay: `kubectl delete -k k8s/` (this **deletes the data**), then `kubectl apply -k k8s/overlays/tier-1k`.
4. **Seed**, then check that nothing is Pending or not Ready:

   ```bash
   PSQL="kubectl -n service-bay exec -i postgres-0 -c postgres -- psql -U postgres" ./seed/seed.sh
   ./k6/in-cluster.sh check          # or: make k8s-check
   ```

5. **Run k6 inside the cluster:**

   ```bash
   TIER=1k ./k6/in-cluster.sh load     # tier-1k baseline: 10 ops/s, SLO judged on the 30 min hold
   TIER=1k ./k6/in-cluster.sh stress   # steps up to 50 ops/s; the report gives the break point
   make k8s-load TIER=1k               # same through make; root: make loadtest TOOL=k8s SCENARIO=load
   ```

`in-cluster.sh` takes the same scenarios, env vars and extra k6 args as `run.sh`. It:

- runs the preflight and refuses to start if any pod is Pending or not Ready (`ALLOW_PENDING=1` overrides this, but then the numbers don't describe the tier). It warns if metrics-server is missing.
- resets the transfer pool through `kubectl exec` into `postgres-0`.
- ships `k6/*.js` in the `loadtest-k6` ConfigMap and starts a `grafana/k6` pod. The pod targets the Services by cluster DNS and prefers a node without customer-service, vehicle-service or postgres pods. Set `K6_NODE=<node>` to pin it to one node.
- streams the k6 output, copies the summary files into `results/`, deletes the pod, and exits with k6's exit code.

Options: `K8S_NAMESPACE`, `K6_IMAGE` (default `grafana/k6:2.3.0`), `K6_CPU` / `K6_MEMORY` (default 250m / 256Mi request, no CPU limit), `KEEP_POD=1`. The pod downloads `k6-summary` from jslib.k6.io, so the cluster needs outbound internet access.

**While it runs,** open Grafana (NodePort 30030) → **PostgreSQL → Saturation (load tests)**:

- **Postgres CPU vs limit** and **CPU throttling by container** come from cAdvisor. A flat line at the limit, or throttling that stays above ~25%, means Postgres is CPU-bound for the tier.
- **DB pool wait per pod** and **connections in use** come from each service's `go_sql_*` metrics. Wait above zero means requests queue for a connection (`DB_MAX_OPEN_CONNS`), not for Postgres.

`./k6/run.sh` through `kubectl port-forward` still works. It sends all traffic through one tunnel to one pod, and the tunnel's own overhead ends up in the latencies, so only use it for smoke runs:

```bash
kubectl -n service-bay port-forward svc/customer-service 8080:8080 &
kubectl -n service-bay port-forward svc/vehicle-service 8081:8081 &
./k6/run.sh smoke
```

## Reading the results

- **k6** prints the standard summary plus:
  - a per-phase table (per step for stress, with the break point)
  - Go runtime gauges (goroutines, heap, RSS and open fds for each service, from `/metrics`)

  It also writes `results/k6-<scenario>-<tier>-<mix>-<time>.{json,txt}`. Add `--out experimental-prometheus-rw` to `run.sh` to push k6 metrics into Prometheus next to the service metrics.
- **sh** writes `results/sh-<scenario>-<tier>-<time>/` with:
  - `report.txt`: tables per operation and per phase (requests, rps, ok%, expected misses and conflicts, err%, p50/p95/p99/max) and the SLO verdict
  - `requests.log`: one line per request
  - `runtime.csv`: Go runtime samples
  - `break.txt`: stress only, the break point

  The exit code is 1 if the SLO failed; stress always exits 0.

**What counts as an error:** 404 on reads of random ids and 409 on updates and transfers are expected outcomes. They are counted separately (`miss` / `conflict`, and `read_miss_rate` / `*_conflict_rate` in k6). Only unexpected statuses, timeouts and connection failures count as errors.

**Runtime samples:** behind a Kubernetes Service with more than one replica, each `/metrics` sample may come from a different pod. Use the per-pod Grafana panels for leak analysis there.

## Limits of the sh engine

Every second, the sh engine launches one `curl --parallel` batch for single-request operations and background chains for the two-step ones. Like k6's arrival-rate executors, it does not slow down when the server does. If more than `MAX_INFLIGHT_JOBS` (default 400) are still running, it skips the tick and logs it in `dropped.log`, the same idea as k6's `dropped_iterations`.

- Connections are reused only within one second's batch (`PARALLEL_MAX`, default 50 connections per batch). Above roughly 300–500 ops/s from a laptop, the load generator becomes the bottleneck, and it can run out of ephemeral ports (TIME_WAIT). Use k6 there.
- Timings are curl's `time_total` per request, so they include connection setup for a batch's first requests.
