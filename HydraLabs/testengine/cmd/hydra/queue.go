package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// The poll queue: one pass over EVERY registered project, cheapest work first.
//
//	class 0  free        a stored test to execute, or a case that will be sent back to its author (no tool, no tokens)
//	class 1  render      automate a case the rules map cleanly (T1): no tokens, seconds
//	class 2  model       automate a case with steps only the model can map (T3): minutes, tens of thousands of tokens
//
// Cost is known before it is paid: a dry render is deterministic and says which steps are unmappable. Within a class,
// projects take turns so one busy app cannot starve another. Class 2 is capped per poll (default 1); the rest wait for
// the next poll, unrecorded, so nothing is lost. A morning window on one local model must not be eaten by one case.

const (
	classFree   = 0
	classRender = 1
	classModel  = 2
)

var classNames = [...]string{"free", "render", "model"}

type job struct {
	cfg                *Config
	key, intent, stamp string
	class              int
	why                string
}

type PollOpts struct {
	JQL        string
	Comment    bool
	Only       string
	Force      bool
	Intent     string
	ModelCases int // class-2 cases allowed per poll
}

// resolved returns the stored case with "Same as TC-1" preconditions resolved against every stored case.
func (c Config) resolved(key string) (*Case, []*Case, error) {
	cases, err := c.loadCases()
	if err != nil {
		return nil, nil, err
	}
	ResolveRefs(cases)
	for _, x := range cases {
		if x.ID == key {
			return x, cases, nil
		}
	}
	return nil, cases, fmt.Errorf("case %s is not stored", key)
}

// classify says what a job will cost, without running it. Execute-with-a-stored-test returns before any render, so a
// stored bundle is never touched by classification.
func classify(c *Config, key, intent string) (int, string) {
	if err := c.requireHome(); err != nil {
		return classFree, "the app's repo does not exist yet: reported at once"
	}
	if intent == "execute" && fileExists(filepath.Join(c.bundleDir(key), "tests")) {
		return classFree, "runs its stored test"
	}
	cs, cases, err := c.resolved(key)
	if err != nil {
		return classFree, "no stored case: reported at once"
	}
	if bad, _ := Maturity(cs); len(bad) > 0 {
		return classFree, "incomplete: goes back to its author"
	}
	pending, _, err := c.render(cs, cases) // dry: deterministic, no browser, no model for the rules (the bridge may settle a tie)
	switch {
	case err != nil:
		return classRender, "render failed: reported by the router"
	case len(pending) > 0 && c.agentOn():
		return classModel, fmt.Sprintf("%d step(s) only the model can map", len(pending))
	case len(pending) > 0:
		return classRender, "unmappable steps, but no model tier: goes to review at once"
	}
	return classRender, "maps cleanly"
}

// order sorts by class, then takes projects in turns inside each class; keys stay in order within a project.
func order(jobs []*job) []*job {
	var out []*job
	for class := classFree; class <= classModel; class++ {
		byProject := map[string][]*job{}
		var projects []string
		for _, j := range jobs {
			if j.class != class {
				continue
			}
			p := j.cfg.Project
			if _, seen := byProject[p]; !seen {
				projects = append(projects, p)
			}
			byProject[p] = append(byProject[p], j)
		}
		sort.Strings(projects)
		for len(byProject) > 0 {
			for _, p := range projects {
				if q := byProject[p]; len(q) > 0 {
					out = append(out, q[0])
					if byProject[p] = q[1:]; len(byProject[p]) == 0 {
						delete(byProject, p)
					}
				}
			}
		}
	}
	return out
}

// schedule splits the ordered jobs into what runs now and what waits (class 2 beyond the cap).
func schedule(ordered []*job, modelCap int) (run, deferred []*job) {
	if modelCap < 1 {
		modelCap = 1
	}
	models := 0
	for _, j := range ordered {
		if j.class == classModel {
			if models >= modelCap {
				deferred = append(deferred, j)
				continue
			}
			models++
		}
		run = append(run, j)
	}
	return
}

