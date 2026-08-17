package agent

import (
	"context"
	"fmt"
	"hash/fnv"
	"log/slog"
)

// Agent runs the perceive -> plan -> act loop until the planner says done, the
// step budget is exhausted, or the UI stalls. The loop is deliberately simple;
// all the cleverness lives in the Planner and the Surface's observation format.
type Agent struct {
	Surface  Surface
	Planner  Planner
	MaxSteps int
	Logger   *slog.Logger
}

func (ag *Agent) Run(ctx context.Context, goal Goal) (*Transcript, error) {
	log := ag.Logger
	if log == nil {
		log = slog.Default()
	}
	maxSteps := ag.MaxSteps
	if maxSteps <= 0 {
		maxSteps = 25
	}

	tr := &Transcript{}
	var lastFingerprint string
	stall := 0

	for n := 1; n <= maxSteps; n++ {
		obs, err := ag.Surface.Observe(ctx)
		if err != nil {
			return tr, fmt.Errorf("observe: %w", err)
		}

		action, err := ag.Planner.Next(ctx, goal, obs, tr.Steps)
		if err != nil {
			return tr, fmt.Errorf("plan: %w", err)
		}

		tr.Steps = append(tr.Steps, Step{N: n, Action: action, Obs: obs})
		log.Debug("step", "n", n, "kind", action.Kind, "ref", action.Ref, "reason", action.Reason)

		if action.Kind == ActDone {
			tr.Done = true
			return tr, nil
		}

		if err := ag.Surface.Act(ctx, action); err != nil {
			return tr, fmt.Errorf("act %s on %q: %w", action.Kind, action.Ref, err)
		}

		// Stall detection: if the screen the planner sees is identical several
		// steps in a row, the agent is stuck (clicking dead elements, waiting on
		// nothing). Bail rather than burning the whole budget.
		fp := fingerprint(obs)
		if fp == lastFingerprint {
			if stall++; stall >= 3 {
				return tr, fmt.Errorf("stalled: UI unchanged for %d steps", stall)
			}
		} else {
			stall = 0
		}
		lastFingerprint = fp
	}

	return tr, fmt.Errorf("hit step limit (%d) without reaching the goal", maxSteps)
}

// fingerprint hashes the parts of an observation that indicate progress. It
// includes the screenshot bytes so the vision path (which has no structured
// elements) still gets meaningful stall detection.
func fingerprint(o Observation) string {
	h := fnv.New64a()
	fmt.Fprint(h, o.URL, o.Title)
	h.Write(o.Screenshot)
	for _, e := range o.Elements {
		fmt.Fprint(h, e.Ref, e.Role, e.Name, e.Value)
	}
	return fmt.Sprintf("%x", h.Sum64())
}
