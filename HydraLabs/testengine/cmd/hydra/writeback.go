package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Write-back: the pipeline tells Jira what it finished, so the board is the record (profile: jiraComments).
// Delivered cases get a result comment; cases a delivery was blocked on get the reason. Never twice for the same news.

func (c Config) testFile(key string) string {
	return filepath.Join("tests", c.App, "test_"+token(key)+".py")
}

func (c Config) runOf(key string) Run {
	var r Run
	if b, err := os.ReadFile(filepath.Join(c.Out, "runs", key+".json")); err == nil {
		json.Unmarshal(b, &r)
	}
	return r
}

// writeBack posts after a delivery. It never fails the delivery: Jira being unreachable is reported, not fatal.
func (c Config) writeBack(d Delivery) {
	if !c.JiraComments {
		return
	}
	j, err := JiraFromEnv()
	if err != nil {
		fmt.Fprintln(os.Stderr, "jira write-back skipped:", err)
		return
	}
	repoName := filepath.Base(expand(c.DeliverRepo))
	for _, k := range d.Delivered {
		if !c.inJira(k) {
			continue
		}
		msg := DeliveredMessage(c.runOf(k), d, repoName, c.testFile(k), d.Runs)
		if ok, e := c.PostOnce(j, k, shortID("delivered", d.Commit), msg); e != nil {
			fmt.Fprintln(os.Stderr, "comment on", k, "failed:", e)
		} else if ok {
			fmt.Fprintln(os.Stderr, "commented on", k)
		}
	}
	for _, o := range d.Outcomes {
		if d.Gate == "" || o.Status == "passed" || !c.inJira(o.Key) {
			continue // only the cases that blocked the delivery are news
		}
		msg := BlockedMessage(o, d.Runs)
		if ok, e := c.PostOnce(j, o.Key, shortID("blocked", o.Status, o.Detail), msg); e != nil {
			fmt.Fprintln(os.Stderr, "comment on", o.Key, "failed:", e)
		} else if ok {
			fmt.Fprintln(os.Stderr, "commented on", o.Key)
		}
	}
}

// Announce backfills the delivered comment for cases that are already on the base branch, using the commit that
// added each test. Without --post it only prints what it would say.
func (c Config) Announce(only []string, post bool) error {
	repo := expand(c.DeliverRepo)
	keys, err := c.passedKeys(only, map[string]bool{})
	if err != nil || len(keys) == 0 {
		return fmt.Errorf("no passing cases to announce (%v)", err)
	}
	var j *Jira
	if post {
		if j, err = JiraFromEnv(); err != nil {
			return err
		}
	}
	for _, k := range keys {
		commit, e := git(repo, "log", c.BaseBranch, "--diff-filter=A", "--format=%h", "-1", "--", c.testFile(k))
		if e != nil || commit == "" {
			fmt.Printf("%s: not on %s yet, nothing to announce\n", k, c.BaseBranch)
			continue
		}
		d := Delivery{Branch: c.BaseBranch, Commit: commit}
		msg := DeliveredMessage(c.runOf(k), d, filepath.Base(repo), c.testFile(k), c.StabilityRuns)
		if !post {
			fmt.Printf("--- would post on %s:\n%s\n", k, strings.TrimSpace(msg))
			continue
		}
		if ok, e := c.PostOnce(j, k, shortID("delivered", commit), msg); e != nil {
			fmt.Printf("%s: failed: %v\n", k, e)
		} else if ok {
			fmt.Printf("%s: commented\n", k)
		} else {
			fmt.Printf("%s: already announced\n", k)
		}
	}
	return nil
}

// inJira: only issues of the project's Jira project get comments; cases fed from files (PQA-1) are not Jira issues.
func (c Config) inJira(key string) bool {
	p := c.JiraProject
	if p == "" {
		p = c.Project
	}
	return p == "" || strings.HasPrefix(key, p+"-")
}
