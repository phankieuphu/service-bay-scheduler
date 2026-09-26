// Smoke: one request per API case, asserted with checks (1 VU, 1 iteration).
// Same cases as sh/smoke.sh. Run it before any load scenario; the run fails
// unless every check passes.
//
//   ./k6/run.sh smoke
//   k6 run k6/smoke.js

import http from 'k6/http';
import { check, group } from 'k6';
import { CUSTOMER_URL, VEHICLE_URL, SEED, TRANSFERS_ENABLED } from './lib/config.js';

const C = `${CUSTOMER_URL}/api/v1`;
const V = `${VEHICLE_URL}/api/v1`;
const H = { headers: { 'Content-Type': 'application/json' }, responseCallback: http.expectedStatuses({ min: 200, max: 499 }) };

export const options = {
  vus: 1,
  iterations: 1,
  thresholds: { checks: ['rate==1'] },
};

const is = (res, name, ...codes) => check(res, { [`${name} -> ${codes.join('|')}`]: (r) => codes.includes(r.status) });
const post = (url, body) => http.post(url, typeof body === 'string' ? body : JSON.stringify(body), H);
const put = (url, body) => http.put(url, JSON.stringify(body), H);
const get = (url) => http.get(url, H);
const del = (url) => http.del(url, null, H);
const BIRTH = '1990-01-01T00:00:00Z';

