# vehicle-service

Vehicle registry for Service Bay: vehicles, who owns them, their warranty, installed materials, and a service-history read model. Hexagonal layout shared with the other services (see the root [CLAUDE.md](../CLAUDE.md)).

## Run

```bash
cp .env.example .env         # note: stale template; real vars are in config/config.go
go run ./cmd/server          # :8080 by default; docker compose runs it on :8081
go test ./...
```

Schema: [`postgres/init/03-vehicle-service.sql`](../postgres/init/03-vehicle-service.sql). Postgres only runs init scripts on an empty volume, so an existing `postgres_data` volume needs `docker compose down -v` (deletes all local data) to pick up schema changes.

## API

All under `/api/v1`. Errors are `{"error": "..."}`: `400` bad input, `404` unknown vehicle, `409` conflict (duplicate, stale `updated_at`, already owned, or scrapped), `500` generic.

| Method & path | Body / query | Success |
|---|---|---|
| `POST /vehicle` | `{vin, vehicle_model_id, license_plate?, warranty_end_date?, status?}` | `201` vehicle |
| `GET /vehicle` | `?cursor=&limit=&vin=&plate=&status=` | `200` `{vehicles, next_cursor?, has_more}` |
| `GET /vehicle/:id` | | `200` vehicle |
| `PATCH /vehicle/:id` | `{status?, warranty_end_date?, updated_at}` | `200` vehicle |
| `POST /vehicle/:id/owner` | `{customer_id, date?}` | `204` |
| `POST /transfer` | `{vehicle_id, from, to, date?}` | `204` |
| `GET /customers/:id/vehicles` | | `200` `{vehicles: [vehicle + owned_from]}` |
| `GET /vehicle/:id/warranty` | `?date=YYYY-MM-DD` (default today) | `200` `{vehicle_id, warranty_end_date, as_of, in_warranty, days_remaining}` |
| `GET /vehicle/:id/materials` | | `200` `{materials: [{id, material_id, description, count, installed_at}]}` |
| `GET /vehicle/:id/history` | `?limit=` (default 50, max 200) | `200` `{history: [{id, appointment_id, dealership_id, completed_at, services}]}` |

Rules worth knowing:

- **VIN** must be a valid 17-character VIN (no I, O or Q); it's uppercased. **Plates** are normalized to `A-Z0-9` (`51a-123.45` → `51A12345`). Both are unique, and `vin`/`plate` searches use the same normalization.
- **Dates** (`warranty_end_date`, `date`) are calendar days: send RFC3339, and the day is read in the offset you send. `warranty_end_date` is `null` when there's no warranty. A warranty covers its whole last day.
- **PATCH** needs the `updated_at` you last read; if the vehicle changed since, it's a `409`, so reload and retry. The response carries the new `updated_at`. `SCRAPPED` is final.
- **Ownership**: `POST /vehicle/:id/owner` is only for a vehicle with no current owner; after that, use `POST /transfer`.

## Events

Produced through the transactional outbox (key = vehicle id); consumed: identity-service's `identity.user-events` (logged only, for now) and scheduler-service's `ServiceCompleted`, which feeds `GET /vehicle/:id/history`. Payloads and topics: [architecture-design.md §3b](../docs/architecture-design.md#3b-vehicle-service-event-contracts).

## Scripts

- `scripts/transfer-race-test.sh`: concurrent `POST /transfer` for one vehicle; exactly one must win.
- `scripts/fake-traffic.sh`: request mix for generating Prometheus metrics.
