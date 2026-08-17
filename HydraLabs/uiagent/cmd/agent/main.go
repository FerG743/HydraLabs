package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"uiagent/agent"
	"uiagent/planner"
	"uiagent/surface"
)

// This is the real agent. Run it on your machine:
//
//	ollama serve                      # in another terminal
//	ollama pull qwen2.5vl:7b          # or qwen3-vl:8b
//	go run -tags robotgo ./cmd/agent -model qwen2.5vl:7b -goal "open the settings and turn on dark mode"
//
// Without -tags robotgo it compiles but the screen surface returns a helpful
// error (so you can confirm the wiring builds before installing the C toolchain).
func main() {
	model := flag.String("model", "qwen2.5vl:7b", "Ollama vision model to use")
	goalText := flag.String("goal", "", "the task for the agent to accomplish")
	maxSteps := flag.Int("max-steps", 25, "maximum actions before giving up")
	flag.Parse()

	if *goalText == "" {
		fmt.Println("usage: provide a goal with -goal \"...\"")
		os.Exit(2)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	ag := &agent.Agent{
		Surface:  surface.Screen{},
		Planner:  planner.NewOllama(*model),
		MaxSteps: *maxSteps,
		Logger:   logger,
	}

	fmt.Printf("MODEL: %s\nGOAL:  %s\n", *model, *goalText)
	fmt.Println("------------------------------------------------------------")

	tr, err := ag.Run(context.Background(), agent.Goal{Task: *goalText})
	for _, s := range tr.Steps {
		target := s.Action.Ref
		if s.Action.Kind == agent.ActClick && s.Action.Ref == "" {
			target = fmt.Sprintf("(%d,%d)", s.Action.X, s.Action.Y)
		}
		if s.Action.Text != "" {
			target += " " + fmt.Sprintf("%q", s.Action.Text)
		}
		if s.Action.Keys != "" {
			target += " " + s.Action.Keys
		}
		fmt.Printf("[%d] %-6s %-16s — %s\n", s.N, s.Action.Kind, target, s.Action.Reason)
	}

	fmt.Println("------------------------------------------------------------")
	if err != nil {
		fmt.Println("result:", err)
		os.Exit(1)
	}
	fmt.Printf("result: done in %d steps\n", len(tr.Steps))
}
