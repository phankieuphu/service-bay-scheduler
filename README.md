# Service Bay

Service Bay is a microservices backend (plus a React web app) for a vehicle service and booking business: customers, their vehicles, ownership and warranty, and, later, bookings at service hubs, billing and notifications.

The Go services follow a hexagonal (ports and adapters) layout, keep one Postgres database per service, and publish domain events to Kafka through a transactional outbox.

---

## Project status

Last updated: 2026-09-29.

| Component | State | What it does today |
|-----------|-------|--------------------|
| **identity-service** (`:8083`) | Implemented | Register, login, refresh and logout (JWT access token + refresh token in Redis), `GET /me`. Publishes `UserCreated` to `identity.user-events`. |
| **customer-service** (`:8080`) | Implemented | Customer CRUD with cursor pagination, optimistic locking on update, soft delete. Publishes to `customer.events`; consumes `identity.user-events`. |
| **vehicle-service** (`:8081`) | Implemented | Register, search (VIN / plate), update status and warranty, assign owner, transfer ownership (row-locked), list a customer's vehicles, warranty / materials / service-history reads. Publishes `vehicle.*` events. |
| **frontend** (`:5173`) | Implemented | React 19 + Vite app: sign in / sign up, customer list and detail, vehicle search, register, detail, ownership and records. |
| Infrastructure | Implemented | Postgres 16, Kafka (KRaft), Redis, Flink, Prometheus, Grafana, Loki + Alloy, in Docker Compose and Kubernetes (with 1k / 100k / 1m load-test tiers). |
| Load tests | Implemented | k6 and curl scenarios (smoke, baseline, load, stress, spike, soak) in [loadtest/](loadtest/). |
| dealership, scheduler, billing, notification, report services | Planned | Databases and schemas exist in [postgres/init/](postgres/init/); no service code yet. See [architecture-design.md](docs/architecture-design.md). |

Known gaps:

