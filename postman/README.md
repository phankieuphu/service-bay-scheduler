# Postman

`service-bay.postman_collection.json` is generated. Edit `gen_collection.py`, then run:

```bash
python3 postman/gen_collection.py
```

## Folders

| Folder | What it is |
|---|---|
| **Full flow** | End-to-end business flows with assertions, meant to be run in order (below). |
| Identity / Customer / Vehicle / Billing / Payment Service | One request per endpoint, for manual poking. Create requests store ids (`customerId`, `vehicleId`, `billId`, `externalPaymentId`) in collection variables. |

## Full flow

It uses the seeded data from `postgres/seed/`:

| Pair | Why |
|---|---|
| customer **685** / vehicle **1** | Vehicle has no warranty, so checkout charges the warranty/expiration fee |
| customer **625** / vehicle **3** | Warranty runs to 2029, so there's no fee |
| customer **148** / vehicle **1** | 148 doesn't own vehicle 1, so checkout answers 409 |

| Flow | Steps |
|---|---|
| 1. Seeded customer, no warranty | Health → customer / ownership / warranty lookups → **checkout** → duplicate checkout (409) → lines → cash deposit (PARTIALLY_PAID) → card checkout for the balance → over-balance (409) → declined test card → successful test card → poll until the bill is **PAID** (Kafka round trip) → cancel a charged card (409) → void a paid bill (409) |
| 2. Seeded customer in warranty | Checkout with no fee → card checkout → customer walks away, cancel → **void** → payment on a void bill (409) |
| 3. New walk-in customer | Create customer → register a vehicle whose warranty has expired → checkout before ownership (409) → assign owner → checkout (fee charged) → pay all in cash → **PAID** |
| 4. Checkout validation | Not the owner (409), unknown customer (404), no services (400), price with 3 decimals (400) |

Run it from the Postman Collection Runner (select the **Full flow** folder, keep the order), or from the terminal with newman. The runner needs Node.js 18+ and fetches newman through `npx`:

```bash
docker compose up -d
postman/run-flow.sh                                   # all four flows
FLOW="4. Checkout validation" postman/run-flow.sh     # one flow (exact folder name)
BILLING_URL=http://localhost:58084 postman/run-flow.sh  # other ports: CUSTOMER_URL, VEHICLE_URL, PAYMENT_URL, IDENTITY_URL
postman/run-flow.sh --reporters cli,junit             # extra args go to newman
```

The exit code is non-zero if any assertion fails.

**Card steps** need Stripe **test** keys on payment-service (`STRIPE_SECRET_KEY=sk_test_...` in the root `.env`). They use payment-service's `test-confirm` endpoint with Stripe's test cards (`pm_card_visa`, `pm_card_chargeDeclined`), so no browser and no `stripe listen` are needed. Without keys, billing answers 502 on card checkout. The flow then reports the card steps as skipped and runs everything else.

The seeded pairs assume a fresh database. If you have transferred vehicle 1 or 3 since seeding, change `seedCustomer*` / `seedVehicle*` in the collection variables.
