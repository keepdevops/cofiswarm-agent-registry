package roster

func isKnownMode(name string) bool {
	switch name {
	case "flat", "pipeline", "cascade", "router":
		return true
	default:
		return false
	}
}

func BuildAgentsResponse(st *Store, modeName string) map[string]any {
	allNames := st.AgentNames()
	active := map[string]bool{}
	for _, n := range allNames {
		active[n] = true
	}
	cfg, _ := st.ModeConfig(modeName)
	var configured []any
	explicit := false
	if cfg != nil {
		if raw, ok := cfg["agents"].([]any); ok && len(raw) > 0 {
			configured = raw
			explicit = true
		}
	}
	effective := []string{}
	stale := []string{}
	emitted := map[string]bool{}
	if explicit {
		for _, item := range configured {
			n, ok := item.(string)
			if !ok {
				continue
			}
			if active[n] {
				if modeName == "pipeline" || !emitted[n] {
					effective = append(effective, n)
					emitted[n] = true
				}
			} else {
				stale = append(stale, n)
			}
		}
	} else {
		effective = append([]string(nil), allNames...)
	}
	out := map[string]any{
		"mode": modeName, "agents": effective,
		"configured_agents": configured, "stale": stale,
		"explicit": explicit, "available": allNames,
	}
	if cfg != nil {
		for _, key := range []string{"max_select", "synthesizer", "variant_policy", "preset",
			"synthesis_policy", "classifier_policy", "stage_context_chars", "order"} {
			if v, ok := cfg[key]; ok {
				out[key] = v
			}
		}
	}
	return out
}
