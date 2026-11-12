package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildRoutesNamesTheFewestClicksToEachPage(t *testing.T) {
	out := t.TempDir()
	c := Config{App: "X", Out: out, AppMap: filepath.Join(out, "appmap"), Python: "python3", ToolsDir: "../../tools/jira"}
	crawl := `{"baseUrl":"http://h/","pages":[` +
		`{"url":"http://h/","elements":[{"href":"/a","locator":"role=link[name=\"A\"]","stable":true}]},` +
		`{"url":"http://h/a","elements":[{"href":"/b","locator":"role=link[name=\"B\"]","stable":true}]},` +
		`{"url":"http://h/b","elements":[{"href":"#x","locator":"role=link[name=\"X\"]","stable":false}]},` +
		`{"url":"http://h/c","elements":[]}]}`
	writeFile(t, c.crawlPath(), crawl)

	if err := c.buildRoutesWith("../../tools/routes.py"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(out, "routes.json"))
	if err != nil {
		t.Fatal(err)
	}
	var rep struct {
		Routes []struct {
			URL    string `json:"url"`
			Clicks int    `json:"clicks"`
			Hops   []struct {
				Locator string `json:"locator"`
			} `json:"hops"`
		} `json:"routes"`
		Unreachable []string `json:"unreachable"`
	}
	if err := json.Unmarshal(b, &rep); err != nil {
		t.Fatalf("%v: %s", err, b)
	}
	for _, r := range rep.Routes {
		if r.URL == "http://h/b" && (r.Clicks != 2 || r.Hops[0].Locator != `role=link[name="A"]` || r.Hops[1].Locator != `role=link[name="B"]`) {
			t.Fatalf("the route to /b is two clicks, A then B: %+v", r)
		}
	}
	if len(rep.Unreachable) != 1 || rep.Unreachable[0] != "http://h/c" {
		t.Fatalf("/c has no link to it: %+v", rep.Unreachable)
	}
}

func TestBuildRoutesWaitsForACrawl(t *testing.T) {
	c := Config{App: "X", Out: t.TempDir(), AppMap: t.TempDir(), Python: "python3"}
	if err := c.buildRoutesWith("../../tools/routes.py"); err != nil {
		t.Fatalf("no app map yet is not an error: %v", err)
	}
	if fileExists(filepath.Join(c.Out, "routes.json")) {
		t.Fatal("nothing to route, nothing written")
	}
}
