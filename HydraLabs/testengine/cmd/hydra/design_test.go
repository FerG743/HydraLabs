package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const designReq = "Operators enter an order number such as 9800074297, choose the store (LIVR or SBB) and process it; " +
	"the order then shows in Un vistazo a tus ordenes."

// fakeModel answers every chat request with the same cases, in the model's closed-choice shape (the T2 tie-break
// gets an answer it cannot parse: -1).
func fakeModel(t *testing.T, cases string) *httptest.Server {
	content := `{"cases": [` + cases + `]}`
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": content}}},
			"usage": map[string]int{"total_tokens": 700}})
	}))
	t.Cleanup(s.Close)
	return s
}

func designConfig(t *testing.T, lm string) Config {
	dir := t.TempDir()
	c := Config{Out: filepath.Join(dir, "out"), AppMap: filepath.Join(dir, "appmap"), App: "PortalQA", Lang: "en",
		ToolsDir: "../../tools/jira", Python: "python3", LMURL: lm, Model: "m"}
	writeJSON(c.crawlPath(), map[string]any{"pages": []any{map[string]any{"url": "http://127.0.0.1:3000/cobro_ordenes", "elements": []any{
		map[string]any{"role": "textbox", "name": "Ingresa el número de orden", "id": "orderNumber", "locator": "#orderNumber", "strategy": "id", "stable": true},
		map[string]any{"role": "combobox", "name": "Ingresa la tienda", "id": "name", "locator": "#name", "strategy": "id", "stable": true, "options": []string{"LIVR", "SBB"}},
		map[string]any{"role": "button", "name": "Agregar", "locator": `role=button[name="Agregar"]`, "strategy": "role+name", "stable": true},
		map[string]any{"role": "button", "name": "Procesar", "locator": `role=button[name="Procesar"]`, "strategy": "role+name", "stable": true},
	}}}})
	return c
}

var goodDraft = Draft{Name: "Add an order and process it", Precondition: "Portal is deployed and reachable.",
	Steps:    []string{"Navigate to `/cobro_ordenes`.", "Enter order number `9800074297`.", "Select store `LIVR`.", `Click "Agregar".`, `Click "Procesar".`},
	Expected: []string{`"Procesar" is disabled before the order is added and enabled after.`, `After processing, the order appears in "Un vistazo a tus ordenes".`}}

const goodCase = `{"name": "Add an order and process it", "precondition": "Portal is deployed and reachable.",
  "steps": [{"action": "navigate", "target": "", "value": "/cobro_ordenes"}, {"action": "enter", "target": "order number", "value": "9800074297"},
            {"action": "select", "target": "store", "value": "LIVR"}, {"action": "click", "target": "Agregar", "value": ""},
            {"action": "click", "target": "'Procesar'", "value": ""}],
  "expected": [{"kind": "disabled_then_enabled", "target": "Procesar", "between": "the order is added"},
               {"kind": "appears_in", "target": "Un vistazo a tus ordenes", "between": ""}]}`

func TestDesignKeepsOnlyDraftsThePipelineCanTrust(t *testing.T) {
	invented := `{"name": "Invented data", "precondition": "Portal is deployed and reachable.",
  "steps": [{"action": "navigate", "target": "", "value": "/cobro_ordenes"}, {"action": "enter", "target": "order number", "value": "5555555555"},
            {"action": "select", "target": "store", "value": "SBB"}, {"action": "click", "target": "Agregar", "value": ""}],
  "expected": [{"kind": "disabled_then_enabled", "target": "Procesar", "between": "the order is added"}]}`
	offList := `{"name": "Off the list", "precondition": "Portal is deployed and reachable.",
  "steps": [{"action": "navigate", "target": "", "value": "/cobro_ordenes"}, {"action": "enter", "target": "order number", "value": "9800074297"}],
  "expected": [{"kind": "looks_right", "target": "the page", "between": ""}]}`
	c := designConfig(t, fakeModel(t, goodCase+","+invented+","+offList).URL)

	d, err := c.Design(designReq, "req.txt", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Drafts) != 3 || d.Tokens < 700 {
		t.Fatalf("%+v", d)
	}
	if !d.Drafts[0].OK {
		t.Errorf("a draft with real values that maps and asserts must pass: %v", d.Drafts[0].Problems)
	}
	if d.Drafts[1].OK || !strings.Contains(strings.Join(d.Drafts[1].Problems, " "), `"5555555555"`) {
		t.Errorf("invented data must be rejected and named: %v", d.Drafts[1].Problems)
	}
	if d.Drafts[2].OK || !strings.Contains(strings.Join(d.Drafts[2].Problems, " "), "would not be asserted") {
		t.Errorf("an expected result outside the closed list asserts nothing and must be rejected: %v", d.Drafts[2].Problems)
	}
	if !fileExists(filepath.Join(d.Dir, "design.json")) || !fileExists(filepath.Join(d.Dir, "DRAFT-1.html")) {
		t.Error("drafts are saved for a person to read")
	}
	if fileExists(filepath.Join(c.Out, "bundles", "DRAFT-1")) || fileExists(filepath.Join(c.Out, "cases", "DRAFT-1.case.json")) {
		t.Error("judging a draft must not leave cases or bundles behind")
	}
}

