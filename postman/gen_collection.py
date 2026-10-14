"""Generates service-bay.postman_collection.json.

Routes and bodies are hand-mirrored from each service's HTTP handlers/DTOs —
update this file when an endpoint changes, then run:

    python3 postman/gen_collection.py [output-path]
"""
import json, os, sys
out = sys.argv[1] if len(sys.argv) > 1 else os.path.join(os.path.dirname(os.path.abspath(__file__)), "service-bay.postman_collection.json")

# Variables substituted into JSON bodies as numbers, not strings.
NUMERIC_VARS = ("customerId", "targetCustomerId", "vehicleId", "billId", "appointmentId",
                "flowCustomerId", "flowVehicleId", "depositAmount", "cardAmount", "cashAmount")

def save(var, expr):
    return [f"if (pm.response.code < 300) {{ const j = pm.response.json(); if ({expr} !== undefined) pm.collectionVariables.set('{var}', {expr}); }}"]

def req(name, method, base, path, body=None, query=None, desc="", tests=None, auth=False, pre=None):
    segs = [s for s in path.strip("/").split("/")]
    url = {"raw": "{{%s}}/%s" % (base, "/".join(segs)) + ("?" + "&".join(f"{q['key']}={q['value']}" for q in query if not q.get("disabled")) if query and any(not x.get("disabled") for x in query) else ""),
           "host": ["{{%s}}" % base], "path": segs}
    if query: url["query"] = query
    r = {"method": method, "header": [], "url": url, "description": desc}
    if body is not None:
        r["header"].append({"key": "Content-Type", "value": "application/json"})
        raw = json.dumps(body, indent=2)
        for v in NUMERIC_VARS: raw = raw.replace('"{{%s}}"' % v, "{{%s}}" % v)
        r["body"] = {"mode": "raw", "raw": raw, "options": {"raw": {"language": "json"}}}
    if auth:
        r["auth"] = {"type": "bearer", "bearer": [{"key": "token", "value": "{{accessToken}}", "type": "string"}]}
    else:
        r["auth"] = {"type": "noauth"}
    item = {"name": name, "request": r}
    ev = []
    if pre: ev.append({"listen": "prerequest", "script": {"type": "text/javascript", "exec": pre}})
    if tests: ev.append({"listen": "test", "script": {"type": "text/javascript", "exec": tests}})
    if ev: item["event"] = ev
    return item

def q(k, v, d="", disabled=False):
    x = {"key": k, "value": v, "description": d}
    if disabled: x["disabled"] = True
    return x

def ops(base):
    return {"name": "Ops", "item": [
        req("Liveness", "GET", base, "/healthz", desc="Liveness probe (go-health-check)."),
        req("Readiness", "GET", base, "/readyz", desc="Readiness probe: checks Postgres/Kafka/Redis dependencies."),
        req("Prometheus metrics", "GET", base, "/metrics"),
    ]}

identity = {"name": "Identity Service", "description": "identity-service (:8083). Auth + JWT issuing.", "item": [
    req("Register", "POST", "identityUrl", "/api/v1/auth/register",
        {"email": "{{userEmail}}", "password": "{{userPassword}}", "role": "CUSTOMER"},
        desc="Role: CUSTOMER (default) | TECHNICIAN | MANAGER | ADMIN. Password 8-72 chars.",
        tests=save("userId", "j.id")),
    req("Login", "POST", "identityUrl", "/api/v1/auth/login",
        {"email": "{{userEmail}}", "password": "{{userPassword}}"},
        desc="Stores accessToken / refreshToken in collection variables.",
        tests=save("accessToken", "j.access_token") + save("refreshToken", "j.refresh_token")),
    req("Refresh token", "POST", "identityUrl", "/api/v1/auth/refresh",
        {"refresh_token": "{{refreshToken}}"},
        desc="Rotates the token pair.",
        tests=save("accessToken", "j.access_token") + save("refreshToken", "j.refresh_token")),
    req("Me", "GET", "identityUrl", "/api/v1/me", desc="Requires Authorization: Bearer <access token>.", auth=True),
    req("Logout", "POST", "identityUrl", "/api/v1/auth/logout",
        {"refresh_token": "{{refreshToken}}"}, desc="Revokes the refresh token. 204 on success."),
    ops("identityUrl"),
]}

