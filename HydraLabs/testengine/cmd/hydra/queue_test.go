package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func jb(project, key string, class int) *job {
	return &job{cfg: &Config{Project: project}, key: key, class: class}
}

func keys(js []*job) string {
	var s []string
	for _, j := range js {
		s = append(s, j.key)
	}
	return strings.Join(s, " ")
}

func TestCheapestWorkRunsFirstAndProjectsTakeTurns(t *testing.T) {
	in := []*job{jb("A", "A-1", classModel), jb("A", "A-2", classRender), jb("A", "A-3", classRender),
		jb("B", "B-1", classRender), jb("B", "B-2", classFree), jb("A", "A-4", classFree)}
	got := keys(order(in))
	want := "A-4 B-2 A-2 B-1 A-3 A-1" // free (A then B), render taking turns (A, B, A), model last
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestModelCasesAreCappedAndTheRestWait(t *testing.T) {
	ordered := order([]*job{jb("A", "A-1", classModel), jb("B", "B-1", classModel), jb("A", "A-2", classModel), jb("A", "A-3", classRender)})
	run, wait := schedule(ordered, 1)
	if keys(run) != "A-3 A-1" || keys(wait) != "B-1 A-2" {
		t.Errorf("cap 1: run [%s] wait [%s]", keys(run), keys(wait))
	}
	if run, wait = schedule(ordered, 2); keys(run) != "A-3 A-1 B-1" || keys(wait) != "A-2" {
		t.Errorf("cap 2: run [%s] wait [%s]", keys(run), keys(wait))
	}
	if run, _ = schedule(ordered, 0); len(run) != 2 {
		t.Errorf("a cap below 1 must still let one model case through, never zero: %s", keys(run))
	}
}

func TestClassifyKnowsTheCostBeforeItIsPaid(t *testing.T) {
	c := fakeEnv(t, "[]", passes) // ABC-1 is complete and maps cleanly
	if cl, _ := classify(&c, "ABC-1", "automate"); cl != classRender {
		t.Errorf("clean case should be render, got %s", classNames[cl])
	}
	c.Ingest(Issue{Key: "ABC-2", Summary: "TC", HTML: "<ul><li>Steps:<ul><li>TBD</li></ul></li></ul>"})
	if cl, why := classify(&c, "ABC-2", "automate"); cl != classFree || !strings.Contains(why, "author") {
		t.Errorf("an incomplete case costs nothing: %s (%s)", classNames[cl], why)
	}
	// a stored test: free, and classification must not touch the stored bundle
	marker := filepath.Join(c.bundleDir("ABC-1"), "tests", "keep.txt")
	os.MkdirAll(filepath.Dir(marker), 0o755)
	os.WriteFile(marker, []byte("stored"), 0o644)
	if cl, _ := classify(&c, "ABC-1", "execute"); cl != classFree || !fileExists(marker) {
		t.Errorf("execute with a stored test is free and must leave the bundle alone: %s", classNames[cl])
	}
	// no stored test: an execute is really an automate
	os.RemoveAll(c.bundleDir("ABC-1"))
	if cl, _ := classify(&c, "ABC-1", "execute"); cl != classRender {
		t.Errorf("execute without a test costs what automating costs: %s", classNames[cl])
	}
}

func TestOnlyTheModelCostsMore(t *testing.T) {
	c := fakeEnv(t, "['1: weird step']", passes)
	if cl, _ := classify(&c, "ABC-1", "automate"); cl != classModel {
		t.Errorf("unmappable steps with a model tier: %s", classNames[cl])
	}
	c.Agent = ""
	if cl, _ := classify(&c, "ABC-1", "automate"); cl != classRender {
		t.Errorf("without a model tier it goes straight to review, which is cheap: %s", classNames[cl])
	}
}

// --- end to end: two projects, one fake Jira, the real queue --------------------------------------------------------

const completeCase = "<ul><li>Steps:<ul><li>Do a thing.</li></ul></li><li>Expected Result:<ul><li>It works.</li></ul></li></ul>"

func fakeProject(t *testing.T, project, pending string) *Config {
	dir := t.TempDir()
	tools := filepath.Join(dir, "tools")
	os.MkdirAll(tools, 0o755)
	w := func(n, body string) { os.WriteFile(filepath.Join(tools, n), []byte(body), 0o644) }
	w("render_pytest.py", "import json,sys,os\na=sys.argv; out=a[a.index('--out')+1]\nos.makedirs(out+'/tests/App',exist_ok=True)\nprint(json.dumps([{'pending':"+pending+"}]))")
	w("run_generated.py", "import json,os,sys\nopen(os.environ['ORDER_LOG'],'a').write(sys.argv[sys.argv.index('--test')+1]+'\\n')\nprint(json.dumps({'status':'passed'}))")
	w("agent.py", "import json,os,sys; os.makedirs(sys.argv[sys.argv.index('--repo')+1],exist_ok=True); print(json.dumps({'finished':True,'problems':[],'summary':'ok','tokens':10}))")
	return &Config{Project: project, Out: filepath.Join(dir, "out"), ToolsDir: tools, AppMap: dir, Python: "python3", App: "App", Lang: "en",
		PortalURL: "http://x", Agent: filepath.Join(tools, "agent.py"), Model: "m", MaxTurns: 5,
		JQL: "project = " + project}
}

func fakeJira(t *testing.T, byProject map[string][]string) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/search/jql") {
			var issues []map[string]string
			for p, ks := range byProject {
				if strings.Contains(r.URL.Query().Get("jql"), "project = "+p) {
					for _, k := range ks {
						issues = append(issues, map[string]string{"key": k})
					}
				}
			}
			json.NewEncoder(w).Encode(map[string]any{"issues": issues})
			return
		}
		key := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		label := "automate"
		if strings.HasSuffix(key, "-9") {
			label = "execute"
		}
		json.NewEncoder(w).Encode(map[string]any{"key": key, "fields": map[string]any{"summary": "TC " + key, "updated": "2026-10-08T10:00:00", "labels": []string{label}},
			"renderedFields": map[string]string{"description": completeCase}})
	}))
	t.Cleanup(srv.Close)
	t.Setenv("JIRA_BASE_URL", srv.URL)
	t.Setenv("JIRA_EMAIL", "a@b")
	t.Setenv("JIRA_TOKEN", "t")
}

