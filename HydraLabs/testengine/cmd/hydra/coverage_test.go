package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(path), 0o755)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCoverageGapsNameTheElementsAndPagesNoBuiltTestReaches(t *testing.T) {
	out := t.TempDir()
	c := Config{App: "X", Out: out, AppMap: filepath.Join(out, "appmap"), Python: "python3", ToolsDir: "../../tools/jira"}
	bundle := filepath.Join(out, "bundles", "B-1")
	writeFile(t, filepath.Join(bundle, "apps", "X", "locators", "x_locators.py"),
		"class XLocators:\n    BOTON_A = 'role=button[name=\"A\"]'\n    BOTON_B = 'role=button[name=\"B\"]'\n")
	writeFile(t, filepath.Join(bundle, "apps", "X", "modules", "m.py"),
		"from apps.X.locators.x_locators import XLocators as L\n\n"+
			"def abrir_flujo(driver):\n    driver.get(Config.BASE_URL + '/tarjetas')\n    driver.page.locator(L.BOTON_A).click()\n")
	writeFile(t, filepath.Join(bundle, "tests", "X", "test_a.py"),
		"from apps.X.modules import m as co\n\ndef test_a(driver):\n    co.abrir_flujo(driver)\n")
	crawl := `{"pages":[` +
		`{"url":"http://h/tarjetas","elements":[{"locator":"role=button[name=\"A\"]"},{"locator":"role=button[name=\"B\"]"}]},` +
		`{"url":"http://h/otra","elements":[{"locator":"role=link[name=\"C\"]"}]}]}`
	writeFile(t, c.crawlPath(), crawl)

	if err := c.coverageGapsWith("../../tools/coverage.py"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(out, "coverage.json"))
	if err != nil {
		t.Fatal(err)
	}
	var rep struct {
		Tests   int `json:"tests"`
		Summary struct {
			CoveredPages    int `json:"coveredPages"`
			Elements        int `json:"elements"`
			CoveredElements int `json:"coveredElements"`
		} `json:"summary"`
		Pages []struct {
			URL  string   `json:"url"`
			Gaps []string `json:"gaps"`
		} `json:"pages"`
	}
	if err := json.Unmarshal(b, &rep); err != nil {
		t.Fatalf("%v: %s", err, b)
	}
	if rep.Tests != 1 || rep.Summary.Elements != 3 || rep.Summary.CoveredElements != 1 || rep.Summary.CoveredPages != 1 {
		t.Fatalf("summary: %+v", rep)
	}
	if len(rep.Pages) != 2 || len(rep.Pages[0].Gaps) != 1 || rep.Pages[0].Gaps[0] != `role=button[name="B"]` {
		t.Fatalf("the gap on /tarjetas is B, got %+v", rep.Pages)
	}
}

func TestCoverageReportsEachPlaylistOnItsOwn(t *testing.T) {
	home := t.TempDir()
	c := Config{Home: home, Out: filepath.Join(home, "out"), AppMap: filepath.Join(home, "appmap"), App: "App",
		Python: "python3", ToolsDir: "../../tools/jira"}
	os.MkdirAll(c.caseDir(), 0o755)
	os.MkdirAll(c.durable("playlists"), 0o755)
	for _, id := range []string{"DWQ-901", "DWQ-902"} {
		body := `{"id":"` + id + `","name":"TC - caso ` + id + `","status":"ready","steps":[{"n":"1","action":"Navigate to /x"}],"expected":[{"text":"ok"}]}`
		os.WriteFile(filepath.Join(c.caseDir(), id+".case.json"), []byte(body), 0o644)
	}
	os.WriteFile(c.durable("playlists", "flow.json"), []byte(`{"id":"flow","steps":[{"case":"DWQ-901"},{"case":"DWQ-902"}]}`), 0o644)
	writeFile(t, c.crawlPath(), `{"pages":[{"url":"http://h/x","elements":[{"locator":"role=button[name=\"A\"]"}]}]}`)

	if err := c.coverageGapsWith("../../tools/coverage.py"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(c.Out, "coverage.json"))
	if err != nil {
		t.Fatal(err)
	}
	var rep struct {
		Flows []struct {
			Name  string `json:"name"`
			Tests int    `json:"tests"`
		} `json:"flows"`
	}
	if err := json.Unmarshal(b, &rep); err != nil {
		t.Fatalf("%v: %s", err, b)
	}
	if len(rep.Flows) != 1 || rep.Flows[0].Name != "flow" || rep.Flows[0].Tests != 2 {
		t.Fatalf("the flow should be reported with its two tests, got %s", b)
	}
}

func TestCoverageGapsWaitForACrawl(t *testing.T) {
	c := Config{App: "X", Out: t.TempDir(), AppMap: t.TempDir(), Python: "python3"}
	if err := c.coverageGapsWith("../../tools/coverage.py"); err != nil {
		t.Fatalf("no app map yet is not an error: %v", err)
	}
	if fileExists(filepath.Join(c.Out, "coverage.json")) {
		t.Fatal("nothing to compare, nothing written")
	}
}

func TestCoverageDataIsEmptyUntilAPollWritesIt(t *testing.T) {
	c := Config{Out: t.TempDir()}
	if got := string(c.coverageData()); got != `{"pages":[]}` {
		t.Fatalf("before the first poll: %s", got)
	}
	writeFile(t, filepath.Join(c.Out, "coverage.json"), `{"pages":[{"url":"http://h/a"}]}`)
	if got := string(c.coverageData()); got != `{"pages":[{"url":"http://h/a"}]}` {
		t.Fatalf("the report as written: %s", got)
	}
}