customer = {"name": "Customer Service", "description": "customer-service (:8080).", "item": [
    req("Create customer", "POST", "customerUrl", "/api/v1/customer",
        {"name": "Jane Doe", "email": "jane.{{$timestamp}}@example.com", "phone": "+84901234567", "birth_day": "1990-05-20T00:00:00Z"},
        tests=save("customerId", "j.id")),
    req("List customers", "GET", "customerUrl", "/api/v1/customer",
        query=[q("limit", "20", "Page size (0 = server default)"), q("cursor", "0", "next_cursor from previous page")],
        desc="Cursor-paginated."),
    req("Get customer", "GET", "customerUrl", "/api/v1/customer/{{customerId}}",
        desc="Stores updated_at for optimistic-lock on Update.",
        tests=save("customerUpdatedAt", "j.updated_at")),
    req("Update customer profile", "PUT", "customerUrl", "/api/v1/customer/{{customerId}}",
        {"name": "Jane Updated", "birth_day": "1990-05-20T00:00:00Z", "updated_at": "{{customerUpdatedAt}}"},
        desc="updated_at must match the last read value, otherwise 409."),
    req("Delete customer", "DELETE", "customerUrl", "/api/v1/customer/{{customerId}}"),
    ops("customerUrl"),
]}

vehicle = {"name": "Vehicle Service", "description": "vehicle-service (:8081).", "item": [
    req("Register vehicle", "POST", "vehicleUrl", "/api/v1/vehicle",
        {"vin": "{{randomVin}}", "license_plate": "51A-123.45", "vehicle_model_id": 1, "warranty_end_date": "2028-12-31T00:00:00Z", "status": "ACTIVE"},
        desc="status: ACTIVE (default) | SOLD | SCRAPPED. VIN/plate normalized server-side.",
        tests=save("vehicleId", "j.id"),
        pre=["const c = 'ABCDEFGHJKLMNPRSTUVWXYZ0123456789';", "let v = ''; for (let i = 0; i < 17; i++) v += c[Math.floor(Math.random() * c.length)];", "pm.collectionVariables.set('randomVin', v);"]),
    req("List vehicles", "GET", "vehicleUrl", "/api/v1/vehicle",
        query=[q("limit", "20"), q("cursor", "0"), q("vin", "", "Filter by VIN", True),
               q("plate", "", "Filter by license plate", True), q("status", "ACTIVE", "ACTIVE | SOLD | SCRAPPED", True)],
        desc="Cursor-paginated with optional filters."),
    req("Get vehicle", "GET", "vehicleUrl", "/api/v1/vehicle/{{vehicleId}}",
        tests=save("vehicleUpdatedAt", "j.updated_at")),
    req("Update vehicle", "PATCH", "vehicleUrl", "/api/v1/vehicle/{{vehicleId}}",
        {"status": "ACTIVE", "warranty_end_date": "2029-12-31T00:00:00Z", "updated_at": "{{vehicleUpdatedAt}}"},
        desc="Omitted fields unchanged. updated_at must match last read value, otherwise 409."),
    req("Assign initial owner", "POST", "vehicleUrl", "/api/v1/vehicle/{{vehicleId}}/owner",
        {"customer_id": "{{customerId}}"},
        desc="Optional \"date\" (RFC3339) — defaults to today, cannot be in the future."),
    req("Transfer vehicle", "POST", "vehicleUrl", "/api/v1/transfer",
        {"vehicle_id": "{{vehicleId}}", "from": "{{customerId}}", "to": "{{targetCustomerId}}"},
        desc="Optional \"date\" (RFC3339). Row-locked: concurrent transfers serialize."),
    req("Get customer vehicles", "GET", "vehicleUrl", "/api/v1/customers/{{customerId}}/vehicles"),
    req("Get warranty", "GET", "vehicleUrl", "/api/v1/vehicle/{{vehicleId}}/warranty",
        query=[q("date", "2026-10-10", "YYYY-MM-DD; defaults to today", True)]),
    req("Get service history", "GET", "vehicleUrl", "/api/v1/vehicle/{{vehicleId}}/history",
        query=[q("limit", "50", "Default 50, max 200")]),
    req("Get vehicle materials", "GET", "vehicleUrl", "/api/v1/vehicle/{{vehicleId}}/materials"),
    ops("vehicleUrl"),
]}

