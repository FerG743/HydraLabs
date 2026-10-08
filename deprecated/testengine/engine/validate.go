package engine

import (
	"fmt"
	"sort"
)

// Validate runs design-time checks so the canvas can surface problems before a
// run starts: known block types, well-formed edges, a real start node, and the
// structural rules (requires_block_upstream). It returns every problem found.
func (g *Graph) Validate(reg *Registry) []error {
	var errs []error

	ids := make(map[string]bool, len(g.Nodes))
	for _, n := range g.Nodes {
		ids[n.ID] = true
		if _, ok := reg.Meta(n.Type); !ok {
			errs = append(errs, fmt.Errorf("node %q has unknown block type %q", n.ID, n.Type))
		}
	}
	if !ids[g.Start] {
		errs = append(errs, fmt.Errorf("start node %q does not exist", g.Start))
	}
	for _, e := range g.Edges {
		if !ids[e.From] {
			errs = append(errs, fmt.Errorf("edge references missing source node %q", e.From))
		}
		if !ids[e.To] {
			errs = append(errs, fmt.Errorf("edge references missing target node %q", e.To))
		}
	}
	if len(errs) > 0 {
		// Structural analysis assumes a well-formed graph; stop here.
		return errs
	}

	// requires_block_upstream is satisfied iff the required type is guaranteed
	// to have run on EVERY path reaching the node.
	mustPass := g.mustPassTypes()
	for _, n := range g.Nodes {
		for _, r := range n.Rules {
			if r.Kind != RuleRequiresBlockUpstream {
				continue
			}
			if !mustPass[n.ID][r.Block] {
				errs = append(errs, fmt.Errorf(
					"node %q requires a %q block upstream on every path, but one is missing",
					n.ID, r.Block))
			}
		}
	}
	return errs
}

// mustPassTypes computes, for each node, the set of block types guaranteed to
// have run on every path reaching it. Assumes a DAG — loops in this engine are
// container blocks (their own subgraph), not back-edges, so the top-level graph
// has no cycles.
func (g *Graph) mustPassTypes() map[string]map[string]bool {
	typeOf := make(map[string]string, len(g.Nodes))
	indeg := make(map[string]int, len(g.Nodes))
	for _, n := range g.Nodes {
		typeOf[n.ID] = n.Type
		indeg[n.ID] = 0
	}
	preds := make(map[string][]string)
	succ := make(map[string][]string)
	for _, e := range g.Edges {
		preds[e.To] = append(preds[e.To], e.From)
		succ[e.From] = append(succ[e.From], e.To)
		indeg[e.To]++
	}

	var queue []string
	for id, d := range indeg {
		if d == 0 {
			queue = append(queue, id)
		}
	}
	sort.Strings(queue)

	result := make(map[string]map[string]bool, len(g.Nodes))
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]

		set := map[string]bool{}
		if ps := preds[id]; len(ps) > 0 {
			first := true
			for _, p := range ps {
				pset := map[string]bool{typeOf[p]: true}
				for t := range result[p] {
					pset[t] = true
				}
				if first {
					set, first = pset, false
					continue
				}
				for t := range set {
					if !pset[t] {
						delete(set, t)
					}
				}
			}
		}
		result[id] = set

		var newly []string
		for _, s := range succ[id] {
			indeg[s]--
			if indeg[s] == 0 {
				newly = append(newly, s)
			}
		}
		sort.Strings(newly)
		queue = append(queue, newly...)
	}
	return result
}
