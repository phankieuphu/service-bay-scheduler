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
| `POST /auth/register` | `{email, password, role?}` | `201` user | `400` validation, `403` role is `MANAGER`/`ADMIN`, `409` email taken, `429` rate limited |
| `POST /auth/login` | `{email, password}` | `200` tokens | `401` wrong email or password (same response for both), `403` account not `ACTIVE`, `429` rate limited |
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

## Rate limiting

Login and registration are rate-limited with fixed-window counters in Redis (`attempts:*` keys). Over a limit, the response is `429` with `Retry-After: <seconds>`.

| Limit | Default | Env |
|---|---|---|
| Failed logins per email (normalized, stored as a SHA-256 hash) | 10 per 15 min | `RATE_LIMIT_LOGIN_PER_EMAIL`, `RATE_LIMIT_LOGIN_WINDOW_SEC` |
| Failed logins per client IP | 100 per 15 min | `RATE_LIMIT_LOGIN_PER_IP` |
| Registrations per client IP (every attempt) | 20 per hour | `RATE_LIMIT_REGISTER_PER_IP`, `RATE_LIMIT_REGISTER_WINDOW_SEC` |

- Only `401` logins count as failures. A successful login resets that email's count but not the IP's.
- A blocked login is rejected before the password is checked, so it costs no bcrypt work. The response is the same whether or not the email exists.
- Set a limit to `0` to disable it. If Redis errors, requests are let through and the error is logged.
- The client IP is the TCP peer address. `X-Forwarded-For` is only believed from proxies listed in `API_TRUSTED_PROXIES` (comma-separated IPs/CIDRs, default none). Otherwise anyone could reset their per-IP count by sending their own header. Behind a load balancer or ingress, list its addresses there. Without that, every request appears to come from the proxy.

## Events

Registration writes the user and a `UserCreated` outbox row in one transaction; the outbox relay publishes it to `identity.user-events`, keyed by user id. Payload contract: [architecture-design.md §3a](../docs/architecture-design.md#3a-identityuser-events-contract).
