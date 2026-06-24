package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/keepdevops/cofiswarm-agent-registry/internal/modes"
	"github.com/keepdevops/cofiswarm-agent-registry/internal/roster"
)

type Server struct {
	store *roster.Store
}

func New(store *roster.Store) *Server { return &Server{store: store} }

func cors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		cors(w)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "engine": "swarm-matrix"})
	})
	mux.HandleFunc("/api/agents", s.handleAgents)
	mux.HandleFunc("/api/modes", func(w http.ResponseWriter, _ *http.Request) {
		cors(w)
		cur := s.store.ActiveMode()
		out := []map[string]any{}
		for _, m := range modes.Catalog {
			out = append(out, map[string]any{
				"name": m.Name, "description": m.Description, "active": m.Name == cur,
			})
		}
		_ = json.NewEncoder(w).Encode(out)
	})
	mux.HandleFunc("/api/modes/active", func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]string{"mode": s.store.ActiveMode()})
		case http.MethodPost:
			var body struct {
				Mode string `json:"mode"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Mode == "" {
				http.Error(w, `{"error":"missing mode"}`, http.StatusBadRequest)
				return
			}
			if !s.store.SetActiveMode(body.Mode) {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error": "unknown mode", "requested": body.Mode, "available": modes.Names(),
				})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"mode": body.Mode})
		default:
			http.Error(w, "GET or POST", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/modes/", s.handleModeSub)
	mux.HandleFunc("/api/agents/", s.handleAgentSub)
	return mux
}

func (s *Server) handleModeSub(w http.ResponseWriter, r *http.Request) {
	cors(w)
	path := strings.TrimPrefix(r.URL.Path, "/api/modes/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[1] != "agents" {
		http.NotFound(w, r)
		return
	}
	modeName := parts[0]
	if !modes.Known(modeName) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "unknown mode", "mode": modeName})
		return
	}
	switch r.Method {
	case http.MethodGet:
		_ = json.NewEncoder(w).Encode(roster.BuildAgentsResponse(s.store, modeName))
	case http.MethodPut:
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := s.store.PutModeAgents(modeName, body); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(roster.BuildAgentsResponse(s.store, modeName))
	default:
		http.Error(w, "GET or PUT", http.StatusMethodNotAllowed)
	}
}

