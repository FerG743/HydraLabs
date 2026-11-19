package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoveDryRunChangesNothing(t *testing.T) {
	reg, root := newRegistry(t)
	if _, err := reg.Onboard("ABC", &Config{}); err != nil {
		t.Fatal(err)
	}
	rm, err := reg.Remove("ABC", true, false)
	if err != nil || rm.Applied || len(rm.Planned) == 0 {
		t.Fatalf("%+v %v", rm, err)
	}
	if !fileExists(filepath.Join(root, "Hydra-MiApp")) || !fileExists(reg.abs("profiles/miapp.json")) {
		t.Fatal("dry run deleted something")
	}
}

func TestRemoveAppliedDropsRepoProfileAndRegistrationKeepingOtherKeys(t *testing.T) {
	reg, root := newRegistry(t)
	if _, err := reg.Onboard("ABC", &Config{}); err != nil {
		t.Fatal(err)
	}
	reg.path = filepath.Join(reg.dir, "projects.json")
	os.WriteFile(reg.path, []byte(`{"_doc":"keep me","projects":{"ABC":{"app":"MiApp","profile":"profiles/miapp.json"},"XYZ":{"app":"Other","profile":"profiles/other.json"}}}`), 0o644)
	if _, err := reg.Remove("ABC", true, true); err != nil {
		t.Fatal(err)
	}
	if fileExists(filepath.Join(root, "Hydra-MiApp")) || fileExists(reg.abs("profiles/miapp.json")) {
		t.Error("repo or profile survived")
	}
	b, _ := os.ReadFile(reg.path)
	if strings.Contains(string(b), `"ABC"`) || !strings.Contains(string(b), `"XYZ"`) || !strings.Contains(string(b), "keep me") {
		t.Errorf("registry after removal:\n%s", b)
	}
}

func TestRemoveRefusesARepoWithUncommittedWork(t *testing.T) {
	reg, root := newRegistry(t)
	if _, err := reg.Onboard("ABC", &Config{}); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(root, "Hydra-MiApp", "unsaved.py"), []byte("x"), 0o644)
	if _, err := reg.Remove("ABC", true, true); err == nil {
		t.Fatal("removal must refuse a dirty repo")
	}
	if !fileExists(filepath.Join(root, "Hydra-MiApp")) {
		t.Fatal("dirty repo was deleted")
	}
}

func TestRemoveEntryWithoutProfileNeverTouchesTheRegistryFolder(t *testing.T) {
	reg, _ := newRegistry(t)
	reg.path = filepath.Join(reg.dir, "projects.json")
	reg.Projects["ABC"] = ProjectEntry{App: "MiApp"}
	os.WriteFile(reg.path, []byte(`{"projects":{"ABC":{"app":"MiApp"}}}`), 0o644)
	if _, err := reg.Remove("ABC", false, true); err != nil {
		t.Fatal(err)
	}
	if !fileExists(reg.dir) {
		t.Fatal("the registry folder was removed")
	}
}
