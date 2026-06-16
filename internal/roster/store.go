package roster

import (
	"encoding/json"
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
}

type SwarmDoc struct {
	Agents      []Agent                `json:"agents"`
	Coordinator map[string]any         `json:"coordinator"`
}

type Store struct {
	mu           sync.RWMutex
	swarmPath    string
	overridePath string
	agents       []Agent
	modesConfig  map[string]any
	activeMode   string
}

func New(swarmPath, overridePath string) (*Store, error) {
	s := &Store{swarmPath: swarmPath, overridePath: overridePath, modesConfig: map[string]any{}}
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
	s.agents = doc.Agents
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
				ActiveMode  string         `json:"active_mode"`
				ModesConfig map[string]any `json:"modes_config"`
			}
			if json.Unmarshal(ob, &ov) == nil {
				s.mu.Lock()
				if ov.ActiveMode != "" {
					s.activeMode = ov.ActiveMode
				}
				for k, v := range ov.ModesConfig {
					s.modesConfig[k] = v
				}
				s.mu.Unlock()
			}
		}
	}
	return nil
}

func (s *Store) persistOverrides() error {
	if s.overridePath == "" {
		return nil
	}
	s.mu.RLock()
	payload := map[string]any{
		"active_mode":  s.activeMode,
		"modes_config": s.modesConfig,
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

func (s *Store) SetPrompt(name, prompt string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.agents {
		if s.agents[i].Name == name {
			s.agents[i].SystemPrompt = prompt
			return true
		}
	}
	return false
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
