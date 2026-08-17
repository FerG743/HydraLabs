package engine

import (
	"context"
	"encoding/json"
)

// Block is the one interface every block implements, regardless of whether it
// is an HTTP request, an assertion, a chaos fault, a load container, or an AI
// agent. The interpreter calls Execute and branches on the returned Status.
type Block interface {
	Execute(ctx context.Context, run *RunContext) (Status, error)
}

// BlockMeta describes a block *type* for the canvas. Reads/Writes declare which
// context keys the block touches, so the editor can draw connections and warn
// when a block reads a key that nothing upstream writes (soft lint).
// DefaultRules are pre-filled onto new instances of this type; the author owns
// them after that.
type BlockMeta struct {
	Type         string   `json:"type"`
	Reads        []string `json:"reads,omitempty"`
	Writes       []string `json:"writes,omitempty"`
	DefaultRules []Rule   `json:"defaultRules,omitempty"`
}

// Factory builds a configured Block instance from a node's raw JSON config.
type Factory struct {
	Meta BlockMeta
	New  func(config json.RawMessage) (Block, error)
}