export default function () {
  const run = `${Date.now()}`;
  const email = `lt-smoke-${run}@example.com`;
  let id = null;
  let updatedAt = null;

  group('infra', () => {
    is(get(`${CUSTOMER_URL}/metrics`), 'customer-service GET /metrics', 200);
    is(get(`${VEHICLE_URL}/metrics`), 'vehicle-service GET /metrics', 200);
  });

  group('customer: create', () => {
    const res = post(`${C}/customer`, { name: 'Smoke Test', email, phone: '+15550001111', birth_day: BIRTH });
    if (is(res, 'POST /customer valid', 201)) {
      id = res.json('id');
      updatedAt = res.json('updated_at');
    }
    is(post(`${C}/customer`, { name: 'Smoke Test', email, birth_day: BIRTH }), 'POST /customer duplicate email', 409);
    is(post(`${C}/customer`, { email: `lt-smoke-${run}-x@example.com`, birth_day: BIRTH }), 'POST /customer missing name', 400);
    is(post(`${C}/customer`, { name: 'x', email: 'not-an-email', birth_day: BIRTH }), 'POST /customer invalid email', 400);
    is(post(`${C}/customer`, { name: 'x', email: `lt-smoke-${run}-y@example.com` }), 'POST /customer missing birth_day', 400);
    is(post(`${C}/customer`, 'not json'), 'POST /customer malformed json', 400);
  });

  group('customer: read', () => {
    if (id) is(get(`${C}/customer/${id}`), 'GET /customer/:id existing', 200);
    is(get(`${C}/customer/999999999`), 'GET /customer/:id unknown', 404);
    is(get(`${C}/customer/abc`), 'GET /customer/:id non-numeric', 400);
    const page = get(`${C}/customer?limit=5`);
    if (is(page, 'GET /customer first page', 200) && page.json('has_more')) {
      is(get(`${C}/customer?cursor=${page.json('next_cursor')}&limit=5`), 'GET /customer?cursor=next_cursor', 200);
    }
    is(get(`${C}/customer?limit=1000`), 'GET /customer limit over max (capped)', 200);
    is(get(`${C}/customer?cursor=-1`), 'GET /customer negative cursor', 400);
    is(get(`${C}/customer?limit=abc`), 'GET /customer non-numeric limit', 400);
  });

  group('customer: update (optimistic lock)', () => {
    if (!id) return;
    is(put(`${C}/customer/${id}`, { name: 'Smoke Renamed', updated_at: updatedAt }), 'PUT /customer/:id current updated_at', 204);
    is(put(`${C}/customer/${id}`, { name: 'Smoke Stale', updated_at: updatedAt }), 'PUT /customer/:id stale updated_at', 409);
    const fresh = get(`${C}/customer/${id}`).json('updated_at');
    is(put(`${C}/customer/${id}`, { updated_at: fresh }), 'PUT /customer/:id nothing to update', 400);
    is(put(`${C}/customer/${id}`, { name: 'x' }), 'PUT /customer/:id missing updated_at', 400);
    is(put(`${C}/customer/999999999`, { name: 'x', updated_at: fresh }), 'PUT /customer/:id unknown', 404);
    is(put(`${C}/customer/abc`, { name: 'x', updated_at: fresh }), 'PUT /customer/:id non-numeric', 400);

    // three concurrent PUTs carrying the same updated_at: exactly one wins
    const race = http.batch([1, 2, 3].map((n) => ['PUT', `${C}/customer/${id}`,
      JSON.stringify({ name: `Race ${n}`, updated_at: fresh }), H]));
    const wins = race.filter((r) => r.status === 204).length;
    check(wins, { 'PUT /customer/:id x3 concurrent -> exactly 1 winner': (w) => w === 1 });
  });

  group('customer: soft delete', () => {
    if (id) {
      is(del(`${C}/customer/${id}`), 'DELETE /customer/:id', 204);
      is(get(`${C}/customer/${id}`), 'GET /customer/:id after delete', 404);
      is(del(`${C}/customer/${id}`), 'DELETE /customer/:id already deleted', 404);
    }
    is(del(`${C}/customer/abc`), 'DELETE /customer/:id non-numeric', 400);
  });

  group('vehicle: read', () => {
    is(get(`${V}/vehicle/${SEED.vehicleIdMin}`), 'GET /vehicle/:id existing', 200);
    is(get(`${V}/vehicle/999999999`), 'GET /vehicle/:id unknown', 404);
    is(get(`${V}/vehicle/abc`), 'GET /vehicle/:id non-numeric', 400);
  });

  group('vehicle: transfer', () => {
    is(post(`${V}/transfer`, {}), 'POST /transfer empty body', 400);
    is(post(`${V}/transfer`, 'not json'), 'POST /transfer malformed json', 400);
    is(post(`${V}/transfer`, { vehicle_id: 999999999, from: 1, to: 2 }), 'POST /transfer unknown vehicle', 404);

    if (!TRANSFERS_ENABLED || !SEED.smokeVehicleId) {
      console.warn('transfer ownership cases skipped: run seed/seed.sh first');
      return;
    }
    const vid = SEED.smokeVehicleId;
    const A = SEED.transferCustomerA;
    const B = SEED.transferCustomerB;
    post(`${V}/transfer`, { vehicle_id: vid, from: B, to: A }); // normalise to owner A

    is(post(`${V}/transfer`, { vehicle_id: vid, from: B, to: A }), 'POST /transfer from non-owner', 409);
    is(post(`${V}/transfer`, { vehicle_id: vid, from: A, to: B, date: new Date().toISOString() }), 'POST /transfer A->B (with date)', 204);
    is(post(`${V}/transfer`, { vehicle_id: vid, from: B, to: A }), 'POST /transfer B->A (no date)', 204);

    // row lock in UnassignVehicleFromCustomer: exactly one concurrent transfer wins
    const targets = [B, B + 1, B + 2];
    const race = http.batch(targets.map((to) => ['POST', `${V}/transfer`,
      JSON.stringify({ vehicle_id: vid, from: A, to }), H]));
    const winners = targets.filter((_, i) => race[i].status === 204);
    check(winners, { 'POST /transfer x3 concurrent same vehicle -> exactly 1 winner': (w) => w.length === 1 });
    if (winners.length) post(`${V}/transfer`, { vehicle_id: vid, from: winners[0], to: A }); // restore
  });
}
