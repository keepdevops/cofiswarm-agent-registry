package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/keepdevops/cofiswarm-agent-registry/internal/httpapi"
	"github.com/keepdevops/cofiswarm-agent-registry/internal/roster"
	"github.com/keepdevops/cofiswarm-observer-sdk/pkg/buspresence"
)

// agentMembers maps the roster to shared-SDK presence members (component_id "agent-<name>",
// mirroring the fields the old internal AnnounceAll published).
func agentMembers(agents []roster.Agent) []buspresence.Member {
	ms := make([]buspresence.Member, 0, len(agents))
	for _, a := range agents {
		ms = append(ms, buspresence.Member{
			ID: "agent-" + a.Name,
			Info: map[string]any{
				"name": a.Name, "engine": a.Engine, "model": a.Model,
				"server_group": a.ServerGroup, "port": a.Port, "tags": a.Tags,
			},
		})
	}
	return ms
}

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
	// Event-driven presence on the bus (default-off). When COFISWARM_BRIDGE_URL is set,
	// announce the roster now and re-announce whenever the middle man broadcasts hello.
	var pub *buspresence.Publisher
	stopWatch := func() {}
	if base := os.Getenv("COFISWARM_BRIDGE_URL"); base != "" {
		pub = buspresence.New(base)
		announce := func() { pub.AnnounceMembers(agentMembers(store.Agents())) }
		announce()
		var wctx context.Context
		wctx, stopWatch = context.WithCancel(context.Background())
		go pub.WatchHello(wctx, announce)
		log.Printf("agent-registry publishing presence to bus via %s", base)
	}

	srv := &http.Server{Addr: *addr, Handler: httpapi.New(store).Handler()}
	go func() {
		log.Printf("agent-registry listening on %s (%d agents)", *addr, len(store.Agents()))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("agent-registry: server error: %v", err)
		}
	}()

	// On SIGINT/SIGTERM: stop re-announcing, say goodbye for every agent (flip offline now,
	// not after the observer's TTL), then drain in-flight requests before exiting.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	log.Printf("agent-registry: shutting down")
	stopWatch()
	if pub != nil {
		pub.GoodbyeMembers(agentMembers(store.Agents()))
	}
	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Printf("agent-registry: graceful shutdown: %v", err)
	}
}
