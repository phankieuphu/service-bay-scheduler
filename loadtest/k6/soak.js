// Soak: medium load for hours to catch memory / goroutine / connection leaks
// (issue #38 §2 — 2–4h). Both services' /metrics are sampled every minute;
// *_goroutines_growth is the current count divided by the count right after
// warm-up, and must stay under 1.5x at constant load. Heap growth is
// reported but not enforced (it saw-tooths with GC).
//
//   ./k6/run.sh soak
//   k6 run -e TIER=100k -e DURATION=4h k6/soak.js
// Overrides: RATE (ops/s), DURATION (2h)

import { rate, vus } from './lib/config.js';
import { runMixedOp } from './lib/mix.js';
import { phaseTagger, k6Stages, phaseNames, totalDuration } from './lib/phases.js';
import { probe as sample, probeScenario } from './lib/probe.js';
import {
  thresholds, sloThresholds, conflictThresholds, phaseReportThresholds, phaseTable,
  SUMMARY_TREND_STATS, summaryHandler,
} from './lib/summary.js';

const RATE = rate('soak');

const STAGES = [
  { duration: '1m', target: RATE, phase: 'ramp_up' },
  { duration: __ENV.DURATION || '2h', target: RATE, phase: 'soak' },
];
const PHASES = phaseNames(STAGES);
const tagPhase = phaseTagger(STAGES);

export const options = {
  scenarios: {
    soak: {
      executor: 'ramping-arrival-rate',
      startRate: 0,
      timeUnit: '1s',
      stages: k6Stages(STAGES),
      ...vus(RATE),
    },
    // 1/min; the baseline is fixed after 4 samples, i.e. ~3 min into the soak
    probe: probeScenario(totalDuration(STAGES), '1m'),
  },
  thresholds: thresholds(
    sloThresholds('kind:api,phase:soak'),
    conflictThresholds,
    phaseReportThresholds(PHASES),
    {
      customer_goroutines_growth: ['value<1.5'],
      vehicle_goroutines_growth: ['value<1.5'],
    },
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

export const handleSummary = summaryHandler('soak', (data) => {
  const growth = ['customer', 'vehicle'].map((svc) => {
    const g = data.metrics[`${svc}_goroutines_growth`];
    const h = data.metrics[`${svc}_heap_growth`];
    if (!g) return `${svc}: not enough samples for a leak check`;
    return `${svc}: goroutines x${g.values.value.toFixed(2)} (max x${g.values.max.toFixed(2)})` +
      (h ? `, heap x${h.values.value.toFixed(2)} (max x${h.values.max.toFixed(2)})` : '') +
      (g.values.value >= 1.5 ? '  <-- possible goroutine leak' : '');
  });
  return `\n== per phase\n${phaseTable(data, PHASES).text}\n\n== leak check (vs. after warm-up)\n${growth.join('\n')}`;
});
