// k6 load benchmark for POST /api/v1/transfer (VehicleService.TransferVehicle).
//
// This complements scripts/transfer-race-test.sh: the bash script fires a
// handful of concurrent requests at ONE vehicle to prove the row-locking
// picks exactly one winner; this script drives sustained load to measure
// throughput/latency of the transfer path (RunInTx + FOR UPDATE lock +
// outbox insert) under contention-free conditions.
//
// Each VU owns a dedicated vehicle (vehicleId = VEHICLE_ID_START + (__VU-1))
// and ping-pongs its ownership between CUSTOMER_A and CUSTOMER_B every
// iteration, so VUs never contend with each other for the same row — the
// benchmark measures steady-state cost, not lock contention.
//
// Precondition: every vehicle in the range
//   [VEHICLE_ID_START, VEHICLE_ID_START + VUS - 1]
// must currently be owned (status='CURRENT' in customer_vehicle) by
// CUSTOMER_A. Check/seed with:
//   docker exec postgres psql -U postgres -d vehicle_db -c \
//     "select vehicle_id, customer_id from customer_vehicle \
//      where status='CURRENT' and vehicle_id >= $VEHICLE_ID_START \
//      order by vehicle_id"
//
// Usage:
//   k6 run scripts/k6/transfer-benchmark.js
//   k6 run -e BASE_URL=http://localhost:8081 -e VUS=20 -e DURATION=60s \
//          -e VEHICLE_ID_START=1000 -e CUSTOMER_A=101 -e CUSTOMER_B=104 \
//          scripts/k6/transfer-benchmark.js
//
// Install k6: https://k6.io/docs/get-started/installation/

import http from 'k6/http';
import { check } from 'k6';
import { Counter } from 'k6/metrics';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8081';
const VUS = parseInt(__ENV.VUS || '10', 10);
const DURATION = __ENV.DURATION || '30s';
const RAMP_UP = __ENV.RAMP_UP || '5s';
const RAMP_DOWN = __ENV.RAMP_DOWN || '5s';

const VEHICLE_ID_START = parseInt(__ENV.VEHICLE_ID_START || '1000', 10);
const CUSTOMER_A = parseInt(__ENV.CUSTOMER_A || '101', 10);
const CUSTOMER_B = parseInt(__ENV.CUSTOMER_B || '104', 10);

const transferSuccess = new Counter('transfer_success');
const transferConflict = new Counter('transfer_conflict');
const transferNotFound = new Counter('transfer_not_found');
const transferOtherError = new Counter('transfer_other_error');

export const options = {
  scenarios: {
    transfer_load: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: RAMP_UP, target: VUS },
        { duration: DURATION, target: VUS },
        { duration: RAMP_DOWN, target: 0 },
      ],
      gracefulRampDown: '5s',
    },
  },
  thresholds: {
    // 204s only; 404/409 recorded separately above and shouldn't occur
    // once each VU has its own dedicated vehicle.
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<300', 'p(99)<800'],
  },
};

// Per-VU state: which customer currently owns this VU's vehicle.
// Module-level state persists across iterations of the same VU in k6.
let owner = CUSTOMER_A;

export default function () {
  const vehicleId = VEHICLE_ID_START + (__VU - 1);
  const to = owner === CUSTOMER_A ? CUSTOMER_B : CUSTOMER_A;

  const payload = JSON.stringify({
    vehicle_id: vehicleId,
    from: owner,
    to: to,
    date: new Date().toISOString(),
  });

  const res = http.post(`${BASE_URL}/api/v1/transfer`, payload, {
    headers: { 'Content-Type': 'application/json' },
    tags: { name: 'TransferVehicle' },
  });

  const ok = check(res, {
    'status is 204': (r) => r.status === 204,
  });

  if (ok) {
    transferSuccess.add(1);
    owner = to; // flip local ownership only after a confirmed win
  } else if (res.status === 409) {
    transferConflict.add(1);
  } else if (res.status === 404) {
    transferNotFound.add(1);
  } else {
    transferOtherError.add(1);
  }
}