billing = {"name": "Billing Service", "description": "billing_service (Spring Boot, :8084).", "item": [
    req("Checkout (service completed)", "POST", "billingUrl", "/api/v1/bills/checkout",
        {"appointmentId": "{{appointmentId}}", "customerId": "{{customerId}}", "vehicleId": "{{vehicleId}}",
         "serviceDate": None,
         "services": [{"description": "Periodic maintenance", "quantity": 1, "unitPrice": 30.00},
                      {"description": "Tire rotation", "quantity": 1, "unitPrice": 15.00}],
         "materials": [{"description": "Oil filter", "quantity": 1, "unitPrice": 12.50},
                       {"description": "Engine oil 1L", "quantity": 4, "unitPrice": 8.00}]},
        desc="Workflow: checks with customer-service and vehicle-service that the customer is ACTIVE, the vehicle is ACTIVE "
             "and currently owned by that customer, then adds the warranty/expiration fee only if the warranty has expired "
             "or there is none (business-requirement.md §6) and issues an itemized bill. serviceDate (YYYY-MM-DD) is optional. "
             "404 unknown customer/vehicle, 409 not owned / inactive / appointment already billed, 502 upstream down.",
        tests=save("billId", "j.bill.id"),
        pre=["pm.collectionVariables.set('appointmentId', Date.now() % 1000000000);"]),
    req("Create bill (manual totals)", "POST", "billingUrl", "/api/v1/bills",
        {"appointmentId": "{{appointmentId}}", "customerId": "{{customerId}}", "vehicleId": "{{vehicleId}}", "warrantyFee": 0, "serviceFee": 500000, "materialCost": 250000},
        desc="Low-level create with no validation or line items; prefer Checkout.",
        tests=save("billId", "j.id"),
        pre=["pm.collectionVariables.set('appointmentId', Date.now() % 1000000000);"]),
    req("Get bill", "GET", "billingUrl", "/api/v1/bills/{{billId}}"),
    req("Get bill lines", "GET", "billingUrl", "/api/v1/bills/{{billId}}/lines", desc="Itemized lines (bills made by Checkout only)."),
    req("List bills by customer", "GET", "billingUrl", "/api/v1/bills", query=[q("customerId", "{{customerId}}")]),
    req("Record cash payment", "POST", "billingUrl", "/api/v1/bills/{{billId}}/payments",
        {"type": "DEPOSIT", "method": "CASH", "amount": 10.00},
        desc="type: DEPOSIT | FINAL. CASH is SUCCEEDED immediately and publishes PaymentReceived."),
    req("Start card payment", "POST", "billingUrl", "/api/v1/bills/{{billId}}/payments",
        {"type": "FINAL", "method": "CARD", "amount": 10.00},
        desc="Creates a PENDING payment and a Stripe PaymentIntent via payment-service; returns clientSecret for Stripe.js "
             "and externalPaymentId (payment-service id). 502 when payment-service/Stripe is unavailable.",
        tests=save("billingPaymentId", "j.id") + save("externalPaymentId", "j.externalPaymentId")),
    req("Cancel card payment", "POST", "billingUrl", "/api/v1/bills/{{billId}}/payments/{{billingPaymentId}}/cancel",
        desc="Abandons a PENDING card payment. 409 if Stripe already charged it."),
    req("List payments", "GET", "billingUrl", "/api/v1/bills/{{billId}}/payments"),
    req("Void bill", "POST", "billingUrl", "/api/v1/bills/{{billId}}/void",
        desc="Only a PENDING bill with no card payment in flight."),
    {"name": "Ops", "item": [
        req("Actuator health", "GET", "billingUrl", "/actuator/health"),
        req("Actuator info", "GET", "billingUrl", "/actuator/info"),
    ]},
]}

