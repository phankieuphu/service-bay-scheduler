"""Generates service-bay.postman_collection.json.

Routes and bodies are hand-mirrored from each service's HTTP handlers/DTOs —
update this file when an endpoint changes, then run:

    python3 postman/gen_collection.py [output-path]
"""
import json, os, sys
out = sys.argv[1] if len(sys.argv) > 1 else os.path.join(os.path.dirname(os.path.abspath(__file__)), "service-bay.postman_collection.json")

# Variables substituted into JSON bodies as numbers, not strings.
NUMERIC_VARS = ("customerId", "targetCustomerId", "vehicleId", "billId", "hubId", "flowCustomerId", "flowVehicleId")

def save(var, expr):
    return [f"if (pm.response.code < 300) {{ const j = pm.response.json(); if ({expr} !== undefined) pm.collectionVariables.set('{var}', {expr}); }}"]

def req(name, method, base, path, body=None, query=None, desc="", tests=None, auth=False, pre=None, headers=None):
    segs = [s for s in path.strip("/").split("/")]
    url = {"raw": "{{%s}}/%s" % (base, "/".join(segs)) + ("?" + "&".join(f"{q['key']}={q['value']}" for q in query if not q.get("disabled")) if query and any(not x.get("disabled") for x in query) else ""),
           "host": ["{{%s}}" % base], "path": segs}
    if query: url["query"] = query
    r = {"method": method, "header": [{"key": k, "value": v} for k, v in (headers or {}).items()], "url": url, "description": desc}
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
        desc="Only CUSTOMER (the default) can self-register; staff are created via Admin > Create user. Password 8-72 chars. 429 when rate-limited.",
        tests=save("userId", "j.id")),
    req("Login", "POST", "identityUrl", "/api/v1/auth/login",
        {"email": "{{userEmail}}", "password": "{{userPassword}}"},
        desc="Stores accessToken in a collection variable. The refresh token comes back as an HttpOnly refresh_token cookie, kept in Postman's cookie jar (needs AUTH_COOKIE_SECURE=false over plain HTTP, as docker-compose sets).",
        tests=save("accessToken", "j.access_token")),
    req("Refresh token", "POST", "identityUrl", "/api/v1/auth/refresh",
        desc="Sends the refresh_token cookie; X-Requested-With is required. Rotates the cookie and returns a new access token.",
        headers={"X-Requested-With": "postman"},
        tests=save("accessToken", "j.access_token")),
    req("Me", "GET", "identityUrl", "/api/v1/me", desc="Requires Authorization: Bearer <access token>.", auth=True),
    req("Logout", "POST", "identityUrl", "/api/v1/auth/logout",
        desc="Revokes the refresh_token cookie's token and clears the cookie. X-Requested-With is required. 204 on success.",
        headers={"X-Requested-With": "postman"}),
    {"name": "Admin", "description": "Needs an ADMIN access token: log in as the bootstrap admin (ADMIN_EMAIL / ADMIN_PASSWORD) first.", "item": [
        req("Create user", "POST", "identityUrl", "/api/v1/admin/users",
            {"email": "staff.{{$timestamp}}@example.com", "password": "{{userPassword}}", "role": "TECHNICIAN"},
            desc="Any role: CUSTOMER | TECHNICIAN | MANAGER | ADMIN. Emits UserCreated.", auth=True,
            tests=save("staffUserId", "j.id")),
        req("Set user status", "PATCH", "identityUrl", "/api/v1/admin/users/{{staffUserId}}",
            {"status": "LOCKED"},
            desc="ACTIVE | INACTIVE | LOCKED. Emits UserUpdated. Not allowed on your own account.", auth=True),
    ]},
    ops("identityUrl"),
]}

def with_auth(group):
    """Bearer-auth every request in group except the Ops folder (probes and
    /metrics need no token)."""
    for item in group["item"]:
        if "request" in item:
            item["request"]["auth"] = {"type": "bearer", "bearer": [{"key": "token", "value": "{{accessToken}}", "type": "string"}]}
    return group

