// hydra: the Jira -> test pipeline's brain. Polls (or is handed) issues, routes each to the cheapest tier
// that can finish it, and keeps a run record with the tokens spent.
//
//	hydra poll --out out --app Portal --portal-url http://host:3000 [--comment]
//	hydra crawl --profile profiles/app.json --watch :8099 --headed   (read-only; watch it at http://localhost:8099)
//	hydra run  --key DWQ-130 --intent automate --html fixtures/DWQ-130.html --summary "TC-1 - ..."
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// cfgFlags: defaults, then the --profile file, then explicit flags. A new web app is a new profile file, not new code.
func cfgFlags(fs *flag.FlagSet, args []string) (*Config, error) {
	here, _ := os.Getwd()
	c := &Config{Out: filepath.Join(here, "out", "hydra"), ToolsDir: filepath.Join(here, "tools", "jira"),
		AppMap: filepath.Join(here, "tools", "jira", "appmap"), Python: "python3", Lang: "en", MaxTurns: 25,
		LMURL: "http://127.0.0.1:1234/v1", MaxPages: 25, Crawler: filepath.Join(here, "tools", "crawl", "crawl.py"), Knowledge: filepath.Join(here, "knowledge"), JQL: `labels in ("automate", "execute") ORDER BY key ASC`} // "execute" is a reserved JQL word: it must be quoted
	for i, a := range args { // find --profile before parsing so its values become the flag defaults
		if v, ok := strings.CutPrefix(strings.TrimLeft(a, "-"), "profile="); ok || (strings.TrimLeft(a, "-") == "profile" && i+1 < len(args)) {
			if !ok {
				v = args[i+1]
			}
			b, err := os.ReadFile(v)
			if err != nil {
				return nil, err
			}
			if err := json.Unmarshal(b, c); err != nil {
				return nil, fmt.Errorf("%s: %w", v, err)
			}
		}
	}
	fs.String("profile", "", "project profile JSON (profiles/<name>.json)")
	fs.StringVar(&c.Out, "out", c.Out, "run record, cases and bundles")
	fs.StringVar(&c.ToolsDir, "tools", c.ToolsDir, "directory with render_pytest.py and run_generated.py")
	fs.StringVar(&c.AppMap, "appmap", c.AppMap, "locators confirmed on the real app, one <App>.json each")
	fs.StringVar(&c.Python, "python", c.Python, "python with playwright installed")
	fs.StringVar(&c.App, "app", c.App, "app folder name in the framework layout")
	fs.StringVar(&c.Lang, "lang", c.Lang, "language of the cases")
	fs.StringVar(&c.PortalURL, "base-url", c.PortalURL, "app under test; without it bundles are written but not verified")
	fs.StringVar(&c.Agent, "agent", c.Agent, "path to agent.py (T3); empty = no model tier")
	fs.StringVar(&c.Model, "model", c.Model, "LM Studio model for T3")
	fs.IntVar(&c.MaxTurns, "max-turns", c.MaxTurns, "hard budget for T3")
	fs.StringVar(&c.LoginFile, "login", c.LoginFile, "crawl login steps (values are ENV: tokens)")
	fs.StringVar(&c.EnvFile, "env-file", c.EnvFile, "the framework's .env, to resolve ENV: tokens")
	fs.BoolVar(&c.SafeClicks, "safe-clicks", c.SafeClicks, "crawl: open comboboxes/tabs to record options")
	fs.BoolVar(&c.AllowWrites, "allow-writes", c.AllowWrites, "let tests send real writes (default: read-only safety net)")
	return c, nil
}

var jsonOut bool

