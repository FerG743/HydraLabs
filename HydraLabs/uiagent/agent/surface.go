package agent

import "context"

// Surface is the agent's hands and eyes — the ONLY thing that touches a real UI.
// A browser (driven over the Chrome DevTools Protocol), a native desktop app
// (driven via OS accessibility APIs), or an in-memory fake all implement this
// same interface, so the agent's brain never depends on how perception and
// action are actually carried out.
type Surface interface {
	Observe(ctx context.Context) (Observation, error)
	Act(ctx context.Context, a Action) error
}

// Planner is the brain's decision step: given the goal, what the surface looks
// like right now, and what has happened so far, choose the next action. The
// Claude-backed planner is one implementation; tests and demos use a
// deterministic one so the loop is verifiable without an API key.
type Planner interface {
	Next(ctx context.Context, goal Goal, obs Observation, history []Step) (Action, error)
}
