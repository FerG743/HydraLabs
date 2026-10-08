package main

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

const fixtures = "../../tools/jira/fixtures/"

func asJSON(t *testing.T, v any) any { // compare as JSON: key order and nil-vs-empty do not matter
	b, _ := json.Marshal(v)
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// The Go port must produce exactly what tools/jira/jira2case.py produced for the same fixtures.
func TestIntakeMatchesPythonGolden(t *testing.T) {
	for _, k := range []string{"DWQ-130", "DWQ-131", "DWQ-132"} {
		src, _ := os.ReadFile(fixtures + k + ".html")
		raw, err := os.ReadFile("testdata/" + k + ".case.json")
		if err != nil {
			t.Fatal(err)
		}
		var want any
		json.Unmarshal(raw, &want)
		n := map[string]string{"DWQ-130": "1", "DWQ-131": "2", "DWQ-132": "3"}[k]
		got, err := ToCase(string(src), k, "TC-"+n+" - Fixture", "en", "DWQ-129", "Medium")
		if err != nil {
			t.Fatal(err)
		}
		if g := asJSON(t, got); !reflect.DeepEqual(g, want) {
			gb, _ := json.MarshalIndent(g, "", " ")
			wb, _ := json.MarshalIndent(want, "", " ")
			t.Errorf("%s differs\n got: %s\nwant: %s", k, gb, wb)
		}
	}
}

func TestResolveRefs(t *testing.T) {
	a := &Case{ID: "DWQ-130", Name: "TC-1 – Manual", Precondition: "Portal is up."}
	b := &Case{ID: "DWQ-131", Name: "TC-2 – Bulk", Precondition: "Same as TC-1.", PreconditionRef: "TC-1"}
	c := &Case{ID: "DWQ-132", Name: "TC-3", Precondition: "Same as TC-9.", PreconditionRef: "TC-9"}
	ResolveRefs([]*Case{a, b, c})
	if !strings.HasSuffix(b.Precondition, "Portal is up.") || b.PreconditionSource != "DWQ-130" {
		t.Errorf("TC-2 not resolved: %q", b.Precondition)
	}
	if c.Precondition != "Same as TC-9." || len(c.Warnings) != 1 || !strings.Contains(c.Warnings[0], "not found") {
		t.Errorf("TC-3 should stay unresolved and say why: %q %v", c.Precondition, c.Warnings)
	}
}
