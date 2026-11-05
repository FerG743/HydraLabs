package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
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
		"",                      // no sign-in
		"",                      // ploy source: jira
		"",                      // jira key: defaults to the project id
		"",                      // jql: the default one
		"",                      // write-back: no
		"",                      // stability runs: 3
		"/, /login",             // crawl paths
		"y",                     // record the crawl as a video
		"3000",                  // pause between elements: a demo
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
	if p["recordVideo"] != true || p["slowMs"] != 3000.0 {
		t.Errorf("the crawl must be recorded and paced as answered: %v", p)
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

func TestAQueryJiraRejectsIsAskedAgain(t *testing.T) {
	offline(t, true)
	jiraCount = func(jql string) (int, error) {
		if jql == "DWQ-140" {
			return 0, errors.New(`jira GET /rest/api/3/search/jql: 400 {"errorMessages":["Error in the JQL Query"]}`)
		}
		return 3, nil
	}
	if _, _, err := checkJQL("DWQ-140", answers{}); err == nil || !strings.Contains(err.Error(), "does not accept") {
		t.Errorf("an invalid query must be asked again: %v", err)
	}
	if _, warn, err := checkJQL("key = DWQ-140", answers{}); err != nil || !strings.Contains(warn, "3 issue(s)") {
		t.Errorf("%q %v", warn, err)
	}
}

func TestInitWithSignInKeepsTheSecretOutOfEverythingButTheEnvFile(t *testing.T) {
	offline(t, true)
	reg, regPath := initRegistry(t)
	envFile := filepath.Join(t.TempDir(), ".hydra.env")
	os.WriteFile(envFile, []byte("JIRA_TOKEN=keep\nHYDRA_SHOP_PASSWORD=old\n"), 0o644)
	old := hydraEnvPath
	hydraEnvPath = envFile
	t.Cleanup(func() { hydraEnvPath = old; os.Unsetenv("HYDRA_SHOP_USER"); os.Unsetenv("HYDRA_SHOP_PASSWORD") })
	given, _ := loadAnswers(writeAnswers(t, `{"core": {"project": "SHOP", "app": "Toolshop", "addons": ["labs"], "baseUrl": "https://qa.shop.test",
		"login": true, "user": "qa@shop.test", "password": "s3cr3t!"}}`))
	var out bytes.Buffer
	a, err := Init(reg, regPath, strings.NewReader(""), &out, given, true, nil)
	if err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	if strings.Contains(out.String(), "s3cr3t!") || !strings.Contains(out.String(), "sign-in   /login as qa@shop.test") {
		t.Errorf("the summary names the user and never the password:\n%s", out.String())
	}
	profilePath := filepath.Join(filepath.Dir(regPath), "profiles", "toolshop.json")
	pb, _ := os.ReadFile(profilePath)
	p := readJSON(t, profilePath)
	if p["loginPath"] != "/login" || p["loginFile"] != filepath.Join(filepath.Dir(regPath), "profiles", "toolshop.login.json") || strings.Contains(string(pb), "s3cr3t!") {
		t.Errorf("profile: %s", pb)
	}
	steps, _ := os.ReadFile(p["loginFile"].(string))
	if strings.Contains(string(steps), "s3cr3t!") || !strings.Contains(string(steps), "ENV:HYDRA_SHOP_PASSWORD") || !strings.Contains(string(steps), "input[type=password]") {
		t.Errorf("login steps hold ENV: tokens only: %s", steps)
	}
	env, _ := os.ReadFile(envFile)
	if st, _ := os.Stat(envFile); st.Mode().Perm() != 0o600 {
		t.Errorf("the env file must be private: %v", st.Mode())
	}
	if !strings.Contains(string(env), "JIRA_TOKEN=keep\n") || !strings.Contains(string(env), "HYDRA_SHOP_PASSWORD=s3cr3t!\n") || strings.Contains(string(env), "=old") {
		t.Errorf("other secrets kept, the old password replaced: %s", env)
	}
	if os.Getenv("HYDRA_SHOP_USER") != "qa@shop.test" {
		t.Error("the first crawl runs in this process: it needs the names set now")
	}

	repo := t.TempDir()
	if out, err := exec.Command("git", "-C", repo, "init", "-q").CombinedOutput(); err != nil {
		t.Skip("no git:", string(out))
	}
	if err := seedLoginTest(repo, a); err != nil {
		t.Fatal(err)
	}
	src, _ := os.ReadFile(filepath.Join(repo, "tests", "Toolshop", "test_login.py"))
	if !strings.Contains(string(src), "login.sign_in(") || !strings.Contains(string(src), `PASSWORD_FIELD = "input[type=password]"`) || strings.Contains(string(src), "s3cr3t!") {
		t.Errorf("test_login.py:\n%s", src)
	}
	if out, err := exec.Command("python3", "-m", "py_compile", filepath.Join(repo, "tests", "Toolshop", "test_login.py")).CombinedOutput(); err != nil && !errors.Is(err, exec.ErrNotFound) {
		t.Errorf("test_login.py is not valid Python: %s", out)
	}
	if !strings.Contains(string(src), "@pytest.mark.signed_out  #") {
		t.Error("the sign-in test must not sign in first")
	}
	if l := readJSON(t, filepath.Join(repo, "hydra", "login.json")); l["path"] != "/login" || len(l["steps"].([]any)) != 3 {
		t.Errorf("hydra/login.json, for the framework's conftest: %v", l)
	}
	if log, _ := git(repo, "log", "--oneline"); !strings.Contains(log, "sign-in test for Toolshop") {
		t.Errorf("the test must be committed: %q", log)
	}
	if err := seedLoginTest(repo, a); err != nil {
		t.Errorf("a second init leaves the test alone: %v", err)
	}
}

func TestInitWithLocalCasesNeedsNoJira(t *testing.T) {
	offline(t, true)
	reg, regPath := initRegistry(t)
	given := answers{"core.project": "SHOP", "core.app": "Toolshop", "core.addons": "ploy,labs", "core.baseUrl": "http://qa", "ploy.source": "local"}
	var out bytes.Buffer
	if _, err := Init(reg, regPath, strings.NewReader(""), &out, given, true, nil); err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	p := readJSON(t, filepath.Join(filepath.Dir(regPath), "profiles", "toolshop.json"))
	if p["caseSource"] != "local" || p["jql"] != nil || strings.Contains(out.String(), "against Jira") || !strings.Contains(out.String(), "cases/SHOP-1.md") {
		t.Errorf("local cases: no Jira question, no query, the folder named in the summary: %v\n%s", p, out.String())
	}
}

func TestInitFindsTheSignInFormSoOnlyTheUserAndPasswordAreAsked(t *testing.T) {
	offline(t, true)
	t.Setenv("HOME", t.TempDir()) // the password goes to ~/.hydra.env
	old, oldEnv := detectLogin, hydraEnvPath
	hydraEnvPath = filepath.Join(t.TempDir(), ".hydra.env")
	t.Cleanup(func() { detectLogin, hydraEnvPath = old, oldEnv })
	given := answers{"core.project": "SHOP", "core.app": "Toolshop", "core.addons": "labs", "core.baseUrl": "http://qa", "core.prod": "n",
		"core.allowWrites": "n", "core.baseBranch": "main", "core.remote": "", "labs.paths": "/", "labs.video": "n", "labs.slowMs": "0", "labs.model": "n"}

	detectLogin = func(string, string) (loginForm, error) {
		return loginForm{Found: true, Path: "/signin", UserField: "#email", PasswordField: "input[type=password]", Submit: `role=button[name="Entrar"]`}, nil
	}
	reg, regPath := initRegistry(t)
	var out bytes.Buffer
	a, err := Init(reg, regPath, strings.NewReader("y\nqa@shop.test\ns3cr3t!\ny\n"), &out, given, false, nil) // login? user, password, confirm
	if err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	if a["core.loginPath"] != "/signin" || a["core.userField"] != "#email" || a["core.submit"] != `role=button[name="Entrar"]` {
		t.Errorf("the form's page and fields come from the app: %v", a)
	}
	if strings.Contains(out.String(), "Selector of") || strings.Contains(out.String(), "Path of the sign-in page") || !strings.Contains(out.String(), "✓ found it at /signin") {
		t.Errorf("nothing about the form is asked when it was found:\n%s", out.String())
	}

	detectLogin = func(string, string) (loginForm, error) { return loginForm{Why: "no password field on this page"}, nil }
	reg, regPath = initRegistry(t)
	out.Reset()
	given["core.project"], given["core.app"] = "SHOP2", "Toolshop2"
	typed := "y\n\nqa@shop.test\ns3cr3t!\n\n\n\ny\n" // login? path (default), user, password, three selectors (defaults), confirm
	if a, err = Init(reg, regPath, strings.NewReader(typed), &out, given, false, nil); err != nil {
		t.Fatalf("%v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "could not find it (no password field on this page)") || !strings.Contains(out.String(), "Selector of the user field") || a["core.passwordField"] != "input[type=password]" {
		t.Errorf("not found: say why and ask as before:\n%s", out.String())
	}
}
