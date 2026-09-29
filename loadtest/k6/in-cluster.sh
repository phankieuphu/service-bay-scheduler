#!/usr/bin/env bash
# Runs a k6 scenario from a pod inside the cluster, so traffic goes through
# the Services' normal load balancing instead of one `kubectl port-forward`
# tunnel. Same scenarios, env vars and results files as k6/run.sh.
#
# Usage: ./k6/in-cluster.sh <smoke|baseline|load|stress|spike|soak> [extra k6 args]
#        ./k6/in-cluster.sh check          # preflight only
#   TIER=1k ./k6/in-cluster.sh load
#   TIER=1k ./k6/in-cluster.sh stress
#
# Extra settings:
#   K8S_NAMESPACE   namespace of the deployment      (service-bay)
#   K6_IMAGE        k6 image                          (grafana/k6:2.3.0)
#   K6_CPU          k6 pod CPU request                (250m, no CPU limit)
#   K6_MEMORY       k6 pod memory request / limit     (256Mi / 1Gi)
#   K6_NODE         pin the k6 pod to this node (default: prefer a node
#                   without customer-/vehicle-service or postgres pods)
#   ALLOW_PENDING=1 start even if some pods are Pending or not Ready
#   KEEP_POD=1      leave the k6 pod behind for debugging
set -uo pipefail

K8S_NAMESPACE="${K8S_NAMESPACE:-service-bay}"
CUSTOMER_URL="${CUSTOMER_URL:-http://customer-service.$K8S_NAMESPACE.svc:8080}"
VEHICLE_URL="${VEHICLE_URL:-http://vehicle-service.$K8S_NAMESPACE.svc:8081}"
PSQL="${PSQL:-kubectl -n $K8S_NAMESPACE exec -i postgres-0 -c postgres -- psql -U postgres}"
export PSQL
. "$(dirname "$0")/../lib/common.sh"
. "$LOADTEST_DIR/lib/k8s.sh"

SCENARIO="${1:-}"
if [ "$SCENARIO" = check ]; then
  k8s_preflight
  exit
fi
[ -f "$LOADTEST_DIR/k6/$SCENARIO.js" ] || die "usage: $0 <check|smoke|baseline|load|stress|spike|soak> [k6 args]"
shift

k8s_preflight || die "preflight failed; fix the cluster or set ALLOW_PENDING=1"
[ "$SCENARIO" = smoke ] || reset_transfer_pool

RESULTS_DIR="${RESULTS_DIR:-$LOADTEST_DIR/results}"
SLO_ERR_RATE="${SLO_ERR_RATE:-$(awk -v p="$SLO_ERR_PCT" 'BEGIN { print p / 100 }')}"
RUN_ID="$(date +%Y%m%d%H%M%S)"
POD="k6-$SCENARIO-$RUN_ID"
mkdir -p "$RESULTS_DIR"

