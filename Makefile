ROLE := agent-registry
.PHONY: build test test-standalone-layout
build:
	go build -o bin/cofiswarm-agent-registry ./cmd/cofiswarm-agent-registry
test: build test-standalone-layout
test-standalone-layout:
	./test/scripts/assert-layout.sh $(ROLE)
