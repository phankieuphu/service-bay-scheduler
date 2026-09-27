# README

## Overview

Base code to create new another repository

---

## Documentation

* [Business Requirement](docs/business-requirement.md)
* [Architecture Design](docs/architecture-design.md)
* [Technical Requirement](docs/technical-requirement.md)
* [Responsibility Planning](docs/responsibility-planning.md)

---

## Repository Purpose

* Clean and normalize data from multiple sources
* Prepare data for banking reports
* Support extensible data ingestion (DB, Queue, etc.)
* Follow layered / hexagonal architecture

---

## Setup Guide

### Local Environment

1. Create environment variables:

```bash
cp .env.example .env
```

2. Update your local configuration in `.env`

3. Run the initialization script:

```bash
sh init.sh
``` 

---

### Docker Setup

```bash
docker compose up -d
```

Grafana: http://localhost:3000 (`admin` / `admin`), with Prometheus metrics and Loki logs. Prometheus: http://localhost:9090. See [Observability](#observability).

---

### Kubernetes Setup

All manifests live under [k8s/](k8s/) and are wired together with a `kustomization.yaml`. Everything runs in the `service-bay` namespace.

1. Pick a cluster. On Docker Desktop, enable Kubernetes (Settings → Kubernetes) and select its context:

```bash
kubectl config use-context docker-desktop
```

2. Build the local images. The manifests reference `:local` tags with `imagePullPolicy: IfNotPresent`, so the cluster must be able to see your locally built images:

```bash
docker build -t customer-service:local ./customer-service
docker build -t vehicle-service:local ./vehicle-service
docker build -t service-bay-postgres:local ./postgres
```

- **Docker Desktop**: images are shared with the cluster automatically, as long as Settings → General → "Use containerd for pulling and storing images" is on. Nothing else to do.
- **minikube**: run `eval $(minikube docker-env)` before building, or `minikube image load <image>` afterwards.
- **kind**: `kind load docker-image customer-service:local vehicle-service:local service-bay-postgres:local`

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
| prometheus       | 9090         | 30090    |
| grafana          | 3000         | 30030    |
| flink (web UI)   | 8081         | 30082    |

Docker Desktop's Kubernetes (kind-based) does **not** expose NodePorts on `localhost`, so use port-forward, one terminal each:

```bash
kubectl port-forward -n service-bay svc/customer-service 8080:8080
kubectl port-forward -n service-bay svc/vehicle-service  8081:8081
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

#### Troubleshooting

| Symptom | Cause / fix |
|---------|-------------|
| `kafka` pod in `CrashLoopBackOff`, log ends with `port is deprecated. Please use KAFKA_ADVERTISED_LISTENERS instead.` | Kubernetes injects `KAFKA_PORT=tcp://...` for the Service named `kafka`, and the `cp-kafka` image treats every `KAFKA_*` variable as a setting. [k8s/kafka/deployment.yaml](k8s/kafka/deployment.yaml) sets `enableServiceLinks: false` to prevent this. Keep that line. |
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
| Metrics | Prometheus → Grafana | Prometheus scrapes `/metrics` ([prometheus/prometheus.yml](prometheus/prometheus.yml)) |
| Logs | Loki → Grafana | [Grafana Alloy](https://grafana.com/docs/alloy/) collects the stdout/stderr of every container or pod and pushes it to Loki |

Grafana is at http://localhost:3000 (`admin` / `admin`). With Docker Compose it's there directly. On k8s, port-forward first (see step 5 above). The Prometheus and Loki datasources are provisioned automatically.

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
| Loki config | [loki/loki-config.yaml](loki/loki-config.yaml) | [k8s/loki/](k8s/loki/) (same config in a ConfigMap) |
| Alloy config | [alloy/config.alloy](alloy/config.alloy): reads containers via the Docker socket | [k8s/alloy/](k8s/alloy/): reads pods in `service-bay` through the Kubernetes API, using a namespace-scoped Role |
| Loki API | http://localhost:3100 | `svc/loki:3100` (ClusterIP) |
| Alloy debug UI | http://localhost:12345 | `svc/alloy:12345` (ClusterIP) |

Loki runs as a single process with filesystem storage and **7-day retention** (`limits_config.retention_period`). That's fine for local and dev use, but not for HA. Keep the k8s Alloy Deployment at **1 replica**: it tails logs through the API server, so a second replica without Alloy clustering would ship every line twice.

If you change `loki/loki-config.yaml`, copy the change into `k8s/loki/configmap.yaml` too. The Alloy configs differ on purpose (Docker vs. Kubernetes discovery), but share the same `loki.process` stage.

Without Grafana, you can still read raw logs with `docker compose logs -f <service>` or `kubectl logs -n service-bay deploy/<name> -f`.

---

## Initializing a New Data Flow

### 1. Define Data Sources

#### From Database

* Implement repository adapters

#### From Queue

* Location: `internal/adapters/consumer`
* Steps:

   * Add a new consumer: `{name}Consumer.go`
   * Define input DTOs in the `/dto` folder

---

### 2. Define a New Service

1. Define service interface:

   * File: `internal/domain/ports/services.go`

2. Implement service logic:

   * Folder: `internal/domain/services`

3. Inputs & outputs:

   * Use DTOs from `internal/adapters/http` if the service is HTTP-based

---

### 3. Define Outbound Adapters (Repositories)

For database or external storage operations:

1. Define repository interface:

   * `internal/domain/ports/repositories.go`

2. Create adapter struct:

   * `internal/adapters/repositories`

3. Implement repository logic

---

## Service Architecture Layers

```
Config
  |
DB Provider
  |
Repository (Storage)
  |
Service (Use Case)
```



---



## Database Configuration

* Define database models in:

```
internal/adapters/database/models
```
---
* **Note**: if your table want to define is SQL please update file **init.sql** your SQL script

## Testing

* Write unit tests for services and repositories
* Mock external dependencies
* Run tests using standard Go **tooling**
* Run `golangci-lint run` for ensure correct syntax
---

## Deployment

* Docker-based deployment
* Environment-driven configuration
* CI/CD friendly

---

## Contribution Guidelines

* Write tests for all new features
* Follow existing code structure
* Code reviews are mandatory

---

## Contact

* Repository owner / admin
* Project team members