payment = {"name": "Payment Service", "description": "payment-service (Spring Boot, :8085). The only service that talks to Stripe. "
           "Create/cancel are internal (called by billing); test-confirm and sync are handy for manual testing.", "item": [
    req("Get payment", "GET", "paymentUrl", "/api/v1/payments/{{externalPaymentId}}"),
    req("Test-confirm (success card)", "POST", "paymentUrl", "/api/v1/payments/{{externalPaymentId}}/test-confirm",
        {"paymentMethod": "pm_card_visa"},
        desc="Stripe test-mode keys only (403 with a live key). Confirms with a Stripe test card and applies the result "
             "like the webhook would, so no browser or `stripe listen` is needed."),
    req("Test-confirm (declined card)", "POST", "paymentUrl", "/api/v1/payments/{{externalPaymentId}}/test-confirm",
        {"paymentMethod": "pm_card_chargeDeclined"},
        desc="The payment stays PENDING with failureReason set; the customer can retry."),
    req("Sync from Stripe", "POST", "paymentUrl", "/api/v1/payments/{{externalPaymentId}}/sync",
        desc="Re-reads the PaymentIntent from Stripe and applies its status (recovers a missed webhook)."),
    req("Cancel payment (internal)", "POST", "paymentUrl", "/api/v1/payments/{{externalPaymentId}}/cancel"),
    req("Create payment (internal)", "POST", "paymentUrl", "/api/v1/payments",
        {"billId": "{{billId}}", "billingPaymentId": 999999, "customerId": "{{customerId}}", "amount": 10.00, "currency": "usd"},
        desc="Normally only billing-service calls this. Idempotent per billingPaymentId."),
    {"name": "Ops", "item": [
        req("Actuator health", "GET", "paymentUrl", "/actuator/health"),
    ]},
]}

# ---------------------------------------------------------------------------
# Full flow: runnable end to end with the Collection Runner or
# postman/run-flow.sh (newman). Uses the seeded data from postgres/seed/:
#   customer 685 owns vehicle 1 (no warranty      -> fee charged)
#   customer 625 owns vehicle 3 (warranty to 2029 -> no fee)
#   customer 148 does NOT own vehicle 1           -> ownership check
# Card steps need Stripe test keys on payment-service; without them they are
# skipped (the runner jumps ahead) and the cash/void paths still run.
# ---------------------------------------------------------------------------

def js(*lines): return list(lines)

def expect_status(code):
    return [f"pm.test('status {code}', () => pm.response.to.have.status({code}));"]

def cents(expr): return f"Math.round(({expr}) * 100)"

def step(name, method, base, path, body=None, tests=(), pre=None, desc=""):
    return req(name, method, base, path, body, desc=desc, tests=list(tests), pre=pre)

SKIP_CARD_IF_NO_STRIPE = js(
    "if (pm.response.code === 502) {",
    "  // payment-service has no Stripe test keys (or Stripe is unreachable): billing marked the payment FAILED.",
    "  console.warn('Card checkout unavailable, skipping card steps: ' + pm.response.text());",
    "  pm.test('card checkout unavailable (no Stripe test keys) - skipped', () => true);",
    "  pm.execution.setNextRequest(pm.collectionVariables.get('skipCardTo'));",
    "  return;",
    "}",
)

