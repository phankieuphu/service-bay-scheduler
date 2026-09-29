// Load: expected peak traffic (issue #38 §2 — 30 min). Ramp up, hold, ramp
// down. The SLO thresholds apply to the hold phase only; watch the HPA scale
// out during ramp_up and back in after ramp_down.
//
//   ./k6/run.sh load
//   k6 run -e TIER=1m -e MIX=mixed k6/load.js
// Overrides: RATE (ops/s), DURATION (hold, 30m), RAMP (2m)

import { rate, vus } from './lib/config.js';
import { runMixedOp } from './lib/mix.js';
import { phaseTagger, k6Stages, phaseNames, totalDuration } from './lib/phases.js';
import { probe as sample, probeScenario } from './lib/probe.js';
import {
  thresholds, sloThresholds, conflictThresholds, phaseReportThresholds, phaseTable,
  SUMMARY_TREND_STATS, summaryHandler,
} from './lib/summary.js';

const RATE = rate('load');
const HOLD = __ENV.DURATION || '30m';
const RAMP = __ENV.RAMP || '2m';

const STAGES = [
  { duration: RAMP, target: RATE, phase: 'ramp_up' },
  { duration: HOLD, target: RATE, phase: 'hold' },
  { duration: '1m', target: 0, phase: 'ramp_down' },
];
const PHASES = phaseNames(STAGES);
const tagPhase = phaseTagger(STAGES);

export const options = {
  scenarios: {
    load: {
      executor: 'ramping-arrival-rate',
      startRate: 0,
      timeUnit: '1s',
      stages: k6Stages(STAGES),
      ...vus(RATE),
    },
    probe: probeScenario(totalDuration(STAGES)),
  },
  thresholds: thresholds(
    sloThresholds('kind:api,phase:hold'),
    conflictThresholds,
    phaseReportThresholds(PHASES),
  ),
  summaryTrendStats: SUMMARY_TREND_STATS,
};

export default function () {
  tagPhase();
  runMixedOp();
}

export function probe() {
  sample();
}

export const handleSummary = summaryHandler('load', (data) => `\n== per phase\n${phaseTable(data, PHASES).text}`);
