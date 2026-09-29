// Baseline: normal traffic at a constant arrival rate (issue #38 §2 — 10 min).
// The reference numbers every later scenario is compared against.
//
//   ./k6/run.sh baseline                       # via the wrapper (recommended)
//   k6 run -e TIER=100k -e MIX=mixed k6/baseline.js
// Overrides: RATE (ops/s), DURATION (10m)

import { rate, vus } from './lib/config.js';
import { runMixedOp } from './lib/mix.js';
import { probe as sample, probeScenario } from './lib/probe.js';
import {
  thresholds, sloThresholds, conflictThresholds, SUMMARY_TREND_STATS, summaryHandler,
} from './lib/summary.js';

const RATE = rate('baseline');
const DURATION = __ENV.DURATION || '10m';

export const options = {
  scenarios: {
    baseline: { executor: 'constant-arrival-rate', rate: RATE, timeUnit: '1s', duration: DURATION, ...vus(RATE) },
    probe: probeScenario(DURATION),
  },
  thresholds: thresholds(sloThresholds(), conflictThresholds),
  summaryTrendStats: SUMMARY_TREND_STATS,
};

export default function () {
  runMixedOp();
}

export function probe() {
  sample();
}

export const handleSummary = summaryHandler('baseline');
