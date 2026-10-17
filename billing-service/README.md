# billing-service

Deposits, bills and payments for Service Bay, with **Stripe** as the payment provider. It's the platform's only Java service (Spring Boot 4, Java 25). See [technical-requirement.md](../docs/technical-requirement.md) for why, and [#35](https://github.com/phankieuphu/service-bay-scheduler/issues/35) for the plan.

## Layout

Same hexagonal shape as the Go services:

| Package (`com.servicebay.billing`) | Holds |
|---|---|
| `domain` | Entities, ports (interfaces), services. **No Spring, JPA or Stripe imports.** `domain.money.Money` is the only way to hold an amount. |
| `adapters.http` | Controllers and DTOs |
| `adapters.persistence` | JPA entities and repositories implementing the domain's ports |
| `adapters.kafka` | Outbox relay and consumers |
| `adapters.stripe` | The only package that touches `stripe-java` |
| `adapters.security` | JWT verification (identity-service's tokens) and billing's own service token |
| `adapters.upstream` | HTTP clients for customer-, vehicle- and scheduler-service |
| `application` | Use cases (deposit, pay, webhooks, pricing, reads), each drawing its own transaction boundaries, plus Spring wiring |

**Money:** a `long` count of minor units plus an ISO 4217 code, never `double`. Minor units follow Stripe's convention. Zero-decimal currencies (VND, JPY, …) have no cents, so `Money.of(150000, "VND")` is 150 000 ₫. `Money.ofMajor` refuses amounts with more precision than the currency has.

**Schema:** Flyway migrations in `src/main/resources/db/migration`, applied on startup. Hibernate only validates against them (`ddl-auto: validate`).

## Run

```bash
cd billing-service
./gradlew bootRun        # needs Postgres (billing_db) and Kafka; env in .env.example
./gradlew test           # unit tests + Testcontainers integration tests (needs Docker)
./gradlew test --tests '*FlowTest'   # the payment flows only, no Docker (in-memory ports)
./gradlew bootJar        # build/libs/billing-service-<version>.jar
```

The Gradle toolchain pins Java 25 and downloads it if it isn't installed. Gradle itself runs on any JDK 17+.

With the whole stack: `docker compose up -d billing-service` (port **8084**).

| Endpoint | |
|---|---|
| `GET /actuator/health/liveness` | the process is up |
| `GET /actuator/health/readiness` | UP once Postgres is reachable |
| `GET /actuator/prometheus` | metrics, scraped by Prometheus |

`server.shutdown=graceful`: on SIGTERM, in-flight requests get up to `SHUTDOWN_TIMEOUT` (15s) to finish.

## API

Every `/api/v1` route needs an identity-service access token (`Authorization: Bearer …`). A customer sees and pays only their own bills: someone else's answers **404**, not 403, so ids can't be probed. Staff (`TECHNICIAN`/`MANAGER`/`ADMIN`) can read every bill. Money is integer minor units plus `currency`, and the client formats it. JSON is snake_case.

| Method | Path | Who | |
|---|---|---|---|
| `POST` | `/bookings/:id/deposit` | the booking's customer | Needs an `Idempotency-Key` header. Starts or resumes the deposit; the amount is the deposit policy applied to the booking's estimate. Returns `payment_id`, `client_secret`, `amount_minor`, `currency`, `status`. `409` if the booking isn't upcoming or the deposit is paid or processing. |
| `POST` | `/bills/:id/payments` | the bill's customer | Needs an `Idempotency-Key` header. Pays the **current balance**, computed by the server. Open intents for another amount (an unpaid deposit, a stale balance) are canceled first. `409` unless the bill is `PENDING`/`PARTIALLY_PAID` with nothing processing. |
| `GET` | `/bills/:id`, `/bookings/:id/bill` | owner, staff | Lines, fees, `total_minor`, payments (card brand and last 4 only), `amount_paid_minor`, `balance_due_minor`, `refund_due_minor`. While `OPEN` (not priced) `total_minor` and `balance_due_minor` are `null`, not 0. |
| `GET` | `/customers/:id/bills`, `/me/bills` | owner, staff | `?cursor=&limit=` (default 20, max 100), newest first; `next_cursor` / `has_more` as in the Go services. |
| `GET` | `/payments/:id` | owner, staff | What the frontend polls after confirming: the webhook decides success, not the redirect. |
| `POST` | `/payments/:id/sync` | staff | Re-reads the PaymentIntent from Stripe and applies it, to recover a missed webhook. |
| `POST` | `/payments/:id/test-confirm` | owner, staff | **Test-mode keys only** (403 otherwise). `{"payment_method":"pm_card_visa"}` (or `pm_card_chargeDeclined`) pays without a browser. |
| `POST` | `/webhooks/stripe` | Stripe | No JWT: the signature over the raw body is the check. `400` bad signature, `503` no `STRIPE_WEBHOOK_SECRET`. |

Errors: `400` bad input or a missing `Idempotency-Key`, `401` no or invalid token, `404` unknown or not yours, `409` the state doesn't allow it, `502` Stripe or another service failed, `503` Stripe isn't configured.

## How a booking gets paid

1. **Booking** (`AppointmentBooked`): billing stores the booking and its estimate in `appointment_ref`.
2. **Deposit** (optional, `POST /bookings/:id/deposit`): an `OPEN` bill (unpriced) is created with a `DEPOSIT` payment, and the browser confirms it with the Payment Element. The webhook marks it `SUCCEEDED` and publishes `PaymentReceived`. A deposit request that beats the event asks scheduler for the caller's own booking.
3. **Completion** (`ServiceCompleted`): the bill is priced as warranty fee (only if the vehicle isn't in warranty that day, per vehicle-service) plus service fees, the deposit counts toward it, and `BillReady` goes out if anything is due.
4. **Balance** (`POST /bills/:id/payments`): a `FINAL` payment for what's due. The webhook settles it, the bill goes `PAID`, and `PaymentReceived` goes out.
5. **Cancellation** (`AppointmentCancelled`): open deposit intents are canceled. An unpaid bill becomes `VOID`; one with a paid deposit stays `OPEN` until the refund policy exists (#89).

Configuration (`.env.example`): `DEPOSIT_POLICY` is `FIXED`, `PERCENT` or `FULL`, with `DEPOSIT_VALUE` (default 20 %). `BILLING_WARRANTY_FEE` defaults to 50.00. Both are placeholders until business-requirement §10 #5 and #6 are decided. `BILLING_CURRENCY` must match scheduler-service's `CATALOG_CURRENCY`.

## Stripe locally

Full setup, the deployed webhook endpoint and troubleshooting: [docs/stripe.md](../docs/stripe.md).

Use **test-mode** keys from https://dashboard.stripe.com/test/apikeys in the repo root `.env`, which docker compose reads:

```bash
STRIPE_SECRET_KEY=sk_test_...
STRIPE_PUBLISHABLE_KEY=pk_test_...        # also VITE_STRIPE_PUBLISHABLE_KEY in frontend/.env
STRIPE_WEBHOOK_SECRET=whsec_...           # printed by the command below
```

```bash
stripe listen --forward-to localhost:8084/webhooks/stripe \
  --events payment_intent.succeeded,payment_intent.payment_failed,payment_intent.canceled,payment_intent.processing,payment_intent.requires_action
                                                              # forwards webhooks, prints whsec_...
stripe trigger payment_intent.succeeded                       # an event not for one of our payments: acknowledged and ignored
```

Test cards in the Payment Element: `4242 4242 4242 4242` succeeds, `4000 0000 0000 0002` is declined, `4000 0025 0000 3155` asks for 3-D Secure. Any future expiry date and any CVC work. Without a browser, `POST /payments/:id/test-confirm` does the same with `pm_card_visa` / `pm_card_chargeDeclined`. `STRIPE_API_BASE` points the SDK at [stripe-mock](https://github.com/stripe/stripe-mock) instead of api.stripe.com.

Before production: confirm Stripe supports the business's country for merchant accounts (#83: it didn't onboard Vietnam-registered businesses last time this was checked). If it doesn't, a local provider plugs in behind `domain.ports.PaymentProvider`.
