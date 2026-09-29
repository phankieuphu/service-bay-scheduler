# Load tests for customer-service / vehicle-service (issue #38).
#
#   make seed                          # seed test data (once per database)
#   make smoke                         # sh + k6 functional checks of every API case
#   make k6-load TIER=100k MIX=mixed   # k6 scenario
#   make sh-spike TIER=1k              # curl-based scenario
#   make k8s-check                     # cluster preflight (no Pending pods, metrics-server)
#   make k8s-load TIER=1k              # k6 from a pod inside the cluster
#
# Scenarios: baseline load stress spike soak. See README.md.

TIER ?= 1k
MIX  ?= mixed
export TIER MIX

SCENARIOS := baseline load stress spike soak

.PHONY: seed reset clean smoke sh-smoke k6-smoke k8s-check k8s-smoke $(addprefix sh-,$(SCENARIOS)) $(addprefix k6-,$(SCENARIOS)) $(addprefix k8s-,$(SCENARIOS))

seed:
	./seed/seed.sh

reset:
	./seed/seed.sh reset

clean:
	./seed/seed.sh clean

smoke: sh-smoke k6-smoke

sh-smoke:
	./sh/smoke.sh

k6-smoke:
	./k6/run.sh smoke

$(addprefix sh-,$(SCENARIOS)): sh-%:
	./sh/$*.sh

$(addprefix k6-,$(SCENARIOS)): k6-%:
	./k6/run.sh $*

k8s-check:
	./k6/in-cluster.sh check

k8s-smoke:
	./k6/in-cluster.sh smoke

$(addprefix k8s-,$(SCENARIOS)): k8s-%:
	./k6/in-cluster.sh $*