func report(r Run) {
	if jsonOut {
		b, _ := json.Marshal(r)
		fmt.Println(string(b))
		return
	}
	fmt.Printf("%-10s %-8s %-12s %-13s tokens=%d %dms\n", r.Key, r.Intent, r.Tier, r.Status, r.Tokens, r.Millis)
	for _, d := range r.Detail {
		fmt.Println("           " + d)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: hydra poll|run [flags]")
		os.Exit(2)
	}
	fs := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
	cfg, err := cfgFlags(fs, os.Args[2:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "hydra:", err)
		os.Exit(1)
	}
	switch os.Args[1] {
	case "run":
		key, intent, htmlPath, summary := fs.String("key", "", ""), fs.String("intent", "automate", "automate|execute"), fs.String("html", "", "rendered description"), fs.String("summary", "", "")
		fs.Parse(os.Args[2:])
		if *htmlPath != "" {
			b, e := os.ReadFile(*htmlPath)
			if e != nil {
				err = e
				break
			}
			if _, err = cfg.Ingest(Issue{Key: *key, Summary: *summary, HTML: string(b)}); err != nil {
				break
			}
		}
		r := cfg.Route(*key, *intent)
		report(r)
		if r.Status == "failed" {
			os.Exit(1)
		}
	case "crawl":
		watch, headed := fs.String("watch", "", "serve a live view of the crawl, e.g. :8099"), fs.Bool("headed", false, "show the real browser")
		fs.Parse(os.Args[2:])
		if err = cfg.Crawl(*watch, *headed); err == nil && *watch != "" {
			fmt.Println("viewer still up for the last page; Ctrl-C to stop")
			select {}
		}
	case "poll":
		jql := fs.String("jql", cfg.JQL, "which issues to look at (profile: jql)")
		comment := fs.Bool("comment", cfg.JiraComments, "post outcomes that need a person to the issue (off until results are trusted)")
		intent, only, force, asJSON := fs.String("intent", "", "with --only: treat the issue as automate|execute even without the label"), fs.String("only", "", "process just this issue key (for a manual trigger)"), fs.Bool("force", false, "ignore the seen-state: run it again"), fs.Bool("json", false, "one JSON object per run (for n8n)")
		fs.Parse(os.Args[2:])
		jsonOut = *asJSON
		err = poll(*cfg, *jql, *comment, *only, *force, *intent)
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "hydra:", err)
		os.Exit(1)
	}
}

// poll: one pass. An issue is handled once per (Jira edit, intent); changing the label or the text re-runs it.
func poll(cfg Config, jql string, comment bool, only string, force bool, forceIntent string) error {
	j, err := JiraFromEnv()
	if err != nil {
		return err
	}
	var issues []Issue
	if only != "" { // a person asked for exactly this issue: no JQL, no seen-state
		is, e := j.Get(only)
		if e != nil {
			return e
		}
		issues, force = []Issue{*is}, true
	} else if issues, err = j.Search(jql); err != nil {
		return err
	}
	statePath := filepath.Join(cfg.Out, "state.json")
	seen := map[string]string{}
	if b, e := os.ReadFile(statePath); e == nil {
		json.Unmarshal(b, &seen)
	}
	type job struct{ key, intent, stamp string }
	var jobs []job
	for _, is := range issues { // phase 1: store every new case so "Same as TC-1" can resolve
		intent := Intent(is.Labels)
		if only != "" && forceIntent != "" {
			intent = forceIntent
		}
		stamp := is.Updated + "|" + intent
		if intent == "" || (seen[is.Key] == stamp && !force) {
			continue
		}
		if _, err := cfg.Ingest(is); err != nil {
			return fmt.Errorf("%s: %w", is.Key, err)
		}
		jobs = append(jobs, job{is.Key, intent, stamp})
	}
	if len(jobs) == 0 {
		if !jsonOut {
			fmt.Println("nothing new")
		}
		return nil
	}
	sort.Slice(jobs, func(a, b int) bool { return jobs[a].key < jobs[b].key })
	for _, jb := range jobs { // phase 2: route
		r := cfg.Route(jb.key, jb.intent)
		report(r)
		seen[jb.key] = jb.stamp
		if needsPerson(r) {
			if comment {
				if e := j.Comment(jb.key, Message(r)); e != nil {
					fmt.Fprintln(os.Stderr, "comment failed:", e)
				}
			}
			if e := ChatNotify(cfg.ChatWebhook, r); e != nil {
				fmt.Fprintln(os.Stderr, "chat notify failed:", e)
			}
		}
	}
	return writeJSON(statePath, seen)
}
