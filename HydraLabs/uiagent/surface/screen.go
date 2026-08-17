//go:build robotgo

// Package surface, robotgo build: captures the real screen and drives the OS
// mouse and keyboard. Build with: go build -tags robotgo ./...
// Requires a C toolchain (Xcode Command Line Tools on macOS) and, on macOS,
// Screen Recording + Accessibility permission for the terminal/binary.
//
// NOTE: this file is compiled on your machine, not in the environment it was
// written in — if robotgo's capture signature differs in your version, adjust
// the Observe body (the API has shifted across releases).
package surface

import (
	"bytes"
	"context"
	"image/png"
	"strings"

	"github.com/go-vgo/robotgo"

	"uiagent/agent"
)

type Screen struct{}

func (Screen) Observe(context.Context) (agent.Observation, error) {
	img := robotgo.CaptureImg() // full primary display as image.Image

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return agent.Observation{}, err
	}
	b := img.Bounds()
	return agent.Observation{
		Title:      "screen",
		Width:      b.Dx(),
		Height:     b.Dy(),
		Screenshot: buf.Bytes(),
	}, nil
}

func (Screen) Act(_ context.Context, a agent.Action) error {
	switch a.Kind {
	case agent.ActClick:
		robotgo.Move(a.X, a.Y)
		robotgo.MilliSleep(60)
		robotgo.Click("left")
	case agent.ActType:
		robotgo.TypeStr(a.Text)
	case agent.ActKey:
		key, mods := keyAndMods(a.Keys)
		if key == "" {
			return nil
		}
		if len(mods) == 0 {
			robotgo.KeyTap(key)
		} else {
			robotgo.KeyTap(key, mods)
		}
	case agent.ActScroll:
		robotgo.Scroll(0, -3)
	case agent.ActDone:
		// nothing to do
	}
	return nil
}

// keyAndMods splits "cmd+s" into key "s" and modifiers ["cmd"] (robotgo wants
// the key first, then modifiers).
func keyAndMods(combo string) (string, []string) {
	parts := strings.Split(combo, "+")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	if len(parts) == 0 || parts[len(parts)-1] == "" {
		return "", nil
	}
	return parts[len(parts)-1], parts[:len(parts)-1]
}
