#!/usr/bin/env bash
# Open-model (arrival-rate) load engine for the sh runners, built on curl.
# Sourced after lib/common.sh.
#
# Every 1s tick it schedules `rate` operations, split across OPS by the MIX
# weights. Single-request operations go into one `curl --parallel` batch (one
# process per tick, not per request); two-step operations (update = GET+PUT,
# delete = POST+DELETE) run as small background chains. Like k6's
# arrival-rate executors, a slow server does NOT slow the schedule down —
# if more than MAX_INFLIGHT_JOBS ticks/chains are still running, the tick is
# dropped and logged instead (k6's dropped_iterations), which is itself a
# saturation signal.
#
# Output, under results/sh-<scenario>-<tier>-<timestamp>/:
#   requests.log   "<tick> <phase> <op> <http_code> <seconds>" per request
#                  (assembled at the end from raw/<tick>.batch|.chain — curl
#                  block-buffers its write-out, so concurrent writers must
#                  never share a file or lines get torn)
#   schedule.log   "<tick> <phase> <target_rate>" per tick
#   dropped.log    ticks skipped because too much was still in flight
#   runtime.csv    Go runtime metrics sampled from both services' /metrics
#   report.txt     the final report
#
# Good for up to a few hundred ops/s from a laptop; beyond that the load
# generator itself becomes the bottleneck — use the k6 runners. Each tick is
# a fresh curl process, so connections are only reused within a tick's
# batch; PARALLEL_MAX bounds the connections a batch opens (the rest queue
# behind them and reuse them). Many short-lived connections per second can
# exhaust ephemeral ports (TIME_WAIT) on the load-generator host.

PARALLEL_MAX="${PARALLEL_MAX:-50}"
MAX_INFLIGHT_JOBS="${MAX_INFLIGHT_JOBS:-400}"
SAMPLE_INTERVAL="${SAMPLE_INTERVAL:-15}"
REQ_TIMEOUT="${REQ_TIMEOUT:-10}"

# to_secs 90 | 90s | 10m | 2h  ->  seconds
to_secs() {
  case "$1" in
    *h) echo $((${1%h} * 3600)) ;;
    *m) echo $((${1%m} * 60)) ;;
    *s) echo "${1%s}" ;;
    *) echo "$1" ;;
  esac
}

# rand_between <min> <max> (inclusive; RANDOM is only 15 bits, so combine two)
rand_between() {
  local span=$(($2 - $1 + 1))
  echo $(($1 + (RANDOM * 32768 + RANDOM) % span))
}

engine_init() {
  SCENARIO="$1"
  RUN_ID="$(date +%Y%m%d-%H%M%S)"
  RESULTS="${RESULTS:-$LOADTEST_DIR/results/sh-$SCENARIO-$TIER-$RUN_ID}"
  LOG="$RESULTS/requests.log"
  RAW="$RESULTS/raw"
  STOP_FILE="$RESULTS/.stop"
  mkdir -p "$RAW"
  : >"$RESULTS/schedule.log"; : >"$RESULTS/dropped.log"

  # shellcheck disable=SC2207
  W=($(mix_weights))
  ACC=(0 0 0 0 0 0 0)
  N=(0 0 0 0 0 0 0)
  TRANSFER_SEQ=0
  if ! transfers_enabled && [ "${W[6]}" -gt 0 ]; then
    log "WARN: no transfer pool seeded (run seed/seed.sh) — transfer share goes to GET /vehicle/:id"
    W[2]=$((W[2] + W[6])); W[6]=0
  fi

  require_up
  reset_transfer_pool

  {
    echo "scenario=$SCENARIO tier=$TIER mix=$MIX run=$RUN_ID"
    echo "customer_url=$CUSTOMER_URL vehicle_url=$VEHICLE_URL"
    echo "weights($OPS)=${W[*]}"
    echo "slo: p95<${SLO_P95_MS}ms p99<${SLO_P99_MS}ms errors<${SLO_ERR_PCT}%"
  } >"$RESULTS/meta.txt"
  log "results: $RESULTS"
  trap 'touch "$STOP_FILE"' INT TERM
  start_sampler
}

# --- operations -------------------------------------------------------------

