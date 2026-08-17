package engine

import (
	"context"
	"fmt"
)

// StepResult records what happened at one node.
type StepResult struct {
	NodeID string
	Type   string
	Status Status
	Reason string // populated when Status == StatusBlocked
	Err    error
}

type RunResult struct {
	Steps []StepResult
}

// Interpreter walks a graph for one RunContext. It is intentionally dumb: its
// only job is "evaluate rules, run the block (or not), read the status, follow
// the matching edge." All cleverness lives in blocks and rules.
type Interpreter struct {
	reg *Registry
}

func NewInterpreter(reg *Registry) *Interpreter {
	return &Interpreter{reg: reg}
}

// Run executes the graph from Start. For each node it evaluates the node's
// state rules FIRST: if any fail, the block does not run and the interpreter
// emits StatusBlocked. Otherwise it builds and runs the block. Either way, the
// resulting status selects the next edge — so a blocked "buy" flows down
// whatever edge you wired for it, exactly like a failed assert.
func (in *Interpreter) Run(ctx context.Context, g *Graph, run *RunContext) (*RunResult, error) {
	res := &RunResult{}
	current := g.Start

	for current != "" {
		node, ok := g.node(current)
		if !ok {
			return res, fmt.Errorf("graph references missing node %q", current)
		}

		step := StepResult{NodeID: node.ID, Type: node.Type}

		if passed, failed, reason := checkState(node.Rules, run); !passed {
			step.Status = StatusBlocked
			step.Reason = reason
			run.Logger.Warn("blocked by rule",
				"node", node.ID, "rule", failed.Kind, "reason", reason)
		} else {
			block, err := in.reg.build(node)
			if err != nil {
				step.Err = err
				res.Steps = append(res.Steps, step)
				return res, err
			}
			status, err := block.Execute(ctx, run)
			step.Status, step.Err = status, err
			if err != nil {
				res.Steps = append(res.Steps, step)
				return res, err
			}
		}

		res.Steps = append(res.Steps, step)

		nextID, ok := g.next(node.ID, step.Status)
		if !ok {
			break
		}
		current = nextID
	}

	return res, nil
}
