# make loadtest SCENARIO=<smoke|baseline|load|stress|spike|soak> TIER=<1k|100k|1m> [MIX=read|mixed|write] [TOOL=k6|sh|k8s]
# See loadtest/README.md.

SCENARIO ?= smoke
TOOL     ?= k6

.PHONY: loadtest loadtest-seed images

# One image at a time: `docker compose build` runs every build in parallel,
# which can run Docker Desktop's VM out of memory.
SERVICES ?= customer-service vehicle-service identity-service scheduler-service billing-service

images:
	@for s in $(SERVICES); do docker compose build $$s || exit 1; done

loadtest:
	$(MAKE) -C loadtest $(TOOL)-$(SCENARIO)

loadtest-seed:
	$(MAKE) -C loadtest seed
