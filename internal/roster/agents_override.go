package roster

import (
	"fmt"
	"log"
	"sort"
	"strings"
)

// This file adds dynamic agent registration on top of the read-only swarm-config roster: agents can
// be upserted/removed over HTTP and survive restarts via overrides.json. The effective roster is
// always base (swarm-config) + overrides - tombstones, recomputed by rebuildAgentsLocked.

// ValidateAgent enforces the minimum schema for a registerable agent at the API boundary: a name, a
// reachable port, and an engine. Fails loudly so a malformed POST never lands a half-agent.
func ValidateAgent(a Agent) error {
	var missing []string
	if strings.TrimSpace(a.Name) == "" {
		missing = append(missing, "name")
	}
	if a.Port <= 0 {
		missing = append(missing, "port>0")
	}
	if strings.TrimSpace(a.Engine) == "" {
		missing = append(missing, "engine")
	}
	if len(missing) > 0 {
		return fmt.Errorf("invalid agent: missing %s", strings.Join(missing, ", "))
	}
	return nil
}

// rebuildAgentsLocked recomputes the effective roster from base + overrides - tombstones. Caller
// holds s.mu. Overrides replace a same-named base agent in place; brand-new agents are appended;
// the result is name-sorted for a stable /api/agents ordering.
func (s *Store) rebuildAgentsLocked() {
	out := make([]Agent, 0, len(s.baseAgents)+len(s.agentOver))
	seen := map[string]bool{}
	for _, a := range s.baseAgents {
		if s.removedNames[a.Name] {
			continue
		}
		if ov, ok := s.agentOver[a.Name]; ok {
			out = append(out, ov)
		} else {
			out = append(out, a)
		}
		seen[a.Name] = true
	}
	for name, ov := range s.agentOver {
		if seen[name] || s.removedNames[name] {
			continue
		}
		out = append(out, ov)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	s.agents = out
}

// findAgentLocked returns the effective agent (override wins over base) by name. Caller holds s.mu.
func (s *Store) findAgentLocked(name string) (Agent, bool) {
	if ov, ok := s.agentOver[name]; ok && !s.removedNames[name] {
		return ov, true
	}
	for _, a := range s.baseAgents {
		if a.Name == name && !s.removedNames[name] {
			return a, true
		}
	}
	return Agent{}, false
}

// AgentByName returns the effective agent by name (the read side of GET /api/agents/{name}).
func (s *Store) AgentByName(name string) (Agent, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.findAgentLocked(name)
}

// UpsertAgent validates and registers (or replaces) an agent, persisting it. created reports whether
// the agent was newly added vs replacing an existing one. A re-add clears any tombstone.
func (s *Store) UpsertAgent(a Agent) (created bool, err error) {
	if err := ValidateAgent(a); err != nil {
		return false, err
	}
	s.mu.Lock()
	_, existed := s.findAgentLocked(a.Name)
	delete(s.removedNames, a.Name)
	s.agentOver[a.Name] = a
	s.rebuildAgentsLocked()
	s.mu.Unlock()
	if perr := s.persistOverrides(); perr != nil {
		log.Printf("[roster] persist upsert %q failed: %v", a.Name, perr)
		return false, perr
	}
	return !existed, nil
}

// RemoveAgent removes an agent: drops a dynamic override and/or tombstones a base agent so it no
// longer appears. Returns false if no such agent was visible. Persists the change.
func (s *Store) RemoveAgent(name string) (bool, error) {
	s.mu.Lock()
	if _, ok := s.findAgentLocked(name); !ok {
		s.mu.Unlock()
		return false, nil
	}
	delete(s.agentOver, name)
	// Tombstone only matters for base agents; harmless (and idempotent) otherwise.
	for _, a := range s.baseAgents {
		if a.Name == name {
			s.removedNames[name] = true
			break
		}
	}
	s.rebuildAgentsLocked()
	s.mu.Unlock()
	if err := s.persistOverrides(); err != nil {
		log.Printf("[roster] persist remove %q failed: %v", name, err)
		return true, err
	}
	return true, nil
}
