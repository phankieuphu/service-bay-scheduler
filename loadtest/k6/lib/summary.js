// Thresholds + end-of-test summary shared by every scenario.

import { textSummary } from 'https://jslib.k6.io/k6-summary/0.1.0/index.js';
import { SLO, RESULTS_DIR, TIER } from './config.js';
import { MIX_NAME } from './mix.js';

export const SUMMARY_TREND_STATS = ['avg', 'min', 'med', 'max', 'p(90)', 'p(95)', 'p(99)', 'count'];

// Merge threshold objects, concatenating lists for the same metric (a plain
// object spread would silently drop the earlier ones).
export function thresholds(...sets) {
  const out = {};
  for (const set of sets) {
    for (const [k, v] of Object.entries(set)) out[k] = [...(out[k] || []), ...v];
  }
  return out;
}

// SLO thresholds from issue #38 over the requests matching `selector`.
export function sloThresholds(selector = 'kind:api', abortOnFail = false) {
  const t = (threshold) => (abortOnFail ? { threshold, abortOnFail: true, delayAbortEval: '1m' } : threshold);
  return {
    [`http_req_duration{${selector}}`]: [t(`p(95)<${SLO.p95}`), t(`p(99)<${SLO.p99}`)],
    [`http_req_failed{${selector}}`]: [t(`rate<${SLO.errRate}`)],
  };
}

// Business-level sanity: conflicts should stay rare when VUs don't share rows.
export const conflictThresholds = {
  transfer_conflict_rate: ['rate<0.05'],
  update_conflict_rate: ['rate<0.2'],
};

// k6 only reports submetrics that a threshold references, so register an
// always-true threshold per phase to get per-phase numbers in the summary.
export function phaseReportThresholds(phases) {
  const out = {};
  for (const p of phases) {
    out[`http_req_duration{kind:api,phase:${p}}`] = ['max>=0'];
    out[`http_req_failed{kind:api,phase:${p}}`] = ['rate>=0'];
    out[`http_reqs{kind:api,phase:${p}}`] = ['count>=0'];
  }
  return out;
}

// Per-phase table: requests, rps, error %, p50/p95/p99. `targets` maps a
// phase to its target ops/s (used by stress to find the break point).
export function phaseTable(data, phases, targets = {}) {
  const m = data.metrics;
  const rows = [];
  let lastGood = null;
  let firstBad = null;
  for (const p of phases) {
    const d = m[`http_req_duration{kind:api,phase:${p}}`];
    const f = m[`http_req_failed{kind:api,phase:${p}}`];
    const r = m[`http_reqs{kind:api,phase:${p}}`];
    if (!d || !r || !r.values.count) continue;
    const err = f ? f.values.rate : 0;
    const ok = d.values['p(95)'] < SLO.p95 && d.values['p(99)'] < SLO.p99 && err < SLO.errRate;
    if (ok && firstBad === null) lastGood = p;
    if (!ok && firstBad === null) firstBad = p;
    rows.push(
      [
        p.padEnd(12),
        String(targets[p] ?? '').padStart(8),
        String(r.values.count).padStart(9),
        (err * 100).toFixed(3).padStart(8),
        d.values.med.toFixed(1).padStart(8),
        d.values['p(95)'].toFixed(1).padStart(8),
        d.values['p(99)'].toFixed(1).padStart(8),
        ok ? '  ok' : '  SLO BREACH',
      ].join(' '),
    );
  }
  const header = ['phase'.padEnd(12), 'target/s'.padStart(8), 'requests'.padStart(9), 'err%'.padStart(8),
    'p50ms'.padStart(8), 'p95ms'.padStart(8), 'p99ms'.padStart(8)].join(' ');
  return { text: [header, ...rows].join('\n'), lastGood, firstBad };
}

// Go runtime gauges sampled by probe.js: min / last / max per service.
function runtimeTable(data) {
  const lines = [];
  for (const svc of ['customer', 'vehicle']) {
    const g = data.metrics[`${svc}_goroutines`];
    const h = data.metrics[`${svc}_heap_bytes`];
    const f = data.metrics[`${svc}_open_fds`];
    if (!g) continue;
    const mib = (v) => (v / 1048576).toFixed(1);
    lines.push(
      `${svc.padEnd(9)} goroutines min/last/max ${g.values.min}/${g.values.value}/${g.values.max}` +
        (h ? `   heap MiB ${mib(h.values.min)}/${mib(h.values.value)}/${mib(h.values.max)}` : '') +
        (f ? `   open fds ${f.values.min}/${f.values.value}/${f.values.max}` : ''),
    );
  }
  return lines.join('\n');
}

// handleSummary factory. `extra(data)` may return additional report text.
export function summaryHandler(name, extra = () => '') {
  return (data) => {
    const stamp = new Date().toISOString().replace(/[:.]/g, '-');
    const base = `${RESULTS_DIR}/k6-${name}-${TIER}-${MIX_NAME}-${stamp}`;
    const report = [
      `\n=== ${name} | tier=${TIER} mix=${MIX_NAME} | SLO p95<${SLO.p95}ms p99<${SLO.p99}ms err<${SLO.errRate * 100}%`,
      extra(data),
      runtimeTable(data) ? `\n== Go runtime (sampled)\n${runtimeTable(data)}` : '',
    ].filter(Boolean).join('\n');
    return {
      stdout: textSummary(data, { indent: ' ', enableColors: true }) + '\n' + report + '\n',
      [`${base}.json`]: JSON.stringify(data, null, 2),
      [`${base}.txt`]: textSummary(data, { indent: ' ', enableColors: false }) + '\n' + report + '\n',
    };
  };
}
