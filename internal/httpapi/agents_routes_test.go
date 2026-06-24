package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/keepdevops/cofiswarm-agent-registry/internal/roster"
)

func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	dir := t.TempDir()
	swarm := filepath.Join(dir, "swarm-config.json")
	doc := `{"agents":[{"name":"synthesis","port":8085,"engine":"llama"}],"coordinator":{}}`
	if err := os.WriteFile(swarm, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := roster.New(swarm, filepath.Join(dir, "overrides.json"))
	if err != nil {
		t.Fatal(err)
	}
	return New(store).Handler()
}

func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestPostAgentValidAndInvalid(t *testing.T) {
	h := newTestServer(t)

	rec := do(t, h, http.MethodPost, "/api/agents", `{"name":"reflector","port":8085,"engine":"llama","model":"/m/r.gguf"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("valid POST: want 201, got %d (%s)", rec.Code, rec.Body)
	}
	// it now appears in the listing
	rec = do(t, h, http.MethodGet, "/api/agents", "")
	var list []map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list) != 2 {
		t.Errorf("expected 2 agents after POST, got %d", len(list))
	}

	// missing port -> 400, validated at the boundary
	rec = do(t, h, http.MethodPost, "/api/agents", `{"name":"broken","engine":"llama"}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid POST: want 400, got %d", rec.Code)
	}
}

func TestListProjectionExposesRagTargeting(t *testing.T) {
	h := newTestServer(t)
	// Register an agent that opts into per-agent RAG targeting.
	rec := do(t, h, http.MethodPost, "/api/agents",
		`{"name":"programmer","port":8086,"engine":"llama","use_rag":true,"rag_top_k":7,"rag_kinds":["code"]}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST: want 201, got %d (%s)", rec.Code, rec.Body)
	}
	// The list projection (what cofiswarm-dispatch reads) must surface the RAG fields.
	rec = do(t, h, http.MethodGet, "/api/agents", "")
	var list []map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	var found map[string]any
	for _, a := range list {
		if a["name"] == "programmer" {
			found = a
		}
	}
	if found == nil {
		t.Fatal("programmer not in list")
	}
	if found["use_rag"] != true {
		t.Errorf("use_rag not in list projection: %+v", found)
	}
	if found["rag_top_k"].(float64) != 7 {
		t.Errorf("rag_top_k not in list projection: %+v", found)
	}
	if kinds, ok := found["rag_kinds"].([]any); !ok || len(kinds) != 1 || kinds[0] != "code" {
		t.Errorf("rag_kinds not in list projection: %+v", found)
	}
	// An agent without use_rag must NOT carry the key (omitempty parity).
	for _, a := range list {
		if a["name"] == "synthesis" {
			if _, present := a["use_rag"]; present {
				t.Errorf("use_rag leaked onto non-RAG agent: %+v", a)
			}
		}
	}
}

func TestGetAgentByName(t *testing.T) {
	h := newTestServer(t)
	rec := do(t, h, http.MethodGet, "/api/agents/synthesis", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET by name: want 200, got %d", rec.Code)
	}
	var a roster.Agent
	if err := json.Unmarshal(rec.Body.Bytes(), &a); err != nil || a.Name != "synthesis" || a.Port != 8085 {
		t.Errorf("unexpected agent payload: %s", rec.Body)
	}
	if rec = do(t, h, http.MethodGet, "/api/agents/ghost", ""); rec.Code != http.StatusNotFound {
		t.Errorf("GET unknown: want 404, got %d", rec.Code)
	}
}

func TestDeleteAgent(t *testing.T) {
	h := newTestServer(t)
	_ = do(t, h, http.MethodPost, "/api/agents", `{"name":"reflector","port":8085,"engine":"llama"}`)

	if rec := do(t, h, http.MethodDelete, "/api/agents/reflector", ""); rec.Code != http.StatusOK {
		t.Fatalf("DELETE: want 200, got %d", rec.Code)
	}
	if rec := do(t, h, http.MethodGet, "/api/agents/reflector", ""); rec.Code != http.StatusNotFound {
		t.Errorf("after DELETE, GET should 404, got %d", rec.Code)
	}
	if rec := do(t, h, http.MethodDelete, "/api/agents/ghost", ""); rec.Code != http.StatusNotFound {
		t.Errorf("DELETE unknown: want 404, got %d", rec.Code)
	}
}

func TestPutPromptPersistedFlag(t *testing.T) {
	h := newTestServer(t)
	rec := do(t, h, http.MethodPut, "/api/agents/synthesis/prompt", `{"system_prompt":"hi"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT prompt: want 200, got %d", rec.Code)
	}
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["persisted"] != true {
		t.Errorf("prompt should now report persisted=true, got %v", out["persisted"])
	}
}
