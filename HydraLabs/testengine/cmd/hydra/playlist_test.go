package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildPlaylistRendersTheFlowIntoOneFolder(t *testing.T) {
	home := t.TempDir()
	c := Config{Home: home, Out: filepath.Join(home, "out"), App: "App", Python: "python3",
		ToolsDir: "../../tools/jira"}
	os.MkdirAll(c.caseDir(), 0o755)
	os.MkdirAll(c.durable("playlists"), 0o755)
	for _, id := range []string{"DWQ-901", "DWQ-902"} {
		body := `{"id":"` + id + `","name":"TC - caso ` + id + `","status":"ready","steps":[{"n":"1","action":"Navigate to /x"}],"expected":[{"text":"ok"}]}`
		os.WriteFile(filepath.Join(c.caseDir(), id+".case.json"), []byte(body), 0o644)
	}
	flow := `{"id":"flow","steps":[{"case":"DWQ-901","captures":{"order_id":"ORDER_NUMBER"}},{"case":"DWQ-902"}]}`
	os.WriteFile(c.durable("playlists", "flow.json"), []byte(flow), 0o644)

	folder, casesFile, flowFile, err := c.buildPlaylist("flow")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(folder, "tests", "App", "test_dwq_901.py")); err != nil {
		t.Fatalf("first case not rendered: %v", err)
	}
	if _, err := os.Stat(filepath.Join(folder, "tests", "App", "test_dwq_902.py")); err != nil {
		t.Fatalf("second case not rendered: %v", err)
	}
	if casesFile != filepath.Join(folder, "cases.json") || flowFile != c.durable("playlists", "flow.json") {
		t.Fatalf("unexpected paths: %s %s", casesFile, flowFile)
	}
	b, _ := os.ReadFile(casesFile)
	if !strings.Contains(string(b), `"status":"ready"`) || !strings.Contains(string(b), "DWQ-902") {
		t.Fatalf("cases file: %s", b)
	}
}

func TestBoardRunsAPlaylistThroughTheSameCommand(t *testing.T) {
	args, err := boardArgs("playlist-check", "DWQ", "/r/projects.json", "order-flow")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(args, " "); got != "playlist check --name order-flow --project DWQ --registry /r/projects.json" {
		t.Fatalf("args: %s", got)
	}
	if _, err := boardArgs("playlist-run", "DWQ", "/r", "../etc"); err == nil {
		t.Fatal("a playlist name with a path was accepted")
	}
}
