package roster

import (
	"os"
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	swarm := filepath.Join(dir, "swarm-config.json")
	over := filepath.Join(dir, "overrides.json")
	doc := `{"agents":[{"name":"synthesis","port":8085,"engine":"llama","model":"/m/base.gguf"}],"coordinator":{}}`
	if err := os.WriteFile(swarm, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := New(swarm, over)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return st, over
}

func TestValidateAgent(t *testing.T) {
	if err := ValidateAgent(Agent{Name: "r", Port: 8085, Engine: "llama"}); err != nil {
		t.Errorf("valid agent rejected: %v", err)
	}
	for _, bad := range []Agent{
		{Port: 8085, Engine: "llama"},      // no name
		{Name: "r", Engine: "llama"},       // no port
		{Name: "r", Port: 8085},            // no engine
		{Name: "r", Port: -1, Engine: "l"}, // bad port
	} {
		if err := ValidateAgent(bad); err == nil {
			t.Errorf("expected rejection for %+v", bad)
		}
	}
}

func TestUpsertNewAndReplace(t *testing.T) {
	st, _ := newTestStore(t)

	created, err := st.UpsertAgent(Agent{Name: "reflector", Port: 8085, Engine: "llama", Model: "/m/r.gguf"})
	if err != nil || !created {
		t.Fatalf("upsert new: created=%v err=%v", created, err)
	}
	if a, ok := st.AgentByName("reflector"); !ok || a.Model != "/m/r.gguf" {
		t.Errorf("reflector not registered: %+v ok=%v", a, ok)
	}

	// replacing a base agent returns created=false and the override wins
	created, err = st.UpsertAgent(Agent{Name: "synthesis", Port: 8085, Engine: "llama", Model: "/m/override.gguf"})
	if err != nil || created {
		t.Fatalf("upsert replace base: created=%v err=%v", created, err)
	}
	if a, _ := st.AgentByName("synthesis"); a.Model != "/m/override.gguf" {
		t.Errorf("base override did not win: %+v", a)
	}
	if len(st.Agents()) != 2 {
		t.Errorf("expected 2 effective agents, got %d", len(st.Agents()))
	}

	// invalid upsert is rejected and changes nothing
	if _, err := st.UpsertAgent(Agent{Name: "bad"}); err == nil {
		t.Error("expected validation error on invalid upsert")
	}
}

func TestRemoveBaseAndDynamic(t *testing.T) {
	st, _ := newTestStore(t)
	_, _ = st.UpsertAgent(Agent{Name: "reflector", Port: 8085, Engine: "llama"})

	// remove a base agent -> tombstoned, gone from effective roster
	ok, err := st.RemoveAgent("synthesis")
	if err != nil || !ok {
		t.Fatalf("remove base: ok=%v err=%v", ok, err)
	}
	if _, found := st.AgentByName("synthesis"); found {
		t.Error("synthesis should be removed")
	}
	// remove dynamic agent
	if ok, _ := st.RemoveAgent("reflector"); !ok {
		t.Error("remove dynamic should succeed")
	}
	// removing unknown -> false
	if ok, _ := st.RemoveAgent("ghost"); ok {
		t.Error("removing unknown should report false")
	}
}

func TestOverridesPersistAcrossReload(t *testing.T) {
	st, over := newTestStore(t)
	swarm := st.swarmPath
	_, _ = st.UpsertAgent(Agent{Name: "reflector", Port: 8085, Engine: "llama", Model: "/m/r.gguf"})
	_, _ = st.RemoveAgent("synthesis")

	if _, err := os.Stat(over); err != nil {
		t.Fatalf("overrides file not written: %v", err)
	}

	// fresh store from the same paths must reflect the persisted state
	st2, err := New(swarm, over)
	if err != nil {
		t.Fatal(err)
	}
	if a, ok := st2.AgentByName("reflector"); !ok || a.Model != "/m/r.gguf" {
		t.Errorf("reflector did not survive reload: %+v ok=%v", a, ok)
	}
	if _, ok := st2.AgentByName("synthesis"); ok {
		t.Error("synthesis tombstone did not survive reload")
	}
}

func TestSetPromptPersists(t *testing.T) {
	st, over := newTestStore(t)
	if !st.SetPrompt("synthesis", "new prompt") {
		t.Fatal("SetPrompt on base agent should succeed")
	}
	st2, _ := New(st.swarmPath, over)
	if a, _ := st2.AgentByName("synthesis"); a.SystemPrompt != "new prompt" {
		t.Errorf("prompt override did not persist across reload: %q", a.SystemPrompt)
	}
}
