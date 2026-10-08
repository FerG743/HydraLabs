package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Onboarding: the first time the pipeline meets a registered project it creates that application's own repo,
// Hydra-<App>, next to the other repos on the machine that runs n8n, from the framework template, with a baseline
// commit so every later review branch shows only generated tests. Registration is explicit (projects.json): a Jira
// token sees dozens of projects and must never turn each of them into a repo.
//
// It then stops at a gate until the app has a base URL (Jira cannot know it), and crawls the app once.

type ProjectEntry struct {
	App     string `json:"app"`
	Profile string `json:"profile"` // relative to the registry file; created with defaults if missing
}

type Registry struct {
	ReposRoot string                  `json:"reposRoot"` // where repos live; default: the folder that holds this one
	Template  string                  `json:"template"`  // the framework template, relative to the registry file
	Projects  map[string]ProjectEntry `json:"projects"`
	dir       string
}

func LoadRegistry(path string) (*Registry, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	r := &Registry{dir: filepath.Dir(path)}
	if err := json.Unmarshal(b, r); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if r.ReposRoot == "" {
		r.ReposRoot = "~/Documents/GitHub"
	}
	return r, nil
}

func (r *Registry) abs(p string) string {
	if p = expand(p); filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(r.dir, p)
}

func (r *Registry) RepoPath(app string) string {
	return filepath.Join(r.abs(r.ReposRoot), "Hydra-"+app)
}

func (r *Registry) Keys() []string {
	var ks []string
	for k := range r.Projects {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

type Onboarding struct {
	Project string `json:"project"`
	Repo    string `json:"repo"`
	Created bool   `json:"created"`
	Gate    string `json:"gate,omitempty"` // what a person must do before the pipeline can continue
}

// defaultProfile is what a new app starts with: read-only, no base URL (the gate), the model tier available.
func defaultProfile(project, app string) map[string]any {
	return map[string]any{"app": app, "baseUrl": "", "allowWrites": false, "paths": []string{"/"}, "maxPages": 25, "safeClicks": true,
		"agent": "agent/agent.py", "model": "prism-ml/bonsai-27b", "maxTurns": 20, "maxTokens": 100000,
		"jql":  fmt.Sprintf(`project = %s AND labels in ("automate", "execute") ORDER BY key ASC`, project),
		"_doc": "Created by onboarding. Set baseUrl (and allowWrites only for a QA environment you may change), then run again."}
}

func gitIdentity(repo string) []string { // the pipeline's commits must work on a machine with no git identity set
	if n, _ := git(repo, "config", "user.name"); n != "" {
		return nil
	}
	return []string{"-c", "user.name=HydraPloy", "-c", "user.email=hydra@localhost"}
}

// Onboard is idempotent: an existing repo with a commit is left exactly as it is.
func (r *Registry) Onboard(project string, c *Config) (Onboarding, error) {
	e, ok := r.Projects[project]
	if !ok {
		return Onboarding{}, fmt.Errorf("project %s is not in the registry", project)
	}
	o := Onboarding{Project: project, Repo: r.RepoPath(e.App)}

	if _, err := git(o.Repo, "rev-parse", "--verify", "HEAD"); err != nil {
		if entries, _ := os.ReadDir(o.Repo); len(entries) > 0 && !fileExists(filepath.Join(o.Repo, ".git")) {
			return o, fmt.Errorf("%s exists and is not a Hydra repo: refusing to write into it", o.Repo)
		}
		if err := r.scaffold(o.Repo, e.App); err != nil {
			return o, err
		}
		o.Created = true
	}

	profile := r.abs(e.Profile)
	if !fileExists(profile) {
		if err := writeJSON(profile, defaultProfile(project, e.App)); err != nil {
			return o, err
		}
		o.Gate = "set baseUrl in " + profile
		return o, nil
	}
	switch {
	case c.PortalURL == "":
		o.Gate = "set baseUrl in " + profile + " (the URL of the app under test)"
	case !fileExists(c.crawlPath()):
		if err := c.Crawl("", false); err != nil { // stage 2, once per app
			o.Gate = "the first crawl of " + c.PortalURL + " failed (is the app up, VPN on?): " + err.Error()
		}
	}
	return o, nil
}

// scaffold: framework template -> new repo, app placeholders replaced, one baseline commit.
func (r *Registry) scaffold(repo, app string) error {
	tpl := r.abs(r.Template)
	if !fileExists(filepath.Join(tpl, "conftest.py")) {
		return fmt.Errorf("framework template not found at %s", tpl)
	}
	if err := copyTree(tpl, repo); err != nil {
		return err
	}
	for _, sub := range []string{"apps", "tests"} { // directories named for the app
		if err := os.Rename(filepath.Join(repo, sub, "__APP__"), filepath.Join(repo, sub, app)); err != nil {
			return err
		}
	}
	for _, f := range []string{"runner_central.py"} {
		b, err := os.ReadFile(filepath.Join(repo, f))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(repo, f), []byte(strings.ReplaceAll(string(b), "__APP__", app)), 0o644); err != nil {
			return err
		}
	}
	if out, err := exec.Command("git", "-C", repo, "init", "-q", "-b", "main").CombinedOutput(); err != nil {
		return fmt.Errorf("git init: %v: %s", err, out)
	}
	if _, err := git(repo, "add", "-A"); err != nil {
		return err
	}
	_, err := git(repo, append(gitIdentity(repo), "commit", "-q", "-m",
		"framework baseline: Playwright + Pytest template for "+app+", no test cases yet\n\nCreated by HydraPloy onboarding. Generated tests arrive on review branches on top of this commit.")...)
	return err
}
