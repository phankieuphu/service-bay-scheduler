# Load tests for customer-service / vehicle-service (issue #38).
#
#   make seed                          # seed test data (once per database)
#   make smoke                         # sh + k6 functional checks of every API case
#   make k6-load TIER=100k MIX=mixed   # k6 scenario
#   make sh-spike TIER=1k              # curl-based scenario
#
# Scenarios: baseline load stress spike soak. See README.md.

TIER ?= 1k
MIX  ?= mixed
export TIER MIX

SCENARIOS := baseline load stress spike soak

.PHONY: seed reset clean smoke sh-smoke k6-smoke $(addprefix sh-,$(SCENARIOS)) $(addprefix k6-,$(SCENARIOS))

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
