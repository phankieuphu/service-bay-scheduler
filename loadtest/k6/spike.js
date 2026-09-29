// Spike: baseline -> 10x baseline in 20s, hold, drop back, and check the
// system recovers (issue #38 §2). The strict SLO applies to warm-up and
// recovery; during the peak the services only have to stay up (errors < 1%).
//
//   ./k6/run.sh spike
//   k6 run -e TIER=100k k6/spike.js
// Overrides: BASE_RATE, SPIKE_RATE (ops/s), PEAK_DURATION (3m),
//            RECOVERY_DURATION (5m)

import { rate, vus } from './lib/config.js';
import { runMixedOp } from './lib/mix.js';
import { phaseTagger, k6Stages, phaseNames, totalDuration } from './lib/phases.js';
import { probe as sample, probeScenario } from './lib/probe.js';
import {
  thresholds, sloThresholds, phaseReportThresholds, phaseTable, SUMMARY_TREND_STATS, summaryHandler,
} from './lib/summary.js';

const BASE = rate('baseline', 'BASE_RATE');
const SPIKE = rate('spike', 'SPIKE_RATE');

const STAGES = [
  { duration: '1m', target: BASE, phase: 'warmup' },
  { duration: '20s', target: SPIKE, phase: 'spike_up' },
  { duration: __ENV.PEAK_DURATION || '3m', target: SPIKE, phase: 'peak' },
  { duration: '20s', target: BASE, phase: 'spike_down' },
  { duration: __ENV.RECOVERY_DURATION || '5m', target: BASE, phase: 'recovery' },
];
const PHASES = phaseNames(STAGES);
const TARGETS = Object.fromEntries(STAGES.map((s) => [s.phase, s.target]));
const tagPhase = phaseTagger(STAGES);

export const options = {
  scenarios: {
    spike: {
      executor: 'ramping-arrival-rate',
      startRate: BASE,
      timeUnit: '1s',
      stages: k6Stages(STAGES),
      ...vus(SPIKE),
    },
    probe: probeScenario(totalDuration(STAGES)),
  },
  thresholds: thresholds(
    phaseReportThresholds(PHASES),
    sloThresholds('kind:api,phase:warmup'),
    sloThresholds('kind:api,phase:recovery'),
    { 'http_req_failed{kind:api,phase:peak}': ['rate<0.01'] },
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

export const handleSummary = summaryHandler('spike', (data) => {
  const { text } = phaseTable(data, PHASES, TARGETS);
  return `\n== per phase (recovery should look like warmup)\n${text}`;
});
