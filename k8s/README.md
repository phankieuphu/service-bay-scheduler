# Kubernetes manifests

Mirrors `docker-compose.yml` for a local cluster (minikube/kind). All resources live in the `service-bay` namespace.

## Build the images the cluster needs

The manifests reference locally-built images, not a registry:

```sh
docker build -t customer-service:local ./customer-service
docker build -t vehicle-service:local ./vehicle-service
docker build -t service-bay-postgres:local ./postgres
```

Load them into your cluster (kind example — minikube uses `minikube image load`):

```sh
kind load docker-image customer-service:local vehicle-service:local service-bay-postgres:local
```

## Apply

```sh
kubectl apply -k k8s/
```

## Load-test tiers (issue #38)

Manifests live in `base/`; `kubectl apply -k k8s/` deploys the base, which is sized for the **1k** tier. Each tier from the [issue #38 sizing plan](https://github.com/phankieuphu/service-bay-scheduler/issues/38) is a kustomize overlay under `overlays/`:

```sh
kubectl apply -k k8s/overlays/tier-100k
TIER=100k ./loadtest/k6/run.sh load      # k6 tiers use the same names
```

| | tier-1k | tier-100k | tier-1m |
|---|---|---|---|
| Design peak / stress RPS | 10 / 25 | 100 / 250 | 1,000 / 2,500 |
| customer-/vehicle-service (each) | 100m / 128Mi, HPA 2–4 | 250m / 256Mi, HPA 2–3 | 500m / 384Mi, HPA 3–6 |
| `DB_MAX_OPEN_CONNS` per pod | 10 | 10 | 20, via PgBouncer |
| PostgreSQL | 250m / 512Mi, 5Gi | 1 CPU / 2Gi, 20Gi | 2 CPU / 8Gi, 100Gi |
| PgBouncer | – | – | 2 × 250m / 128Mi |
| Redis (no persistence) | 50m / 64Mi | 100m / 256Mi | 250m / 1Gi |
| Kafka / ZooKeeper | 250m / 1Gi, 100m / 256Mi | 500m / 2Gi, 100m / 256Mi | 1 CPU / 4Gi, 250m / 512Mi |
| Flink JM / TM | 250m / 1Gi, 500m / 1.5Gi | 500m / 1.5Gi, 1 CPU / 2Gi | 1 CPU / 2Gi, 2 × 2 CPU / 4Gi |
| Prometheus (15d retention) | 250m / 512Mi, 10Gi | 500m / 1Gi, 20Gi | 1 CPU / 3Gi, 50Gi |
| Suggested nodes | 2 × (2 vCPU, 8 GiB) | 3 × (4 vCPU, 8 GiB) | 8 × (4 vCPU, 16 GiB) |

Sizing rules used everywhere (from the issue):

- **Memory limit = memory request** for every component.
- **App pods:** CPU limit is 2–4× the request, and `GOMEMLIMIT` is about 85% of the memory limit. Go 1.26 already sets `GOMAXPROCS` from the CPU limit.
- **Postgres, Redis, Kafka, ZooKeeper:** CPU limit = request, so they get Guaranteed QoS.
- **JVM heaps follow the limit:** `KAFKA_HEAP_OPTS` is about 50% of the limit. Flink's `*.memory.process.size` equals the container limit. The images' default heaps would get OOM-killed inside these limits.
- **DB connection budget:** services × HPA max × `DB_MAX_OPEN_CONNS` must stay under 80% of `max_connections` (100). tier-1m exceeds that (240), so it adds PgBouncer in transaction mode, which uses at most 48 Postgres connections. `max_prepared_statements` is set because GORM uses `PrepareStmt: true`.

**Switching tiers on an existing cluster:** CPU, memory, replicas and config change in place. Disk sizes do not: a StatefulSet's `volumeClaimTemplates` is immutable, and the `local-path` storage class (Docker Desktop, kind) can't grow a PVC, so `apply` reports those two objects as forbidden. To get a tier's disk sizes, start from a fresh namespace (`kubectl delete -k k8s/`, which **deletes the data**, then apply the overlay). `local-path` doesn't enforce sizes, so locally you can ignore those errors.

**Not implemented yet** (the issue's 1M column goes beyond what the app can use today):

- **Postgres read replica and Redis replica:** the services have no read/replica routing, so an extra instance would sit idle.
- **3-broker Kafka and 3-node ZooKeeper:** this needs Kafka turned into a StatefulSet with per-broker IDs and listeners, plus a move to KRaft. Kafka also has no persistent volume yet.
- **Monitoring add-ons:** Alertmanager, kube-state-metrics, metrics-server and the exporters belong to separate PRs in the issue's plan. metrics-server is required for the HPAs to act; without it `kubectl get hpa` shows `<unknown>`.

## Access

NodePort services are exposed on the cluster node's IP:

- customer-service: 30080
- vehicle-service: 30081
- prometheus: 30090
- grafana: 30030 (admin/admin)
- flink web UI: 30082

postgres, zookeeper, kafka, redis, loki and alloy are ClusterIP-only, matching their internal-only role in `docker-compose.yml`.

## Logs

`alloy/` runs Grafana Alloy (1 replica, namespace-scoped Role for `pods` and `pods/log`). It tails every pod in `service-bay` through the API server and pushes to `loki/` (single-binary, 2Gi PVC, 7-day retention). Grafana has Loki provisioned as a datasource: open **Explore → Loki** and query e.g. `{service="vehicle-service", level="ERROR"}`. See the Observability section of the root README for details.
