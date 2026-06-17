package buspresence

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/keepdevops/cofiswarm-agent-registry/internal/roster"
)

func TestAnnounceAllPublishesPresence(t *testing.T) {
	var bodies [][]byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/publish" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, b)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer ts.Close()

	New(ts.URL).AnnounceAll([]roster.Agent{{Name: "architect", Engine: "llama", Model: "m.gguf"}})

	if len(bodies) != 1 {
		t.Fatalf("got %d publishes, want 1", len(bodies))
	}
	var msg struct {
		Topic   string `json:"topic"`
		Payload struct {
			ComponentID string `json:"component_id"`
			Status      string `json:"status"`
			Info        struct {
				Name   string `json:"name"`
				Engine string `json:"engine"`
			} `json:"info"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(bodies[0], &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Topic != "swarm.observer.presence" {
		t.Errorf("topic = %q", msg.Topic)
	}
	if msg.Payload.ComponentID != "agent-architect" || msg.Payload.Status != "online" {
		t.Errorf("payload = %+v", msg.Payload)
	}
	if msg.Payload.Info.Name != "architect" || msg.Payload.Info.Engine != "llama" {
		t.Errorf("info = %+v", msg.Payload.Info)
	}
}