func TestPollRunsEverythingCheapestFirstAndHoldsBackTheSecondModelCase(t *testing.T) {
	jsonOut = true
	defer func() { jsonOut = false }()
	log := filepath.Join(t.TempDir(), "order.log")
	t.Setenv("ORDER_LOG", log)
	a, b := fakeProject(t, "A", "['1: weird step']"), fakeProject(t, "B", "[]") // A needs the model, B maps cleanly
	// B-9 is an execute that already has its test
	os.MkdirAll(filepath.Join(b.bundleDir("B-9"), "tests"), 0o755)
	fakeJira(t, map[string][]string{"A": {"A-1", "A-2"}, "B": {"B-1", "B-9"}})

	if _, err := Poll([]*Config{a, b}, PollOpts{ModelCases: 1}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(log)
	if strings.Fields(string(got))[0] != "b_9" {
		t.Errorf("the stored test goes first: %s", got)
	}
	if want := "b_9 b_1 a_1"; strings.Join(strings.Fields(string(got)), " ") != want {
		t.Errorf("run order:\n got  %s\n want %s (free, render, then the one model case)", strings.Join(strings.Fields(string(got)), " "), want)
	}
	// the held-back model case is not recorded, so the next poll takes it
	if seen := loadState(a); seen["A-1"] == "" || seen["A-2"] != "" {
		t.Errorf("A-1 ran and is remembered; A-2 waits and must not be: %v", seen)
	}
	if fileExists(filepath.Join(a.Out, "runs", "A-2.json")) {
		t.Error("a deferred case must not get a run record")
	}
	// next poll: nothing else is new except the waiting case
	if _, err := Poll([]*Config{a, b}, PollOpts{ModelCases: 1}); err != nil {
		t.Fatal(err)
	}
	got, _ = os.ReadFile(log)
	if want := "b_9 b_1 a_1 a_2"; strings.Join(strings.Fields(string(got)), " ") != want {
		t.Errorf("second poll should run only A-2: %s", got)
	}
}

func TestOnboardingGateOfOneProjectDoesNotStopTheOthers(t *testing.T) {
	jsonOut = true
	defer func() { jsonOut = false }()
	t.Setenv("ORDER_LOG", filepath.Join(t.TempDir(), "o.log"))
	reg, _ := newRegistry(t)
	reg.Projects["ABC"] = ProjectEntry{App: "MiApp", Profile: "profiles/miapp.json"}
	gatedCfg := &Config{Project: "ABC", Registry: reg, App: "MiApp"} // no baseUrl: stops at the gate
	ok := fakeProject(t, "B", "[]")
	fakeJira(t, map[string][]string{"B": {"B-1"}})
	gated, err := Poll([]*Config{gatedCfg, ok}, PollOpts{ModelCases: 1})
	if err != nil || !gated {
		t.Fatalf("the gate must be reported (gated=%v err=%v)", gated, err)
	}
	if !fileExists(filepath.Join(ok.Out, "runs", "B-1.json")) {
		t.Error("project B must still be processed")
	}
}

func TestStatusReadsTheAppsRepoAndRunRecords(t *testing.T) {
	t.Setenv("GIT_AUTHOR_NAME", "t")
	t.Setenv("GIT_AUTHOR_EMAIL", "t@t")
	t.Setenv("GIT_COMMITTER_NAME", "t")
	t.Setenv("GIT_COMMITTER_EMAIL", "t@t")
	reg, _ := newRegistry(t)
	if _, err := reg.Onboard("ABC", &Config{}); err != nil { // a real repo from the template
		t.Fatal(err)
	}
	repo := reg.RepoPath("MiApp")
	c := &Config{Project: "ABC", App: "MiApp", Home: repo, Out: filepath.Join(repo, "hydra", "out"), BaseBranch: "main"}
	writeJSON(filepath.Join(c.Out, "runs", "ABC-1.json"), Run{Key: "ABC-1", Intent: "automate", Tier: "T1-render", Status: "passed", Tokens: 5})
	writeJSON(filepath.Join(c.Out, "runs", "ABC-2.json"), Run{Key: "ABC-2", Intent: "automate", Tier: "T0-gate", Status: "needs-clarification"})
	git(repo, "checkout", "-q", "-b", "hydra/waiting")
	os.WriteFile(filepath.Join(repo, "x.txt"), []byte("x"), 0o644)
	git(repo, "add", "x.txt")
	git(repo, "commit", "-q", "-m", "x")
	git(repo, "checkout", "-q", "main")

	st := c.Status()
	if !st.Onboarded || len(st.Cases) != 2 || st.Passing != 1 || st.NeedsPerson != 1 || st.Tokens != 5 {
		t.Errorf("%+v", st)
	}
	if len(st.AwaitingReview) != 1 || st.AwaitingReview[0] != "hydra/waiting" {
		t.Errorf("a branch not merged into main is awaiting review: %v", st.AwaitingReview)
	}
	if st.CrawlStored || st.TestsOnBase != 0 {
		t.Errorf("nothing delivered or crawled yet: %+v", st)
	}
	missing := (&Config{Project: "NEW", App: "X", Home: filepath.Join(t.TempDir(), "nope"), Out: t.TempDir()}).Status()
	if missing.Onboarded {
		t.Error("an app with no repo is reported as not onboarded, not as an error")
	}
}
