// Weighted operation mix. One k6 iteration = one operation, so arrival rates
// are operations/s — the same unit the sh runners use. Keep the weights in
// sync with mix_weights in lib/common.sh.

import {
  getCustomer, listCustomers, getVehicle, createCustomer,
  updateCustomer, deleteCustomer, transferVehicle,
} from './api.js';
import { TRANSFERS_ENABLED } from './config.js';

const OPS = {
  cust_get: getCustomer,
  cust_list: listCustomers,
  veh_get: getVehicle,
  cust_create: createCustomer,
  cust_update: updateCustomer,
  cust_delete: deleteCustomer,
  transfer: transferVehicle,
};

// percent per op: read = reads only, mixed = the 90/10 split from issue #38,
// write = 20/80 to load the transactional-outbox write path.
const MIXES = {
  read: { cust_get: 40, cust_list: 15, veh_get: 45 },
  mixed: { cust_get: 36, cust_list: 13, veh_get: 41, cust_create: 3, cust_update: 3, cust_delete: 1, transfer: 3 },
  write: { cust_get: 8, cust_list: 3, veh_get: 9, cust_create: 25, cust_update: 25, cust_delete: 5, transfer: 25 },
};

const MIX = __ENV.MIX || 'mixed';
if (!MIXES[MIX]) throw new Error(`unknown MIX ${MIX} (read|mixed|write)`);

const weights = { ...MIXES[MIX] };
if (!TRANSFERS_ENABLED && weights.transfer) {
  // no seeded transfer pool: give its share to vehicle reads
  weights.veh_get = (weights.veh_get || 0) + weights.transfer;
  delete weights.transfer;
}

const table = [];
let total = 0;
for (const [op, w] of Object.entries(weights)) {
  total += w;
  table.push([total, op]);
}

export function randomOp() {
  const r = Math.random() * total;
  for (const [edge, op] of table) if (r < edge) return op;
  return table[table.length - 1][1];
}

export function runMixedOp() {
  return OPS[randomOp()]();
}

export const MIX_NAME = MIX;
