# Deploying to Railway (backend) and Vercel (frontend)

The backend runs on Railway: one project, with one Railway service per
deployable, all built from this repo. Each deployable has a `railway.json`
next to its `Dockerfile`. Railway picks it up through the service's
**Config-as-code path**.

| Railway service    | Source                           | Root directory      | Config file path                  | Public domain |
|--------------------|----------------------------------|---------------------|-----------------------------------|---------------|
| `postgres`         | this repo                        | `/postgres`         | `/postgres/railway.json`          | no            |
| `kafka`            | this repo                        | `/kafka`            | `/kafka/railway.json`             | no            |
| `redis`            | Railway **Redis** template       | n/a                 | n/a                               | no            |
| `identity-service` | this repo                        | `/identity-service` | `/identity-service/railway.json`  | yes           |
| `customer-service` | this repo                        | `/customer-service` | `/customer-service/railway.json`  | yes           |
| `vehicle-service`  | this repo                        | `/vehicle-service`  | `/vehicle-service/railway.json`   | yes           |

Service names matter: the variable references below (`${{postgres.…}}`,
`${{kafka.…}}`, `${{redis.…}}`) use them.

The observability stack (Prometheus, Grafana, Loki, Alloy) and Flink are
not deployed. They stay local-only (docker-compose / k8s).

## 1. Stateful services

### postgres

This builds `postgres/Dockerfile`, so the `postgres/init` scripts create one
database per service on first boot. Railway's managed Postgres template
doesn't run those scripts.

- Attach a **volume** mounted at `/var/lib/postgresql/data`.
- Variables:

  ```
  POSTGRES_USER=postgres
  POSTGRES_PASSWORD=<generate>
  POSTGRES_DB=postgres
  PGDATA=/var/lib/postgresql/data/pgdata   # volume root has lost+found; initdb needs an empty dir
  ```

The init scripts run only when the volume is empty. To re-run them, wipe the volume.

### kafka

Single-node KRaft broker (`kafka/Dockerfile`, same image as docker-compose).

- Attach a **volume** mounted at `/var/lib/kafka/data`.
- Variables:

  ```
  KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://${{RAILWAY_PRIVATE_DOMAIN}}:9092
  ```

### redis

Add it from Railway's **Redis** template, and name the service `redis`.

## 2. Go services

The services read `API_PORT`, not Railway's `PORT`, so `API_PORT` has to be
set explicitly. When generating the public domain, use the same port as the
target port. The deploy healthcheck is `GET /readyz`, which fails only when
Postgres is unreachable.

Shared variables (set them on each of the three services, or as project
shared variables):

```
DB_HOST=${{postgres.RAILWAY_PRIVATE_DOMAIN}}
DB_PORT=5432
DB_USERNAME=${{postgres.POSTGRES_USER}}
DB_PASSWORD=${{postgres.POSTGRES_PASSWORD}}
DB_SSLMODE=disable
KAFKA_BROKERS=${{kafka.RAILWAY_PRIVATE_DOMAIN}}:9092
REDIS_HOST=${{redis.RAILWAY_PRIVATE_DOMAIN}}
REDIS_PORT=6379
REDIS_PASSWORD=${{redis.REDISPASSWORD}}
```

Per service:

```
# identity-service
API_PORT=8083
DB_NAME=identity_db
KAFKA_PRODUCER_TOPIC=identity.user-events
JWT_SECRET=<generate, 32+ chars>       # required, no default

# customer-service
API_PORT=8080
DB_NAME=customer_db
KAFKA_PRODUCER_TOPIC=customer.events
KAFKA_CONSUMER_TOPIC=identity.user-events
KAFKA_CONSUMER_GROUP=customer-service

# vehicle-service
API_PORT=8081
DB_NAME=vehicle_db
KAFKA_PRODUCER_TOPIC=vehicle.events
KAFKA_CONSUMER_TOPIC=identity.user-events
KAFKA_CONSUMER_GROUP=vehicle-service
```

## 3. frontend (Vercel)

The frontend is a static Vite build, so it is deployed to Vercel rather than
Railway. Import the repo in Vercel and set **Root Directory** to `frontend`.
`frontend/vercel.json` pins the framework, install, build and output settings.

Add these Environment Variables in the Vercel project. Vite inlines `VITE_*`
at **build** time, so changing them requires a redeploy.

```
VITE_IDENTITY_API_URL=https://<identity-service public domain>/api/v1
VITE_CUSTOMER_API_URL=https://<customer-service public domain>/api/v1
VITE_VEHICLE_API_URL=https://<vehicle-service public domain>/api/v1
```

The browser calls the Railway APIs cross-origin. That works because all three
services send `Access-Control-Allow-Origin: *`. Vercel preview deployments
use the same variables, so previews also hit the production APIs unless you
scope different values to the Preview environment.

## Notes

- `watchPatterns` in each `railway.json` limit rebuilds to commits that touch
  that service's directory.
- Railway's private network can be IPv6-only in older environments. The Go
  servers (`:PORT`) and Kafka (`PLAINTEXT://:9092`) both bind dual-stack.