func TestDesignNeedsAModelAndACrawl(t *testing.T) {
	c := designConfig(t, "http://127.0.0.1:1")
	c.Model = ""
	if _, err := c.Design(designReq, "x", 1); err == nil || !strings.Contains(err.Error(), "--model") {
		t.Errorf("%v", err)
	}
	c.Model = "m"
	os.Remove(c.crawlPath())
	if _, err := c.Design(designReq, "x", 1); err == nil || !strings.Contains(err.Error(), "crawl") {
		t.Errorf("%v", err)
	}
}

func TestADraftReadsBackAsTheSameCase(t *testing.T) {
	cs, err := ToCase(draftHTML(goodDraft), "DRAFT-1", goodDraft.Name, "en", "", "")
	if err != nil || len(cs.Steps) != 5 || cs.Steps[1].Action != "Enter order number `9800074297`." || len(cs.Expected) != 2 ||
		cs.Precondition != goodDraft.Precondition {
		t.Errorf("what Jira will show must be what was judged: %+v %v", cs, err)
	}
}

func TestCreatedDraftsWaitForAPerson(t *testing.T) {
	var got map[string]any
	jira := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		json.Unmarshal(b, &got)
		w.Write([]byte(`{"key": "DWQ-200"}`))
	}))
	defer jira.Close()
	key, err := (&Jira{Base: jira.URL, Email: "e", Token: "t"}).CreateDraft("DWQ", "DWQ-129", "Task", goodDraft)
	if err != nil || key != "DWQ-200" {
		t.Fatal(key, err)
	}
	f := got["fields"].(map[string]any)
	if labels := f["labels"].([]any); len(labels) != 1 || labels[0] != "hydra-draft" {
		t.Errorf("a draft is labeled hydra-draft and never automate: %v", labels)
	}
	if f["parent"].(map[string]any)["key"] != "DWQ-129" || f["project"].(map[string]any)["key"] != "DWQ" {
		t.Errorf("%v", f)
	}
	desc, _ := json.Marshal(f["description"])
	if !strings.Contains(string(desc), `{"marks":[{"type":"code"}],"text":"9800074297","type":"text"}`) {
		t.Errorf("values go in code marks, as people write them: %s", desc)
	}
}

func TestTheModelsChoicesBecomeTheRenderersSentences(t *testing.T) {
	var m modelCase
	if err := json.Unmarshal([]byte(goodCase), &m); err != nil {
		t.Fatal(err)
	}
	d := m.draft()
	if strings.Join(d.Steps, "|") != strings.Join(goodDraft.Steps, "|") || strings.Join(d.Expected, "|") != strings.Join(goodDraft.Expected, "|") {
		t.Errorf("got\n%q\n%q", d.Steps, d.Expected) // the stray quotes around 'Procesar' are dropped too
	}
}

func TestADraftMustStartByOpeningAPage(t *testing.T) {
	noNav := strings.Replace(goodCase, `{"action": "navigate", "target": "", "value": "/cobro_ordenes"}, `, "", 1)
	c := designConfig(t, fakeModel(t, noNav).URL)
	d, err := c.Design(designReq, "x", 1)
	if err != nil || d.Drafts[0].OK || !strings.Contains(strings.Join(d.Drafts[0].Problems, " "), "step 1 must be navigate") {
		t.Errorf("%v %+v", err, d.Drafts)
	}
}

func TestNavigateTakesThePathFromEitherField(t *testing.T) {
	var m modelCase
	json.Unmarshal([]byte(`{"steps": [{"action": "navigate", "target": "/cobro_ordenes", "value": ""}]}`), &m)
	if got := m.draft().Steps[0]; got != "Navigate to `/cobro_ordenes`." {
		t.Error(got)
	}
}

func TestTheStartPageComesFromTheRequirementWhenTheModelForgetsIt(t *testing.T) {
	noNav := strings.Replace(goodCase, `{"action": "navigate", "target": "", "value": "/cobro_ordenes"}, `, "", 1)
	c := designConfig(t, fakeModel(t, noNav).URL)
	d, err := c.Design(designReq+" It happens on /cobro_ordenes.", "x", 1)
	if err != nil || !d.Drafts[0].OK || d.Drafts[0].Draft.Steps[0] != "Navigate to `/cobro_ordenes`." {
		t.Errorf("%v %+v", err, d.Drafts)
	}
}

func TestCreateUsesTheReviewedRunAndNeverCreatesTwice(t *testing.T) {
	calls := 0
	jira := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Write([]byte(`{"key": "DWQ-300"}`))
	}))
	defer jira.Close()
	t.Setenv("JIRA_BASE_URL", jira.URL)
	t.Setenv("JIRA_EMAIL", "e")
	t.Setenv("JIRA_TOKEN", "t")
	c := designConfig(t, fakeModel(t, goodCase).URL)
	c.JiraProject = "DWQ"
	if _, err := c.Design(designReq, "x", 1); err != nil {
		t.Fatal(err)
	}
	d, err := c.LoadDesign("last")
	if err != nil || len(d.Drafts) != 1 || !d.Drafts[0].OK {
		t.Fatalf("%v %+v", err, d)
	}
	if err := c.createDrafts(&d, "DWQ-129", "Task"); err != nil || d.Drafts[0].Issue != "DWQ-300" {
		t.Fatalf("%v %+v", err, d.Drafts)
	}
	again, _ := c.LoadDesign("last") // the run now records the key
	c.createDrafts(&again, "DWQ-129", "Task")
	if calls != 1 {
		t.Errorf("a draft already in Jira must not be created again: %d calls", calls)
	}
}
