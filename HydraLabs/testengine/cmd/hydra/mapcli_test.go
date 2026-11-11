package main

import (
	"path/filepath"
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

func TestBoardReadsTheSameKPIsAsTheMap(t *testing.T) {
	out := t.TempDir()
	c := Config{App: "X", Out: out, AppMap: filepath.Join(out, "appmap")}
	writeFile(t, c.crawlPath(), `{"pages":[{"url":"http://h/a","elements":[{"locator":"role=button[name=\"Go\"]","role":"button"}]}]}`)
	writeFile(t, filepath.Join(out, "coverage.json"), `{"pages":[{"url":"http://h/a","covered":false,"gaps":["role=button[name=\"Go\"]"]}]}`)
	v, err := c.loadMapView()
	if err != nil || v.KPIs == nil || v.KPIs.CoveredLocators != 0 || v.KPIs.UniqueLocators != 1 {
		t.Fatalf("kpis: %v %+v", err, v.KPIs)
	}
}