- vehicle-service and customer-service log `identity.user-events` but don't create local profiles from them yet.
- The frontend calls the services directly (CORS is open, `*`); there is no API gateway.
- The service Dockerfiles build `GOARCH=amd64` binaries (see [Troubleshooting](#troubleshooting) for Apple Silicon).

---

## Documentation

* [Business Requirement](docs/business-requirement.md)
* [Architecture Design](docs/architecture-design.md): service map, event contracts, data ownership
* [Technical Requirement](docs/technical-requirement.md)
* [Responsibility Planning](docs/responsibility-planning.md)
* Service API references: [identity-service](identity-service/README.md), [customer-service](customer-service/README.md), [vehicle-service](vehicle-service/README.md)
* [Kubernetes manifests and load-test tiers](k8s/README.md)
* [Load tests](loadtest/README.md)

---

## Repository layout

```
.
├── identity-service/    Go module: auth, users, JWT
├── customer-service/    Go module: customer profiles
├── vehicle-service/     Go module: vehicles, ownership, warranty
├── frontend/            React + TypeScript + Vite web app
├── postgres/            init SQL (one database per service) and seed data
├── k8s/                 Kustomize base, tier overlays, metrics-server add-on
├── loadtest/            k6 and sh load-test scenarios
├── prometheus/ grafana/ loki/ alloy/   observability config
├── docs/                requirements and architecture
└── docker-compose.yml   full local stack
```

---

## Prerequisites

| Tool | Version | Needed for |
|------|---------|------------|
| Docker + Docker Compose | recent Docker Desktop, or Engine with the compose plugin | the full stack |
| Go | 1.26+ | running or testing a service outside Docker |
| Node.js + npm | Node 22 LTS or newer | the frontend |
| kubectl | any recent version | Kubernetes only |
| k6, psql | | load tests only |

Ports used on `localhost`: 3000, 3100, 5173, 5432, 6379, 8080–8083, 9090, 9092, 12345. Stop anything else listening on them first.

---

## Quick start (Docker Compose)

This runs all three services and every piece of infrastructure. You only need Docker.

1. Clone and enter the repo:

   ```bash
   git clone git@github.com:phankieuphu/service-bay-scheduler.git
   cd service-bay-scheduler
   ```

2. Build and start the stack:

   ```bash
   docker compose up -d --build
   ```

   On first start, Postgres runs [postgres/init/](postgres/init/): it creates one database per service (`identity_db`, `customer_db`, `vehicle_db`, …), their tables, and seed customers and vehicles. The services wait for Postgres to be healthy.

3. Check that everything is up:

   ```bash
   docker compose ps
   curl localhost:8083/healthz   # identity-service
   curl localhost:8080/readyz    # customer-service (checks Postgres, Kafka, Redis)
   curl localhost:8081/readyz    # vehicle-service
   ```

   If a service exited because Kafka wasn't ready yet, run `docker compose up -d` again.

4. Try the API:

   ```bash
   # create a user and log in
   curl -s -X POST localhost:8083/api/v1/auth/register \
     -H 'Content-Type: application/json' \
     -d '{"email":"demo@example.com","password":"password123"}'
   curl -s -X POST localhost:8083/api/v1/auth/login \
     -H 'Content-Type: application/json' \
     -d '{"email":"demo@example.com","password":"password123"}'

   # list seeded customers and vehicles
   curl -s 'localhost:8080/api/v1/customer?limit=5'
   curl -s 'localhost:8081/api/v1/vehicle?limit=5'
   ```

5. Start the frontend (next section) and open http://localhost:5173.

| URL | What |
|-----|------|
| http://localhost:8083/api/v1 | identity-service |
| http://localhost:8080/api/v1 | customer-service |
| http://localhost:8081/api/v1 | vehicle-service |
| http://localhost:3000 | Grafana (`admin` / `admin`) |
| http://localhost:9090 | Prometheus |
| http://localhost:8082 | Flink web UI |
| http://localhost:12345 | Alloy debug UI |
| `localhost:5432` | Postgres (`postgres` / `postgres`) |
| `localhost:9092` | Kafka |
| `localhost:6379` | Redis |

Useful commands:

```bash
docker compose logs -f vehicle-service             # follow one service's logs
docker compose up -d --build vehicle-service       # rebuild after a code change
docker compose down                                # stop, keep data
docker compose down -v                             # stop and delete all data
```

Postgres only runs the init scripts when its data volume is empty. After changing a schema in `postgres/init/`, run `docker compose down -v` (this deletes all local data) and start again.

---

## Run the frontend

The frontend is a Vite dev server that calls the three services directly.

```bash
cd frontend
cp .env.example .env    # API URLs, defaults point at the compose ports
npm install
npm run dev             # http://localhost:5173
```

| Variable | Default |
|----------|---------|
| `VITE_IDENTITY_API_URL` | `http://localhost:8083/api/v1` |
| `VITE_CUSTOMER_API_URL` | `http://localhost:8080/api/v1` |
| `VITE_VEHICLE_API_URL` | `http://localhost:8081/api/v1` |

Other scripts: `npm run build` (type-check and production build into `dist/`), `npm run preview`, `npm run lint` (oxlint).

---

## Run a service with Go (without Docker)

Use this when you're working on one service and want fast restarts. Start the infrastructure in Docker, stop the container of the service you're running yourself, then run it with Go:

```bash
docker compose up -d postgres kafka redis
docker compose stop vehicle-service      # if it was running, so the port is free

cd vehicle-service                       # or customer-service / identity-service
cp .env.example .env                     # loaded automatically on start
go run ./cmd/server
```

Each service is its own Go module (there is no `go.work`), so always `cd` into its directory first. Every environment variable and its default is defined in the service's `config/config.go`. The main ones:

| Variable | identity-service | customer-service | vehicle-service |
|----------|------------------|------------------|-----------------|
| `API_PORT` | 8083 | 8080 | 8081 |
| `DB_NAME` | `identity_db` | `customer_db` | `vehicle_db` |
| `KAFKA_PRODUCER_TOPIC` | `identity.user-events` | `customer.events` | `vehicle.events` |
| `KAFKA_CONSUMER_TOPIC` | – | `identity.user-events` | `identity.user-events` |
| `JWT_SECRET` | required, ≥ 32 bytes | – | – |

Shared: `DB_HOST`, `DB_PORT`, `DB_USERNAME`, `DB_PASSWORD`, `DB_SSLMODE`, `DB_MAX_OPEN_CONNS`, `KAFKA_BROKERS` (`localhost:9092` from the host), `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`, `LOG_LEVEL`, `LOG_FORMAT`.

Every service exposes `GET /healthz` (liveness), `GET /readyz` (dependencies) and `GET /metrics` (Prometheus).

---

## Testing

Unit tests live next to the code (mostly `internal/domain/services/*_test.go`) and use hand-written mocks of the `ports` interfaces, so they need no database or Kafka.

```bash
cd vehicle-service     # or customer-service / identity-service
go test ./...
go vet ./...
go test ./internal/domain/services/ -run TestVehicleService_TransferVehicle   # a single test
```

Test files must end in `_test.go` (underscore). A file ending in `-test.go` compiles as normal source, and `go test` never runs its tests.

Against a running stack:

```bash
./vehicle-service/scripts/transfer-race-test.sh   # concurrent transfers of one vehicle; exactly one must win
cd loadtest && ./seed/seed.sh && ./sh/smoke.sh      # every API case, asserted
```

See [loadtest/README.md](loadtest/README.md) for load, stress, spike and soak runs (`make loadtest SCENARIO=load TIER=100k` from the repo root).

---

## Run on Kubernetes

All manifests live under [k8s/](k8s/) and are wired together with a `kustomization.yaml`. Everything runs in the `service-bay` namespace.

1. Pick a cluster. On Docker Desktop, enable Kubernetes (Settings → Kubernetes) and select its context:

```bash
kubectl config use-context docker-desktop
```

2. Build the local images. The manifests reference `:local` tags with `imagePullPolicy: IfNotPresent`, so the cluster must be able to see your locally built images:

```bash
docker build -t customer-service:local ./customer-service
docker build -t vehicle-service:local ./vehicle-service
docker build -t identity-service:local ./identity-service
docker build -t service-bay-postgres:local ./postgres
```

- **Docker Desktop**: images are shared with the cluster automatically, as long as Settings → General → "Use containerd for pulling and storing images" is on. Nothing else to do.
- **minikube**: run `eval $(minikube docker-env)` before building, or `minikube image load <image>` afterwards.
- **kind**: `kind load docker-image customer-service:local vehicle-service:local identity-service:local service-bay-postgres:local`

3. Apply all manifests via Kustomize:

```bash
kubectl apply -k k8s/
```

4. Wait until every pod is `Running`:

```bash
kubectl get pods -n service-bay -w
```

`customer-service` and `vehicle-service` usually restart a few times during a fresh start. They exit if Postgres or Kafka isn't accepting connections yet, and Kafka has no readiness probe. They settle on their own within about a minute.

5. Access the services. They're exposed as `NodePort`:

| Service          | Service port | NodePort |
|------------------|--------------|----------|
| customer-service | 8080         | 30080    |
| vehicle-service  | 8081         | 30081    |
| identity-service | 8083         | 30083    |
| prometheus       | 9090         | 30090    |
| grafana          | 3000         | 30030    |
| flink (web UI)   | 8081         | 30082    |

Docker Desktop's Kubernetes (kind-based) does **not** expose NodePorts on `localhost`, so use port-forward, one terminal each:

```bash
kubectl port-forward -n service-bay svc/customer-service 8080:8080
kubectl port-forward -n service-bay svc/vehicle-service  8081:8081
kubectl port-forward -n service-bay svc/identity-service 8083:8083
kubectl port-forward -n service-bay svc/grafana          3000:3000   # admin/admin
kubectl port-forward -n service-bay svc/prometheus       9090:9090
kubectl port-forward -n service-bay svc/flink-jobmanager 8082:8081
kubectl port-forward -n service-bay svc/alloy            12345:12345 # optional: log collector debug UI
```

`loki` and `alloy` are ClusterIP-only. You read logs through Grafana (see [Observability](#observability)), so they need no NodePort.

With `minikube`, get a reachable URL instead:

```bash
minikube service customer-service -n service-bay --url
```

6. Redeploy after a code change:

```bash
docker build -t vehicle-service:local ./vehicle-service
kubectl rollout restart deployment/vehicle-service -n service-bay
```

7. Tear down:

```bash
kubectl delete -k k8s/
```

### Troubleshooting

| Symptom | Cause / fix |
|---------|-------------|
| `kafka` pod in `CrashLoopBackOff`, log ends with `port is deprecated. Please use KAFKA_ADVERTISED_LISTENERS instead.` | Kubernetes injects `KAFKA_PORT=tcp://...` for the Service named `kafka`, and the `cp-kafka` image treats every `KAFKA_*` variable as a setting. [k8s/base/kafka/deployment.yaml](k8s/base/kafka/deployment.yaml) sets `enableServiceLinks: false` to prevent this. Keep that line. |
| Services in `CrashLoopBackOff` with `failed to init Kafka producer ... connection refused` | Kafka isn't up yet (or is crashing; see the row above). Check `kubectl logs -n service-bay deploy/kafka`. Once Kafka is running, the services recover on their next restart, or run `kubectl rollout restart deploy/customer-service deploy/vehicle-service -n service-bay`. |
| `ErrImagePull` / `ImagePullBackOff` on a `:local` image | The cluster can't see your local build. Redo step 2 for your cluster type. |
| `exec format error` on Apple Silicon | The service Dockerfiles hard-code `GOARCH=amd64`. Remove it to build a native arm64 binary. |
| No logs in Grafana's Loki datasource | Check the collector: `kubectl logs -n service-bay deploy/alloy`. A few `connection refused` warnings for `loki:3100` right after a fresh apply are normal, because Alloy retries until Loki is ready. If they continue, check `kubectl logs -n service-bay deploy/loki`. |
| Grafana has no Loki datasource after an apply | Grafana only reads provisioning at startup: `kubectl rollout restart deploy/grafana -n service-bay`. |

Debugging a pod:

```bash
kubectl logs -n service-bay deploy/<name> [--previous]
kubectl describe pod -n service-bay <pod>
```

---

## Observability

| What | Tool | Where it comes from |
|------|------|---------------------|
| Metrics | Prometheus → Grafana | Services and Flink expose `/metrics`; Postgres and Kafka go through exporters (`postgres-exporter` sidecar on :9187, `kafka-exporter` on :9308). Compose uses static targets ([prometheus/prometheus.yml](prometheus/prometheus.yml)); k8s discovers every pod annotated `prometheus.io/scrape: "true"` + `prometheus.io/port` ([k8s/base/prometheus/configmap.yaml](k8s/base/prometheus/configmap.yaml)), so each replica is scraped on its own |
| Logs | Loki → Grafana | [Grafana Alloy](https://grafana.com/docs/alloy/) collects the stdout/stderr of every container or pod and pushes it to Loki |

Grafana is at http://localhost:3000 (`admin` / `admin`). With Docker Compose it's there directly. On k8s, port-forward first (see step 5 above). The Prometheus and Loki datasources are provisioned automatically, and so are these dashboards from [k8s/base/grafana/dashboards/](k8s/base/grafana/dashboards) (the single source for both setups; compose mounts the directory, k8s ships it as the `grafana-dashboards` ConfigMap):

| Dashboard | Watch during a load test |
|-----------|--------------------------|
| **Go Goroutines** | goroutine leaks, scheduler pressure, CPU per pod ([grafana/docs.md](grafana/docs.md)) |
| **PostgreSQL** | connections vs `max_connections` (pool budget), transactions/s, row locks, cache hit ratio, deadlocks |
| **Kafka** | messages in/s per topic, consumer-group lag (appears once a group has committed offsets) |

To add a dashboard, drop its JSON into that directory without a hard-coded datasource `uid` (panels then use the default Prometheus datasource), list it under `configMapGenerator` in [k8s/base/kustomization.yaml](k8s/base/kustomization.yaml), and re-apply.

### Viewing logs

In Grafana, open **Explore**, pick the **Loki** datasource, and query with LogQL:

```logql
{service="vehicle-service"}                                   # one service
{service=~"customer-service|vehicle-service", level="ERROR"}  # errors from the Go services
{service="vehicle-service"} | json | msg=~"outbox.*"           # filter on a JSON field
{service="kafka"} |= "error"                                  # plain-text search
```

Labels you can filter on:

| Label | Value | Available on |
|-------|-------|--------------|
| `service` | Compose service name / pod `app` label, e.g. `vehicle-service`, `kafka` | both |
| `container` | Container name | both |
| `level` | slog level (`DEBUG`/`INFO`/`WARN`/`ERROR`), parsed from JSON logs | `customer-service`, `vehicle-service` only |
| `pod`, `namespace` | Pod name, `service-bay` | k8s only |

`level` exists only because the Go services log JSON (`LOG_FORMAT=json`, the default). If you switch a service to `LOG_FORMAT=text`, filter with `|= "level=ERROR"` instead.

### How it's wired

| | Docker Compose | Kubernetes |
|-|----------------|------------|
| Loki config | [loki/loki-config.yaml](loki/loki-config.yaml) | [k8s/base/loki/](k8s/base/loki/) (same config in a ConfigMap) |
| Alloy config | [alloy/config.alloy](alloy/config.alloy): reads containers via the Docker socket | [k8s/base/alloy/](k8s/base/alloy/): reads pods in `service-bay` through the Kubernetes API, using a namespace-scoped Role |
| Loki API | http://localhost:3100 | `svc/loki:3100` (ClusterIP) |
| Alloy debug UI | http://localhost:12345 | `svc/alloy:12345` (ClusterIP) |

Loki runs as a single process with filesystem storage and **7-day retention** (`limits_config.retention_period`). That's fine for local and dev use, but not for HA. Keep the k8s Alloy Deployment at **1 replica**: it tails logs through the API server, so a second replica without Alloy clustering would ship every line twice.

If you change `loki/loki-config.yaml`, copy the change into `k8s/base/loki/configmap.yaml` too. The Alloy configs differ on purpose (Docker vs. Kubernetes discovery), but share the same `loki.process` stage.

Without Grafana, you can still read raw logs with `docker compose logs -f <service>` or `kubectl logs -n service-bay deploy/<name> -f`.

---

## Architecture in brief

```
HTTP handler (gin) ──┐
                     ├─→ domain Service (ports.*Service) → Repository (ports.*Repository) → GORM → Postgres
Kafka consumer ──────┘                  │
                                        └─→ outbox row (same transaction) → OutboxRelay → Kafka
```

* **`internal/domain/ports/`**: every interface (services, repositories, cache, transaction manager, producer). Tests mock against these.
* **`internal/domain/services/`**: business logic. Constructors take interfaces, not concrete adapters.
* **`internal/adapters/`**: `http/` (gin handlers and DTOs), `repository/` (GORM), `database/` (connection, transactions, models), `kafka/` (producer, consumer, outbox relay), `cache/` (Redis), `metrics/` (Prometheus).
* **Transactional outbox**: a write and the event it triggers are committed in one `RunInTx`. The outbox relay polls every 2 s and publishes to Kafka in order.
* **Errors**: repositories return `ports.ErrNotFound` / `ports.ErrConflict`, and handlers map them to `404` / `409`.

### Adding to a service

1. Add the interface method in `internal/domain/ports/`.
2. Implement it in `internal/domain/services/` and add a `_test.go` using the existing mocks.
3. Add the repository method in `internal/adapters/repository/`, and a GORM model in `internal/adapters/database/models/` if there's a new table.
4. Add the SQL to that service's file in `postgres/init/` (then `docker compose down -v` locally).
5. Add the handler and DTOs in `internal/adapters/http/`, and register the route in `server.go`.
6. For a new Kafka event, write it to the outbox inside the same transaction, and document its contract in [architecture-design.md](docs/architecture-design.md).

A new service copies an existing one (customer-service is the smallest) and gets its own database in `postgres/init/00-create-databases.sql`, a compose entry and a `k8s/base/` folder.

---

## Contribution guidelines

* Write tests for new features.
* Follow the existing layout and error conventions.
* Code review is required: open a PR against `master`.
