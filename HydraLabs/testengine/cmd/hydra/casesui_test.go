package main

import (
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
)

func casesRepo(t *testing.T) Config {
	t.Helper()
	home := t.TempDir()
	if out, err := exec.Command("git", "-C", home, "init", "-q", "-b", "main").CombinedOutput(); err != nil {
		t.Skip("no git:", string(out))
	}
	git(home, append(gitIdentity(home), "commit", "-q", "--allow-empty", "-m", "baseline")...)
	return Config{Home: home, Project: "SHOP", BaseBranch: "main", Lang: "en"}
}

func TestTheCasesPageWritesAndCommitsAFilePollReads(t *testing.T) {
	c := casesRepo(t)
	f := caseForm{Name: "Open the orders", Precondition: "The QA user\nis signed in.", Steps: []string{"Navigate to /orders.", "  ", `Click "New order".`},
		Expected: []string{`The page shows "Orders".`}}
	key, ask, err := c.saveCase(f)
	if err != nil || key != "SHOP-1" || len(ask) != 0 {
		t.Fatalf("%s %v %v", key, ask, err)
	}
	if st, _ := git(c.Home, "status", "--porcelain"); st != "" {
		t.Errorf("saved means committed (delivery needs a clean repo): %q", st)
	}
	issues, _ := localIssues(c.localCaseDir(), "")
	back, cs := formOf(issues[0], "en")
	if back.Name != f.Name || back.Precondition != "The QA user is signed in." || len(back.Steps) != 2 || back.Steps[1] != `Click "New order".` || back.Intent != "automate" || len(cs.Warnings) != 0 {
		t.Errorf("what the page saves must read back the same (blank steps dropped, lines joined): %+v %v", back, cs.Warnings)
	}

	back.Steps = append(back.Steps, "Click \"Orders\".")
	if key, _, err := c.saveCase(back); err != nil || key != "SHOP-1" {
		t.Errorf("editing keeps the key: %s %v", key, err)
	}
	if key, ask, _ := c.saveCase(caseForm{Name: "Later", Steps: []string{"TBD"}, Expected: []string{"ok"}}); key != "SHOP-2" || len(ask) == 0 {
		t.Errorf("a new case gets the next number, and what blocks automation is said at once: %s %v", key, ask)
	}
	if log, _ := git(c.Home, "log", "--oneline"); strings.Count(log, "case SHOP-") != 3 {
		t.Errorf("one commit per save: %s", log)
	}
	if _, _, err := c.saveCase(caseForm{Name: "x", Steps: []string{"Navigate to /."}}); err == nil {
		t.Error("no expected result: refused")
	}
	git(c.Home, "checkout", "-q", "-b", "hydra/delivery")
	if _, _, err := c.saveCase(f); err == nil || !strings.Contains(err.Error(), "delivery") {
		t.Errorf("never commit onto a delivery branch: %v", err)
	}
}

func TestTheCasesPageAnswersOnlyLocalhost(t *testing.T) {
	h := localOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	for host, want := range map[string]int{"localhost:8098": 200, "127.0.0.1:8098": 200, "evil.example:8098": 403} {
		rec := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		r.Host = host
		h.ServeHTTP(rec, r)
		if rec.Code != want {
			t.Errorf("%s: %d", host, rec.Code)
		}
	}
}

func TestCaseKeysSortAsNumbers(t *testing.T) {
	if !keyLess("SHOP-2", "SHOP-10") || keyLess("SHOP-10", "SHOP-2") {
		t.Error("SHOP-2 comes before SHOP-10")
	}
	if nextKey("SHOP", []Issue{{Key: "SHOP-2"}, {Key: "SHOP-10"}, {Key: "OTHER-99"}}) != "SHOP-11" {
		t.Error("next after the highest of this project")
	}
}
