package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// hydra show prints the case's Markdown, and its latest recorded run when there is one.
func TestCaseTextShowsTheCaseAndLastRun(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "cases"), 0o755)
	os.WriteFile(filepath.Join(dir, "cases", "PQA-1.md"), []byte("# Tarjetas shows Agregar\n"), 0o644)
	c := Config{Home: dir, Out: dir}
	got, err := c.caseText("PQA-1")
	if err != nil || !strings.Contains(got, "Tarjetas shows Agregar") || strings.Contains(got, "Last run") {
		t.Fatalf("no runs yet: %q %v", got, err)
	}
	if _, err := c.caseText("PQA-9"); err == nil {
		t.Fatal("missing case should fail")
	}
	c.logExecution(Execution{At: "2026-11-20T10:00:00", Key: "PQA-1", Status: "passed", Trigger: "manual"})
	got, _ = c.caseText("PQA-1")
	if !strings.Contains(got, "Last run:") && strings.Contains(got, "passed (manual)") {
		t.Fatalf("with a run: %q", got)
	}
}

func TestSplitCaseKeyKeepsTheFlags(t *testing.T) {
	key, rest := splitCaseKey([]string{"--project", "PQA", "pqa-1"})
	if key != "PQA-1" || strings.Join(rest, " ") != "--project PQA" {
		t.Fatalf("key %q rest %v", key, rest)
	}
}
