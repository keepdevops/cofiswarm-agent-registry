package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/keepdevops/cofiswarm-agent-registry/internal/httpapi"
	"github.com/keepdevops/cofiswarm-agent-registry/internal/roster"
)

func main() {
	addr := flag.String("listen", ":8012", "listen address")
	swarm := flag.String("swarm-config", "", "swarm-config.json path")
	overrides := flag.String("state", "", "modes override json path")
	flag.Parse()
	if *swarm == "" {
		if v := os.Getenv("COFISWARM_SWARM_CONFIG"); v != "" {
			*swarm = v
		} else {
			*swarm = "/etc/cofiswarm/config/swarm-config.json"
		}
	}
	if *overrides == "" {
		if v := os.Getenv("COFISWARM_AGENT_REGISTRY_STATE"); v != "" {
			*overrides = v
		} else {
			*overrides = "/var/lib/cofiswarm/agent-registry/overrides.json"
		}
	}
	store, err := roster.New(*swarm, *overrides)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("agent-registry listening on %s (%d agents)", *addr, len(store.Agents()))
	log.Fatal(http.ListenAndServe(*addr, httpapi.New(store).Handler()))
}
