package main

import (
	"strings"
	"testing"
	"time"
)

func waitIdle(t *testing.T, r *runner) runState {
	t.Helper()
	for i := 0; i < 200; i++ {
		if s := r.state(); !s.Running {
			return s
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("still running")
	return runState{}
}

func TestTheBoardRunsOneCommandAtATimeAndKeepsItsOutput(t *testing.T) {
	r := &runner{}
	if err := r.start("poll", "/bin/sh", t.TempDir(), []string{"-c", "echo mapped; sleep 0.3; echo done >&2"}); err != nil {
		t.Skip("no /bin/sh:", err)
	}
	if err := r.start("crawl", "/bin/sh", "", []string{"-c", "true"}); err == nil || !strings.Contains(err.Error(), "poll is still running") {
		t.Errorf("a second command while one runs: %v", err)
	}
	s := waitIdle(t, r)
	if s.Exit != "ok" || !strings.Contains(s.Log, "mapped") || !strings.Contains(s.Log, "done") || s.Label != "poll" {
		t.Errorf("finished run: %+v", s)
	}
	r.start("crawl", "/bin/sh", "", []string{"-c", "sleep 5"})
	if !r.stop() {
		t.Fatal("stop must find the running command")
	}
	if s := waitIdle(t, r); s.Exit != "stopped" {
		t.Errorf("a stopped run says so: %+v", s)
	}
	if r.stop() {
		t.Error("nothing to stop when idle")
	}
	r.start("x", "/bin/sh", "", []string{"-c", "exit 3"})
	if s := waitIdle(t, r); s.Exit != "exit status 3" {
		t.Errorf("a failing command: %+v", s)
	}
}

func TestOnlyTheBoardsOwnCommandsCanRun(t *testing.T) {
	for _, tc := range [][3]string{
		{"automate", "SHOP-4", "poll --project SHOP --registry /r --only SHOP-4 --intent automate"},
		{"execute", "SHOP-4", "poll --project SHOP --registry /r --only SHOP-4 --intent execute"},
		{"deliver", "", "deliver --project SHOP --registry /r"},
		{"poll", "", "poll --project SHOP --registry /r"},
	} {
		if a, err := boardArgs(tc[0], "SHOP", "/r", tc[1]); err != nil || strings.Join(a, " ") != tc[2] {
			t.Errorf("%s %s: %v %v", tc[0], tc[1], a, err)
		}
	}
	for _, bad := range [][2]string{{"init", ""}, {"automate", "SHOP-1 --force"}, {"execute", "--registry=/etc"}, {"execute", ""}, {"rm", ""}} {
		if _, err := boardArgs(bad[0], "SHOP", "/r", bad[1]); err == nil {
			t.Errorf("must refuse %q %q", bad[0], bad[1])
		}
	}
}

func TestTheCLIAutomatesOrExecutesOneCaseByItsKey(t *testing.T) {
	if a, err := oneCase("execute", []string{"dwq-l1", "--json"}); err != nil || strings.Join(a, " ") != "--json --only DWQ-L1 --intent execute" {
		t.Errorf("%v %v", a, err)
	}
	if a, _ := oneCase("automate", []string{"--project", "DWQ", "DWQ-131"}); strings.Join(a, " ") != "--project DWQ --only DWQ-131 --intent automate" {
		t.Errorf("%v", a)
	}
	if _, err := oneCase("automate", []string{"--json"}); err == nil || !strings.Contains(err.Error(), "usage") {
		t.Errorf("no case key: say how to call it: %v", err)
	}
}
