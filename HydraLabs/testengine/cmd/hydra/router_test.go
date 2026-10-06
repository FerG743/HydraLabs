package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Fake tools stand in for render_pytest.py, run_generated.py and agent.py, so the tier order is tested alone.
func fakeEnv(t *testing.T, pending, runner string) Config {
	dir := t.TempDir()
	tools := filepath.Join(dir, "tools")
	os.MkdirAll(tools, 0o755)
	w := func(name, body string) { os.WriteFile(filepath.Join(tools, name), []byte(body), 0o644) }
	w("render_pytest.py", "import json,sys,os\na=sys.argv; out=a[a.index('--out')+1]\nos.makedirs(out+'/tests/App',exist_ok=True)\nprint(json.dumps([{'pending':"+pending+"}]))")
	w("run_generated.py", runner)
	w("agent.py", "import json,os,sys; os.makedirs(sys.argv[sys.argv.index('--repo')+1],exist_ok=True); print(json.dumps({'finished':True,'problems':[],'summary':'ok','tokens':1234}))")
	c := Config{Out: filepath.Join(dir, "out"), ToolsDir: tools, AppMap: dir, Python: "python3", App: "App", Lang: "en",
		PortalURL: "http://x", Agent: filepath.Join(tools, "agent.py"), Model: "m", MaxTurns: 5}
	html := "<ul><li>Steps:<ul><li>Do a thing.</li></ul></li><li>Expected Result:<ul><li>It works.</li></ul></li></ul>"
	if _, err := c.Ingest(Issue{Key: "ABC-1", Summary: "TC", HTML: html}); err != nil {
		t.Fatal(err)
	}
	return c
}

const passes = "import json; print(json.dumps({'status':'passed'}))"
const failsOnce = "import json,os,sys\np=sys.argv[sys.argv.index('--out')+1]+'/ran'\nif os.path.exists(p): print(json.dumps({'status':'passed'}))\nelse:\n  open(p,'w').close(); print(json.dumps({'status':'failed','error':'locator'})); sys.exit(1)"
const blocked = "import json,sys; print(json.dumps({'status':'failed','error':'x','aborted_writes':['POST /a']})); sys.exit(1)"

func TestT1PassesWithZeroTokens(t *testing.T) {
	r := fakeEnv(t, "[]", passes).Route("ABC-1", "automate")
	if r.Tier != "T1-render" || r.Status != "passed" || r.Tokens != 0 {
		t.Errorf("%+v", r)
	}
}

func TestUnmappedStepEscalatesToAgentAndLogsTokens(t *testing.T) {
	r := fakeEnv(t, "['1: weird step']", passes).Route("ABC-1", "automate")
	if r.Tier != "T3-agent" || r.Status != "passed" || r.Tokens != 1234 {
		t.Errorf("%+v", r)
	}
}

// the agent is called once; if its output also fails the independent run, the case goes to a human, not to a retry loop
func TestFailingT1OutputEscalatesOnceThenReview(t *testing.T) {
	r := fakeEnv(t, "[]", failsOnce).Route("ABC-1", "automate")
	if r.Tier != "T4-review" || r.Tokens != 1234 {
		t.Errorf("%+v", r)
	}
}

func TestWriteBlockedDoesNotSpendTokens(t *testing.T) {
	r := fakeEnv(t, "[]", blocked).Route("ABC-1", "automate")
	if r.Tier != "T1-render" || r.Status != "needs-writes" || r.Tokens != 0 {
		t.Errorf("%+v", r)
	}
}

func TestNoModelTierStopsAtReview(t *testing.T) {
	c := fakeEnv(t, "['1: weird']", passes)
	c.Agent = ""
	if r := c.Route("ABC-1", "automate"); r.Tier != "T4-review" || r.Tokens != 0 {
		t.Errorf("%+v", r)
	}
}

func TestExecuteRunsStoredBundle(t *testing.T) {
	c := fakeEnv(t, "[]", passes)
	c.Route("ABC-1", "automate")
	if r := c.Route("ABC-1", "execute"); r.Tier != "T0-existing" || r.Status != "passed" {
		t.Errorf("%+v", r)
	}
}

func TestMaturityGateStopsBeforeAnyTool(t *testing.T) {
	c := fakeEnv(t, "[]", passes)
	html := "<ul><li>Steps:<ul><li>Open the page.</li><li>TBD</li></ul></li></ul>"
	c.Ingest(Issue{Key: "ABC-2", Summary: "TC", HTML: html})
	r := c.Route("ABC-2", "automate")
	if r.Tier != "T0-gate" || r.Status != "needs-clarification" || r.Tokens != 0 {
		t.Fatalf("%+v", r)
	}
	if len(r.Detail) < 2 { // placeholder step + missing expected result, each phrased as a question
		t.Errorf("want specific questions, got %v", r.Detail)
	}
}

func TestMaturityAcceptsCompleteCase(t *testing.T) {
	c := &Case{Steps: []Step{{"1", "Open the page."}}, Expected: []Expected{{"It loads."}}, Precondition: "Logged in."}
	if bad, adv := Maturity(c); len(bad) != 0 || len(adv) != 0 {
		t.Errorf("bad=%v advice=%v", bad, adv)
	}
}

const assertion = "import json,sys; print(json.dumps({'status':'failed','error':'AssertionError: button should be enabled'})); sys.exit(1)"

func TestAssertionFailureIsReportedNotEscalated(t *testing.T) {
	r := fakeEnv(t, "[]", assertion).Route("ABC-1", "automate")
	if r.Tier != "T1-render" || r.Status != "failed" || r.Tokens != 0 {
		t.Fatalf("a possible app bug must stop at T1 with no model: %+v", r)
	}
	if !strings.Contains(strings.Join(r.Detail, "\n"), "possible app bug") {
		t.Errorf("%v", r.Detail)
	}
}
