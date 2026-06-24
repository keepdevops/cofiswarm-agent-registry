package roster

import (
	"encoding/json"
	"log"
	"os"
	"sync"
)

type Agent struct {
	Name              string   `json:"name"`
	Port              int      `json:"port"`
	Engine            string   `json:"engine"`
	Description       string   `json:"description,omitempty"`
	Backend           string   `json:"backend,omitempty"`
	Model             string   `json:"model,omitempty"`
	DraftModel        string   `json:"draft_model,omitempty"`
	DraftMax          int      `json:"draft_max,omitempty"`
	InferenceBackend  string   `json:"inference_backend,omitempty"`
	SystemPrompt      string   `json:"system_prompt,omitempty"`
	ServerGroup       string   `json:"server_group,omitempty"`
	MaxTokens         int      `json:"max_tokens,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	// Long-context + KV-memory tuning (consumed by cofiswarm-launcher at spawn).
	Context       int      `json:"context,omitempty"`
	CtxCap        int      `json:"ctx_cap,omitempty"`
	FlashAttn     bool     `json:"flash_attn,omitempty"`
	ExtraArgs     []string `json:"extra_args,omitempty"`
	KVCacheType   string   `json:"kv_cache_type,omitempty"`
	RopeScaling   string   `json:"rope_scaling,omitempty"`
	RopeFreqBase  float64  `json:"rope_freq_base,omitempty"`
	RopeFreqScale float64  `json:"rope_freq_scale,omitempty"`
	YarnOrigCtx   int      `json:"yarn_orig_ctx,omitempty"`
	YarnExtFactor float64  `json:"yarn_ext_factor,omitempty"`
	TurboQuant    bool     `json:"turbo_quant,omitempty"`
	// Per-agent RAG targeting (carried verbatim from swarm-config; consumed by
	// cofiswarm-dispatch's prepare step to inject retrieved context for this agent).
	UseRAG   bool     `json:"use_rag,omitempty"`
	RagTopK  int      `json:"rag_top_k,omitempty"`
	RagKinds []string `json:"rag_kinds,omitempty"`
}

type SwarmDoc struct {
	Agents      []Agent                `json:"agents"`
	Coordinator map[string]any         `json:"coordinator"`
}

type Store struct {
	mu           sync.RWMutex
	swarmPath    string
	overridePath string
	baseAgents   []Agent          // roster from swarm-config.json (read-only source)
	agentOver    map[string]Agent // dynamic upserts/edits, persisted to overrides.json
	removedNames map[string]bool  // tombstones hiding base agents, persisted
	agents       []Agent          // effective roster = base + overrides - removed
	modesConfig  map[string]any
	activeMode   string
}

func New(swarmPath, overridePath string) (*Store, error) {
	s := &Store{
		swarmPath: swarmPath, overridePath: overridePath,
		modesConfig: map[string]any{}, agentOver: map[string]Agent{}, removedNames: map[string]bool{},
	}
	return s, s.Reload()
}

func (s *Store) Reload() error {
	b, err := os.ReadFile(s.swarmPath)
	if err != nil {
		return err
	}
	var doc SwarmDoc
	if err := json.Unmarshal(b, &doc); err != nil {
		return err
	}
	s.mu.Lock()
	s.baseAgents = doc.Agents
	if doc.Coordinator != nil {
		if m, ok := doc.Coordinator["modes"].(map[string]any); ok {
			s.modesConfig = m
		}
		if dm, ok := doc.Coordinator["default_mode"].(string); ok {
			s.activeMode = dm
		}
	}
	s.mu.Unlock()
	if s.overridePath != "" {
		if ob, err := os.ReadFile(s.overridePath); err == nil {
			var ov struct {
				ActiveMode    string           `json:"active_mode"`
				ModesConfig   map[string]any   `json:"modes_config"`
				Agents        map[string]Agent `json:"agents"`
				RemovedAgents []string         `json:"removed_agents"`
			}
			if json.Unmarshal(ob, &ov) == nil {
				s.mu.Lock()
				if ov.ActiveMode != "" {
					s.activeMode = ov.ActiveMode
				}
				for k, v := range ov.ModesConfig {
					s.modesConfig[k] = v
				}
				if ov.Agents != nil {
					s.agentOver = ov.Agents
				}
				s.removedNames = map[string]bool{}
				for _, n := range ov.RemovedAgents {
					s.removedNames[n] = true
				}
				s.mu.Unlock()
			}
		}
	}
	s.mu.Lock()
	s.rebuildAgentsLocked()
	s.mu.Unlock()
	return nil
}

func (s *Store) persistOverrides() error {
	if s.overridePath == "" {
		return nil
	}
	s.mu.RLock()
	removed := make([]string, 0, len(s.removedNames))
	for n := range s.removedNames {
		removed = append(removed, n)
	}
	payload := map[string]any{
		"active_mode":    s.activeMode,
		"modes_config":   s.modesConfig,
		"agents":         s.agentOver,
		"removed_agents": removed,
	}
	s.mu.RUnlock()
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dirOf(s.overridePath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(s.overridePath, b, 0o644)
}

func dirOf(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			return p[:i]
		}
	}
	return "."
}

func (s *Store) Agents() []Agent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Agent, len(s.agents))
	copy(out, s.agents)
	return out
}

func (s *Store) AgentNames() []string {
	agents := s.Agents()
	out := make([]string, len(agents))
	for i, a := range agents {
		out[i] = a.Name
	}
	return out
}


func (s *Store) HasAgent(name string) bool {
	for _, a := range s.Agents() {
		if a.Name == name {
			return true
		}
	}
	return false
}

// SetPrompt overrides an agent's system prompt and PERSISTS it (closes the prior in-memory-only
// gap). Works for base agents (creates an override carrying the new prompt) and dynamic agents.
func (s *Store) SetPrompt(name, prompt string) bool {
	s.mu.Lock()
	cur, ok := s.findAgentLocked(name)
	if !ok {
		s.mu.Unlock()
		return false
	}
	cur.SystemPrompt = prompt
	s.agentOver[name] = cur
	s.rebuildAgentsLocked()
	s.mu.Unlock()
	if err := s.persistOverrides(); err != nil {
		log.Printf("[roster] persist prompt override for %q failed: %v", name, err)
	}
	return true
}

func (s *Store) ActiveMode() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.activeMode
}

func (s *Store) SetActiveMode(name string) bool {
	if !isKnownMode(name) {
		return false
	}
	s.mu.Lock()
	s.activeMode = name
	s.mu.Unlock()
	_ = s.persistOverrides()
	return true
}

func (s *Store) ModesConfig() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]any, len(s.modesConfig))
	for k, v := range s.modesConfig {
		out[k] = v
	}
	return out
}

func (s *Store) ModeConfig(name string) (map[string]any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.modesConfig[name]
	if !ok {
		return nil, false
	}
	m, ok := v.(map[string]any)
	return m, ok
}

func (s *Store) PutModeAgents(modeName string, body map[string]any) error {
	s.mu.Lock()
	if s.modesConfig[modeName] == nil {
		s.modesConfig[modeName] = map[string]any{}
	}
	cfg, _ := s.modesConfig[modeName].(map[string]any)
	for _, key := range []string{"agents", "max_select", "synthesizer", "order"} {
		if v, ok := body[key]; ok {
			cfg[key] = v
		}
	}
	s.modesConfig[modeName] = cfg
	s.mu.Unlock()
	return s.persistOverrides()
}