# write_batch <tick> <phase> — curl config for this tick's single-request ops.
write_batch() {
  awk -v tick="$1" -v phase="$2" -v seed="$RANDOM$1" -v run="$RUN_ID" \
      -v cu="$CUSTOMER_URL/api/v1" -v vu="$VEHICLE_URL/api/v1" \
      -v n_cget="${N[0]}" -v n_clist="${N[1]}" -v n_vget="${N[2]}" \
      -v n_create="${N[3]}" -v n_tr="${N[6]}" \
      -v cmin="$CUSTOMER_ID_MIN" -v cmax="$CUSTOMER_ID_MAX" \
      -v vmin="$VEHICLE_ID_MIN" -v vmax="$VEHICLE_ID_MAX" \
      -v tseq="$TRANSFER_SEQ" -v tstart="$TRANSFER_VEHICLE_START" -v tpool="$TRANSFER_POOL" \
      -v ta="$TRANSFER_CUSTOMER_A" -v tb="$TRANSFER_CUSTOMER_B" -v timeout="$REQ_TIMEOUT" '
    function rid(lo, hi) { return lo + int(rand() * (hi - lo + 1)) }
    function start(url, op) {
      if (blocks++) print "next"
      print "url = \"" url "\""
      print "output = \"/dev/null\""
      print "max-time = " timeout
      print "write-out = \"" tick " " phase " " op " %{http_code} %{time_total}\\n\""
    }
    function post(url, op, body) {
      start(url, op)
      print "request = \"POST\""
      print "header = \"Content-Type: application/json\""
      print "data = \"" body "\""
    }
    BEGIN {
      srand(seed)
      for (i = 0; i < n_cget; i++) start(cu "/customer/" rid(cmin, cmax), "cust_get")
      for (i = 0; i < n_clist; i++) {
        # half the listings are the first page (hot, cached), half deep pages
        cursor = rand() < 0.5 ? 0 : rid(cmin, cmax)
        start(cu "/customer?cursor=" cursor "&limit=20", "cust_list")
      }
      for (i = 0; i < n_vget; i++) start(vu "/vehicle/" rid(vmin, vmax), "veh_get")
      for (i = 0; i < n_create; i++)
        post(cu "/customer", "cust_create",
             "{\\\"name\\\":\\\"Load Test\\\",\\\"email\\\":\\\"lt-" run "-" tick "-" i "@example.com\\\",\\\"phone\\\":\\\"+15550000000\\\",\\\"birth_day\\\":\\\"1990-01-01T00:00:00Z\\\"}")
      # Deterministic A<->B schedule: the k-th transfer moves vehicle
      # start+(k mod pool); even passes over the pool go A->B, odd ones B->A.
      # Consecutive uses of one vehicle are `pool` transfers apart, so
      # requests never contend for the same row.
      for (i = 0; i < n_tr; i++) {
        k = tseq + i
        vid = tstart + (k % tpool)
        if (int(k / tpool) % 2 == 0) { from = ta; to = tb } else { from = tb; to = ta }
        post(vu "/transfer", "transfer",
             "{\\\"vehicle_id\\\":" vid ",\\\"from\\\":" from ",\\\"to\\\":" to "}")
      }
    }'
}

# op_update <tick> <phase> — optimistic-locked update: GET for updated_at, then PUT.
op_update() {
  local id out meta body updated
  id=$(rand_between "$LT_CUSTOMER_MIN" "$LT_CUSTOMER_MAX")
  out=$(curl -s -m "$REQ_TIMEOUT" -w '\n%{http_code} %{time_total}' "$CUSTOMER_URL/api/v1/customer/$id")
  meta="${out##*$'\n'}"
  body="${out%$'\n'*}"
  echo "$1 $2 cust_update_get $meta" >>"$RAW/$1.chain"
  [ "${meta%% *}" = "200" ] || return 0
  updated=$(printf '%s' "$body" | sed -n 's/.*"updated_at"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')
  curl -s -m "$REQ_TIMEOUT" -o /dev/null -w "$1 $2 cust_update_put %{http_code} %{time_total}\n" \
    -X PUT -H 'Content-Type: application/json' \
    -d "{\"name\":\"Load Test $id $1\",\"updated_at\":\"$updated\"}" \
    "$CUSTOMER_URL/api/v1/customer/$id" >>"$RAW/$1.chain"
}

