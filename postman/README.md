# Postman

`service-bay.postman_collection.json` is generated. Edit `gen_collection.py`, then run `python3 postman/gen_collection.py`.

## Full flow

One booking, from sign-up to a paid bill, with assertions at every step. Run the **Full flow** folder in order: in Postman's Collection Runner, or from a terminal against a running stack with

```bash
docker compose up -d
postman/run-flow.sh            # newman via npx (Node.js 18+); non-zero exit on any failed assertion
```

| Part | What it does |
|---|---|
| 1. Accounts | Logs in as the bootstrap admin (staff token), registers a fresh customer and logs in, waits for customer-service to create their profile, then (as staff) registers a vehicle whose warranty has expired and assigns it to them. |
| 2. Booking and deposit | Picks the first free slot tomorrow and books an oil change and a brake inspection (105.50). Pays a 20 % deposit (21.10), checks that retrying with the same `Idempotency-Key` returns the same payment, has a card declined, then pays with a good one. The bill is `OPEN`, with no balance yet. |
| 3. Completion and pricing | Staff completes the appointment (a customer gets 403). Waits for billing to price the bill from `ServiceCompleted`: a warranty fee, because the warranty expired, plus both services comes to 155.50, minus the deposit. Checks staff can read it, no token gets 401, and the visit is in the vehicle's service history. |
| 4. Paying the balance | Pays exactly the balance, then checks the bill is `PAID` and shows only card brand and last 4. A second payment answers 409. |

Card steps use billing-service's `test-confirm` endpoint with Stripe's test payment methods, so no browser is needed. They need **Stripe test keys** on billing-service (`STRIPE_SECRET_KEY=sk_test_...` in the root `.env`). Without them billing answers 503, the card steps are reported as skipped, and the flow still checks booking, completion and pricing.

The staff login uses the bootstrap admin from `docker-compose.yml` (`adminEmail` / `adminPassword` collection variables).
