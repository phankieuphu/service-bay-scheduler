# identity-service

Login identities (User, Credential, Role) for Service Bay: registration, login, token refresh/logout, and the `identity.user-events` stream that customer-service and vehicle-service consume. Same hexagonal layout as the other services (see the root [CLAUDE.md](../CLAUDE.md)); it has a Kafka producer + outbox relay but no consumer, since nothing is upstream of it.

## Run

```bash
cp .env.example .env
go run ./cmd/server          # :8083
go test ./...
```

Or `docker compose up -d identity-service` from the repo root. The schema is [`postgres/init/01-identity-service.sql`](../postgres/init/01-identity-service.sql); Postgres only runs init scripts on an empty data volume, so an existing `postgres_data` volume needs `docker compose down -v` (which deletes all local data) to pick it up.

## API

All under `/api/v1`. Errors are `{"error": "..."}`.

| Method & path | Body | Success | Errors |
|---|---|---|---|
| `POST /auth/register` | `{email, password, role?}` | `201` user | `400` validation, `403` role is `MANAGER`/`ADMIN`, `409` email taken |
| `POST /auth/login` | `{email, password}` | `200` tokens | `401` wrong email or password (same response for both), `403` account not `ACTIVE` |
| `POST /auth/refresh` | `{refresh_token}` | `200` new tokens | `401` unknown, used, revoked or expired token |
| `POST /auth/logout` | `{refresh_token}` | `204` | — (idempotent) |
| `GET /me` | `Authorization: Bearer <access_token>` | `200` user | `401` |

`role` defaults to `CUSTOMER`; `TECHNICIAN` is also self-registrable. Passwords are 8–72 characters (72 is bcrypt's input limit). Emails are trimmed, lowercased, and unique case-insensitively.

Token response:

```json
{ "access_token": "eyJ...", "token_type": "Bearer", "expires_in": 900, "refresh_token": "..." }
```

## Tokens

- **Access token**: HS256 JWT, 15 min (`JWT_ACCESS_TTL_SEC`). Claims: `sub` (user id), `role`, `iss`, `iat`, `exp`. Stateless, so it stays valid until expiry even after logout.
- **Refresh token**: 256-bit random opaque string, 7 days (`JWT_REFRESH_TTL_SEC`). Stored in Redis as `refresh:<sha256(token)>` → user id, never in plain text. Each refresh atomically consumes it (`GETDEL`) and issues a new one, so a replayed token is rejected. Redis is therefore a critical `/readyz` dependency here.

## Events

Registration writes the user and a `UserCreated` outbox row in one transaction; the outbox relay publishes it to `identity.user-events`, keyed by user id. Payload contract: [architecture-design.md §3a](../docs/architecture-design.md#3a-identityuser-events-contract).