flow_paid = {"name": "1. Seeded customer, no warranty: cash deposit + card balance -> PAID", "item": [
    step("1.01 Billing is up", "GET", "billingUrl", "/actuator/health",
         tests=expect_status(200) + js("pm.test('UP', () => pm.expect(pm.response.json().status).to.eql('UP'));")),
    step("1.02 Payment service is up", "GET", "paymentUrl", "/actuator/health",
         tests=expect_status(200) + js("pm.test('UP', () => pm.expect(pm.response.json().status).to.eql('UP'));")),
    step("1.03 Get seeded customer", "GET", "customerUrl", "/api/v1/customer/{{flowCustomerId}}",
         pre=js("pm.collectionVariables.set('flowCustomerId', pm.collectionVariables.get('seedCustomerNoWarranty'));",
                "pm.collectionVariables.set('flowVehicleId', pm.collectionVariables.get('seedVehicleNoWarranty'));"),
         tests=expect_status(200) + js("pm.test('customer is ACTIVE', () => pm.expect(pm.response.json().status).to.eql('ACTIVE'));")),
    step("1.04 Customer currently owns the vehicle", "GET", "vehicleUrl", "/api/v1/customers/{{flowCustomerId}}/vehicles",
         tests=expect_status(200) + js(
             "const ids = pm.response.json().vehicles.map(v => v.id);",
             "pm.test('vehicle is in the customer list', () => pm.expect(ids).to.include(Number(pm.collectionVariables.get('flowVehicleId'))));")),
    step("1.05 Warranty status", "GET", "vehicleUrl", "/api/v1/vehicle/{{flowVehicleId}}/warranty",
         tests=expect_status(200) + js("pm.collectionVariables.set('expectFee', !pm.response.json().in_warranty);")),
    step("1.06 Checkout", "POST", "billingUrl", "/api/v1/bills/checkout",
         {"appointmentId": "{{appointmentId}}", "customerId": "{{flowCustomerId}}", "vehicleId": "{{flowVehicleId}}",
          "services": [{"description": "Periodic maintenance (10,000 km)", "quantity": 1, "unitPrice": 30.00},
                       {"description": "Brake inspection", "quantity": 1, "unitPrice": 20.00}],
          "materials": [{"description": "Oil filter", "quantity": 1, "unitPrice": 12.50},
                        {"description": "Engine oil 1L", "quantity": 4, "unitPrice": 8.00}]},
         pre=js("pm.collectionVariables.set('appointmentId', Date.now() % 1000000000);",
                "pm.collectionVariables.set('externalPaymentId', '');",
                "pm.collectionVariables.set('skipCardTo', '1.15 Payments are listed');"),
         tests=expect_status(201) + js(
             "const j = pm.response.json();",
             "const expectFee = pm.collectionVariables.get('expectFee') === true || pm.collectionVariables.get('expectFee') === 'true';",
             "pm.test('warranty fee follows the warranty', () => pm.expect(j.warranty.feeCharged).to.eql(expectFee));",
             "pm.test('lines add up to the total', () => {",
             f"  const sum = j.lines.reduce((s, l) => s + {cents('l.amount')}, 0);",
             f"  pm.expect(sum).to.eql({cents('j.bill.totalAmount')});",
             "});",
             "pm.test('service fee = 50.00, materials = 44.50', () => {",
             f"  pm.expect({cents('j.bill.serviceFee')}).to.eql(5000);",
             f"  pm.expect({cents('j.bill.materialCost')}).to.eql(4450);",
             "});",
             "pm.test('bill is PENDING', () => pm.expect(j.bill.status).to.eql('PENDING'));",
             "pm.collectionVariables.set('billId', j.bill.id);",
             f"pm.collectionVariables.set('billTotalCents', {cents('j.bill.totalAmount')});")),
    step("1.07 Same appointment cannot be billed twice", "POST", "billingUrl", "/api/v1/bills/checkout",
         {"appointmentId": "{{appointmentId}}", "customerId": "{{flowCustomerId}}", "vehicleId": "{{flowVehicleId}}",
          "services": [{"description": "Duplicate", "unitPrice": 1.00}]},
         tests=expect_status(409)),
    step("1.08 Bill lines", "GET", "billingUrl", "/api/v1/bills/{{billId}}/lines",
         tests=expect_status(200) + js(
             "const kinds = pm.response.json().map(l => l.kind);",
             "pm.test('2 services and 2 materials', () => {",
             "  pm.expect(kinds.filter(k => k === 'SERVICE').length).to.eql(2);",
             "  pm.expect(kinds.filter(k => k === 'MATERIAL').length).to.eql(2);",
             "});")),
    step("1.09 Cash deposit", "POST", "billingUrl", "/api/v1/bills/{{billId}}/payments",
         {"type": "DEPOSIT", "method": "CASH", "amount": "{{depositAmount}}"},
         pre=js("pm.collectionVariables.set('depositAmount', 20.00);"),
         tests=expect_status(201) + js("pm.test('SUCCEEDED at once', () => pm.expect(pm.response.json().status).to.eql('SUCCEEDED'));")),
    step("1.10 Bill is PARTIALLY_PAID", "GET", "billingUrl", "/api/v1/bills/{{billId}}",
         tests=expect_status(200) + js("pm.test('PARTIALLY_PAID', () => pm.expect(pm.response.json().status).to.eql('PARTIALLY_PAID'));")),
    step("1.11 Card checkout for the balance", "POST", "billingUrl", "/api/v1/bills/{{billId}}/payments",
         {"type": "FINAL", "method": "CARD", "amount": "{{cardAmount}}"},
         pre=js("const balance = Number(pm.collectionVariables.get('billTotalCents')) - 2000;",
                "pm.collectionVariables.set('cardAmount', (balance / 100).toFixed(2));"),
         tests=SKIP_CARD_IF_NO_STRIPE + expect_status(201) + js(
             "const j = pm.response.json();",
             "pm.test('PENDING with a client secret', () => { pm.expect(j.status).to.eql('PENDING'); pm.expect(j.clientSecret).to.be.a('string'); });",
             "pm.collectionVariables.set('billingPaymentId', j.id);",
             "pm.collectionVariables.set('externalPaymentId', j.externalPaymentId);")),
    step("1.12 Nothing left to charge while the card is pending", "POST", "billingUrl", "/api/v1/bills/{{billId}}/payments",
         {"type": "FINAL", "method": "CASH", "amount": 1.00},
         tests=expect_status(409)),
    step("1.13 Declined card attempt", "POST", "paymentUrl", "/api/v1/payments/{{externalPaymentId}}/test-confirm",
         {"paymentMethod": "pm_card_chargeDeclined"},
         tests=expect_status(200) + js(
             "const j = pm.response.json();",
             "pm.test('still PENDING, reason recorded', () => { pm.expect(j.status).to.eql('PENDING'); pm.expect(j.failureReason).to.be.a('string'); });")),
    step("1.14 Successful card attempt", "POST", "paymentUrl", "/api/v1/payments/{{externalPaymentId}}/test-confirm",
         {"paymentMethod": "pm_card_visa"},
         tests=expect_status(200) + js("pm.test('SUCCEEDED', () => pm.expect(pm.response.json().status).to.eql('SUCCEEDED'));",
                                       "pm.collectionVariables.set('pollAttempts', 0);")),
    step("1.15 Payments are listed", "GET", "billingUrl", "/api/v1/bills/{{billId}}/payments",
         tests=expect_status(200) + js("pm.test('deposit recorded', () => pm.expect(pm.response.json()[0].status).to.eql('SUCCEEDED'));")),
    step("1.16 Wait for bill PAID (Kafka round trip)", "GET", "billingUrl", "/api/v1/bills/{{billId}}",
         desc="billing settles the card payment from payment-service's Kafka event, so this polls for up to ~15 s. Skipped when card steps were skipped.",
         pre=js("if (!pm.collectionVariables.get('externalPaymentId')) { pm.execution.skipRequest(); }"),
         tests=expect_status(200) + js(
             "const attempts = Number(pm.collectionVariables.get('pollAttempts') || 0);",
             "if (pm.response.json().status !== 'PAID' && attempts < 15) {",
             "  pm.collectionVariables.set('pollAttempts', attempts + 1);",
             "  setTimeout(() => {}, 1000);",
             "  pm.execution.setNextRequest(pm.info.requestName);",
             "  return;",
             "}",
             "pm.test('bill is PAID', () => pm.expect(pm.response.json().status).to.eql('PAID'));")),
    step("1.17 A charged card cannot be canceled", "POST", "billingUrl", "/api/v1/bills/{{billId}}/payments/{{billingPaymentId}}/cancel",
         pre=js("if (!pm.collectionVariables.get('externalPaymentId')) { pm.execution.skipRequest(); }"),
         tests=expect_status(409)),
    step("1.18 A paid bill cannot be voided", "POST", "billingUrl", "/api/v1/bills/{{billId}}/void",
         pre=js("if (!pm.collectionVariables.get('externalPaymentId')) { pm.execution.skipRequest(); }"),
         tests=expect_status(409)),
]}

