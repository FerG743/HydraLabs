package main

import (
	"regexp"
	"testing"
)

func TestJoin(t *testing.T) {
	step := []map[string]any{{"action": "a", "expected": "b"}}
	stub := []map[string]any{{"action": "Dependencia de insumos"}}
	matrix := []Matrix{
		{ID: "1", Steps: step}, // joins suite "01"
		{ID: "2", Steps: step}, // suite row is free text
		{ID: "3", Steps: step}, // suite row is empty
		{ID: "4", Steps: stub}, // blocked by regex
		{ID: "5", Steps: step}, // not in suite
		{ID: "5", Steps: step}, // duplicate id
		{ID: "6", Steps: nil},  // no steps
	}
	suite := []Suite{
		{ID: "01", Status: "ready", Rows: []map[string]string{{"boleta": "47"}}},
		{ID: "02", Status: "free-text", Note: "use case 01"},
		{ID: "03", Status: "empty"},
		{ID: "99", Status: "ready", Rows: []map[string]string{{"boleta": "1"}}}, // orphan
	}
	out := join(matrix, suite, regexp.MustCompile("Dependencia"))

	want := []string{"ready", "depends", "no-data", "blocked", "no-data", "no-data", "blocked"}
	for i, c := range out.Cases {
		if c.Status != want[i] {
			t.Errorf("case %s: got %s, want %s", c.ID, c.Status, want[i])
		}
	}
	if out.Cases[0].Data == nil || out.Cases[0].Data.Rows[0]["boleta"] != "47" {
		t.Error("case 1 should carry suite rows")
	}
	if len(out.Cases[5].Warnings) == 0 {
		t.Error("duplicate id should warn")
	}
	if len(out.Orphans) != 1 || out.Orphans[0] != "99" {
		t.Errorf("orphans: %v", out.Orphans)
	}
}