# Scripts go in one ConfigMap. Keys can't contain '/', so lib/x.js is stored
# as lib--x.js and mapped back to lib/x.js by the volume's items.
cm_args=()
items=""
for f in "$LOADTEST_DIR"/k6/*.js "$LOADTEST_DIR"/k6/lib/*.js; do
  rel="${f#"$LOADTEST_DIR"/k6/}"
  key="$(printf '%s' "$rel" | sed 's|/|--|g')"
  cm_args+=(--from-file="$key=$f")
  items="$items
            - { key: \"$key\", path: \"$rel\" }"
done
kube create configmap loadtest-k6 "${cm_args[@]}" --dry-run=client -o yaml \
  | kube apply -f - >/dev/null || die "could not create ConfigMap loadtest-k6"

# Same variables run.sh passes with -e; k6 reads them from the pod env.
env_yaml=""
for var in CUSTOMER_URL VEHICLE_URL TIER MIX \
  CUSTOMER_ID_MIN CUSTOMER_ID_MAX LT_CUSTOMER_MIN LT_CUSTOMER_MAX VEHICLE_ID_MIN VEHICLE_ID_MAX \
  TRANSFER_VEHICLE_START TRANSFER_POOL TRANSFER_CUSTOMER_A TRANSFER_CUSTOMER_B SMOKE_VEHICLE_ID \
  RATE DURATION RAMP MAX_RATE STEPS STEP_DURATION BASE_RATE SPIKE_RATE PEAK_DURATION RECOVERY_DURATION \
  SLO_P95_MS SLO_P99_MS SLO_ERR_RATE PROBE_WARMUP_SAMPLES; do
  eval "val=\${$var:-}"
  # shellcheck disable=SC2154
  [ -n "$val" ] && env_yaml="$env_yaml
        - { name: $var, value: \"$val\" }"
done

args_yaml=""
for a in "$@"; do
  args_yaml="$args_yaml
        - \"$(printf '%s' "$a" | sed 's/\\/\\\\/g; s/"/\\"/g')\""
done

if [ -n "${K6_NODE:-}" ]; then
  placement="nodeName: $K6_NODE"
else
  placement="affinity:
    podAntiAffinity:
      preferredDuringSchedulingIgnoredDuringExecution:
        - weight: 100
          podAffinityTerm:
            topologyKey: kubernetes.io/hostname
            labelSelector:
              matchExpressions:
                - { key: app, operator: In, values: [customer-service, vehicle-service, postgres] }"
fi

cleanup() {
  [ -n "${LOGS_PID:-}" ] && kill "$LOGS_PID" 2>/dev/null
  [ "${KEEP_POD:-0}" = 1 ] || kube delete pod "$POD" --wait=false >/dev/null 2>&1
}
trap cleanup EXIT
trap 'exit 130' INT TERM

# k6 writes its summary files to /results, then the container waits until
# this script has copied them out (it touches /tmp/collected).
kube apply -f - >/dev/null <<EOF || die "could not create pod $POD"
apiVersion: v1
kind: Pod
metadata:
  name: $POD
  labels: { app: k6-loadtest, scenario: $SCENARIO }
spec:
  restartPolicy: Never
  activeDeadlineSeconds: 21600
  $placement
  containers:
    - name: k6
      image: ${K6_IMAGE:-grafana/k6:2.3.0}
      command:
        - sh
        - -c
        - |
          k6 run "\$@" /scripts/$SCENARIO.js
          echo \$? > /tmp/exit-code
          while [ ! -f /tmp/collected ]; do sleep 2; done
        - k6
      args: ${args_yaml:-[]}
      env:
        - { name: RESULTS_DIR, value: /results }
        - { name: RUN_ID, value: "$RUN_ID" }
        - { name: K6_NO_USAGE_REPORT, value: "true" }$env_yaml
      resources:
        requests: { cpu: ${K6_CPU:-250m}, memory: ${K6_MEMORY:-256Mi} }
        limits: { memory: ${K6_MEMORY_LIMIT:-1Gi} }
      volumeMounts:
        - { name: scripts, mountPath: /scripts }
        - { name: results, mountPath: /results }
  volumes:
    - name: scripts
      configMap:
        name: loadtest-k6
        items:$items
    - name: results
      emptyDir: {}
EOF

log "k6 in-cluster $SCENARIO (tier=$TIER mix=$MIX) pod=$POD -> $RESULTS_DIR"
kube wait --for=condition=Ready "pod/$POD" --timeout="${K6_START_TIMEOUT:-180s}" >/dev/null || {
  kube get pod "$POD" -o wide >&2
  kube get events --field-selector "involvedObject.name=$POD" --sort-by=.lastTimestamp >&2
  die "k6 pod did not start"
}
log "k6 pod running on node $(kube get pod "$POD" -o jsonpath='{.spec.nodeName}')"

kube logs -f "$POD" &
LOGS_PID=$!

rc=""
while [ -z "$rc" ]; do
  sleep 5
  phase="$(kube get pod "$POD" -o jsonpath='{.status.phase}' 2>/dev/null)"
  [ "$phase" = Running ] || die "k6 pod is $phase before k6 finished"
  rc="$(kube exec "$POD" -- cat /tmp/exit-code 2>/dev/null)"
done

kube exec "$POD" -- tar cf - -C /results . | tar xf - -C "$RESULTS_DIR" \
  || log "WARN: could not copy results out of $POD"
kube exec "$POD" -- touch /tmp/collected >/dev/null 2>&1
sleep 1
log "k6 exit code $rc; results in $RESULTS_DIR"
exit "$rc"
