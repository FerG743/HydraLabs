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
	if a, err := boardArgs("poll", "SHOP", "/r/projects.json", "SHOP-4"); err != nil || strings.Join(a, " ") != "poll --project SHOP --registry /r/projects.json --only SHOP-4" {
		t.Errorf("%v %v", a, err)
	}
	if a, _ := boardArgs("deliver", "SHOP", "/r", ""); strings.Join(a, " ") != "deliver --project SHOP --registry /r" {
		t.Errorf("%v", a)
	}
	for _, bad := range [][2]string{{"init", ""}, {"poll", "SHOP-1 --force"}, {"poll", "--registry=/etc"}, {"rm", ""}} {
		if _, err := boardArgs(bad[0], "SHOP", "/r", bad[1]); err == nil {
			t.Errorf("must refuse %q %q", bad[0], bad[1])
		}
	}
}
