package modes

type Mode struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

var Catalog = []Mode{
	{Name: "flat", Description: "Broadcast the prompt to every agent in parallel; no reducer."},
	{Name: "pipeline", Description: "Sequential chain — each agent receives the previous agent's output."},
	{Name: "cascade", Description: "Mixture-of-agents — parallel broadcast then a synthesizer reduces all responses into one final answer."},
	{Name: "router", Description: "Classifier agent picks a subset; prompt is broadcast to those agents only."},
}

func Known(name string) bool {
	for _, m := range Catalog {
		if m.Name == name {
			return true
		}
	}
	return false
}

func Names() []string {
	out := make([]string, len(Catalog))
	for i, m := range Catalog {
		out[i] = m.Name
	}
	return out
}
