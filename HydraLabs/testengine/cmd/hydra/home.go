package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// An app's home is its own repo (Hydra-<App>); everything the pipeline produces for the app lives in its hydra/ folder.
//
//	hydra/            DURABLE, committed on review branches (never straight to main)
//	  README.md         what this folder is
//	  crawl/            crawl.json + CRAWL.md  (hydra crawl --store)
//	  locators/         learned.json (what the crawl taught the bridge) and confirmed.json (written by a person)
//	  knowledge.md      what is peculiar about the app (written by a person, read by the model tier)
//	  cases/            the Jira text each delivered test was generated from (<KEY>.case.json) and the same case as
//	                    readable step-by-step Gherkin (<KEY>.feature)
//	hydra/out/        WORKING, git-ignored: run records, bundles, crawl screenshots/video/events, agent logs, state
//
// Working files never dirty the repo, so delivery can insist on a clean tree; durable files only ever change on a branch.

func (c Config) durable(rel ...string) string {
	return filepath.Join(append([]string{c.Home, "hydra"}, rel...)...)
}

// applyHome points the working locations into the app's repo, unless the profile or a flag chose them explicitly.
func (c *Config) applyHome(defOut, defAppMap string) {
	if c.Home == "" {
		return
	}
	if c.Out == defOut {
		c.Out = filepath.Join(c.Home, "hydra", "out")
	}
	if c.AppMap == defAppMap {
		c.AppMap = filepath.Join(c.Home, "hydra", "out", "appmap")
	}
}

func (c Config) notesPath() string {
	if c.Home != "" {
		return c.durable("knowledge.md")
	}
	return filepath.Join(c.Knowledge, c.App+".md")
}

func copyFile(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0o644)
}

// seedWorking: on a fresh clone the working copies of the crawl and the learned locators are missing; the committed
// ones are the starting point (so nothing has to be crawled or asked of the model again).
func (c Config) seedWorking() {
	if c.Home == "" {
		return
	}
	for _, p := range [][2]string{{c.durable("crawl", "crawl.json"), c.crawlPath()}, {c.durable("locators", "learned.json"), c.learnedPath()}} {
		if !fileExists(p[1]) && fileExists(p[0]) {
			copyFile(p[0], p[1])
		}
	}
}

// ensureIgnored makes sure git ignores hydra/out/ even in a repo created before the rule existed (locally, via
// .git/info/exclude: it changes no tracked file). Repos from the template already have the rule in .gitignore.
func ensureIgnored(repo string) {
	if _, err := git(repo, "check-ignore", "-q", "hydra/out/x"); err == nil {
		return
	}
	f := filepath.Join(repo, ".git", "info", "exclude")
	if b, _ := os.ReadFile(f); !strings.Contains(string(b), "hydra/out/") {
		os.MkdirAll(filepath.Dir(f), 0o755)
		if fh, err := os.OpenFile(f, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
			fh.WriteString("\n# HydraPloy working files\nhydra/out/\n")
			fh.Close()
		}
	}
}

// requireHome stops a command from creating an app's working folders before the app's repo exists (onboarding would
// then refuse the folder as "not a Hydra repo").
func (c Config) requireHome() error {
	if c.Home != "" && !fileExists(filepath.Join(c.Home, ".git")) {
		return fmt.Errorf("%s is not a repo yet: run hydra onboard --project %s first", c.Home, c.Project)
	}
	return nil
}

// stageDurable copies what a delivery should carry into the repo's hydra/ folder and returns the paths to commit:
// the learned locators and the Jira text each delivered test came from.
func (c Config) stageDurable(repo string, keys []string) ([]string, error) {
	var paths []string
	if fileExists(c.learnedPath()) {
		if err := copyFile(c.learnedPath(), filepath.Join(repo, "hydra", "locators", "learned.json")); err != nil {
			return nil, err
		}
		paths = append(paths, "hydra/locators")
	}
	n := 0
	for _, k := range keys {
		src := filepath.Join(c.caseDir(), k+".case.json")
		if fileExists(src) {
			if err := copyFile(src, filepath.Join(repo, "hydra", "cases", k+".case.json")); err != nil {
				return nil, err
			}
			n++
		}
	}
	features, err := c.writeFeatures(filepath.Join(repo, "hydra", "cases"))
	if err != nil {
		return nil, err
	}
	if n > 0 || features > 0 {
		paths = append(paths, "hydra/cases")
	}
	return paths, nil
}

// writeFeatures renders every stored case as Gherkin next to it (deterministic, no model), so a reviewer reads the
// case step by step in the repo; cases delivered before this existed get theirs on the next delivery.
func (c Config) writeFeatures(dir string) (int, error) {
	if c.ToolsDir == "" {
		return 0, nil
	}
	cases, _ := filepath.Glob(filepath.Join(dir, "*.case.json"))
	for _, src := range cases {
		in, err := os.Open(src)
		if err != nil {
			return 0, err
		}
		cmd := exec.Command(c.Python, filepath.Join(c.ToolsDir, "gherkin_gen.py"))
		cmd.Stdin = in
		feature, err := cmd.Output()
		in.Close()
		if err != nil {
			return 0, fmt.Errorf("gherkin for %s: %w", filepath.Base(src), err)
		}
		if err := os.WriteFile(strings.TrimSuffix(src, ".case.json")+".feature", feature, 0o644); err != nil {
			return 0, err
		}
	}
	return len(cases), nil
}