# op_create_delete <tick> <phase> <n> — soft-deletes a customer this run just
# created, so seeded rows are never deleted.
op_create_delete() {
  local out meta id
  out=$(curl -s -m "$REQ_TIMEOUT" -w '\n%{http_code} %{time_total}' \
    -X POST -H 'Content-Type: application/json' \
    -d "{\"name\":\"Load Test Delete\",\"email\":\"lt-$RUN_ID-del-$1-$3@example.com\",\"birth_day\":\"1990-01-01T00:00:00Z\"}" \
    "$CUSTOMER_URL/api/v1/customer")
  meta="${out##*$'\n'}"
  echo "$1 $2 cust_delete_create $meta" >>"$RAW/$1.chain"
  [ "${meta%% *}" = "201" ] || return 0
  id=$(printf '%s' "${out%$'\n'*}" | sed -n 's/.*"id"[[:space:]]*:[[:space:]]*\([0-9]*\).*/\1/p')
  [ -n "$id" ] || return 0
  curl -s -m "$REQ_TIMEOUT" -o /dev/null -w "$1 $2 cust_delete %{http_code} %{time_total}\n" \
    -X DELETE "$CUSTOMER_URL/api/v1/customer/$id" >>"$RAW/$1.chain"
}

# fire_tick <tick> <rate> <phase>
fire_tick() {
  local tick="$1" rate="$2" phase="$3" i inflight cfg
  for i in 0 1 2 3 4 5 6; do
    ACC[i]=$((ACC[i] + rate * W[i]))
    N[i]=$((ACC[i] / 100))
    ACC[i]=$((ACC[i] % 100))
  done

  inflight=$(jobs -rp | wc -l | tr -d ' ')
  if [ "$inflight" -ge "$MAX_INFLIGHT_JOBS" ]; then
    echo "$tick $phase $rate $inflight" >>"$RESULTS/dropped.log"
    return 0
  fi

  cfg="$RAW/$tick.cfg"
  write_batch "$tick" "$phase" >"$cfg"
  if [ -s "$cfg" ]; then
    (curl -s -Z --parallel-max "$PARALLEL_MAX" -K "$cfg" >"$RAW/$tick.batch" 2>/dev/null; rm -f "$cfg") &
  else
    rm -f "$cfg"
  fi
  TRANSFER_SEQ=$((TRANSFER_SEQ + N[6]))

  i=0; while [ $i -lt "${N[4]}" ]; do op_update "$tick" "$phase" & i=$((i + 1)); done
  i=0; while [ $i -lt "${N[5]}" ]; do op_create_delete "$tick" "$phase" "$i" & i=$((i + 1)); done
}

# run_stages <start_rate> "<secs>:<target>:<phase>"...
# The rate ramps linearly from the previous stage's target to this one's, like
# k6's ramping-arrival-rate. ON_TICK, if set, is called as `$ON_TICK <tick>`
# after every tick (the stress runner uses it to stop at the SLO break).
run_stages() {
  local prev="$1" t0 tick=0 stage rest secs target phase s rate
  shift
  t0=$(now_ms)
  for stage in "$@"; do
    secs="${stage%%:*}"; rest="${stage#*:}"; target="${rest%%:*}"; phase="${rest#*:}"
    log "phase $phase: ${secs}s, $prev -> $target ops/s"
    s=0
    while [ "$s" -lt "$secs" ]; do
      [ -f "$STOP_FILE" ] && break 2
      rate=$((prev + (target - prev) * (s + 1) / secs))
      fire_tick "$tick" "$rate" "$phase"
      echo "$tick $phase $rate" >>"$RESULTS/schedule.log"
      s=$((s + 1)); tick=$((tick + 1))
      [ -n "${ON_TICK:-}" ] && "$ON_TICK" "$tick"
      sleep_ms $((t0 + tick * 1000 - $(now_ms)))
    done
    prev="$target"
  done
  stop_sampler
  log "waiting for in-flight requests..."
  wait
  find "$RAW" -name '*.batch' -o -name '*.chain' | xargs cat >"$LOG"
  rm -rf "$RAW"
}