flow_void = {"name": "2. Seeded customer in warranty: card abandoned -> bill voided", "item": [
    step("2.01 Checkout (warranty covers the fee)", "POST", "billingUrl", "/api/v1/bills/checkout",
         {"appointmentId": "{{appointmentId}}", "customerId": "{{flowCustomerId}}", "vehicleId": "{{flowVehicleId}}",
          "serviceDate": "2026-10-15",
          "services": [{"description": "Warranty inspection", "quantity": 1, "unitPrice": 25.00}]},
         pre=js("pm.collectionVariables.set('appointmentId', Date.now() % 1000000000);",
                "pm.collectionVariables.set('flowCustomerId', pm.collectionVariables.get('seedCustomerInWarranty'));",
                "pm.collectionVariables.set('flowVehicleId', pm.collectionVariables.get('seedVehicleInWarranty'));",
                "pm.collectionVariables.set('externalPaymentId', '');",
                "pm.collectionVariables.set('skipCardTo', '2.04 Void the bill');"),
         tests=expect_status(201) + js(
             "const j = pm.response.json();",
             "pm.test('in warranty: no fee', () => { pm.expect(j.warranty.inWarranty).to.eql(true); pm.expect(j.warranty.feeCharged).to.eql(false); });",
             "pm.test('no WARRANTY_FEE line', () => pm.expect(j.lines.map(l => l.kind)).to.not.include('WARRANTY_FEE'));",
             "pm.collectionVariables.set('billId', j.bill.id);")),
    step("2.02 Start card checkout", "POST", "billingUrl", "/api/v1/bills/{{billId}}/payments",
         {"type": "FINAL", "method": "CARD", "amount": 25.00},
         tests=SKIP_CARD_IF_NO_STRIPE + expect_status(201) + js(
             "pm.collectionVariables.set('billingPaymentId', pm.response.json().id);",
             "pm.collectionVariables.set('externalPaymentId', pm.response.json().externalPaymentId);")),
    step("2.03 Customer walks away: cancel the card payment", "POST", "billingUrl", "/api/v1/bills/{{billId}}/payments/{{billingPaymentId}}/cancel",
         tests=expect_status(200) + js("pm.test('FAILED (canceled)', () => pm.expect(pm.response.json().status).to.eql('FAILED'));")),
    step("2.04 Void the bill", "POST", "billingUrl", "/api/v1/bills/{{billId}}/void",
         tests=expect_status(200) + js("pm.test('VOID', () => pm.expect(pm.response.json().status).to.eql('VOID'));")),
    step("2.05 A void bill takes no payments", "POST", "billingUrl", "/api/v1/bills/{{billId}}/payments",
         {"type": "FINAL", "method": "CASH", "amount": 25.00},
         tests=expect_status(409)),
]}

