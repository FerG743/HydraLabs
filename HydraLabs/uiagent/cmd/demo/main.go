package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"uiagent/agent"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ag := &agent.Agent{
		Surface:  newFakeSurface(),
		Planner:  fakePlanner{},
		MaxSteps: 25,
		Logger:   logger,
	}

	goal := agent.Goal{Task: "Log in, enable dark mode, and save the settings"}
	fmt.Println("GOAL:", goal.Task)
	fmt.Println("------------------------------------------------------------")

	tr, err := ag.Run(context.Background(), goal)
	printTranscript(tr)

	fmt.Println("------------------------------------------------------------")
	if err != nil {
		fmt.Println("result: FAILED —", err)
		os.Exit(1)
	}
	fmt.Printf("result: %s in %d steps\n", outcome(tr.Done), len(tr.Steps))
}

func printTranscript(tr *agent.Transcript) {
	for _, s := range tr.Steps {
		target := s.Action.Ref
		if s.Action.Text != "" {
			target += " \"" + s.Action.Text + "\""
		}
		fmt.Printf("[%d] %-11s  %-5s %-16s — %s\n",
			s.N, "("+s.Obs.Title+")", s.Action.Kind, target, s.Action.Reason)
	}
}

func outcome(done bool) string {
	if done {
		return "SUCCESS"
	}
	return "INCOMPLETE"
}