# --- runtime metrics sampler -----------------------------------------------

start_sampler() {
  echo "epoch,service,goroutines,heap_alloc_bytes,rss_bytes,open_fds" >"$RESULTS/runtime.csv"
  (
    while [ ! -f "$STOP_FILE" ]; do
      for svc in customer vehicle; do
        [ "$svc" = customer ] && url="$CUSTOMER_URL" || url="$VEHICLE_URL"
        curl -s -m 5 "$url/metrics" | awk -v ts="$(date +%s)" -v svc="$svc" '
          $1 == "go_goroutines" { g = $2 }
          $1 == "go_memstats_heap_alloc_bytes" { h = $2 }
          $1 == "process_resident_memory_bytes" { r = $2 }
          $1 == "process_open_fds" { f = $2 }
          END { if (g != "") printf "%s,%s,%d,%.0f,%.0f,%d\n", ts, svc, g, h, r, f }'
      done >>"$RESULTS/runtime.csv"
      sleep "$SAMPLE_INTERVAL"
    done
  ) &
  SAMPLER_PID=$!
}

stop_sampler() {
  [ -n "${SAMPLER_PID:-}" ] && kill "$SAMPLER_PID" 2>/dev/null
  wait "${SAMPLER_PID:-}" 2>/dev/null
  SAMPLER_PID=""
}

# --- reporting ---------------------------------------------------------------

# Classify a response: ok, an expected miss/conflict, or an error.
AWK_CLASSIFY='
  function klass(op, code) {
    if (op == "cust_get" || op == "veh_get") { if (code == 200) return "ok"; if (code == 404) return "miss" }
    else if (op == "cust_list" || op == "cust_update_get") { if (code == 200) return "ok" }
    else if (op == "cust_create" || op == "cust_delete_create") { if (code == 201) return "ok" }
    else if (op == "cust_update_put" || op == "transfer") { if (code == 204) return "ok"; if (code == 409) return "conflict" }
    else if (op == "cust_delete") { if (code == 204) return "ok" }
    return "error"
  }
  function pct(p) { i = int(p * n + 0.999999); if (i < 1) i = 1; return t[i] }
'

# stats_by <field> <title> <duration_s> [filter-awk-expr]
# field: 2 = phase, 3 = operation, 0 = everything in one row.
stats_by() {
  local field="$1" title="$2" dur="$3" filter="${4:-1}"
  echo
  echo "== $title"
  printf '%-20s %8s %8s %7s %6s %6s %7s %8s %8s %8s %8s\n' \
    group requests rps ok% miss confl err% p50ms p95ms p99ms maxms
  awk "$filter" "$LOG" | awk -v f="$field" '{ print (f == 0 ? "all" : $f), $0 }' |
    sort -k1,1 -k6,6g |
    awk -v dur="$dur" -v f="$field" "$AWK_CLASSIFY"'
      # per-phase rps uses the phase'"'"'s own duration (its ticks in schedule.log)
      FILENAME != "-" { pd[$2]++; next }
      function flush() {
        if (n == 0) return
        d = (f == 2 && pd[g]) ? pd[g] : dur
        printf "%-20s %8d %8.1f %7.2f %6d %6d %7.3f %8.1f %8.1f %8.1f %8.1f\n",
          g, n, n / d, 100 * c["ok"] / n, c["miss"], c["conflict"], 100 * c["error"] / n,
          pct(.50), pct(.95), pct(.99), t[n]
        n = 0; delete c
      }
      $1 != g { flush(); g = $1 }
      { t[++n] = $6 * 1000; c[klass($4, $5)]++ }
      END { flush() }' "$RESULTS/schedule.log" -
}

# window_stats <from_tick> <to_tick> -> "<requests> <p95_ms> <err_pct>"
# Reads the raw per-tick files, so it works while the run is in progress.
window_stats() {
  local t="$1"
  while [ "$t" -lt "$2" ]; do
    cat "$RAW/$t.batch" "$RAW/$t.chain" 2>/dev/null
    t=$((t + 1))
  done | sort -k5,5g |
    awk "$AWK_CLASSIFY"'
      { t[++n] = $5 * 1000; if (klass($3, $4) == "error") e++ }
      END { if (n == 0) print 0, 0, 0; else printf "%d %.1f %.3f\n", n, pct(.95), 100 * e / n }'
}

