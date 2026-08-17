//go:build !robotgo

package surface

import (
	"context"
	"errors"

	"uiagent/agent"
)

// Screen here is a stub so the module builds without a C toolchain. The real,
// screen-driving implementation lives in screen.go behind the `robotgo` build
// tag — build with `-tags robotgo` to get it.
type Screen struct{}

var errNoRobotgo = errors.New(
	"screen surface needs `-tags robotgo` (plus a C toolchain and, on macOS, " +
		"Screen Recording + Accessibility permission)")

func (Screen) Observe(context.Context) (agent.Observation, error) {
	return agent.Observation{}, errNoRobotgo
}

func (Screen) Act(context.Context, agent.Action) error { return errNoRobotgo }
