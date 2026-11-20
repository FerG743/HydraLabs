package main

import (
	"path/filepath"
	"testing"
)

// Tests run from testengine/cmd/hydra: the engine must still be found from there.
func TestEngineDirFoundFromASubfolder(t *testing.T) {
	if !fileExists(filepath.Join(engineDir(), "tools", "jira", "render_pytest.py")) {
		t.Fatalf("engine dir %s has no render_pytest.py", engineDir())
	}
}
