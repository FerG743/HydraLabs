package main

import (
	"strings"
	"testing"
)

func ready(name string) string {
	return `{"id":"1","name":"` + name + `","status":"ready","steps":[{"action":"x"}],"data":{"rows":[{"k":"v"}]}}`
}

func TestCheck(t *testing.T) {
	dupBlocked := `{"cases":[{"id":"9","name":"a","status":"blocked","steps":[]},{"id":"9","name":"b","status":"blocked","steps":[]}]}`
	if e := check([]byte(dupBlocked)); len(e) != 0 {
		t.Fatalf("duplicate blocked stubs must not fail the run: %v", e)
	}
	ok := `{"cases":[{"id":"1","name":"a","status":"ready","steps":[{"action":"x"}],"data":{"rows":[{"k":"v"}]}},
	                  {"id":"2","name":"b","status":"blocked","steps":[]}]}`
	if e := check([]byte(ok)); len(e) != 0 {
		t.Fatalf("valid input rejected: %v", e)
	}
	bad := map[string]string{
		"duplicate id": `{"cases":[` + ready("a") + `,` + ready("b") + `]}`,
		"empty name":   `{"cases":[{"id":"1","name":"","status":"blocked","steps":[]}]}`,
		"status":       `{"cases":[{"id":"1","name":"a","status":"done","steps":[]}]}`,
		"no data rows": `{"cases":[{"id":"1","name":"a","status":"ready","steps":[{"action":"x"}]}]}`,
		"no action":    `{"cases":[{"id":"1","name":"a","status":"blocked","steps":[{"action":""}]}]}`,
	}
	for want, in := range bad {
		if e := strings.Join(check([]byte(in)), "|"); !strings.Contains(e, want) {
			t.Errorf("want error mentioning %q, got %q", want, e)
		}
	}
}
