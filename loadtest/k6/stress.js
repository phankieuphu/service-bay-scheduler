// Stress: step the arrival rate up to MAX_RATE and find where the SLO breaks
// (issue #38 §2 — "record max RPS"). Every step is tagged (step1, step2, ...)
// and evaluated on its own, so the summary shows the last step that still
// met the SLO — that step's target is the max sustainable ops/s. The run
// only aborts early if errors go past 10% overall (no point hammering a
// dead service).
//
//   ./k6/run.sh stress
//   k6 run -e TIER=100k k6/stress.js
// Overrides: MAX_RATE (ops/s ceiling), STEPS (10), STEP_DURATION (2m)
//
// For a per-pod ops/s-per-vCPU number (the input to the HPA sizing in issue
// #38), scale each service to 1 replica and disable the HPA first.

import { rate, vus } from './lib/config.js';
import { runMixedOp } from './lib/mix.js';
import { phaseTagger, k6Stages, phaseNames, totalDuration, secondsOf } from './lib/phases.js';
import { probe as sample, probeScenario } from './lib/probe.js';
import {
  thresholds, phaseReportThresholds, phaseTable, SUMMARY_TREND_STATS, summaryHandler,
} from './lib/summary.js';

const MAX_RATE = rate('stress', 'MAX_RATE');
const STEPS = parseInt(__ENV.STEPS || '10', 10);
const STEP_DURATION = __ENV.STEP_DURATION || '2m';

// Each step: 15s ramp to the new target, then hold for the rest of the step.
const STAGES = [];
const TARGETS = {};
for (let i = 1; i <= STEPS; i++) {
  const target = Math.round((MAX_RATE * i) / STEPS);
  TARGETS[`step${i}`] = target;
  STAGES.push({ duration: '15s', target, phase: `step${i}` });
  STAGES.push({ duration: `${secondsOf(STEP_DURATION) - 15}s`, target, phase: `step${i}` });
}
const PHASES = phaseNames(STAGES);
const tagPhase = phaseTagger(STAGES);

export const options = {
  scenarios: {
    stress: {
      executor: 'ramping-arrival-rate',
      startRate: 0,
      timeUnit: '1s',
      stages: k6Stages(STAGES),
      ...vus(MAX_RATE),
    },
    probe: probeScenario(totalDuration(STAGES)),
  },
  thresholds: thresholds(
    { 'http_req_failed{kind:api}': [{ threshold: 'rate<0.10', abortOnFail: true, delayAbortEval: '30s' }] },
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

export const handleSummary = summaryHandler('stress', (data) => {
  const { text, lastGood, firstBad } = phaseTable(data, PHASES, TARGETS);
  const dropped = data.metrics.dropped_iterations ? data.metrics.dropped_iterations.values.count : 0;
  const verdict = firstBad
    ? `BREAK POINT: SLO first breached at ${firstBad} (${TARGETS[firstBad]} ops/s); ` +
      `max sustainable ≈ ${lastGood ? `${TARGETS[lastGood]} ops/s (${lastGood})` : 'below the first step'}`
    : `NO BREAK up to ${MAX_RATE} ops/s — raise MAX_RATE`;
  return `\n== per step\n${text}\n\n${verdict}\ndropped_iterations=${dropped} (k6 ran out of VUs: the system stopped keeping up)`;
});
