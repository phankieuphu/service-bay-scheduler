#!/usr/bin/env bash
# Kubernetes helpers for loadtest/ runners. Sourced after lib/common.sh.
# Bash 3.2 compatible.

K8S_NAMESPACE="${K8S_NAMESPACE:-service-bay}"
KUBECTL="${KUBECTL:-kubectl}"

kube() { $KUBECTL -n "$K8S_NAMESPACE" "$@"; }

# k8s_preflight — refuse to start a load test on a cluster that can't run the
# tier. A Pending pod means the tier's requests don't fit the nodes, so the
# run would measure a smaller deployment than the overlay describes. Also
# warns when HPAs can't read CPU (no metrics-server), because then they never
# scale. Returns 1 on a blocking problem. ALLOW_PENDING=1 downgrades the
# Pending/not-ready failures to warnings.
k8s_preflight() {
  local fail=0 pending notready line

  $KUBECTL version --request-timeout=5s >/dev/null 2>&1 \
    || { log "FAIL: cannot reach the cluster ($($KUBECTL config current-context 2>/dev/null))"; return 1; }
  kube get namespace "$K8S_NAMESPACE" >/dev/null 2>&1 \
    || { log "FAIL: namespace $K8S_NAMESPACE not found; apply an overlay first"; return 1; }

  log "nodes (allocatable):"
  $KUBECTL get nodes --no-headers \
    -o custom-columns='NAME:.metadata.name,CPU:.status.allocatable.cpu,MEMORY:.status.allocatable.memory' \
    | while read -r line; do log "  $line"; done

  # k6 load-test pods are excluded: an old one may still be terminating.
  pending="$(kube get pods -l 'app!=k6-loadtest' --field-selector=status.phase=Pending \
    --no-headers -o custom-columns=NAME:.metadata.name 2>/dev/null)"
  if [ -n "$pending" ]; then
    log "FAIL: pods stuck in Pending (the tier doesn't fit the nodes):"
    for line in $pending; do
      log "  $line: $(kube get events --field-selector "involvedObject.name=$line,reason=FailedScheduling" \
        --sort-by=.lastTimestamp -o jsonpath='{.items[-1:].message}' 2>/dev/null)"
    done
    log "  Give the cluster more CPU/memory (see k8s/README.md, 'Suggested nodes') or use a smaller tier."
    fail=1
  fi

  notready="$(kube get pods -l 'app!=k6-loadtest' --field-selector=status.phase=Running --no-headers \
    -o custom-columns='NAME:.metadata.name,READY:.status.conditions[?(@.type=="Ready")].status' 2>/dev/null \
    | awk '$2 != "True" { print $1 }')"
  if [ -n "$notready" ]; then
    log "FAIL: pods running but not Ready: $(echo $notready)"
    fail=1
  fi

  if ! $KUBECTL get apiservice v1beta1.metrics.k8s.io \
    -o jsonpath='{.status.conditions[?(@.type=="Available")].status}' 2>/dev/null | grep -q True; then
    log "WARN: metrics-server is not available, so the HPAs show <unknown> and never scale."
    log "      Install it: kubectl apply -k k8s/addons/metrics-server"
  fi

  if [ "$fail" = 1 ] && [ "${ALLOW_PENDING:-0}" = 1 ]; then
    log "WARN: continuing anyway (ALLOW_PENDING=1); results do NOT describe this tier."
    fail=0
  fi
  [ "$fail" = 0 ] && log "preflight ok: every pod in $K8S_NAMESPACE is scheduled and Ready"
  return "$fail"
}