# report [slo_phases] — prints + saves the report; returns 1 if the SLO is
# breached over the given phases (space-separated; default: all phases).
# Set SLO_ENFORCE=0 to report without failing (stress does).
report() {
  local slo_phases="${1:-}" dur slo_dur filter verdict=0 out="$RESULTS/report.txt"
  dur=$(wc -l <"$RESULTS/schedule.log" | tr -d ' ')
  [ "$dur" -gt 0 ] || dur=1
  filter=1
  if [ -n "$slo_phases" ]; then
    filter="BEGIN { split(\"$slo_phases\", ps, \" \"); for (i in ps) want[ps[i]] = 1 } want[\$2]"
  fi
  slo_dur=$(awk "$filter" "$RESULTS/schedule.log" | wc -l | tr -d ' ')
  [ "$slo_dur" -gt 0 ] || slo_dur=1
  {
    cat "$RESULTS/meta.txt"
    echo "duration=${dur}s scheduled_ops=$(awk '{ s += $3 } END { print s + 0 }' "$RESULTS/schedule.log")" \
         "dropped_ticks=$(wc -l <"$RESULTS/dropped.log" | tr -d ' ')"
    stats_by 3 "per operation" "$dur"
    stats_by 2 "per phase" "$dur"
    stats_by 0 "SLO window (phases: ${slo_phases:-all})" "$slo_dur" "$filter"
    runtime_summary
  } | tee "$out"

  # last line of the SLO table: all requests p50ms p95ms p99ms maxms
  local err p95 p99
  # shellcheck disable=SC2046
  set -- $(stats_by 0 x "$slo_dur" "$filter" | tail -n 1)
  err="${7:-100}"; p95="${9:-0}"; p99="${10:-0}"
  case "${2:-}" in ''|*[!0-9]*) err=100 ;; esac # no requests recorded at all
  echo >>"$out"
  if awk -v e="$err" -v p95="$p95" -v p99="$p99" \
       -v se="$SLO_ERR_PCT" -v s95="$SLO_P95_MS" -v s99="$SLO_P99_MS" \
       'BEGIN { exit !(e < se && p95 < s95 && p99 < s99) }'; then
    echo "SLO: PASS (p95=${p95}ms p99=${p99}ms err=${err}%)" | tee -a "$out"
  else
    echo "SLO: FAIL (p95=${p95}ms p99=${p99}ms err=${err}%; target p95<${SLO_P95_MS} p99<${SLO_P99_MS} err<${SLO_ERR_PCT}%)" | tee -a "$out"
    verdict=1
  fi
  [ "${SLO_ENFORCE:-1}" = "1" ] || verdict=0
  return "$verdict"
}

# runtime_summary — first/last/max of the sampled Go runtime metrics. A
# goroutine count or heap that keeps climbing at constant load is a leak.
runtime_summary() {
  [ "$(wc -l <"$RESULTS/runtime.csv")" -gt 1 ] || return 0
  echo
  echo "== Go runtime (sampled every ${SAMPLE_INTERVAL}s; behind a k8s Service each sample may hit a different pod)"
  printf '%-10s %28s %28s %8s\n' service "goroutines first/last/max" "heap MiB first/last/max" samples
  awk -F, 'NR > 1 {
      s = $2; n[s]++
      if (n[s] == 1) { g0[s] = $3; h0[s] = $4 }
      g1[s] = $3; h1[s] = $4
      if ($3 > gm[s]) gm[s] = $3; if ($4 > hm[s]) hm[s] = $4
    }
    END { for (s in n) printf "%-10s %28s %28s %8d\n", s, g0[s] "/" g1[s] "/" gm[s],
      sprintf("%.1f/%.1f/%.1f", h0[s] / 1048576, h1[s] / 1048576, hm[s] / 1048576), n[s] }' "$RESULTS/runtime.csv"
}
