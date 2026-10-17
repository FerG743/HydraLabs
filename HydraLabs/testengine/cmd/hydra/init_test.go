package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// initRegistry: a projects.json with one project and a field init does not know, in a temp dir.
func initRegistry(t *testing.T) (*Registry, string) {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "projects.json")
	os.WriteFile(p, []byte(`{"_doc": "keep me", "reposRoot": "`+dir+`", "projects": {"DWQ": {"app": "CobroOrdenes", "profile": "profiles/cobro-ordenes.json"}}}`), 0o644)
	reg, err := LoadRegistry(p)
	if err != nil {
		t.Fatal(err)
	}
	return reg, p
}

// offline replaces the checks that touch the network for the length of a test.
func offline(t *testing.T, appUp bool) {
	r, j, g := reachable, jiraCount, remoteOK
	reachable = func(string) error {
		if appUp {
			return nil
		}
		return errors.New("connection refused")
	}
	jiraCount = func(string) (int, error) { return 0, errors.New("set JIRA_BASE_URL, JIRA_EMAIL and JIRA_TOKEN") }
	remoteOK = func(string) error { return nil }
	t.Cleanup(func() { reachable, jiraCount, remoteOK = r, j, g })
}

func readJSON(t *testing.T, p string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	json.Unmarshal(b, &m)
	return m
}

func TestInitAsksFixesATypoAndWritesOnlyAfterYes(t *testing.T) {
	offline(t, true)
	reg, regPath := initRegistry(t)
	typed := strings.Join([]string{
		"shop",                  // project id: lower case is accepted and upper-cased
		"toolshop",              // app: rejected (must start with a capital)...
		"Toolshop",              // ...asked again
		"",                      // add-ons: default (all)
		"https://qa.shop.test/", // trailing slash dropped
		"n",                     // not production
		"",                      // writes: default no
		"",                      // base branch: main
		"",                      // no remote
		"",                      // ploy source: jira
		"",                      // jira key: defaults to the project id
		"",                      // jql: the default one
		"",                      // write-back: no
		"",                      // stability runs: 3
		"/, /login",             // crawl paths
		"",                      // model: no
		"y",                     // confirm
	}, "\n") + "\n"
	var out bytes.Buffer
	previewed := false
	a, err := Init(reg, regPath, strings.NewReader(typed), &out, answers{}, false, func(answers) string { previewed = true; return "3 elements" })
	if err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "✗ one word starting with a capital") || a["core.app"] != "Toolshop" || a["core.project"] != "SHOP" {
		t.Errorf("typo not re-asked:\n%s", out.String())
	}
	if !previewed || !strings.Contains(out.String(), "could not check it against Jira") {
		t.Errorf("preview and the Jira warning belong in the conversation:\n%s", out.String())
	}

	p := readJSON(t, filepath.Join(filepath.Dir(regPath), "profiles", "toolshop.json"))
	if p["baseUrl"] != "https://qa.shop.test" || p["allowWrites"] != false || p["baseBranch"] != "main" || p["stabilityRuns"] != 3.0 {
		t.Errorf("profile: %v", p)
	}
	if p["jiraProject"] != "SHOP" {
		t.Errorf("the Jira project must be saved, not only used in the default query: %v", p["jiraProject"])
	}
	if !strings.Contains(p["jql"].(string), "project = SHOP AND labels") || p["agent"] != nil || p["model"] != nil || p["storeCrawl"] != true {
		t.Errorf("jql default from the key, no model tier, crawl stored: %v", p)
	}
	if paths := p["paths"].([]any); len(paths) != 2 || paths[1] != "/login" {
		t.Errorf("paths: %v", p["paths"])
	}
	r := readJSON(t, regPath)
	if r["_doc"] != "keep me" || r["projects"].(map[string]any)["SHOP"] == nil || r["projects"].(map[string]any)["DWQ"] == nil {
		t.Errorf("registry must gain SHOP and keep everything else: %v", r)
	}
	if reg.Projects["SHOP"].Profile != "profiles/toolshop.json" {
		t.Errorf("in-memory registry not updated: %+v", reg.Projects)
	}
}

func TestInitFromAnswersFileWithoutPrompts(t *testing.T) {
	offline(t, false)
	reg, regPath := initRegistry(t)
	given, _ := loadAnswers(writeAnswers(t, `{"core": {"project": "PAY", "app": "Payments", "addons": ["ploy"], "baseUrl": "https://pay.example", "prod": true},
		"ploy": {"source": "none"}}`))
	var out bytes.Buffer
	a, err := Init(reg, regPath, strings.NewReader(""), &out, given, true, nil)
	if err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "not reachable right now") || !strings.Contains(out.String(), "ploy without labs") {
		t.Errorf("an unreachable app and a missing add-on are warnings, said up front:\n%s", out.String())
	}
	if _, asked := a["core.allowWrites"]; asked {
		t.Error("production: the writes question must not even be asked")
	}
	p := readJSON(t, filepath.Join(filepath.Dir(regPath), "profiles", "payments.json"))
	if p["allowWrites"] != false || p["jql"] != nil || p["paths"] == nil || p["agent"] != nil || p["_environment"] == nil {
		t.Errorf("production, no Jira, no labs: %v", p)
	}
	if ad := p["addons"].([]any); len(ad) != 1 || ad[0] != "ploy" {
		t.Errorf("addons: %v", p["addons"])
	}
}

func TestInitWritesNothingWhenNotConfirmedOrInvalid(t *testing.T) {
	offline(t, true)
	reg, regPath := initRegistry(t)
	before, _ := os.ReadFile(regPath)
	given := answers{"core.project": "NEW", "core.app": "NewApp", "core.addons": "labs", "core.baseUrl": "http://qa"}
	if _, err := Init(reg, regPath, strings.NewReader("\n\n\n\n\n\n\nn\n"), &bytes.Buffer{}, given, false, nil); err == nil || !strings.Contains(err.Error(), "cancelled") {
		t.Errorf("a no must cancel: %v", err)
	}
	if _, err := Init(reg, regPath, strings.NewReader(""), &bytes.Buffer{}, answers{"core.project": "DWQ"}, true, nil); err == nil || !strings.Contains(err.Error(), "already registered") {
		t.Errorf("a registered project must be refused: %v", err)
	}
	if _, err := Init(reg, regPath, strings.NewReader(""), &bytes.Buffer{}, answers{"core.project": "X1", "core.app": "Xapp", "core.addons": "qa-magic"}, true, nil); err == nil || !strings.Contains(err.Error(), "unknown add-on") {
		t.Errorf("an unknown add-on from a file is an error, not a silent skip: %v", err)
	}
	after, _ := os.ReadFile(regPath)
	if !bytes.Equal(before, after) || fileExists(filepath.Join(filepath.Dir(regPath), "profiles", "new-app.json")) {
		t.Error("nothing may be written without a yes")
	}
}

func TestAutomateNeedsLabs(t *testing.T) {
	if !(Config{}).hasAddon("labs") {
		t.Error("a profile from before add-ons keeps everything")
	}
	c := Config{Out: t.TempDir(), App: "Payments", Addons: []string{"ploy"}}
	if r := c.Route("PAY-1", "automate"); r.Status != "needs-review" || !strings.Contains(r.Detail[0], "labs add-on") {
		t.Errorf("%+v", r)
	}
}

func writeAnswers(t *testing.T, s string) string {
	p := filepath.Join(t.TempDir(), "answers.json")
	os.WriteFile(p, []byte(s), 0o644)
	return p
}
