# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Service Bay is a microservices system for a vehicle service/booking business. Only two services are implemented so far: **customer-service** and **vehicle-service**, each an independent Go module living in its own top-level directory. Both are near-identical clones of the same hexagonal-architecture template (same layer names, same file layout, same patterns) — understanding one gets you 90% of the other. Supporting infra (Postgres, Kafka, Redis, Flink, Prometheus, Grafana) is wired up in `docker-compose.yml` and mirrored under `k8s/`.

The target system-wide service map (identity, customer, vehicle, dealership, scheduler, billing, notification, report services) is documented in [docs/architecture-design.md](docs/architecture-design.md) — read it for the intended event contracts and cross-service data-ownership rules before adding a new service or a new Kafka event.

## Commands

Each service is its own Go module (`vehicle-service/go.mod`, `customer-service/go.mod`; no `go.work`), so `cd` into the service directory first.

```bash
cd vehicle-service   # or customer-service

go build ./...                          # compile
go test ./...                           # run all tests in the module
go test ./internal/domain/services/...  # run one package's tests
go test ./internal/domain/services/ -run TestVehicleService_TransferVehicle  # single test
go vet ./...
golangci-lint run                       # not installed in this environment; CI doesn't run it either
```

Local run (either service): copy `.env.example` to `.env` — **note it's a stale generic template** (MySQL/AWS SQS vars that no longer apply); the real, current env vars are defined in `config/config.go` of each service (Postgres/Kafka/Redis/API/Logger). Then run `go run ./cmd/server`, or bring up the full stack:

```bash
docker compose up -d          # customer-service:8080, vehicle-service:8081, postgres, kafka, redis, flink UI:8082, prometheus:30090-equivalent, grafana
kubectl apply -k k8s/         # namespace `service-bay`; see README.md for building/loading local images first
```

`vehicle-service/scripts/transfer-race-test.sh` fires concurrent `POST /api/v1/transfer` requests for the same vehicle to verify the row-locking in `TransferVehicle` lets exactly one request win (see Architecture below).

## Architecture

Both services follow the same layering (Config → DB Provider → Repository → Service, with HTTP and Kafka as parallel inbound adapters):

```
HTTP handler (gin) ──┐
                      ├─→ domain entity → domain Service (ports.*Service) → Repository (ports.*Repository) → GORM model → Postgres
Kafka consumer ───────┘
```

- **`internal/domain/ports/`** defines every interface (`*Service`, `*Repository`, `Cache`, `TxManager`, `OutboxRepository`, `Producer`) — this is the seam every layer is mocked against in tests.
- **`internal/domain/services/`** holds the business logic. Constructors take interfaces, not concrete adapters, so unit tests build a service against hand-written mocks (see `vehicle-service-test.go` for the mock types — `MockVehicleRepository`, `MockOutboxRepository`, `MockTxManager`, `MockCache`, etc. — reused across that package's tests).
- **`internal/adapters/`** — concrete implementations: `http/` (gin handlers + DTOs), `repository/` (GORM), `database/provider/` (connection + transaction plumbing), `kafka/` (producer/consumer/outbox relay), `cache/` (Redis), `metrics/` (Prometheus).

### Transactional outbox

State changes and the Kafka event they trigger must land atomically, so nothing publishes to Kafka synchronously from inside a write path:

1. A service method wraps its work in `txManager.RunInTx(ctx, func(ctx) error { ... })` ([`database/provider/transaction.go`](vehicle-service/internal/adapters/database/provider/transaction.go)). This stashes the `*gorm.DB` transaction in `ctx`.
2. Repository calls inside that closure fetch the DB via `database_provider.DBFromContext(ctx, fallback)`, which returns the active tx if present, otherwise a plain context-scoped connection. This is how every repository silently participates in whatever transaction the calling service started, with no explicit tx parameter threading.
3. The same closure writes an `entity.OutboxMessage` row (topic/key/payload) via `outbox.Create` — same transaction, same commit/rollback as the state change.
4. A separate `kafka.OutboxRelay` ([`adapters/kafka/outbox-relay.go`](vehicle-service/internal/adapters/kafka/outbox-relay.go)), started as its own goroutine in `application.go`, polls the outbox table on a ticker (every 2s), publishes unpublished rows to Kafka, and marks them published. It stops at the first publish failure in a batch (rather than skipping ahead) to preserve per-key ordering, and leaves the rest for the next poll.

`VehicleService.TransferVehicle` ([`domain/services/vehicle-service.go`](vehicle-service/internal/domain/services/vehicle-service.go)) is the fullest example: unassign old owner → assign new owner → write outbox row, all in one `RunInTx`. The unassign step in [`repository/vehicle-customer-repository.go`](vehicle-service/internal/adapters/repository/vehicle-customer-repository.go) takes a `SELECT ... FOR UPDATE` row lock on the current-owner row before checking/changing it, which is what makes concurrent transfer requests for the same vehicle serialize correctly (exercised by `scripts/transfer-race-test.sh`).

### Kafka topics

Each service's producer and consumer intentionally point at **different** topics, so a service never replays its own output:

- Consumes `identity.user-events` (identity-service's user stream) — intended to sync local Customer/Vehicle profiles when a user with the matching role is created/updated. Consumer group name matches the service name.
- Produces its own domain events (`customer.events`, `vehicle.*.v1` — see `internal/constants/topics.go` per service) via the outbox relay.
- `Kafka.ConsumerTopic` and `Kafka.ProducerTopic` must never collide (see the comment on the `Kafka` config struct) — a service consuming its own producer topic would create a feedback loop.
- File names under `adapters/kafka/` (e.g. `customer-consumer.go` inside *vehicle-service*) reflect what upstream event they react to, not necessarily the service's own domain — don't assume a file's name tells you which topic it's wired to; check the `NewConsumer(cfg.Kafka, ...)` call in `application.go`.

### Error conventions

Repositories/services return sentinel errors from `ports/errors.go` (`ports.ErrNotFound`, `ports.ErrConflict`, `ports.ErrCacheMiss`) checked with `errors.Is`. HTTP handlers map these to status codes (404/409/500) — see `TransferVehicle` in [`http/handler/vehicle-handler.go`](vehicle-service/internal/adapters/http/handler/vehicle-handler.go) for the pattern to follow when adding a new handler.

### Known repo quirks worth knowing before touching tests

- Each service's `domain/services/` package has **two** differently-named "test" files: one ending `_test.go` (a real Go test file, picked up by `go test`) and one ending `-test.go` (dash, not underscore — compiled as an ordinary source file, its `Test*` functions are silently never run by `go test`). This split is mid-refactor (see current git status); when adding tests, make sure the file ends in `_test.go` or it won't execute.
- `internal/adapters/database/models/` still has file names like `customer-vehicle-model.go` for what other services call an "account model" in stale docs — the per-service `docs/architecture.md` files (`vehicle-service/docs/architecture.md`, `customer-service/docs/architecture.md`) describe an earlier, non-functional snapshot of the code (panicking repositories, MySQL driver, no tests) that has since been implemented; treat them as historical, not current — verify against the actual source before relying on their "Known Gaps" tables.