customer = {"name": "Customer Service", "description": "customer-service (:8080). Needs a staff access token (log in as the bootstrap admin); DELETE needs MANAGER or ADMIN.", "item": [
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

vehicle = {"name": "Vehicle Service", "description": "vehicle-service (:8081). Needs an access token; writes need a staff role.", "item": [
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

def bearer(item, token_var):
    """Sends item with the bearer token held in token_var."""
    item["request"]["auth"] = {"type": "bearer", "bearer": [{"key": "token", "value": "{{%s}}" % token_var, "type": "string"}]}
    return item

scheduler = {"name": "Scheduler Service", "description": "scheduler-service (:8085). Needs an access token; booking is for CUSTOMER accounts, a hub's day and completion for staff.", "item": [
    req("List hubs", "GET", "schedulerUrl", "/api/v1/hubs", tests=save("hubId", "j.hubs[0].id")),
    req("List services", "GET", "schedulerUrl", "/api/v1/services"),
    req("Availability", "GET", "schedulerUrl", "/api/v1/hubs/{{hubId}}/availability",
        query=[q("date", "2026-10-20", "YYYY-MM-DD in the hub's time zone"), q("service_id", "1"), q("service_id", "3")]),
    req("Book (CUSTOMER)", "POST", "schedulerUrl", "/api/v1/appointments",
        {"hub_id": "{{hubId}}", "vehicle_id": "{{vehicleId}}", "service_ids": [1, 3], "start": "2026-10-20T10:00:00+07:00"},
        tests=save("appointmentId", "j.id")),
    req("My appointments (CUSTOMER)", "GET", "schedulerUrl", "/api/v1/me/appointments"),
    req("Cancel (CUSTOMER)", "POST", "schedulerUrl", "/api/v1/appointments/{{appointmentId}}/cancel"),
    req("Hub day (staff)", "GET", "schedulerUrl", "/api/v1/hubs/{{hubId}}/appointments", query=[q("date", "2026-10-20")]),
    req("Complete (staff)", "POST", "schedulerUrl", "/api/v1/appointments/{{appointmentId}}/complete",
        desc="Publishes ServiceCompleted; billing-service prices the bill from it."),
    ops("schedulerUrl"),
]}

billing = {"name": "Billing Service", "description": "billing-service (:8084). Needs an access token. A customer sees and pays only their own bills (others answer 404); staff read every bill. Money is integer minor units plus currency.", "item": [
    req("Pay deposit (CUSTOMER)", "POST", "billingUrl", "/api/v1/bookings/{{appointmentId}}/deposit",
        headers={"Idempotency-Key": "{{$guid}}"},
        desc="Amount = the deposit policy applied to the booking's estimate. Returns client_secret for the Payment Element. Retrying with the same Idempotency-Key returns the same payment.",
        tests=save("paymentId", "j.payment_id")),
    req("Get booking's bill", "GET", "billingUrl", "/api/v1/bookings/{{appointmentId}}/bill", tests=save("billId", "j.id")),
    req("Get bill", "GET", "billingUrl", "/api/v1/bills/{{billId}}"),
    req("My bills (CUSTOMER)", "GET", "billingUrl", "/api/v1/me/bills", query=[q("limit", "20"), q("cursor", "", "next_cursor from the previous page", True)]),
    req("Customer's bills", "GET", "billingUrl", "/api/v1/customers/{{customerId}}/bills", query=[q("limit", "20")]),
    req("Pay balance (CUSTOMER)", "POST", "billingUrl", "/api/v1/bills/{{billId}}/payments",
        headers={"Idempotency-Key": "{{$guid}}"},
        desc="Charges the bill's current balance (the server computes it).", tests=save("paymentId", "j.payment_id")),
    req("Get payment", "GET", "billingUrl", "/api/v1/payments/{{paymentId}}"),
    req("Test-confirm (test keys only)", "POST", "billingUrl", "/api/v1/payments/{{paymentId}}/test-confirm",
        {"payment_method": "pm_card_visa"}, desc="pm_card_visa succeeds, pm_card_chargeDeclined is declined. 403 with a live key."),
    req("Sync from Stripe (staff)", "POST", "billingUrl", "/api/v1/payments/{{paymentId}}/sync"),
    {"name": "Ops", "item": [
        req("Liveness", "GET", "billingUrl", "/actuator/health/liveness"),
        req("Readiness", "GET", "billingUrl", "/actuator/health/readiness"),
    ]},
]}

# ---------------------------------------------------------------------------
# Full flow: one booking from sign-up to a PAID bill, with assertions. Run the
# folder in order (Collection Runner, or postman/run-flow.sh with newman).
# It logs in as the bootstrap admin for staff steps and registers a fresh
# customer for the rest. Card steps need Stripe test keys on billing-service;
# without them (503) they're reported as skipped.
# ---------------------------------------------------------------------------

def js(*lines): return list(lines)

def status(code): return [f"pm.test('status {code}', () => pm.response.to.have.status({code}));"]

def step(name, method, base, path, token, body=None, tests=(), pre=None, desc="", headers=None, query=None):
    return bearer(req(name, method, base, path, body, query=query, desc=desc, tests=list(tests), pre=pre, headers=headers), token) \
        if token else req(name, method, base, path, body, query=query, desc=desc, tests=list(tests), pre=pre, headers=headers)

# Polls the current request until ok() (a JS expression over j) holds, up to
# 60 tries a second apart: for steps that wait on a Kafka round trip, which
# can include a consumer-group rebalance right after a service restarts.
def poll(ok, then=()):
    return js(
        "const attempts = Number(pm.collectionVariables.get('pollAttempts') || 0);",
        "const j = pm.response.code === 200 ? pm.response.json() : null;",
        f"if (!(j && ({ok})) && attempts < 60) {{",
        "  pm.collectionVariables.set('pollAttempts', attempts + 1);",
        "  setTimeout(() => {}, 1000);",
        "  pm.execution.setNextRequest(pm.info.requestName);",
        "  return;",
        "}",
        "pm.collectionVariables.set('pollAttempts', 0);",
        f"pm.test('settled', () => pm.expect(j && ({ok})).to.be.true);",
        *then)

SKIP_IF_NO_STRIPE = js(
    "if (pm.response.code === 503) {",
    "  console.warn('Stripe is not configured on billing-service: ' + pm.response.text());",
    "  pm.test('Stripe not configured - card steps skipped', () => true);",
    "  pm.collectionVariables.set('stripe', '');",
    "  pm.execution.setNextRequest(pm.collectionVariables.get('skipTo'));",
    "  return;",
    "}",
    "pm.collectionVariables.set('stripe', '1');")

SKIP_UNLESS_STRIPE = js("if (!pm.collectionVariables.get('stripe')) { pm.execution.skipRequest(); }")

full_flow = {"name": "Full flow", "description": "Sign-up → booking → deposit → completion → priced bill → balance paid. Run in order.", "item": [
    {"name": "1. Accounts", "item": [
        step("1.01 Staff login (bootstrap admin)", "POST", "identityUrl", "/api/v1/auth/login", None,
             {"email": "{{adminEmail}}", "password": "{{adminPassword}}"},
             tests=status(200) + js("pm.collectionVariables.set('staffToken', pm.response.json().access_token);")),
        step("1.02 Register a customer", "POST", "identityUrl", "/api/v1/auth/register", None,
             {"email": "{{flowEmail}}", "password": "{{userPassword}}", "role": "CUSTOMER"},
             pre=js("pm.collectionVariables.set('flowEmail', `flow.${Date.now()}@example.com`);"),
             tests=status(201)),
        step("1.03 Customer login", "POST", "identityUrl", "/api/v1/auth/login", None,
             {"email": "{{flowEmail}}", "password": "{{userPassword}}"},
             tests=status(200) + js("pm.collectionVariables.set('customerToken', pm.response.json().access_token);")),
        step("1.04 Wait for the customer profile", "GET", "customerUrl", "/api/v1/customer/me", "customerToken",
             desc="customer-service creates it from identity's UserCreated event, a moment after sign-up.",
             tests=poll("j.id > 0", js("pm.collectionVariables.set('flowCustomerId', j.id);"))),
        step("1.05 Register a vehicle (warranty expired)", "POST", "vehicleUrl", "/api/v1/vehicle", "staffToken",
             {"vin": "{{randomVin}}", "vehicle_model_id": 1, "warranty_end_date": "2025-01-01T00:00:00Z", "status": "ACTIVE"},
             pre=js("const c = 'ABCDEFGHJKLMNPRSTUVWXYZ0123456789';",
                    "let v = ''; for (let i = 0; i < 17; i++) v += c[Math.floor(Math.random() * c.length)];",
                    "pm.collectionVariables.set('randomVin', v);"),
             tests=status(201) + js("pm.collectionVariables.set('flowVehicleId', pm.response.json().id);")),
        step("1.06 Assign it to the customer", "POST", "vehicleUrl", "/api/v1/vehicle/{{flowVehicleId}}/owner", "staffToken",
             {"customer_id": "{{flowCustomerId}}"}, tests=status(204)),
    ]},
    {"name": "2. Booking and deposit", "item": [
        step("2.01 Hubs", "GET", "schedulerUrl", "/api/v1/hubs", "customerToken",
             tests=status(200) + js("const hub = pm.response.json().hubs[0];",
                                    "pm.collectionVariables.set('hubId', hub.id);",
                                    "pm.collectionVariables.set('hubTz', hub.timezone);")),
        step("2.02 Free slots tomorrow", "GET", "schedulerUrl", "/api/v1/hubs/{{hubId}}/availability", "customerToken",
             query=[q("date", "{{bookDate}}"), q("service_id", "1"), q("service_id", "3")],
             pre=js("const tz = pm.collectionVariables.get('hubTz') || 'Asia/Ho_Chi_Minh';",
                    "pm.collectionVariables.set('bookDate', new Date(Date.now() + 86400000).toLocaleDateString('en-CA', { timeZone: tz }));"),
             tests=status(200) + js("const slot = pm.response.json().slots.find(s => s.available);",
                                    "pm.test('a free slot', () => pm.expect(slot).to.exist);",
                                    "pm.collectionVariables.set('slotStart', slot.start);")),
        step("2.03 Book oil change + brake inspection", "POST", "schedulerUrl", "/api/v1/appointments", "customerToken",
             {"hub_id": "{{hubId}}", "vehicle_id": "{{flowVehicleId}}", "service_ids": [1, 3], "start": "{{slotStart}}"},
             tests=status(201) + js("const j = pm.response.json();",
                                    "pm.test('estimate 105.50', () => pm.expect(j.total_amount).to.eql('105.50'));",
                                    "pm.collectionVariables.set('appointmentId', j.id);")),
        step("2.04 Pay the deposit", "POST", "billingUrl", "/api/v1/bookings/{{appointmentId}}/deposit", "customerToken",
             headers={"Idempotency-Key": "{{depositKey}}"},
             pre=js("pm.collectionVariables.set('depositKey', pm.variables.replaceIn('{{$guid}}'));",
                    "pm.collectionVariables.set('skipTo', '3.01 Staff completes the appointment');"),
             tests=SKIP_IF_NO_STRIPE + status(201) + js(
                 "const j = pm.response.json();",
                 "pm.test('20% of the estimate, with a client secret', () => {",
                 "  pm.expect(j.type).to.eql('DEPOSIT'); pm.expect(j.amount_minor).to.eql(2110); pm.expect(j.client_secret).to.be.a('string');",
                 "});",
                 "pm.collectionVariables.set('depositPaymentId', j.payment_id);")),
        step("2.05 Retrying with the same key returns the same payment", "POST", "billingUrl",
             "/api/v1/bookings/{{appointmentId}}/deposit", "customerToken", headers={"Idempotency-Key": "{{depositKey}}"},
             pre=SKIP_UNLESS_STRIPE,
             tests=status(201) + js("pm.test('same payment', () => pm.expect(pm.response.json().payment_id).to.eql(Number(pm.collectionVariables.get('depositPaymentId'))));")),
        step("2.06 A declined card", "POST", "billingUrl", "/api/v1/payments/{{depositPaymentId}}/test-confirm", "customerToken",
             {"payment_method": "pm_card_chargeDeclined"}, pre=SKIP_UNLESS_STRIPE,
             tests=status(200) + js("const j = pm.response.json();",
                                    "pm.test('still payable, with the reason', () => { pm.expect(j.status).to.eql('REQUIRES_PAYMENT'); pm.expect(j.failure_message).to.be.a('string'); });")),
        step("2.07 A good card", "POST", "billingUrl", "/api/v1/payments/{{depositPaymentId}}/test-confirm", "customerToken",
             {"payment_method": "pm_card_visa"}, pre=SKIP_UNLESS_STRIPE,
             tests=status(200) + js("pm.test('SUCCEEDED', () => pm.expect(pm.response.json().status).to.eql('SUCCEEDED'));")),
        step("2.08 The bill is open, with the deposit on it", "GET", "billingUrl", "/api/v1/bookings/{{appointmentId}}/bill", "customerToken",
             pre=SKIP_UNLESS_STRIPE,
             tests=status(200) + js("const j = pm.response.json();",
                                    "pm.test('OPEN, not priced, 21.10 paid', () => {",
                                    "  pm.expect(j.status).to.eql('OPEN'); pm.expect(j.balance_due_minor).to.be.null; pm.expect(j.amount_paid_minor).to.eql(2110);",
                                    "});")),
    ]},
    {"name": "3. Completion and pricing", "item": [
        step("3.01 Staff completes the appointment", "POST", "schedulerUrl", "/api/v1/appointments/{{appointmentId}}/complete", "staffToken",
             tests=status(200) + js("pm.test('COMPLETED', () => pm.expect(pm.response.json().status).to.eql('COMPLETED'));")),
        step("3.02 A customer can't complete", "POST", "schedulerUrl", "/api/v1/appointments/{{appointmentId}}/complete", "customerToken",
             tests=status(403)),
        step("3.03 Wait for the priced bill", "GET", "billingUrl", "/api/v1/bookings/{{appointmentId}}/bill", "customerToken",
             desc="billing prices it from ServiceCompleted: scheduler's outbox relay, then billing's consumer.",
             tests=poll("j.status !== 'OPEN'", js(
                 "pm.test('warranty fee (expired) + both services', () => {",
                 "  pm.expect(j.lines.map(l => l.kind)).to.eql(['WARRANTY_FEE', 'SERVICE', 'SERVICE']);",
                 "  pm.expect(j.total_minor).to.eql(15550);",
                 "});",
                 "const paid = pm.collectionVariables.get('stripe') ? 2110 : 0;",
                 "pm.test('the deposit counts toward it', () => pm.expect(j.balance_due_minor).to.eql(15550 - paid));",
                 "pm.collectionVariables.set('billId', j.id);"))),
        step("3.04 Staff can read it", "GET", "billingUrl", "/api/v1/bills/{{billId}}", "staffToken", tests=status(200)),
        step("3.05 Without a token: 401", "GET", "billingUrl", "/api/v1/bills/{{billId}}", None, tests=status(401)),
        step("3.06 It's in the vehicle's service history", "GET", "vehicleUrl", "/api/v1/vehicle/{{flowVehicleId}}/history", "staffToken",
             tests=poll("j.history.some(e => e.appointment_id === Number(pm.collectionVariables.get('appointmentId')))")),
    ]},
    {"name": "4. Paying the balance", "item": [
        step("4.01 Start paying the balance", "POST", "billingUrl", "/api/v1/bills/{{billId}}/payments", "customerToken",
             headers={"Idempotency-Key": "{{$guid}}"},
             pre=js("pm.collectionVariables.set('skipTo', '4.05 My bills');"),
             tests=SKIP_IF_NO_STRIPE + status(201) + js(
                 "const j = pm.response.json();",
                 "const paid = 2110;",
                 "pm.test('FINAL for the balance', () => { pm.expect(j.type).to.eql('FINAL'); pm.expect(j.amount_minor).to.eql(15550 - paid); });",
                 "pm.collectionVariables.set('finalPaymentId', j.payment_id);")),
        step("4.02 Pay it", "POST", "billingUrl", "/api/v1/payments/{{finalPaymentId}}/test-confirm", "customerToken",
             {"payment_method": "pm_card_visa"}, pre=SKIP_UNLESS_STRIPE,
             tests=status(200) + js("pm.test('SUCCEEDED', () => pm.expect(pm.response.json().status).to.eql('SUCCEEDED'));")),
        step("4.03 The bill is PAID", "GET", "billingUrl", "/api/v1/bills/{{billId}}", "customerToken", pre=SKIP_UNLESS_STRIPE,
             tests=status(200) + js("const j = pm.response.json();",
                                    "pm.test('PAID, nothing due', () => { pm.expect(j.status).to.eql('PAID'); pm.expect(j.balance_due_minor).to.eql(0); });",
                                    "pm.test('card shown as brand and last 4 only', () => pm.expect(j.payments.find(p => p.type === 'FINAL').card_last4).to.match(/^\\d{4}$/));")),
        step("4.04 Nothing left to pay", "POST", "billingUrl", "/api/v1/bills/{{billId}}/payments", "customerToken",
             headers={"Idempotency-Key": "{{$guid}}"}, pre=SKIP_UNLESS_STRIPE, tests=status(409)),
        step("4.05 My bills", "GET", "billingUrl", "/api/v1/me/bills", "customerToken",
             tests=status(200) + js("pm.test('this bill is listed', () => pm.expect(pm.response.json().bills.map(b => b.id)).to.include(Number(pm.collectionVariables.get('billId'))));")),
    ]},
]}


def var(k, v): return {"key": k, "value": v, "type": "string"}
coll = {
    "info": {"name": "Service Bay", "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
             "description": "All HTTP APIs for Service Bay. 'Full flow' runs one booking from sign-up to a paid bill with assertions (postman/README.md). In the per-service folders, run Identity > Register/Login first; create/register requests store ids into collection variables."},
    "item": [full_flow, identity, with_auth(customer), with_auth(vehicle), with_auth(scheduler), with_auth(billing)],
    "variable": [var("identityUrl", "http://localhost:8083"), var("customerUrl", "http://localhost:8080"),
                 var("vehicleUrl", "http://localhost:8081"), var("billingUrl", "http://localhost:8084"),
                 var("schedulerUrl", "http://localhost:8085"),
                 # The bootstrap admin from docker-compose.yml (identity-service ADMIN_EMAIL / ADMIN_PASSWORD).
                 var("adminEmail", "admin@servicebay.local"), var("adminPassword", "admin-local-dev-pass"),
                 var("userEmail", "dev@example.com"), var("userPassword", "Passw0rd!"),
                 var("accessToken", ""), var("userId", ""),
                 var("customerId", "1"), var("targetCustomerId", "2"), var("vehicleId", "1"), var("billId", "1"),
                 var("customerUpdatedAt", ""), var("randomVin", ""), var("vehicleUpdatedAt", ""),
                 var("hubId", "1"), var("appointmentId", ""), var("paymentId", ""),
                 # Full flow state.
                 var("staffToken", ""), var("customerToken", ""), var("flowEmail", ""), var("flowCustomerId", ""),
                 var("flowVehicleId", ""), var("hubTz", ""), var("bookDate", ""), var("slotStart", ""),
                 var("depositKey", ""), var("depositPaymentId", ""), var("finalPaymentId", ""),
                 var("stripe", ""), var("skipTo", ""), var("pollAttempts", "0")],
}
with open(out, "w") as f:
    json.dump(coll, f, indent=2)
    f.write("\n")
