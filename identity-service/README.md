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
| `POST /auth/register` | `{email, password, role?}` | `201` user | `400` validation, `403` role other than `CUSTOMER`, `409` email taken, `429` rate limited |
| `POST /auth/login` | `{email, password}` | `200` access token, sets the refresh cookie | `401` wrong email or password (same response for both), `403` account not `ACTIVE`, `429` rate limited |
| `POST /auth/refresh` | none (refresh cookie + `X-Requested-With`) | `200` new access token, rotates the cookie | `401` missing, unknown, used, revoked or expired token (cookie cleared), `403` no `X-Requested-With` |
| `POST /auth/logout` | none (refresh cookie + `X-Requested-With`) | `204`, clears the cookie | `403` no `X-Requested-With` (otherwise idempotent) |
| `GET /me` | `Authorization: Bearer <access_token>` | `200` user | `401` |
| `POST /admin/users` | `{email, password, role}` (ADMIN token) | `201` user | `400` validation, `401`, `403` not an active admin, `409` email taken |
| `PATCH /admin/users/:id` | `{status}`: `ACTIVE`/`INACTIVE`/`LOCKED` (ADMIN token) | `200` user | `400` validation or own account, `401`, `403`, `404` |

`role` defaults to `CUSTOMER`, and `CUSTOMER` is the only self-registrable role. `TECHNICIAN`, `MANAGER` and `ADMIN` are staff roles that other services grant access to, so an admin creates them with `POST /admin/users`. Passwords are 8–72 characters (72 is bcrypt's input limit). Emails are trimmed, lowercased, and unique case-insensitively.

Token response (the refresh token is never in the body):

```json
{ "access_token": "eyJ...", "token_type": "Bearer", "expires_in": 900 }
```

## Tokens

- **Access token**: HS256 JWT, 15 min (`JWT_ACCESS_TTL_SEC`). Claims: `sub` (user id), `role`, `iss`, `iat`, `exp`. Stateless, so it stays valid until expiry even after logout.
- **Refresh token**: 256-bit random opaque string, 7 days (`JWT_REFRESH_TTL_SEC`). Stored in Redis as `refresh:<sha256(token)>` → user id, never in plain text. Each refresh atomically consumes it (`GETDEL`) and issues a new one, so a replayed token is rejected. Redis is therefore a critical `/readyz` dependency here.
- **Refresh cookie**: the refresh token travels only in a `refresh_token` cookie that is `HttpOnly; Secure; SameSite=Strict; Path=/api/v1/auth`, so page scripts, including any XSS, can't read it, and it's only sent to the auth routes. `/auth/refresh` and `/auth/logout` read it from there and also require an `X-Requested-With` header. A cross-site form can't send that header, and a cross-origin `fetch` that does has to pass a CORS preflight.
- **CORS**: a cookie means credentialed CORS, which rules out `*`. Only origins in `CORS_ALLOWED_ORIGINS` (comma-separated, default `http://localhost:5173`) get `Access-Control-Allow-Origin` (their own origin echoed back) and `Access-Control-Allow-Credentials: true`. Browser clients call the auth routes with `credentials: 'include'`.
- `SameSite=Strict` assumes the frontend and the API are **same-site** (same registrable domain, e.g. `app.example.com` and `api.example.com`, or `localhost` on different ports). A frontend on a different site wouldn't send the cookie at all.
- `AUTH_COOKIE_SECURE=false` drops `Secure`, for plain-HTTP local setups that won't store Secure cookies (Safari, Postman). Chrome and Firefox accept `Secure` on `http://localhost`. docker-compose sets it to `false`.

## Admin

- `/admin/*` needs an `ADMIN` access token. Each call also re-reads the caller from the database and requires an `ACTIVE` `ADMIN`, so a demoted or locked admin is refused even while their token is still valid (up to 15 min).
- `POST /admin/users` creates a user with any role and emits `UserCreated`, like registration does.
- `PATCH /admin/users/:id` sets a user's status and emits `UserUpdated`. A `LOCKED` or `INACTIVE` user can't log in or refresh. Their current access token stays valid until it expires. An admin can't change their own status, so the last admin can't lock themselves out.
- **First admin:** set `ADMIN_EMAIL` and `ADMIN_PASSWORD`. On startup, if no `ADMIN` exists, one is created with them. An existing non-admin account with that email is never promoted. Once the admin exists, the variables are ignored, and you can remove them. docker-compose sets `admin@servicebay.local` / `admin-local-dev-pass`.

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
