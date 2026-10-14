# scheduler-service

Booking for Service Bay: which hubs and services exist, which start times are free, and a customer's own appointments. Go, same hexagonal layout as the other Go services (see the root [CLAUDE.md](../CLAUDE.md)). Port `8085`, database `scheduler_db`.

## Run

```bash
docker compose up -d --build scheduler-service     # from the repo root; needs customer-service and vehicle-service for booking
# or
cp .env.example .env && go run ./cmd/server
```

Migrations in [migrations/](migrations/) are applied on startup (`DB_AUTO_MIGRATE=false` to turn off; `go run ./cmd/migrate status|up|down`). The hubs, bays and service catalog come from the dev seed, [postgres/seed/init-scheduler.sql](../postgres/seed/init-scheduler.sql), which `db-seed` loads.

| Variable | Default | |
|---|---|---|
| `DB_*` | `localhost:5432`, `scheduler_db` | as in the other services |
| `API_PORT` | `8085` | |
| `KAFKA_BROKERS` | `localhost:9092` | outbox relay target |
| `CUSTOMER_API_URL` | `http://localhost:8080/api/v1` | `GET /customer/me` |
| `VEHICLE_API_URL` | `http://localhost:8081/api/v1` | `GET /customers/:id/vehicles` |
| `UPSTREAM_TIMEOUT_SEC` | `5` | per call to the two above |
| `JWT_SECRET`, `JWT_ISSUER` | `""`, `identity-service` | must match identity-service |

## API

Every `/api/v1` route needs an identity-service access token (`Authorization: Bearer …`). Times are RFC3339 UTC with milliseconds; money is a decimal string.

| Method | Path | Who | |
|---|---|---|---|
| `GET` | `/hubs` | any signed-in user | active hubs, opening hours (`HH:MM` in `timezone`), bookable bay count |
| `GET` | `/services` | any | the bookable catalog: fee, duration |
| `GET` | `/hubs/:id/availability?date=YYYY-MM-DD&service_id=1&service_id=3` | any | start times every 30 min on that local day; `available` is false in the past or when no bay is free for the services' total duration |
| `POST` | `/appointments` | `CUSTOMER` | `{"hub_id":1,"vehicle_id":7,"service_ids":[1,3],"start":"2026-10-20T10:00:00+07:00"}` → `201` appointment |
| `GET` | `/me/appointments` | `CUSTOMER` | the caller's appointments, latest first |
| `POST` | `/appointments/:id/cancel` | `CUSTOMER` | cancel an upcoming `BOOKED` appointment of the caller's |

Errors: `400` invalid input (unknown service, outside opening hours, a vehicle that isn't the caller's), `404` not found or no linked customer profile, `409` the slot is taken / the vehicle is already booked then / not cancellable, `502` customer-service or vehicle-service failed.

## How booking stays correct

1. Hub, services and opening hours are checked against the local snapshot (`hub`, `hub_bay`, `catalog_service`).
2. customer-service says who the caller is (`GET /customer/me`, with the caller's own token), and vehicle-service that they own the vehicle (architecture-design.md §4 option 1, §5a).
3. In one transaction: a per-vehicle advisory lock, then a check that the vehicle has no overlapping live appointment, then the first free bay is picked and `appointment`, `appointment_vehicle`, `appointment_bay`, `appointment_service` and the outbox row are inserted.
4. The `no_bay_double_booking` exclusion constraint on `appointment_bay` is the final guard: if a concurrent booking took the bay first, the insert fails with `23P01` and the request gets `409`.

Fees are copied onto `appointment_service.fee_charged` at booking time (§5c). Events: `scheduler.appointment.booked.v1` and `scheduler.appointment.cancelled.v1`, contract in [architecture-design.md §3d](../docs/architecture-design.md).

## Not built yet

Staff views of a hub's day, check-in / start / complete, technician assignment, `ServiceCompleted` (§3b), and keeping the hub/bay/catalog snapshot current from dealership-service events.
