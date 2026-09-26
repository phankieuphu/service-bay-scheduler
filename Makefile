# make loadtest SCENARIO=<smoke|baseline|load|stress|spike|soak> TIER=<1k|100k|1m> [MIX=read|mixed|write] [TOOL=k6|sh]
# See loadtest/README.md.

SCENARIO ?= smoke
TOOL     ?= k6

.PHONY: loadtest loadtest-seed

loadtest:
	$(MAKE) -C loadtest $(TOOL)-$(SCENARIO)

loadtest-seed:
	$(MAKE) -C loadtest seed
