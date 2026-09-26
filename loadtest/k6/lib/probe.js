// Samples Go runtime metrics from both services' /metrics endpoint (exposed
// by promhttp's default registry) into k6 gauges, so every run's summary
// shows goroutine / heap / RSS min..max — a count that keeps climbing at
// constant load is a leak. Behind a multi-replica k8s Service each sample may
// hit a different pod; use the Grafana per-pod panels for the real picture.

import http from 'k6/http';
import { Gauge } from 'k6/metrics';
import { CUSTOMER_URL, VEHICLE_URL } from './config.js';

const SERVICES = { customer: CUSTOMER_URL, vehicle: VEHICLE_URL };
const SERIES = {
  goroutines: 'go_goroutines',
  heap_bytes: 'go_memstats_heap_alloc_bytes',
  rss_bytes: 'process_resident_memory_bytes',
  open_fds: 'process_open_fds',
};

const gauges = {};
for (const svc of Object.keys(SERVICES)) {
  for (const key of Object.keys(SERIES)) gauges[`${svc}_${key}`] = new Gauge(`${svc}_${key}`);
  // current / baseline, where baseline is the first sample after warm-up
  gauges[`${svc}_goroutines_growth`] = new Gauge(`${svc}_goroutines_growth`);
  gauges[`${svc}_heap_growth`] = new Gauge(`${svc}_heap_growth`);
}

// Samples taken before the baseline is fixed (load still ramping up).
const WARMUP_SAMPLES = parseInt(__ENV.PROBE_WARMUP_SAMPLES || '4', 10);
const seen = {};
const baseline = {};

export function probe() {
  for (const [svc, url] of Object.entries(SERVICES)) {
    const res = http.get(`${url}/metrics`, { tags: { name: 'GET /metrics', kind: 'probe' } });
    if (res.status !== 200) continue;
    const values = {};
    for (const [key, series] of Object.entries(SERIES)) {
      const m = res.body.match(new RegExp(`^${series} (\\S+)$`, 'm'));
      if (m) {
        values[key] = parseFloat(m[1]);
        gauges[`${svc}_${key}`].add(values[key]);
      }
    }
    seen[svc] = (seen[svc] || 0) + 1;
    if (seen[svc] === WARMUP_SAMPLES + 1) baseline[svc] = values;
    const b = baseline[svc];
    if (b && b.goroutines) gauges[`${svc}_goroutines_growth`].add(values.goroutines / b.goroutines);
    if (b && b.heap_bytes) gauges[`${svc}_heap_growth`].add(values.heap_bytes / b.heap_bytes);
  }
}

// One sample every `every` for the whole test. A single VU, so the
// warm-up baseline above is shared by every sample.
export const probeScenario = (duration, every = '15s') => ({
  executor: 'constant-arrival-rate',
  exec: 'probe',
  rate: 1,
  timeUnit: every,
  duration,
  preAllocatedVUs: 1,
  maxVUs: 1,
});
