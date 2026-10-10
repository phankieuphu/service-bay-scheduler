"""Generates service-bay.postman_collection.json.

Routes and bodies are hand-mirrored from each service's HTTP handlers/DTOs —
update this file when an endpoint changes, then run:

    python3 postman/gen_collection.py [output-path]
"""
import json, os, sys
out = sys.argv[1] if len(sys.argv) > 1 else os.path.join(os.path.dirname(os.path.abspath(__file__)), "service-bay.postman_collection.json")

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
        for v in ("customerId", "targetCustomerId", "vehicleId", "billId"): raw = raw.replace('"{{%s}}"' % v, "{{%s}}" % v)
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
    req("Create bill", "POST", "billingUrl", "/api/v1/bills",
        {"appointmentId": 1, "customerId": "{{customerId}}", "vehicleId": "{{vehicleId}}", "warrantyFee": 0, "serviceFee": 500000, "materialCost": 250000},
        desc="appointmentId/customerId/vehicleId required; fees must be non-negative.",
        tests=save("billId", "j.id")),
    req("Get bill", "GET", "billingUrl", "/api/v1/bills/{{billId}}"),
    req("List bills by customer", "GET", "billingUrl", "/api/v1/bills", query=[q("customerId", "{{customerId}}")]),
    req("Record payment", "POST", "billingUrl", "/api/v1/bills/{{billId}}/payments",
        {"type": "DEPOSIT", "method": "CASH", "amount": 100000},
        desc="type: DEPOSIT | FINAL. amount > 0."),
    req("List payments", "GET", "billingUrl", "/api/v1/bills/{{billId}}/payments"),
    req("Void bill", "POST", "billingUrl", "/api/v1/bills/{{billId}}/void"),
    {"name": "Ops", "item": [
        req("Actuator health", "GET", "billingUrl", "/actuator/health"),
        req("Actuator info", "GET", "billingUrl", "/actuator/info"),
    ]},
]}

def var(k, v): return {"key": k, "value": v, "type": "string"}
coll = {
    "info": {"name": "Service Bay", "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
             "description": "All HTTP APIs for Service Bay. Run Identity > Register/Login first; create/register requests store IDs (customerId, vehicleId, billId) into collection variables for later requests."},
    "item": [identity, customer, vehicle, billing],
    "variable": [var("identityUrl", "http://localhost:8083"), var("customerUrl", "http://localhost:8080"),
                 var("vehicleUrl", "http://localhost:8081"), var("billingUrl", "http://localhost:8084"),
                 var("userEmail", "dev@example.com"), var("userPassword", "Passw0rd!"),
                 var("accessToken", ""), var("refreshToken", ""), var("userId", ""),
                 var("customerId", "1"), var("targetCustomerId", "2"), var("vehicleId", "1"), var("billId", "1"),
                 var("customerUpdatedAt", ""), var("randomVin", ""), var("vehicleUpdatedAt", "")],
}
with open(out, "w") as f:
    json.dump(coll, f, indent=2)
    f.write("\n")
