package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// The execution log: one JSON line per execution, appended and never rewritten, so a case's history survives every
// later run (runs/<KEY>.json only keeps the latest). It lives in hydra/out/ (git-ignored): a log that grows with
// every run cannot ride on review branches.
// shortcut: one file on the machine that runs Hydra; ship the same records to a shared store (ReportPortal, Xray...)
// when more than one runner or a team-wide view is needed.

type Execution struct {
	At          string   `json:"at"` // RFC 3339, local time
	Project     string   `json:"project,omitempty"`
	App         string   `json:"app"`
	Key         string   `json:"key"`
	Command     string   `json:"command"`           // run | poll | deliver
	Trigger     string   `json:"trigger"`           // why it ran: manual, poll, deliver, or what CI passes (deploy, nightly...)
	Intent      string   `json:"intent,omitempty"`  // automate | execute
	Tier        string   `json:"tier,omitempty"`    // which router tier finished it
	Status      string   `json:"status"`            // the run's or the framework's verdict, unchanged
	Detail      []string `json:"detail,omitempty"`  // the reason, as reported
	Attempt     int      `json:"attempt,omitempty"` // deliver: stability run n of Of
	Of          int      `json:"of,omitempty"`
	Millis      int64    `json:"ms,omitempty"`
	Tokens      int      `json:"tokens"`
	Env         string   `json:"env,omitempty"`         // base URL of the app under test
	AppVersion  string   `json:"appVersion,omitempty"`  // the build under test, when the caller knows it (--app-version)
	TestVersion string   `json:"testVersion,omitempty"` // commit of the app's test repo the run used
	Host        string   `json:"host,omitempty"`
	Artifacts   []string `json:"artifacts,omitempty"` // logs, junit, reports
}

func (c Config) executionsPath() string { return filepath.Join(c.Out, "executions.jsonl") }

// logExecution fills what every record shares and appends it. A failure to record is reported, never fatal: the
// run itself already happened and its own outputs are intact.
func (c Config) logExecution(e Execution) {
	e.At = time.Now().Format(time.RFC3339)
	e.Project, e.App, e.Env, e.AppVersion = c.Project, c.App, c.PortalURL, c.AppVersion
	if e.Command == "" {
		e.Command = c.Command
	}
	if e.Trigger == "" {
		e.Trigger = c.Trigger
	}
	if e.Trigger == "" {
		e.Trigger = e.Command
	}
	if e.TestVersion == "" && c.Home != "" {
		e.TestVersion, _ = git(c.Home, "rev-parse", "--short", "HEAD")
	}
	e.Host, _ = os.Hostname()
	b, _ := json.Marshal(e)
	os.MkdirAll(c.Out, 0o755)
	f, err := os.OpenFile(c.executionsPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err == nil {
		_, err = f.Write(append(b, '\n'))
		f.Close()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "hydra: execution not recorded (%s %s): %v\n", e.Key, e.Status, err)
	}
}

// executions reads the log, oldest first; key "" means every case. A damaged line is skipped, not fatal.
func (c Config) executions(key string) []Execution {
	f, err := os.Open(c.executionsPath())
	if err != nil {
		return nil
	}
	defer f.Close()
	var out []Execution
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024) // a detail can carry a long pytest trace
	for sc.Scan() {
		var e Execution
		if json.Unmarshal(sc.Bytes(), &e) == nil && (key == "" || e.Key == key) {
			out = append(out, e)
		}
	}
	return out
}

func printHistory(es []Execution, last int, asJSON bool) {
	if last > 0 && len(es) > last {
		es = es[len(es)-last:]
	}
	if asJSON {
		b, _ := json.MarshalIndent(es, "", "  ")
		fmt.Println(string(b))
		return
	}
	if len(es) == 0 {
		fmt.Println("no executions recorded yet")
		return
	}
	for _, e := range es {
		at := e.At
		if t, err := time.Parse(time.RFC3339, e.At); err == nil {
			at = t.Format("2006-01-02 15:04")
		}
		run := e.Command
		if e.Of > 0 {
			run = fmt.Sprintf("%s %d/%d", e.Command, e.Attempt, e.Of)
		}
		why := ""
		if len(e.Detail) > 0 {
			why = strings.SplitN(e.Detail[0], "\n", 2)[0]
			if len(why) > 70 {
				why = why[:70] + "…"
			}
		}
		fmt.Printf("%s  %-9s %-12s %-10s %-14s %6dms %6d tokens  test@%-8s %s\n",
			at, e.Key, run, e.Trigger, e.Status, e.Millis, e.Tokens, e.TestVersion, why)
	}
}
