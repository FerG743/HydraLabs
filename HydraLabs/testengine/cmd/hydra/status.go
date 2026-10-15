package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// hydra status: one read-only table over every tracked project, built from what each app's repo already holds (its
// run records and its git state). No file of its own: a second copy of the state could only drift from the real one.

type CaseStatus struct {
	Key     string `json:"key"`
	Intent  string `json:"intent"`
	Tier    string `json:"tier"`
	Status  string `json:"status"`
	Tokens  int    `json:"tokens"`
	Updated string `json:"updated"`
	Runs    int    `json:"runs"`       // executions recorded for this case
	RunsOK  int    `json:"runsPassed"` // ...of which passed
}

type ProjectStatus struct {
	Project        string       `json:"project"`
	App            string       `json:"app"`
	Repo           string       `json:"repo"`
	Onboarded      bool         `json:"onboarded"`
	Cases          []CaseStatus `json:"cases"`
	Passing        int          `json:"passing"`
	NeedsPerson    int          `json:"needsPerson"`
	Tokens         int          `json:"tokens"`
	Base           string       `json:"base"`           // the test repo's base branch
	TestsOnBase    int          `json:"testsOnBase"`    // generated tests already merged into the base branch
	CrawlStored    bool         `json:"crawlStored"`    // a crawl is committed on the base branch
	AwaitingReview []string     `json:"awaitingReview"` // hydra/* branches not merged yet
}

func (c Config) Status() ProjectStatus {
	st := ProjectStatus{Project: c.Project, App: c.App, Repo: c.Home}
	files, _ := filepath.Glob(filepath.Join(c.Out, "runs", "*.json"))
	for _, f := range files {
		b, err := os.ReadFile(f)
		var r Run
		if err != nil || json.Unmarshal(b, &r) != nil {
			continue
		}
		cs := CaseStatus{Key: r.Key, Intent: r.Intent, Tier: r.Tier, Status: r.Status, Tokens: r.Tokens}
		if fi, e := os.Stat(f); e == nil {
			cs.Updated = fi.ModTime().Format("2006-01-02 15:04")
		}
		st.Cases = append(st.Cases, cs)
		st.Tokens += r.Tokens
		if r.Status == "passed" {
			st.Passing++
		}
		if needsPerson(r) {
			st.NeedsPerson++
		}
	}
	sort.Slice(st.Cases, func(a, b int) bool { return st.Cases[a].Key < st.Cases[b].Key })
	hist := map[string][2]int{}
	for _, e := range c.executions("") {
		h := hist[e.Key]
		h[0]++
		if e.Status == "passed" {
			h[1]++
		}
		hist[e.Key] = h
	}
	for i := range st.Cases {
		st.Cases[i].Runs, st.Cases[i].RunsOK = hist[st.Cases[i].Key][0], hist[st.Cases[i].Key][1]
	}

	if !fileExists(filepath.Join(c.Home, ".git")) {
		return st
	}
	st.Onboarded = true
	base := c.BaseBranch
	if base == "" {
		base = "main"
	}
	st.Base = base
	if out, err := git(c.Home, "ls-tree", "-r", "--name-only", base, "tests/"+c.App); err == nil {
		for _, f := range strings.Fields(out) {
			if strings.HasSuffix(f, ".py") && strings.Contains(filepath.Base(f), "test_") {
				st.TestsOnBase++
			}
		}
	}
	_, err := git(c.Home, "cat-file", "-e", base+":hydra/crawl/crawl.json")
	st.CrawlStored = err == nil
	if out, err := git(c.Home, "branch", "--no-merged", base, "--list", "hydra/*", "--format=%(refname:short)"); err == nil {
		st.AwaitingReview = strings.Fields(out)
	}
	return st
}

func printStatus(all []ProjectStatus, asJSON bool) {
	if asJSON {
		b, _ := json.MarshalIndent(all, "", "  ")
		fmt.Println(string(b))
		return
	}
	for _, p := range all {
		if !p.Onboarded {
			fmt.Printf("%-6s %-16s not onboarded yet (no repo at %s): hydra onboard --project %s\n\n", p.Project, p.App, p.Repo, p.Project)
			continue
		}
		crawl := "no"
		if p.CrawlStored {
			crawl = "yes"
		}
		fmt.Printf("%-6s %-16s %d tracked | %d passing | %d need a person | %d tokens | %d tests on %s | crawl stored: %s\n",
			p.Project, p.App, len(p.Cases), p.Passing, p.NeedsPerson, p.Tokens, p.TestsOnBase, p.Base, crawl)
		for _, c := range p.Cases {
			fmt.Printf("         %-9s %-8s %-12s %-14s %6d tokens  %s  history: %d/%d passed\n", c.Key, c.Intent, c.Tier, c.Status, c.Tokens, c.Updated, c.RunsOK, c.Runs)
		}
		if len(p.AwaitingReview) > 0 {
			fmt.Printf("         awaiting review: %s\n", strings.Join(p.AwaitingReview, ", "))
		}
		fmt.Println()
	}
}
