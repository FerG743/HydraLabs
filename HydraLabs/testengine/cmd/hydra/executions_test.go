package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExecutionsAreAppendedNeverRewritten(t *testing.T) {
	c := Config{Out: t.TempDir(), App: "App", Project: "ABC", PortalURL: "http://qa", Command: "run"}
	c.logExecution(Execution{Key: "ABC-1", Status: "failed", Detail: []string{"env-down"}})
	c.Trigger, c.AppVersion = "deploy", "build-42"
	c.logExecution(Execution{Key: "ABC-1", Status: "passed"})
	c.logExecution(Execution{Key: "ABC-2", Status: "passed", Command: "deliver", Attempt: 1, Of: 3})

	got := c.executions("ABC-1")
	if len(got) != 2 || got[0].Status != "failed" || got[1].Status != "passed" {
		t.Fatalf("both runs of ABC-1, oldest first: %+v", got)
	}
	if got[0].Trigger != "run" || got[1].Trigger != "deploy" || got[1].AppVersion != "build-42" {
		t.Errorf("trigger defaults to the command, CI can name it and the build: %+v", got)
	}
	if got[0].Project != "ABC" || got[0].Env != "http://qa" || got[0].At == "" || got[0].Host == "" {
		t.Errorf("shared fields not filled: %+v", got[0])
	}
	if all := c.executions(""); len(all) != 3 || all[2].Command != "deliver" || all[2].Of != 3 {
		t.Errorf("every case, and an explicit command wins: %+v", all)
	}
}

func TestADamagedLineDoesNotHideTheRest(t *testing.T) {
	c := Config{Out: t.TempDir(), App: "App"}
	c.logExecution(Execution{Key: "A-1", Status: "passed"})
	f, _ := os.OpenFile(c.executionsPath(), os.O_APPEND|os.O_WRONLY, 0o644)
	f.WriteString("{not json\n")
	f.Close()
	c.logExecution(Execution{Key: "A-1", Status: "failed"})
	if got := c.executions("A-1"); len(got) != 2 {
		t.Errorf("a damaged line is skipped, not fatal: %+v", got)
	}
}

func TestStatusCountsEachCasesHistory(t *testing.T) {
	c := Config{Out: t.TempDir(), App: "App"}
	writeJSON(filepath.Join(c.Out, "runs", "A-1.json"), Run{Key: "A-1", Status: "passed"})
	for _, s := range []string{"failed", "passed", "passed"} {
		c.logExecution(Execution{Key: "A-1", Status: s})
	}
	st := c.Status()
	if len(st.Cases) != 1 || st.Cases[0].Runs != 3 || st.Cases[0].RunsOK != 2 {
		t.Errorf("history 2/3 expected: %+v", st.Cases)
	}
}
