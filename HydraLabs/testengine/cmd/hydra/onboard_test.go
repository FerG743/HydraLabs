package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newRegistry(t *testing.T) (*Registry, string) {
	root := t.TempDir()
	tpl, _ := filepath.Abs("../../templates/framework")
	reg := &Registry{ReposRoot: root, Template: tpl, dir: t.TempDir(),
		Projects: map[string]ProjectEntry{"ABC": {App: "MiApp", Profile: "profiles/miapp.json"}}}
	return reg, root
}

func TestOnboardingCreatesTheAppsOwnRepo(t *testing.T) {
	reg, root := newRegistry(t)
	o, err := reg.Onboard("ABC", &Config{})
	if err != nil || !o.Created {
		t.Fatalf("%+v %v", o, err)
	}
	repo := filepath.Join(root, "Hydra-MiApp")
	if o.Repo != repo {
		t.Errorf("repo is named Hydra-<app>: got %s", o.Repo)
	}
	for _, f := range []string{"conftest.py", "utils/driver_wrapper.py", "apps/MiApp/data/expected_result.csv", "tests/MiApp/.gitkeep", ".gitignore"} {
		if !fileExists(filepath.Join(repo, f)) {
			t.Errorf("missing %s", f)
		}
	}
	runner, _ := os.ReadFile(filepath.Join(repo, "runner_central.py"))
	if strings.Contains(string(runner), "__APP__") || !strings.Contains(string(runner), "choices=['MiApp', 'all']") {
		t.Errorf("runner not specialized:\n%s", runner)
	}
	if n, _ := git(repo, "rev-list", "--count", "HEAD"); n != "1" {
		t.Errorf("want exactly one baseline commit, got %s", n)
	}
	if b, _ := git(repo, "branch", "--show-current"); b != "main" {
		t.Errorf("branch %s", b)
	}
	if st, _ := git(repo, "status", "--porcelain"); st != "" {
		t.Errorf("repo should be clean after onboarding: %q", st)
	}
}

func TestNewProjectStopsAtTheBaseUrlGate(t *testing.T) {
	reg, _ := newRegistry(t)
	o, _ := reg.Onboard("ABC", &Config{})
	if !strings.Contains(o.Gate, "baseUrl") {
		t.Fatalf("gate: %q", o.Gate)
	}
	b, err := os.ReadFile(reg.abs("profiles/miapp.json"))
	if err != nil || !strings.Contains(string(b), `project = ABC AND labels in`) || !strings.Contains(string(b), `"allowWrites": false`) {
		t.Errorf("default profile must be project-scoped and read-only: %s %v", b, err)
	}
}

func TestOnboardingIsIdempotent(t *testing.T) {
	reg, root := newRegistry(t)
	reg.Onboard("ABC", &Config{})
	before, _ := git(filepath.Join(root, "Hydra-MiApp"), "rev-parse", "HEAD")
	os.WriteFile(filepath.Join(root, "Hydra-MiApp", "mine.txt"), []byte("a person's work"), 0o644)
	o, err := reg.Onboard("ABC", &Config{})
	after, _ := git(filepath.Join(root, "Hydra-MiApp"), "rev-parse", "HEAD")
	if err != nil || o.Created || before != after || !fileExists(filepath.Join(root, "Hydra-MiApp", "mine.txt")) {
		t.Errorf("a second run must change nothing: %+v %v", o, err)
	}
}

func TestOnboardingNeverWritesIntoAForeignFolder(t *testing.T) {
	reg, root := newRegistry(t)
	os.MkdirAll(filepath.Join(root, "Hydra-MiApp"), 0o755)
	os.WriteFile(filepath.Join(root, "Hydra-MiApp", "notes.txt"), []byte("not ours"), 0o644)
	if _, err := reg.Onboard("ABC", &Config{}); err == nil || fileExists(filepath.Join(root, "Hydra-MiApp", "conftest.py")) {
		t.Errorf("must refuse: %v", err)
	}
}

func TestUnregisteredProjectIsRefused(t *testing.T) {
	reg, root := newRegistry(t)
	if _, err := reg.Onboard("ZZZ", &Config{}); err == nil {
		t.Error("an unregistered project must not create anything")
	}
	if entries, _ := os.ReadDir(root); len(entries) != 0 {
		t.Errorf("created %d entries", len(entries))
	}
}
