ROLE := agent-registry
.PHONY: build test test-standalone-layout sync-agents check-agents
build:
	go build -o bin/cofiswarm-agent-registry ./cmd/cofiswarm-agent-registry
test: build test-standalone-layout check-agents
test-standalone-layout:
	./test/scripts/assert-layout.sh $(ROLE)
# Sync the data/agents/ mirror from the cofiswarm-config SoT (config/agents/).
sync-agents:
	./scripts/sync-agents.sh
# Fail when the mirror drifts from the SoT (skips cleanly if the SoT isn't present).
check-agents:
	./scripts/sync-agents.sh --check
