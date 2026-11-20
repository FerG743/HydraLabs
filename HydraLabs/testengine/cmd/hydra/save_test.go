package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Passing drafts become cases/<PROJECT>-n.md after the last existing number; a failed draft and an already-saved one are skipped.
func TestSaveDraftsWritesPassingDraftsOnce(t *testing.T) {
	dir := t.TempDir()
	c := Config{Home: dir, Out: dir, Project: "PQA"}
	os.MkdirAll(c.localCaseDir(), 0o755)
	os.WriteFile(filepath.Join(c.localCaseDir(), "PQA-3.md"), []byte("# old\n"), 0o644)
	os.WriteFile(filepath.Join(c.localCaseDir(), "OTHER-9.md"), []byte("# other\n"), 0o644)
	d := Design{Drafts: []DraftResult{
		{OK: true, Draft: Draft{Name: "Tarjetas shows Agregar", Precondition: "No sign-in is needed.", Steps: []string{"Navigate to /Tarjetas."}, Expected: []string{"The page shows \"Agregar\": it appears in \"Agregar\"."}}},
		{OK: false, Draft: Draft{Name: "rejected"}},
	}}
	keys, err := c.saveDrafts(&d)
	if err != nil || len(keys) != 1 || keys[0] != "PQA-4" {
		t.Fatalf("keys %v err %v", keys, err)
	}
	b, _ := os.ReadFile(filepath.Join(c.localCaseDir(), "PQA-4.md"))
	for _, want := range []string{"# Tarjetas shows Agregar", "labels: automate", "- Preconditions: No sign-in is needed.", "  1. Navigate to /Tarjetas.", "- Expected result:", "  - The page shows"} {
		if !strings.Contains(string(b), want) {
			t.Fatalf("missing %q in:\n%s", want, b)
		}
	}
	if again, _ := c.saveDrafts(&d); len(again) != 0 {
		t.Fatalf("saved twice: %v", again)
	}
	if _, err := os.Stat(filepath.Join(c.localCaseDir(), "PQA-5.md")); err == nil {
		t.Fatal("a second case was written for a draft already saved")
	}
}
