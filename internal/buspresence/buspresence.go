// Package buspresence publishes event-driven presence for the configured agent roster to
// the observer bus, via cofiswarm-zmq-bridge's HTTP API (no NATS client needed).
//
// This is the diagram's "event-driven, no-heartbeat presence" applied to cofiswarm's
// config-driven roster: on startup the registry announces its agents, and on
// swarm.observer.hello (the middle man restarted) it re-announces. Nothing is polled.
package buspresence

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/keepdevops/cofiswarm-agent-registry/internal/roster"
)

const (
	presenceTopic = "swarm.observer.presence"
	helloTopic    = "swarm.observer.hello"
)

// Publisher emits presence to the bus and re-announces on hello.
type Publisher struct {
	base   string
	client *http.Client
}

// New builds a publisher targeting the bridge base URL (e.g. http://127.0.0.1:5555).
func New(bridgeBase string) *Publisher {
	return &Publisher{
		base:   strings.TrimRight(bridgeBase, "/"),
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

// AnnounceAll publishes an "online" presence event for every agent in the roster.
func (p *Publisher) AnnounceAll(agents []roster.Agent) {
	for _, a := range agents {
		p.publish(presenceTopic, map[string]any{
			"component_id": "agent-" + a.Name,
			"status":       "online",
			"info": map[string]any{
				"name": a.Name, "engine": a.Engine, "model": a.Model,
				"server_group": a.ServerGroup, "port": a.Port, "tags": a.Tags,
			},
		})
	}
	log.Printf("buspresence: announced %d agents", len(agents))
}

func (p *Publisher) publish(topic string, payload map[string]any) {
	body, err := json.Marshal(map[string]any{"topic": topic, "payload": payload})
	if err != nil {
		log.Printf("buspresence: marshal %s: %v", topic, err)
		return
	}
	resp, err := p.client.Post(p.base+"/v1/publish", "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("buspresence: publish %s: %v", topic, err)
		return
	}
	_ = resp.Body.Close()
}

// WatchHello re-announces the roster whenever the middle man broadcasts hello. The agents
// func is read at announce time so a reloaded roster is reflected. Reconnects with backoff.
func (p *Publisher) WatchHello(ctx context.Context, agents func() []roster.Agent) {
	url := p.base + "/v1/subscribe?topic=" + helloTopic
	backoff := time.Second
	for ctx.Err() == nil {
		if err := p.streamHello(ctx, url, agents); err != nil && ctx.Err() == nil {
			log.Printf("buspresence: hello watch error: %v (retry %s)", err, backoff)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func (p *Publisher) streamHello(ctx context.Context, url string, agents func() []roster.Agent) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := (&http.Client{}).Do(req) // no timeout: long-lived SSE
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		if strings.HasPrefix(sc.Text(), "data:") {
			log.Printf("buspresence: hello received -> re-announcing")
			p.AnnounceAll(agents())
		}
	}
	return sc.Err()
}
