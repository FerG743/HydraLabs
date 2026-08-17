package engine

import "fmt"

// RuleKind enumerates the small, typed set of rules a block can carry. This is
// deliberately not a general expression language yet — these three cover the
// common cases. Promote to a predicate DSL later by adding a new kind; nothing
// else has to change.
type RuleKind string

const (
	// RuleRequiresKey: a key must be present in the RunContext before the
	// block runs. Evaluated at runtime ("buy needs an auth_token").
	RuleRequiresKey RuleKind = "requires_key"

	// RuleKeyEquals: a key must be present and equal a value. Runtime.
	RuleKeyEquals RuleKind = "key_equals"

	// RuleRequiresBlockUpstream: a block of the given type must appear on
	// every path that reaches this node. Evaluated at validation time, not
	// runtime ("there must be a login block before buy").
	RuleRequiresBlockUpstream RuleKind = "requires_block_upstream"
)

// Rule is the author-controlled precondition stored on a node. A block type may
// ship default rules (see BlockMeta) that the editor pre-fills onto new nodes,
// but the engine treats the rules on the node as authoritative — the author can
// edit or delete them.
type Rule struct {
	Kind  RuleKind `json:"kind"`
	Key   string   `json:"key,omitempty"`   // requires_key, key_equals
	Value any      `json:"value,omitempty"` // key_equals
	Block string   `json:"block,omitempty"` // requires_block_upstream (block type)
}

// evalState evaluates a single runtime (state) rule against the context.
// Structural rules return ok=true here — they are checked by the validator,
// not at runtime.
func (r Rule) evalState(run *RunContext) (ok bool, reason string) {
	switch r.Kind {
	case RuleRequiresKey:
		if run.Has(r.Key) {
			return true, ""
		}
		return false, fmt.Sprintf("requires key %q to be set", r.Key)

	case RuleKeyEquals:
		// NOTE: compared as strings to sidestep JSON's float64-vs-int mismatch.
		// A production impl should compare type-aware.
		v, present := run.Get(r.Key)
		if present && fmt.Sprint(v) == fmt.Sprint(r.Value) {
			return true, ""
		}
		return false, fmt.Sprintf("requires key %q to equal %v", r.Key, r.Value)

	default:
		return true, ""
	}
}

// checkState evaluates every state rule on a node. On the first failure it
// returns ok=false with the offending rule and a human-readable reason.
func checkState(rules []Rule, run *RunContext) (ok bool, failed Rule, reason string) {
	for _, r := range rules {
		if passed, why := r.evalState(run); !passed {
			return false, r, why
		}
	}
	return true, Rule{}, ""
}
