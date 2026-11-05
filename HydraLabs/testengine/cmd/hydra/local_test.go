package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const signIn = `# Sign in with a valid user
labels: automate

- Preconditions: The QA user exists.
- Steps:
  1. Navigate to /login.
  2. Enter the user and click "Entrar"
     with the ` + "`qa@shop.test`" + ` account.
- Expected result:
  - The home page shows <Orders>.
`

func TestALocalCaseParsesLikeAJiraOne(t *testing.T) {
	is := mdIssue(signIn)
	cs, err := ToCase(is.HTML, "SHOP-1", is.Summary, "en", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if cs.Name != "Sign in with a valid user" || cs.Precondition != "The QA user exists." || len(cs.Warnings) != 0 {
		t.Errorf("%+v", cs)
	}
	if len(cs.Steps) != 2 || cs.Steps[1].Action != "Enter the user and click \"Entrar\" with the `qa@shop.test` account." {
		t.Errorf("steps (a wrapped line continues its step, code kept as a value anchor): %+v", cs.Steps)
	}
	if len(cs.Expected) != 1 || cs.Expected[0].Text != "The home page shows <Orders>." {
		t.Errorf("expected: %+v", cs.Expected)
	}
	if Intent(is.Labels) != "automate" || Intent(mdIssue("# x\nlabels: execute\n").Labels) != "execute" || Intent(mdIssue("# x\n").Labels) != "automate" {
		t.Error("labels: line sets the intent; without one a dropped-in file is automated")
	}
}

func TestLocalCasesArePolledWithoutJira(t *testing.T) {
	home := t.TempDir()
	c := &Config{Home: home, Out: filepath.Join(home, "hydra", "out"), Lang: "en", CaseSource: "local"}
	os.MkdirAll(c.localCaseDir(), 0o755)
	os.WriteFile(filepath.Join(c.localCaseDir(), "README.md"), []byte(localCasesReadme), 0o644)
	os.WriteFile(filepath.Join(c.localCaseDir(), "shop-1.md"), []byte(signIn), 0o644)

	jobs, seen, err := collect(c, nil, PollOpts{}) // no Jira client at all
	if err != nil || len(jobs) != 1 || jobs[0].key != "SHOP-1" || jobs[0].intent != "automate" {
		t.Fatalf("%v %+v", err, jobs)
	}
	if cs, _, err := c.resolved("SHOP-1"); err != nil || len(cs.Steps) != 2 {
		t.Errorf("the case must be stored for the router: %v", err)
	}
	seen["SHOP-1"] = jobs[0].stamp
	writeJSON(statePath(c), seen)
	if jobs, _, _ := collect(c, nil, PollOpts{}); len(jobs) != 0 {
		t.Error("an unchanged file is not run again")
	}
	os.WriteFile(filepath.Join(c.localCaseDir(), "shop-1.md"), []byte(strings.Replace(signIn, "/login", "/signin", 1)), 0o644)
	if jobs, _, _ := collect(c, nil, PollOpts{}); len(jobs) != 1 {
		t.Error("an edited file is a changed case: it runs again")
	}
	if jobs, _, err := collect(c, nil, PollOpts{Only: "SHOP-1"}); err != nil || len(jobs) != 1 {
		t.Errorf("--only runs it even when seen: %v", err)
	}
	if _, _, err := collect(c, nil, PollOpts{Only: "SHOP-9"}); err == nil {
		t.Error("--only with a missing file must say so")
	}
	os.WriteFile(filepath.Join(c.localCaseDir(), "login test.md"), []byte(signIn), 0o644)
	if _, _, err := collect(c, nil, PollOpts{}); err == nil || !strings.Contains(err.Error(), "SHOP-1.md") {
		t.Errorf("a file that is not named for a case is an error, not silently skipped: %v", err)
	}
}

func TestPollWithOnlyLocalProjectsNeedsNoJiraToken(t *testing.T) {
	t.Setenv("JIRA_BASE_URL", "")
	t.Setenv("JIRA_TOKEN", "")
	home := t.TempDir()
	c := &Config{Home: home, Out: filepath.Join(home, "hydra", "out"), CaseSource: "local"}
	if _, err := Poll([]*Config{c}, PollOpts{}); err != nil {
		t.Errorf("an empty cases/ folder is nothing new, not a Jira error: %v", err)
	}
}

func TestAJiraProjectAlsoRunsTheCasesWrittenInItsRepo(t *testing.T) {
	fakeJira(t, map[string][]string{"DWQ": {"DWQ-1", "DWQ-2"}})
	j, err := JiraFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	c := &Config{Home: home, Out: filepath.Join(home, "hydra", "out"), Lang: "en", Project: "DWQ", JQL: "project = DWQ"}
	os.MkdirAll(c.localCaseDir(), 0o755)
	os.WriteFile(filepath.Join(c.localCaseDir(), "DWQ-L1.md"), []byte(signIn), 0o644)

	jobs, _, err := collect(c, j, PollOpts{JQL: c.JQL})
	if err != nil || keys(jobs) != "DWQ-1 DWQ-2 DWQ-L1" {
		t.Fatalf("Jira's and the repo's cases together: [%s] %v", keys(jobs), err)
	}

	t.Setenv("JIRA_BASE_URL", "http://127.0.0.1:1") // Jira unreachable: a local case asked for by name does not need it
	j2, _ := JiraFromEnv()
	if jobs, _, err := collect(c, j2, PollOpts{Only: "DWQ-L1"}); err != nil || keys(jobs) != "DWQ-L1" {
		t.Errorf("--only on a file reads the file: [%s] %v", keys(jobs), err)
	}

	os.WriteFile(filepath.Join(c.localCaseDir(), "DWQ-2.md"), []byte(signIn), 0o644)
	if _, _, err := collect(c, j, PollOpts{JQL: c.JQL}); err == nil || !strings.Contains(err.Error(), "rename the file") {
		t.Errorf("a file named like a Jira issue would mix two cases' runs and tests: %v", err)
	}
}