flow_walk_in = {"name": "3. New walk-in customer: register, assign owner, checkout, pay cash", "item": [
    step("3.01 Create customer", "POST", "customerUrl", "/api/v1/customer",
         {"name": "Walk-in Customer", "email": "walkin.{{$timestamp}}@example.com", "phone": "+84901234567", "birth_day": "1990-05-20T00:00:00Z"},
         tests=expect_status(201) + js("pm.collectionVariables.set('flowCustomerId', pm.response.json().id);")),
    step("3.02 Register vehicle (warranty already expired)", "POST", "vehicleUrl", "/api/v1/vehicle",
         {"vin": "{{randomVin}}", "vehicle_model_id": 1, "warranty_end_date": "2025-01-01T00:00:00Z", "status": "ACTIVE"},
         pre=js("const c = 'ABCDEFGHJKLMNPRSTUVWXYZ0123456789';",
                "let v = ''; for (let i = 0; i < 17; i++) v += c[Math.floor(Math.random() * c.length)];",
                "pm.collectionVariables.set('randomVin', v);"),
         tests=expect_status(201) + js("pm.collectionVariables.set('flowVehicleId', pm.response.json().id);")),
    step("3.03 Checkout before ownership is rejected", "POST", "billingUrl", "/api/v1/bills/checkout",
         {"appointmentId": "{{appointmentId}}", "customerId": "{{flowCustomerId}}", "vehicleId": "{{flowVehicleId}}",
          "services": [{"description": "Oil change", "unitPrice": 30.00}]},
         pre=js("pm.collectionVariables.set('appointmentId', Date.now() % 1000000000);"),
         tests=expect_status(409)),
    step("3.04 Assign the vehicle to the customer", "POST", "vehicleUrl", "/api/v1/vehicle/{{flowVehicleId}}/owner",
         {"customer_id": "{{flowCustomerId}}"},
         tests=js("pm.test('assigned', () => pm.expect(pm.response.code).to.be.oneOf([200, 201, 204]));")),
    step("3.05 Checkout (expired warranty: fee charged)", "POST", "billingUrl", "/api/v1/bills/checkout",
         {"appointmentId": "{{appointmentId}}", "customerId": "{{flowCustomerId}}", "vehicleId": "{{flowVehicleId}}",
          "services": [{"description": "Oil change", "unitPrice": 30.00}],
          "materials": [{"description": "Engine oil 1L", "quantity": 4, "unitPrice": 8.00}]},
         tests=expect_status(201) + js(
             "const j = pm.response.json();",
             "pm.test('fee charged, warranty ended 2025-01-01', () => {",
             "  pm.expect(j.warranty.feeCharged).to.eql(true);",
             "  pm.expect(j.lines[0].description).to.include('warranty ended 2025-01-01');",
             "});",
             "pm.collectionVariables.set('billId', j.bill.id);",
             "pm.collectionVariables.set('cashAmount', j.bill.totalAmount);")),
    step("3.06 Pay everything in cash", "POST", "billingUrl", "/api/v1/bills/{{billId}}/payments",
         {"type": "FINAL", "method": "CASH", "amount": "{{cashAmount}}"},
         tests=expect_status(201)),
    step("3.07 Bill is PAID", "GET", "billingUrl", "/api/v1/bills/{{billId}}",
         tests=expect_status(200) + js("pm.test('PAID', () => pm.expect(pm.response.json().status).to.eql('PAID'));")),
    step("3.08 Customer's bills", "GET", "billingUrl", "/api/v1/bills?customerId={{flowCustomerId}}",
         tests=expect_status(200) + js("pm.test('one bill', () => pm.expect(pm.response.json().length).to.eql(1));")),
]}

