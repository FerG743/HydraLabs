package main

import (
	"strings"
	"testing"
)

func TestMapSaysWhichProjectWhenNoneIsGiven(t *testing.T) {
	err := Config{}.printMap(false, "", false)
	if err == nil || !strings.Contains(err.Error(), "--project") {
		t.Fatalf("want a hint to pass --project, got %v", err)
	}
}

func TestProjectFirstPagesCommandRunsTheMap(t *testing.T) {
	reg := &Registry{Projects: map[string]ProjectEntry{"DWQ": {}}}
	got, ok := projectPagesArgs(reg, []string{"DWQ", "pages", "--gaps"})
	if !ok || strings.Join(got, " ") != "map --project DWQ --gaps" {
		t.Fatalf("got %v %v", got, ok)
	}
	if _, ok := projectPagesArgs(reg, []string{"NOPE", "pages"}); ok {
		t.Fatal("an unknown project was accepted")
	}
	if _, ok := projectPagesArgs(reg, []string{"DWQ", "poll"}); ok {
		t.Fatal("another command was rewritten")
	}
}
