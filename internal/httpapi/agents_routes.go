package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/keepdevops/cofiswarm-agent-registry/internal/roster"
)

// listProjection is the historical /api/agents element shape (a routing-relevant subset). Kept
// stable for existing consumers; GET /api/agents/{name} returns the full record instead.
func listProjection(a roster.Agent) map[string]any {
	obj := map[string]any{"name": a.Name, "port": a.Port, "engine": a.Engine}
	if a.Description != "" {
		obj["description"] = a.Description
	}
	if a.Backend != "" {
		obj["backend"] = a.Backend
	}
	if a.Model != "" {
		obj["model"] = a.Model
	}
	if a.DraftModel != "" {
		obj["draft_model"] = a.DraftModel
	}
	if a.DraftMax > 0 {
		obj["draft_max"] = a.DraftMax
	}
	if a.InferenceBackend != "" {
		obj["inference_backend"] = a.InferenceBackend
	}
	// Per-agent RAG targeting — surfaced in the list so cofiswarm-dispatch can resolve
	// which agents opt into RAG without a full-record fetch per agent.
	if a.UseRAG {
		obj["use_rag"] = true
	}
	if a.RagTopK > 0 {
		obj["rag_top_k"] = a.RagTopK
	}
	if len(a.RagKinds) > 0 {
		obj["rag_kinds"] = a.RagKinds
	}
	return obj
}

// handleAgents serves the roster collection: GET lists (subset projection), POST upserts a fully
// specified agent (schema-validated; persisted) — the dynamic-registration entry point.
func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	cors(w)
	switch r.Method {
	case http.MethodGet:
		list := []map[string]any{}
		for _, a := range s.store.Agents() {
			list = append(list, listProjection(a))
		}
		_ = json.NewEncoder(w).Encode(list)
	case http.MethodPost:
		var a roster.Agent
		if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON: " + err.Error()})
			return
		}
		created, err := s.store.UpsertAgent(a)
		if err != nil {
			log.Printf("[agents] reject POST %q: %v", a.Name, err)
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		if created {
			w.WriteHeader(http.StatusCreated)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"agent": a, "created": created})
	default:
		http.Error(w, "GET or POST", http.StatusMethodNotAllowed)
	}
}

// handleAgentSub serves /api/agents/{name}[/prompt]:
//   GET    /api/agents/{name}         -> full agent record (404 if unknown)
//   DELETE /api/agents/{name}         -> remove agent (404 if unknown)
//   PUT    /api/agents/{name}/prompt  -> override + persist the system prompt
func (s *Server) handleAgentSub(w http.ResponseWriter, r *http.Request) {
	cors(w)
	path := strings.TrimPrefix(r.URL.Path, "/api/agents/")
	parts := strings.Split(path, "/")
	name := parts[0]
	if name == "" {
		http.NotFound(w, r)
		return
	}

	// /api/agents/{name}/prompt
	if len(parts) == 2 && parts[1] == "prompt" {
		s.handleAgentPrompt(w, r, name)
		return
	}
	if len(parts) != 1 {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		a, ok := s.store.AgentByName(name)
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": "unknown agent", "name": name})
			return
		}
		_ = json.NewEncoder(w).Encode(a)
	case http.MethodDelete:
		removed, err := s.store.RemoveAgent(name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !removed {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": "unknown agent", "name": name})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"name": name, "removed": true})
	default:
		http.Error(w, "GET or DELETE", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAgentPrompt(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodPut {
		http.Error(w, "PUT only", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		SystemPrompt string `json:"system_prompt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.SystemPrompt == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "missing 'system_prompt' string"})
		return
	}
	if !s.store.SetPrompt(name, body.SystemPrompt) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "unknown agent", "name": name})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"name": name, "system_prompt": body.SystemPrompt, "persisted": true,
	})
}