flow_rejects = {"name": "4. Checkout validation", "item": [
    step("4.01 Vehicle owned by someone else", "POST", "billingUrl", "/api/v1/bills/checkout",
         {"appointmentId": "{{appointmentId}}", "customerId": "{{seedCustomerNotOwner}}", "vehicleId": "{{seedVehicleNoWarranty}}",
          "services": [{"description": "Oil change", "unitPrice": 30.00}]},
         pre=js("pm.collectionVariables.set('appointmentId', Date.now() % 1000000000);"),
         tests=expect_status(409) + js("pm.test('says not owned', () => pm.expect(pm.response.json().error).to.include('not currently owned'));")),
    step("4.02 Unknown customer", "POST", "billingUrl", "/api/v1/bills/checkout",
         {"appointmentId": "{{appointmentId}}", "customerId": 999999999, "vehicleId": "{{seedVehicleNoWarranty}}",
          "services": [{"description": "Oil change", "unitPrice": 30.00}]},
         tests=expect_status(404)),
    step("4.03 No services", "POST", "billingUrl", "/api/v1/bills/checkout",
         {"appointmentId": "{{appointmentId}}", "customerId": "{{seedCustomerNoWarranty}}", "vehicleId": "{{seedVehicleNoWarranty}}",
          "services": []},
         tests=expect_status(400)),
    step("4.04 Price with 3 decimals", "POST", "billingUrl", "/api/v1/bills/checkout",
         {"appointmentId": "{{appointmentId}}", "customerId": "{{seedCustomerNoWarranty}}", "vehicleId": "{{seedVehicleNoWarranty}}",
          "services": [{"description": "Oil change", "unitPrice": 30.005}]},
         tests=expect_status(400)),
]}

full_flow = {"name": "Full flow", "description": "End-to-end business flows with assertions. Run the whole folder in order with the "
             "Collection Runner or postman/run-flow.sh. Uses seeded customers/vehicles (postgres/seed). Card steps need Stripe "
             "test keys on payment-service and are skipped otherwise.",
             "item": [flow_paid, flow_void, flow_walk_in, flow_rejects]}

def var(k, v): return {"key": k, "value": v, "type": "string"}
coll = {
    "info": {"name": "Service Bay", "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
             "description": "All HTTP APIs for Service Bay. 'Full flow' runs the business flows end to end with assertions (see postman/README.md). In the per-service folders, create/register requests store IDs (customerId, vehicleId, billId) into collection variables for later requests."},
    "item": [full_flow, identity, customer, vehicle, billing, payment],
    "variable": [var("identityUrl", "http://localhost:8083"), var("customerUrl", "http://localhost:8080"),
                 var("vehicleUrl", "http://localhost:8081"), var("billingUrl", "http://localhost:8084"),
                 var("paymentUrl", "http://localhost:8085"),
                 var("userEmail", "dev@example.com"), var("userPassword", "Passw0rd!"),
                 var("accessToken", ""), var("refreshToken", ""), var("userId", ""),
                 var("customerId", "1"), var("targetCustomerId", "2"), var("vehicleId", "1"), var("billId", "1"),
                 var("customerUpdatedAt", ""), var("randomVin", ""), var("vehicleUpdatedAt", ""),
                 var("appointmentId", ""), var("billingPaymentId", ""), var("externalPaymentId", ""),
                 # Seeded pairs from postgres/seed/ used by the Full flow folder.
                 var("seedCustomerNoWarranty", "685"), var("seedVehicleNoWarranty", "1"),
                 var("seedCustomerInWarranty", "625"), var("seedVehicleInWarranty", "3"),
                 var("seedCustomerNotOwner", "148"),
                 var("flowCustomerId", ""), var("flowVehicleId", ""), var("expectFee", ""),
                 var("billTotalCents", ""), var("depositAmount", ""), var("cardAmount", ""), var("cashAmount", ""),
                 var("skipCardTo", ""), var("pollAttempts", "0")],
}
with open(out, "w") as f:
    json.dump(coll, f, indent=2)
    f.write("\n")
