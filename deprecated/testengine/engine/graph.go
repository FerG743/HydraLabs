package engine

import "encoding/json"

// Node is one block instance in a test. Config is the block-specific settings
// blob (unmarshaled by the block's factory). Rules are the author-controlled
// preconditions for this node.
type Node struct {
	ID     string          `json:"id"`
	Type   string          `json:"type"`
	Config json.RawMessage `json:"config,omitempty"`
	Rules  []Rule          `json:"rules,omitempty"`
}

// Edge connects two nodes and fires on a given status. An empty On is the
// default edge — it matches any status not claimed by a more specific edge,
// which keeps linear (non-branching) flows to a single edge per node.
type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
	On   Status `json:"on,omitempty"`
}

// Graph is the whole test: a start node plus nodes and edges. This is exactly
// the JSON shape the canvas emits and the engine consumes.
type Graph struct {
	Start string `json:"start"`
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

func LoadGraph(data []byte) (*Graph, error) {
	var g Graph
	if err := json.Unmarshal(data, &g); err != nil {
		return nil, err
	}
	return &g, nil
}

func (g *Graph) node(id string) (Node, bool) {
	for _, n := range g.Nodes {
		if n.ID == id {
			return n, true
		}
	}
	return Node{}, false
}

// next returns the destination for the edge leaving `from` that matches
// `status`. An exact status match wins; otherwise the default (empty On) edge
// is used. Returns ok=false when no edge applies, which ends the path.
func (g *Graph) next(from string, status Status) (string, bool) {
	fallback, haveFallback := "", false
	for _, e := range g.Edges {
		if e.From != from {
			continue
		}
		if e.On == status {
			return e.To, true
		}
		if e.On == "" {
			fallback, haveFallback = e.To, true
		}
	}
	return fallback, haveFallback
}
