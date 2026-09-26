// One function per API operation. Every request is tagged with a stable
// `name` (so /customer/123 and /customer/456 aggregate together) and
// `kind: api` (so SLO thresholds ignore the /metrics probe). Each request
// declares which statuses are expected, so http_req_failed only counts real
// failures; expected 404s / 409s are tracked in their own rates instead.

import http from 'k6/http';
import { check } from 'k6';
import { Rate } from 'k6/metrics';
import { CUSTOMER_URL, VEHICLE_URL, SEED } from './config.js';

const C = `${CUSTOMER_URL}/api/v1`;
const V = `${VEHICLE_URL}/api/v1`;
const JSON_HEADERS = { 'Content-Type': 'application/json' };

export const readMissRate = new Rate('read_miss_rate');
export const updateConflictRate = new Rate('update_conflict_rate');
export const transferConflictRate = new Rate('transfer_conflict_rate');

const params = (name, ...expected) => ({
  headers: JSON_HEADERS,
  tags: { name, kind: 'api' },
  responseCallback: http.expectedStatuses(...expected),
});

export const randInt = (min, max) => min + Math.floor(Math.random() * (max - min + 1));

// Per-VU state (module scope is per VU in k6).
const created = []; // ids this VU created — the only customers it deletes
let transferOwner = SEED.transferCustomerA;

export function getCustomer(id = randInt(SEED.customerIdMin, SEED.customerIdMax)) {
  const res = http.get(`${C}/customer/${id}`, params('GET /customer/:id', 200, 404));
  check(res, { 'GET /customer/:id 200|404': (r) => r.status === 200 || r.status === 404 });
  readMissRate.add(res.status === 404);
  return res;
}

export function listCustomers() {
  // half the listings hit the first (hot, 30s-cached) page, half deep pages
  const cursor = Math.random() < 0.5 ? 0 : randInt(SEED.customerIdMin, SEED.customerIdMax);
  const res = http.get(`${C}/customer?cursor=${cursor}&limit=20`, params('GET /customer', 200));
  check(res, { 'GET /customer 200': (r) => r.status === 200 });
  return res;
}

export function getVehicle(id = randInt(SEED.vehicleIdMin, SEED.vehicleIdMax)) {
  const res = http.get(`${V}/vehicle/${id}`, params('GET /vehicle/:id', 200, 404));
  check(res, { 'GET /vehicle/:id 200|404': (r) => r.status === 200 || r.status === 404 });
  readMissRate.add(res.status === 404);
  return res;
}

export function createCustomer(tag = 'c') {
  const email = `lt-${__ENV.RUN_ID || 'k6'}-${tag}-${__VU}-${__ITER}-${Date.now()}@example.com`;
  const res = http.post(
    `${C}/customer`,
    JSON.stringify({ name: 'Load Test', email, phone: '+15550000000', birth_day: '1990-01-01T00:00:00Z' }),
    params('POST /customer', 201),
  );
  check(res, { 'POST /customer 201': (r) => r.status === 201 });
  if (res.status === 201) {
    created.push(res.json('id'));
    if (created.length > 100) created.shift();
  }
  return res;
}

// Optimistic-locked update: read updated_at, send it back with the change.
// A 409 means another VU updated the same customer in between — expected at
// low rates, tracked in update_conflict_rate.
export function updateCustomer(id = randInt(SEED.ltCustomerMin, SEED.ltCustomerMax)) {
  const got = getCustomer(id);
  if (got.status !== 200) return got;
  const res = http.put(
    `${C}/customer/${id}`,
    JSON.stringify({ name: `Load Test ${id} ${__ITER}`, updated_at: got.json('updated_at') }),
    params('PUT /customer/:id', 204, 409),
  );
  check(res, { 'PUT /customer/:id 204|409': (r) => r.status === 204 || r.status === 409 });
  updateConflictRate.add(res.status === 409);
  return res;
}

// Soft-deletes a customer this VU created (creating one first if needed), so
// seeded data is never deleted.
export function deleteCustomer() {
  if (created.length === 0) {
    createCustomer('del');
    if (created.length === 0) return null;
  }
  const id = created.pop();
  const res = http.del(`${C}/customer/${id}`, null, params('DELETE /customer/:id', 204));
  check(res, { 'DELETE /customer/:id 204': (r) => r.status === 204 });
  return res;
}

// Each VU owns one vehicle of the seeded pool and ping-pongs it between
// customers A and B, so VUs never contend for the same row lock. A 409 means
// the DB disagrees about the current owner (e.g. a previous run stopped
// mid-way): flip our view and carry on — the next iteration is back in sync.
export function transferVehicle() {
  const vehicleId = SEED.transferVehicleStart + ((__VU - 1) % SEED.transferPool);
  const from = transferOwner;
  const to = from === SEED.transferCustomerA ? SEED.transferCustomerB : SEED.transferCustomerA;
  const res = http.post(
    `${V}/transfer`,
    JSON.stringify({ vehicle_id: vehicleId, from, to, date: new Date().toISOString() }),
    params('POST /transfer', 204, 409),
  );
  check(res, { 'POST /transfer 204': (r) => r.status === 204 });
  transferConflictRate.add(res.status === 409);
  if (res.status === 204 || res.status === 409) transferOwner = to;
  return res;
}
