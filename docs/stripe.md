# Stripe setup

billing-service takes deposits and bill payments through Stripe PaymentIntents. This guide covers the keys, the webhook and how to check that a payment goes through, locally and when deployed. For the API and the booking-to-paid-bill flow, see [billing-service/README.md](../billing-service/README.md#how-a-booking-gets-paid). For the event contracts, see [architecture-design.md §3e](architecture-design.md).

Use **test-mode** keys everywhere outside production.

## How a payment is confirmed

```text
browser ──(Payment Element, client_secret)──▶ Stripe
Stripe  ──POST /webhooks/stripe (signed)────▶ billing-service ──▶ re-reads the PaymentIntent ──▶ payment SUCCEEDED / FAILED
browser ──GET /payments/:id (polls)─────────▶ billing-service
```

The webhook decides whether a payment succeeded, not the browser redirect. Without a working webhook, payments made in the browser stay `REQUIRES_PAYMENT`/`PROCESSING` until staff call `POST /payments/:id/sync`.

billing-service doesn't trust the webhook payload. It verifies the `Stripe-Signature` header with `STRIPE_WEBHOOK_SECRET`, then fetches the PaymentIntent from Stripe and applies that. Only `payment_intent.*` events change state. Any other event is acknowledged with 200 and ignored.

## The three values

| Variable | Where it comes from | Used by | Missing → |
|---|---|---|---|
| `STRIPE_SECRET_KEY` (`sk_test_…`) | [Dashboard → API keys](https://dashboard.stripe.com/test/apikeys) | billing-service | payment endpoints answer **503** "Stripe secret key is not configured" |
| `STRIPE_WEBHOOK_SECRET` (`whsec_…`) | `stripe listen` locally; the webhook endpoint's signing secret when deployed | billing-service | `/webhooks/stripe` answers **503** "Stripe webhook secret is not configured" |
| `STRIPE_PUBLISHABLE_KEY` / `VITE_STRIPE_PUBLISHABLE_KEY` (`pk_test_…`) | Dashboard → API keys | billing-service / frontend | the pay buttons explain that Stripe isn't set up |

Optional: `STRIPE_RETURN_URL` is where Stripe sends the browser back after a redirect-based payment method (docker compose defaults it to `http://localhost:5173/`). `STRIPE_API_BASE` points the SDK at [stripe-mock](https://github.com/stripe/stripe-mock) instead of api.stripe.com.

## Local setup (docker compose)

### 1. Put the keys in the root `.env`

Docker compose fills billing-service's environment from your shell or from the **repo root** `.env` (gitignored). It does **not** read `billing-service/.env`. That file is only for running `./gradlew bootRun` yourself, and even then you have to load it into your shell first.

```bash
# <repo root>/.env
STRIPE_SECRET_KEY=sk_test_...
STRIPE_PUBLISHABLE_KEY=pk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...     # from step 2
```

```bash
# frontend/.env
VITE_STRIPE_PUBLISHABLE_KEY=pk_test_...
```

### 2. Forward webhooks with the Stripe CLI

Stripe can't reach `localhost`, so the CLI receives the events and forwards them.

```bash
brew install stripe/stripe-cli/stripe
stripe login
stripe listen \
  --events payment_intent.succeeded,payment_intent.payment_failed,payment_intent.canceled,payment_intent.processing,payment_intent.requires_action \
  --forward-to localhost:8084/webhooks/stripe
```

Current CLI versions refuse to start without `--events` (or `--all-snapshot`). The list above is every event billing-service acts on.

`stripe listen` prints `Your webhook signing secret is whsec_...`. Copy it into `STRIPE_WEBHOOK_SECRET`. It stays the same between runs on the same machine and login, so usually you copy it only once. Keep `stripe listen` running while you test.

Without Homebrew, run the CLI in Docker. Inside a container `localhost` is the container itself, so forward to `host.docker.internal`:

```bash
docker run --rm -it -v ~/.config/stripe:/root/.config/stripe stripe/stripe-cli login
docker run --rm -it -v ~/.config/stripe:/root/.config/stripe stripe/stripe-cli \
  listen --events payment_intent.succeeded,payment_intent.payment_failed,payment_intent.canceled,payment_intent.processing,payment_intent.requires_action \
  --forward-to host.docker.internal:8084/webhooks/stripe
```

### 3. Recreate billing-service

A plain `restart` keeps the old environment, so recreate the container:

```bash
docker compose up -d --force-recreate billing-service
```

### 4. Check it

```bash
# Both lengths should be > 0. This prints the lengths only, never the keys.
docker compose exec billing-service sh -c 'echo ${#STRIPE_SECRET_KEY} ${#STRIPE_WEBHOOK_SECRET}'

# The startup warning "STRIPE_SECRET_KEY is not set" should be gone.
docker compose logs billing-service | grep -i stripe

# Sends a test event: stripe listen shows "<-- [200] POST .../webhooks/stripe",
# and billing logs it as an intent that "isn't one of ours".
stripe trigger payment_intent.succeeded
```

Then run a real payment: in the frontend, book an appointment, pay the deposit with card `4242 4242 4242 4242` (any future expiry, any CVC) and watch it turn **paid**. To run the whole flow without a browser, use `postman/run-flow.sh` ([postman/README.md](../postman/README.md)). It pays through `POST /payments/:id/test-confirm`, which applies the result directly and doesn't need the webhook.

Test cards:

| Card | Result |
|---|---|
| `4242 4242 4242 4242` | succeeds |
| `4000 0000 0000 0002` | declined |
| `4000 0025 0000 3155` | asks for 3-D Secure |

## Deployed (Kubernetes / a public URL)

Here Stripe calls billing-service directly, so you register an endpoint in the dashboard instead of running `stripe listen`.

1. [Dashboard → Developers → Webhooks](https://dashboard.stripe.com/test/webhooks) → **Add endpoint**.
2. Endpoint URL: `https://<billing-service public host>/webhooks/stripe`. The route takes no JWT; the signature is the check.
3. Events: the five `payment_intent.*` events listed in step 2 above.
4. Copy the endpoint's **signing secret** (`whsec_…`). It's different from the CLI's secret, and test and live mode each have their own.
5. Give it to billing-service. On Kubernetes:

   ```bash
   kubectl -n service-bay create secret generic billing-stripe \
     --from-literal=STRIPE_SECRET_KEY=sk_test_... \
     --from-literal=STRIPE_PUBLISHABLE_KEY=pk_test_... \
     --from-literal=STRIPE_WEBHOOK_SECRET=whsec_...
   kubectl -n service-bay rollout restart deployment/billing-service
   ```

   The secret is optional in [deployment.yaml](../k8s/base/billing-service/deployment.yaml), so without it the pod starts and payments answer 503.
6. Set `STRIPE_RETURN_URL` to the deployed frontend's URL.
7. Send a test event from the endpoint's page in the dashboard. It should show `200`.

## Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| `Stripe secret key is not configured` / 503 on deposit or pay | `STRIPE_SECRET_KEY` is empty inside the container | put it in the **root** `.env`, not `billing-service/.env`; `--force-recreate` |
| `/webhooks/stripe` → 503 | `STRIPE_WEBHOOK_SECRET` empty | step 2, then recreate |
| `/webhooks/stripe` → 400 `invalid Stripe signature` | the secret doesn't match the sender: CLI secret used for a dashboard endpoint (or the other way round), or test/live mixed up | use the secret of whatever is actually sending |
| `stripe listen`: `must specify events to forward` | newer CLI | add `--events …` (step 2) |
| `zsh: command not found: stripe` | CLI not installed | `brew install stripe/stripe-cli/stripe`, or the Docker form |
| Card accepted in the browser but the payment never turns paid | the webhook isn't arriving | check `stripe listen` is running and shows `[200]`; staff can recover with `POST /payments/:id/sync` |
| `test-confirm` → 403 | it only works with `sk_test_` keys | use a test-mode key |
| `stripe listen` shows `connection refused` | billing-service isn't up on 8084 | `docker compose ps billing-service`; in Docker, forward to `host.docker.internal` |
