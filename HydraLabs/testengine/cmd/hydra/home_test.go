package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppOutputsLiveInTheAppsRepo(t *testing.T) {
	c := Config{Out: "/pipeline/out", AppMap: "/pipeline/appmap", Home: "/repos/Hydra-App", App: "App"}
	c.applyHome("/pipeline/out", "/pipeline/appmap")
	if c.Out != "/repos/Hydra-App/hydra/out" || c.AppMap != "/repos/Hydra-App/hydra/out/appmap" {
		t.Errorf("working files belong in the repo's git-ignored hydra/out: %s %s", c.Out, c.AppMap)
	}
	if c.notesPath() != "/repos/Hydra-App/hydra/knowledge.md" || c.handPath() != "/repos/Hydra-App/hydra/locators/confirmed.json" {
		t.Errorf("what a person writes is committed in hydra/: %s %s", c.notesPath(), c.handPath())
	}
}

func TestAnExplicitOutIsRespected(t *testing.T) {
	c := Config{Out: "/custom/out", AppMap: "/custom/map", Home: "/repos/Hydra-App"}
	c.applyHome("/pipeline/out", "/pipeline/appmap")
	if c.Out != "/custom/out" || c.AppMap != "/custom/map" {
		t.Errorf("a path chosen in the profile or by flag must not be overridden: %s %s", c.Out, c.AppMap)
	}
}

func TestWithoutAHomeNothingChanges(t *testing.T) {
	c := Config{Out: "/pipeline/out", AppMap: "/pipeline/appmap", Knowledge: "/pipeline/knowledge", App: "App"}
	c.applyHome("/pipeline/out", "/pipeline/appmap")
	if c.Out != "/pipeline/out" || c.notesPath() != "/pipeline/knowledge/App.md" || c.handPath() != "/pipeline/appmap/App.json" {
		t.Errorf("the --profile mode keeps the old layout: %+v", c)
	}
}

func TestAFreshCloneIsSeededFromTheCommittedFiles(t *testing.T) {
	home := t.TempDir()
	c := Config{Home: home, App: "App", AppMap: filepath.Join(home, "hydra", "out", "appmap")}
	os.MkdirAll(filepath.Join(home, "hydra", "locators"), 0o755)
	os.WriteFile(c.durable("locators", "learned.json"), []byte(`{"locators":{"A":"#a"}}`), 0o644)
	c.seedWorking()
	if b, err := os.ReadFile(c.learnedPath()); err != nil || !strings.Contains(string(b), "#a") {
		t.Fatalf("working learned file not seeded: %v", err)
	}
	os.WriteFile(c.learnedPath(), []byte(`{"locators":{"A":"#newer"}}`), 0o644)
	c.seedWorking() // must never overwrite a working copy that already exists
	if b, _ := os.ReadFile(c.learnedPath()); !strings.Contains(string(b), "#newer") {
		t.Error("seeding overwrote the working copy")
	}
}

func TestOldReposGetTheIgnoreRuleLocally(t *testing.T) {
	repo := t.TempDir()
	exec.Command("git", "-C", repo, "init", "-q", "-b", "main").Run()
	if _, err := git(repo, "check-ignore", "-q", "hydra/out/x"); err == nil {
		t.Fatal("precondition: not ignored yet")
	}
	ensureIgnored(repo)
	ensureIgnored(repo) // idempotent
	if _, err := git(repo, "check-ignore", "-q", "hydra/out/x"); err != nil {
		t.Error("hydra/out/ must be ignored now")
	}
	b, _ := os.ReadFile(filepath.Join(repo, ".git", "info", "exclude"))
	if strings.Count(string(b), "hydra/out/") != 1 {
		t.Errorf("rule added once, not twice: %q", b)
	}
	if st, _ := git(repo, "status", "--porcelain"); st != "" {
		t.Errorf("a local exclude must not touch tracked files: %q", st)
	}
}

func TestDeliveryCarriesLearnedLocatorsAndTheCaseText(t *testing.T) {
	home, out := t.TempDir(), t.TempDir()
	c := Config{Home: home, Out: out, App: "App", AppMap: filepath.Join(out, "appmap"), Python: "python3", ToolsDir: "../../tools/jira"}
	os.MkdirAll(c.AppMap, 0o755)
	os.WriteFile(c.learnedPath(), []byte(`{"locators":{"A":"#a"}}`), 0o644)
	writeJSON(filepath.Join(c.caseDir(), "ABC-1.case.json"), map[string]any{"id": "ABC-1", "name": "TC-1 - Login", "language": "en",
		"steps": []map[string]string{{"n": "1", "action": "Click `Entrar`."}}, "expected": []map[string]string{{"text": "The home page shows."}}})
	writeJSON(filepath.Join(home, "hydra", "cases", "ABC-0.case.json"), map[string]any{"id": "ABC-0", "name": "TC-0 - Old", "language": "en",
		"steps": []map[string]string{{"n": "1", "action": "Open the app."}}, "expected": []map[string]string{}}) // delivered before features existed
	paths, err := c.stageDurable(home, []string{"ABC-1", "ABC-NOPE"})
	if err != nil || strings.Join(paths, ",") != "hydra/locators,hydra/cases" {
		t.Fatalf("%v %v", paths, err)
	}
	for _, f := range []string{"hydra/locators/learned.json", "hydra/cases/ABC-1.case.json", "hydra/cases/ABC-0.feature"} {
		if !fileExists(filepath.Join(home, f)) {
			t.Errorf("missing %s", f)
		}
	}
	feature, _ := os.ReadFile(filepath.Join(home, "hydra", "cases", "ABC-1.feature"))
	if !strings.Contains(string(feature), "When Click Entrar.") || !strings.Contains(string(feature), "Then The home page shows.") {
		t.Errorf("the case must be readable step by step as Gherkin:\n%s", feature)
	}
	if fileExists(filepath.Join(home, "hydra", "cases", "ABC-NOPE.case.json")) {
		t.Error("a case with no stored text must be skipped, not invented")
	}
}

func TestACommandCannotCreateAnAppsFoldersBeforeItsRepoExists(t *testing.T) {
	c := Config{Home: filepath.Join(t.TempDir(), "Hydra-New"), Project: "NEW"}
	if err := c.requireHome(); err == nil || !strings.Contains(err.Error(), "hydra onboard --project NEW") {
		t.Errorf("%v", err)
	}
	if (Config{}).requireHome() != nil {
		t.Error("the --profile mode has no home to require")
	}
}
