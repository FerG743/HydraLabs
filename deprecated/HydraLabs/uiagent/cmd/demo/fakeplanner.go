package main

import (
	"context"
	"strings"

	"uiagent/agent"
)

// fakePlanner stands in for the Claude-backed planner. It reads the current
// observation and picks the obvious next action toward the goal — deterministic,
// so the whole loop is testable without an API key. The real planner will hand
// the same observation + goal + history to Claude and parse the action back; the
// loop around it doesn't change at all.
type fakePlanner struct{}

func (fakePlanner) Next(_ context.Context, goal agent.Goal, obs agent.Observation, _ []agent.Step) (agent.Action, error) {
	wantDark := strings.Contains(strings.ToLower(goal.Task), "dark")

	if u, ok := find(obs, "textbox", "username"); ok && u.Value == "" {
		return typeInto(u.Ref, "alice", "fill in the username"), nil
	}
	if p, ok := find(obs, "textbox", "password"); ok && p.Value == "" {
		return typeInto(p.Ref, "hunter2", "fill in the password"), nil
	}
	if b, ok := find(obs, "button", "Log in"); ok {
		return click(b.Ref, "submit the login form"), nil
	}
	if l, ok := find(obs, "link", "Settings"); ok {
		return click(l.Ref, "navigate to settings"), nil
	}
	if t, ok := find(obs, "checkbox", "Dark mode"); ok && wantDark {
		if t.Value == "off" {
			return click(t.Ref, "enable dark mode"), nil
		}
		if save, ok := find(obs, "button", "Save"); ok {
			if save.Value == "saved" {
				return done("dark mode is on and settings are saved"), nil
			}
			return click(save.Ref, "save the settings"), nil
		}
	}
	return done("nothing left to do for this goal"), nil
}

func find(obs agent.Observation, role, name string) (agent.Element, bool) {
	for _, e := range obs.Elements {
		if e.Role == role && strings.EqualFold(e.Name, name) {
			return e, true
		}
	}
	return agent.Element{}, false
}

func click(ref, reason string) agent.Action {
	return agent.Action{Kind: agent.ActClick, Ref: ref, Reason: reason}
}
func typeInto(ref, text, reason string) agent.Action {
	return agent.Action{Kind: agent.ActType, Ref: ref, Text: text, Reason: reason}
}
func done(reason string) agent.Action {
	return agent.Action{Kind: agent.ActDone, Reason: reason}
}
