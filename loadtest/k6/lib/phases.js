// Tags every metric a VU emits with the phase of the scenario it's in
// ("hold", "peak", "step3", ...), so thresholds and the summary can be
// evaluated per phase instead of over the whole run — e.g. only the hold
// phase of a load test must meet the SLO, and the stress test's per-step
// numbers show where the SLO broke.

import exec from 'k6/execution';

// stages: [{ duration: '2m', target: 100, phase: 'ramp_up' }, ...]
export function phaseTagger(stages) {
  const bounds = [];
  let acc = 0;
  for (const s of stages) {
    acc += secondsOf(s.duration) * 1000;
    bounds.push([acc, s.phase, s.target]);
  }
  return function tag() {
    const elapsed = Date.now() - exec.scenario.startTime;
    let found = bounds[bounds.length - 1];
    for (const b of bounds) {
      if (elapsed < b[0]) {
        found = b;
        break;
      }
    }
    exec.vu.metrics.tags.phase = found[1];
    return { phase: found[1], target: found[2] };
  };
}

// Strip `phase` so the array can be passed to k6 as executor stages.
export const k6Stages = (stages) => stages.map(({ duration, target }) => ({ duration, target }));

// Total duration of the stages, as a k6 duration string.
export const totalDuration = (stages) => `${stages.reduce((t, s) => t + secondsOf(s.duration), 0)}s`;

// Unique phase names, in order.
export const phaseNames = (stages) => [...new Set(stages.map((s) => s.phase))];

export function secondsOf(d) {
  let total = 0;
  const re = /(\d+)(h|m|s)/g;
  let m;
  while ((m = re.exec(d)) !== null) total += parseInt(m[1], 10) * { h: 3600, m: 60, s: 1 }[m[2]];
  return total;
}