// collect: Jira -> the jobs of one project that are new (or forced). Every new case is stored first so "Same as TC-1" resolves.
func collect(cfg *Config, j *Jira, o PollOpts) ([]*job, map[string]string, error) {
	var issues []Issue
	force := o.Force
	if o.Only != "" { // a person asked for exactly this issue: no JQL, no seen-state
		is, err := j.Get(o.Only)
		if err != nil {
			return nil, nil, err
		}
		issues, force = []Issue{*is}, true
	} else {
		var err error
		if issues, err = j.Search(o.JQL); err != nil {
			return nil, nil, err
		}
	}
	seen := loadState(cfg)
	var jobs []*job
	for _, is := range issues {
		intent := Intent(is.Labels)
		if o.Only != "" && o.Intent != "" {
			intent = o.Intent
		}
		stamp := is.Updated + "|" + intent
		if intent == "" || (seen[is.Key] == stamp && !force) {
			continue
		}
		if _, err := cfg.Ingest(is); err != nil {
			return nil, nil, fmt.Errorf("%s: %w", is.Key, err)
		}
		jobs = append(jobs, &job{cfg: cfg, key: is.Key, intent: intent, stamp: stamp})
	}
	sort.Slice(jobs, func(a, b int) bool { return jobs[a].key < jobs[b].key })
	return jobs, seen, nil
}

func statePath(c *Config) string { return filepath.Join(c.Out, "state.json") }

func loadState(c *Config) map[string]string {
	seen := map[string]string{}
	if b, err := os.ReadFile(statePath(c)); err == nil {
		json.Unmarshal(b, &seen)
	}
	return seen
}

// Poll runs one pass over the given projects. gated is true if any project stopped at an onboarding gate.
func Poll(cfgs []*Config, o PollOpts) (gated bool, err error) {
	j, err := JiraFromEnv()
	if err != nil {
		return false, err
	}
	var all []*job
	states := map[*Config]map[string]string{}
	for _, cfg := range cfgs {
		if cfg.Registry != nil && cfg.Project != "" { // a registered project is onboarded before anything else happens
			ob, e := cfg.Registry.Onboard(cfg.Project, cfg)
			if e != nil {
				fmt.Fprintf(os.Stderr, "hydra: %s: %v\n", cfg.Project, e)
				err = e
				continue
			}
			if ob.Created {
				report(Run{Key: cfg.Project, Intent: "onboard", Tier: "T0-onboarding", Status: "onboarded", Detail: []string{"created " + ob.Repo}})
			}
			if ob.Gate != "" {
				report(Run{Key: cfg.Project, Intent: "onboard", Tier: "T0-onboarding", Status: "needs-onboarding", Detail: []string{ob.Gate}})
				gated = true
				continue
			}
		}
		po := o
		po.JQL, po.Comment = cfg.JQL, cfg.JiraComments
		if len(cfgs) == 1 && o.JQL != "" {
			po.JQL, po.Comment = o.JQL, o.Comment
		}
		jobs, seen, e := collect(cfg, j, po)
		if e != nil {
			fmt.Fprintf(os.Stderr, "hydra: %s: %v\n", cfg.Project, e)
			err = e
			continue
		}
		states[cfg] = seen
		all = append(all, jobs...)
	}
	if len(all) == 0 {
		if !jsonOut && err == nil && !gated {
			fmt.Println("nothing new")
		}
		return gated, err
	}

	for _, jb := range all {
		jb.class, jb.why = classify(jb.cfg, jb.key, jb.intent)
	}
	run, deferred := schedule(order(all), o.ModelCases)
	if !jsonOut {
		n := [3]int{}
		for _, jb := range all {
			n[jb.class]++
		}
		fmt.Printf("queue: %d free, %d render, %d model (model cap %d per poll, %d waiting)\n", n[0], n[1], n[2], max(o.ModelCases, 1), len(deferred))
	}

	for _, jb := range run {
		r := jb.cfg.Route(jb.key, jb.intent)
		report(r)
		states[jb.cfg][jb.key] = jb.stamp
		if needsPerson(r) {
			if cfg := jb.cfg; cfg.JiraComments || (len(cfgs) == 1 && o.Comment) {
				msg := Message(r)
				if _, e := cfg.PostOnce(j, jb.key, shortID(r.Status, msg), msg); e != nil {
					fmt.Fprintln(os.Stderr, "comment failed:", e)
				}
			}
			if e := ChatNotify(jb.cfg.ChatWebhook, r); e != nil {
				fmt.Fprintln(os.Stderr, "chat notify failed:", e)
			}
		}
	}
	for _, jb := range deferred { // not recorded as seen: the next poll picks them up
		report(Run{Key: jb.key, Intent: jb.intent, Tier: "queue", Status: "deferred",
			Detail: []string{fmt.Sprintf("%s: waits for the next poll (model cap is %d per poll)", jb.why, max(o.ModelCases, 1))}})
	}
	for cfg, seen := range states {
		if e := writeJSON(statePath(cfg), seen); e != nil && err == nil {
			err = e
		}
	}
	return gated, err
}
