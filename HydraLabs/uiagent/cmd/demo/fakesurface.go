package main

import (
	"context"
	"fmt"
	"strings"

	"uiagent/agent"
)

// fakeSurface is a tiny simulated app: a login screen -> dashboard -> settings.
// It implements agent.Surface, so the real chromedp browser surface is a
// drop-in replacement later — the agent's brain can't tell the difference.
type fakeSurface struct {
	screen   string // "login" | "dashboard" | "settings"
	username string
	password string
	darkMode bool
	saved    bool
}

func newFakeSurface() *fakeSurface { return &fakeSurface{screen: "login"} }

func (s *fakeSurface) Observe(context.Context) (agent.Observation, error) {
	switch s.screen {
	case "login":
		return agent.Observation{
			URL: "app://login", Title: "Sign in",
			Elements: []agent.Element{
				{Ref: "e1", Role: "textbox", Name: "username", Value: s.username},
				{Ref: "e2", Role: "textbox", Name: "password", Value: strings.Repeat("•", len(s.password))},
				{Ref: "e3", Role: "button", Name: "Log in"},
			},
		}, nil
	case "dashboard":
		return agent.Observation{
			URL: "app://dashboard", Title: "Dashboard",
			Elements: []agent.Element{
				{Ref: "e1", Role: "link", Name: "Settings"},
				{Ref: "e2", Role: "link", Name: "Profile"},
			},
		}, nil
	case "settings":
		return agent.Observation{
			URL: "app://settings", Title: "Settings",
			Elements: []agent.Element{
				{Ref: "e1", Role: "checkbox", Name: "Dark mode", Value: onoff(s.darkMode)},
				{Ref: "e2", Role: "button", Name: "Save", Value: savedLabel(s.saved)},
				{Ref: "e3", Role: "link", Name: "Back"},
			},
		}, nil
	}
	return agent.Observation{}, fmt.Errorf("unknown screen %q", s.screen)
}

func (s *fakeSurface) Act(_ context.Context, a agent.Action) error {
	switch s.screen {
	case "login":
		switch {
		case a.Kind == agent.ActType && a.Ref == "e1":
			s.username = a.Text
		case a.Kind == agent.ActType && a.Ref == "e2":
			s.password = a.Text
		case a.Kind == agent.ActClick && a.Ref == "e3":
			if s.username != "" && s.password != "" {
				s.screen = "dashboard"
			}
		default:
			return fmt.Errorf("not a valid action on the login screen")
		}
	case "dashboard":
		if a.Kind == agent.ActClick && a.Ref == "e1" {
			s.screen = "settings"
		} else {
			return fmt.Errorf("not a valid action on the dashboard")
		}
	case "settings":
		switch {
		case a.Kind == agent.ActClick && a.Ref == "e1":
			s.darkMode = !s.darkMode
			s.saved = false
		case a.Kind == agent.ActClick && a.Ref == "e2":
			s.saved = true
		case a.Kind == agent.ActClick && a.Ref == "e3":
			s.screen = "dashboard"
		default:
			return fmt.Errorf("not a valid action on the settings screen")
		}
	}
	return nil
}

func onoff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

func savedLabel(b bool) string {
	if b {
		return "saved"
	}
	return ""
}
