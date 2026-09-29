#!/usr/bin/env bash
# Soak: medium load for hours to surface memory / goroutine / connection
# leaks (issue #38 §2 — 2–4h). Go runtime metrics are sampled from both
# services throughout (results/.../runtime.csv); at constant load the
# goroutine count and heap should plateau, not climb.
#
# Usage: TIER=1k|100k|1m MIX=read|mixed|write ./sh/soak.sh
# Overrides: RATE (ops/s), DURATION (2h), SAMPLE_INTERVAL (60s)
set -uo pipefail
SAMPLE_INTERVAL="${SAMPLE_INTERVAL:-60}"
. "$(dirname "$0")/../lib/common.sh"
. "$(dirname "$0")/../lib/engine.sh"

RATE="${RATE:-$(tier_rate soak)}"
DURATION="$(to_secs "${DURATION:-2h}")"

engine_init soak
run_stages 0 "60:$RATE:ramp_up" "$DURATION:$RATE:soak"
report "soak"
status=$?

# Leak heuristic: compare the first and last 10% of samples (skipping the
# ramp) — more than 50% growth in goroutines or heap is flagged.
awk -F, -v skip=$((60 / SAMPLE_INTERVAL + 1)) '
  NR > 1 { s = $2; k = ++n[s]; if (k > skip) { g[s, k] = $3; h[s, k] = $4 } }
  END {
    for (s in n) {
      m = n[s] - skip; if (m < 10) { printf "leak-check %s: not enough samples (%d)\n", s, m; continue }
      w = int(m / 10); ga = gb = ha = hb = 0
      for (i = 1; i <= w; i++) { ga += g[s, skip + i]; ha += h[s, skip + i]; gb += g[s, n[s] - w + i]; hb += h[s, n[s] - w + i] }
      flag = (gb > ga * 1.5 || hb > ha * 1.5) ? "WARN: growing — possible leak" : "ok: stable"
      printf "leak-check %s: goroutines %.0f -> %.0f, heap %.1f -> %.1f MiB  %s\n",
        s, ga / w, gb / w, ha / w / 1048576, hb / w / 1048576, flag
    }
  }' "$RESULTS/runtime.csv" | tee -a "$RESULTS/report.txt"
exit "$status"
