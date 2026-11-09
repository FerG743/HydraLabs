package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// The board's buttons: one hydra command at a time, the same binary and flags a person would type, its output kept
// for the page. Only the commands below can run; the page never sends a command line.

// boardArgs: the action a button asked for -> hydra's arguments. Automate and Execute take one case.
func boardArgs(action, project, registry, key string) ([]string, error) {
	base := []string{"--project", project, "--registry", registry}
	switch action {
	case "crawl", "deliver":
		return append([]string{action}, base...), nil
	case "poll":
		return append([]string{"poll"}, base...), nil
	case "automate", "execute": // build the case from its text / run its test as it is (with the test repo's data)
		if !caseKey.MatchString(key) {
			return nil, fmt.Errorf("%q is not a case key", key)
		}
		return append([]string{"poll"}, append(base, "--only", key, "--intent", action)...), nil
	}
	return nil, fmt.Errorf("unknown action %q", action)
}

type runner struct {
	mu      sync.Mutex
	cmd     *exec.Cmd
	label   string
	started time.Time
	ended   time.Time
	exit    string // "" while running or before the first run; "ok", "exit status 1", "stopped"
	log     []byte
}

const runLogMax = 200 << 10 // the page shows the end of the output; the start of a long run is dropped

func (r *runner) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.log = append(r.log, p...)
	if len(r.log) > runLogMax {
		r.log = r.log[len(r.log)-runLogMax:]
	}
	return len(p), nil
}

// start runs exe args in dir unless something is already running.
func (r *runner) start(label, exe, dir string, args []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cmd != nil {
		return fmt.Errorf("%s is still running: wait for it or stop it", r.label)
	}
	cmd := exec.Command(exe, args...)
	cmd.Dir, cmd.Stdout, cmd.Stderr = dir, r, r
	r.log = []byte(fmt.Sprintf("$ hydra %s\n", strings.Join(args, " ")))
	if err := cmd.Start(); err != nil {
		return err
	}
	r.cmd, r.label, r.started, r.exit = cmd, label, time.Now(), ""
	go func() {
		err := cmd.Wait()
		r.mu.Lock()
		defer r.mu.Unlock()
		switch {
		case r.exit == "stopped":
		case err != nil:
			r.exit = err.Error()
		default:
			r.exit = "ok"
		}
		r.cmd, r.ended = nil, time.Now()
	}()
	return nil
}

// stop interrupts the running command: hydra's crawl stops its browser on an interrupt.
// shortcut: no process group, so a verify subprocess of an interrupted poll finishes on its own (seconds); kill the group if that ever matters.
func (r *runner) stop() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cmd == nil {
		return false
	}
	r.exit = "stopped"
	if r.cmd.Process.Signal(os.Interrupt) != nil {
		r.cmd.Process.Kill()
	}
	return true
}

type runState struct {
	Running bool   `json:"running"`
	Label   string `json:"label"`
	Seconds int    `json:"seconds"`
	Exit    string `json:"exit"`
	Log     string `json:"log"`
}

func (r *runner) state() runState {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := runState{Running: r.cmd != nil, Label: r.label, Exit: r.exit, Log: string(r.log)}
	if s.Running {
		s.Seconds = int(time.Since(r.started).Seconds())
	} else if !r.ended.IsZero() {
		s.Seconds = int(r.ended.Sub(r.started).Seconds())
	}
	return s
}
