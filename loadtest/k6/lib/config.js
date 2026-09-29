// Shared configuration for the k6 scenarios. Everything is overridable with
// `-e NAME=value`; k6/run.sh passes the ids from loadtest/.env.seed for you.

const int = (name, def) => {
  const v = __ENV[name];
  return v === undefined || v === '' ? def : parseInt(v, 10);
};

export const CUSTOMER_URL = __ENV.CUSTOMER_URL || 'http://localhost:8080';
export const VEHICLE_URL = __ENV.VEHICLE_URL || 'http://localhost:8081';

// Id ranges written by seed/seed.sh. The fallbacks match postgres/seed/*.sql;
// without a seeded transfer pool, transfers are replaced by vehicle reads.
export const SEED = {
  customerIdMin: int('CUSTOMER_ID_MIN', 1),
  customerIdMax: int('CUSTOMER_ID_MAX', 1000),
  ltCustomerMin: int('LT_CUSTOMER_MIN', int('CUSTOMER_ID_MIN', 1)),
  ltCustomerMax: int('LT_CUSTOMER_MAX', int('CUSTOMER_ID_MAX', 1000)),
  vehicleIdMin: int('VEHICLE_ID_MIN', 1),
  vehicleIdMax: int('VEHICLE_ID_MAX', 5000),
  transferVehicleStart: int('TRANSFER_VEHICLE_START', 0),
  transferPool: int('TRANSFER_POOL', 0),
  transferCustomerA: int('TRANSFER_CUSTOMER_A', 0),
  transferCustomerB: int('TRANSFER_CUSTOMER_B', 0),
  smokeVehicleId: int('SMOKE_VEHICLE_ID', 0),
};

export const TRANSFERS_ENABLED =
  SEED.transferPool > 0 && SEED.transferCustomerA > 0 && SEED.transferCustomerB > 0;

// Target operations/s per tier, from the traffic model in issue #38
// (1k / 100k / 1M registered users). `stress` is the ceiling the stress ramp
// climbs to (2x the tier's stress target). Keep in sync with lib/common.sh.
const TIERS = {
  '1k': { baseline: 5, load: 10, stress: 50, spike: 50, soak: 5 },
  '100k': { baseline: 30, load: 100, stress: 500, spike: 300, soak: 50 },
  '1m': { baseline: 300, load: 1000, stress: 5000, spike: 3000, soak: 500 },
};

export const TIER = (__ENV.TIER || '1k').toLowerCase();
if (!TIERS[TIER]) throw new Error(`unknown TIER ${TIER} (1k|100k|1m)`);

// rate('load') -> ops/s for the tier, unless overridden with -e RATE=...
export const rate = (scenario, override = 'RATE') => int(override, TIERS[TIER][scenario]);

// SLO from issue #38's acceptance criteria.
export const SLO = {
  p95: int('SLO_P95_MS', 200),
  p99: int('SLO_P99_MS', 500),
  errRate: parseFloat(__ENV.SLO_ERR_RATE || '0.001'),
};

// VU budget for an arrival-rate executor peaking at `peak` ops/s. An op is
// at most 2 requests; maxVUs covers ~1s per op before k6 starts dropping
// iterations (which the summary reports as dropped_iterations).
export function vus(peak) {
  const maxVUs = Math.max(20, Math.ceil(peak * 1.0));
  if (TRANSFERS_ENABLED && maxVUs > SEED.transferPool) {
    console.warn(`maxVUs ${maxVUs} > TRANSFER_POOL ${SEED.transferPool}: some VUs share a vehicle and will see 409s`);
  }
  return { preAllocatedVUs: Math.max(5, Math.ceil(peak * 0.2)), maxVUs };
}

export const RESULTS_DIR = __ENV.RESULTS_DIR || 'results';
