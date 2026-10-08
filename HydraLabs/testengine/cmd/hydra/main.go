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
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
)

// cfgFlags: defaults, then the --profile file, then explicit flags. A new web app is a new profile file, not new code.
func cfgFlags(fs *flag.FlagSet, args []string) (*Config, error) {
	here, _ := os.Getwd()
	c := &Config{Out: filepath.Join(here, "out", "hydra"), ToolsDir: filepath.Join(here, "tools", "jira"),
		AppMap: filepath.Join(here, "tools", "jira", "appmap"), Python: "python3", Lang: "en", MaxTurns: 25,
		LMURL: "http://127.0.0.1:1234/v1", MaxPages: 25, BaseBranch: "main", Crawler: filepath.Join(here, "tools", "crawl", "crawl.py"), Knowledge: filepath.Join(here, "knowledge"), JQL: `labels in ("automate", "execute") ORDER BY key ASC`} // "execute" is a reserved JQL word: it must be quoted
	profilePath, project := argValue(args, "profile"), argValue(args, "project")
	regPath := argValue(args, "registry")
	if regPath == "" && fileExists(filepath.Join(here, "projects.json")) {
		regPath = filepath.Join(here, "projects.json")
	}
	if regPath != "" && (project != "" || profilePath == "") {
		reg, err := LoadRegistry(regPath)
		if err != nil {
			return nil, err
		}
		c.Registry = reg
	}
	if project != "" {
		if c.Registry == nil {
			return nil, fmt.Errorf("--project needs a registry (projects.json)")
		}
		e, ok := c.Registry.Projects[project]
		if !ok {
			return nil, fmt.Errorf("project %s is not in the registry (%s)", project, strings.Join(c.Registry.Keys(), ", "))
		}
		c.Project, c.App = project, e.App
		if profilePath == "" {
			profilePath = c.Registry.abs(e.Profile)
		}
	}
	if profilePath != "" {
		b, err := os.ReadFile(profilePath)
		if err != nil && !(project != "" && os.IsNotExist(err)) { // a new project has no profile yet: onboarding writes it
			return nil, err
		} else if err == nil {
			if err := json.Unmarshal(b, c); err != nil {
				return nil, fmt.Errorf("%s: %w", profilePath, err)
			}
		}
	}
	if c.DeliverRepo == "" && c.Registry != nil && c.App != "" {
		c.DeliverRepo = c.Registry.RepoPath(c.App) // every app delivers to its own Hydra-<App> repo
	}
	fs.String("profile", "", "project profile JSON (profiles/<name>.json)")
	fs.String("project", "", "registered Jira project key (projects.json): the profile and the repo follow from it")
	fs.String("registry", "", "projects.json (default: next to the binary's working dir)")
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

// argValue finds "--name v" or "--name=v" in args before flag parsing (the values become the flag defaults).
func argValue(args []string, name string) string {
	for i, a := range args {
		a = strings.TrimLeft(a, "-")
		if v, ok := strings.CutPrefix(a, name+"="); ok {
			return v
		}
		if a == name && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

// pollAll: with a registry and no --project, poll every registered project (or the one a --only key belongs to),
// each in its own process so a failing project cannot stop the others.
func pollAll(args []string) (handled bool, code int) {
	if argValue(args, "project") != "" || argValue(args, "profile") != "" {
		return false, 0
	}
	here, _ := os.Getwd()
	regPath := argValue(args, "registry")
	if regPath == "" {
		regPath = filepath.Join(here, "projects.json")
	}
	reg, err := LoadRegistry(regPath)
	if err != nil {
		return false, 0
	}
	keys := reg.Keys()
	if only := argValue(args, "only"); only != "" {
		keys = []string{strings.SplitN(only, "-", 2)[0]}
		if _, ok := reg.Projects[keys[0]]; !ok {
			fmt.Fprintf(os.Stderr, "hydra: %s belongs to project %s, which is not in the registry (%s)\n", only, keys[0], strings.Join(reg.Keys(), ", "))
			return true, 1
		}
	}
	for _, k := range keys {
		cmd := exec.Command(os.Args[0], append([]string{"poll", "--project", k}, args...)...)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			code = 1
		}
	}
	return true, code
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
	if os.Args[1] == "poll" {
		if handled, code := pollAll(os.Args[2:]); handled {
			os.Exit(code)
		}
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
		fs.IntVar(&cfg.SlowMS, "slow", cfg.SlowMS, "reveal elements one at a time, this many ms apart (for demos)")
		fs.Parse(os.Args[2:])
		if err = cfg.Crawl(*watch, *headed); err == nil && *watch != "" {
			fmt.Println("viewer still up; Ctrl-C to stop")
			sig := make(chan os.Signal, 1)
			signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
			<-sig
		}
	case "onboard":
		fs.Parse(os.Args[2:])
		if cfg.Registry == nil || cfg.Project == "" {
			err = fmt.Errorf("onboard needs --project KEY (registered in projects.json)")
			break
		}
		var o Onboarding
		if o, err = cfg.Registry.Onboard(cfg.Project, cfg); err == nil {
			b, _ := json.MarshalIndent(o, "", "  ")
			fmt.Println(string(b))
			if o.Gate != "" {
				os.Exit(3)
			}
		}
	case "deliver":
		only, exclude := fs.String("only", "", "comma-separated keys to deliver (default: every passing case)"), fs.String("exclude", "", "comma-separated keys to leave out")
		fs.StringVar(&cfg.DeliverRepo, "repo", cfg.DeliverRepo, "framework repo that receives the tests")
		fs.Parse(os.Args[2:])
		ex := map[string]bool{}
		for _, k := range strings.Split(*exclude, ",") {
			ex[strings.TrimSpace(k)] = true
		}
		var on []string
		if *only != "" {
			on = strings.Split(*only, ",")
		}
		var d Delivery
		if d, err = cfg.Deliver(on, ex); err == nil {
			b, _ := json.MarshalIndent(d, "", "  ")
			fmt.Println(string(b))
			if d.Gate != "" {
				os.Exit(3) // distinct from 1 (error): the pipeline worked and is asking a person
			}
		}
	case "poll":
		jql := fs.String("jql", cfg.JQL, "which issues to look at (profile: jql)")
		comment := fs.Bool("comment", cfg.JiraComments, "post outcomes that need a person to the issue (off until results are trusted)")
		intent, only, force, asJSON := fs.String("intent", "", "with --only: treat the issue as automate|execute even without the label"), fs.String("only", "", "process just this issue key (for a manual trigger)"), fs.Bool("force", false, "ignore the seen-state: run it again"), fs.Bool("json", false, "one JSON object per run (for n8n)")
		fs.Parse(os.Args[2:])
		jsonOut = *asJSON
		if cfg.Registry != nil && cfg.Project != "" { // a registered project is onboarded before anything else happens
			o, e := cfg.Registry.Onboard(cfg.Project, cfg)
			if e != nil {
				err = e
				break
			}
			if o.Created {
				report(Run{Key: cfg.Project, Intent: "onboard", Tier: "T0-onboarding", Status: "onboarded", Detail: []string{"created " + o.Repo}})
			}
			if o.Gate != "" {
				report(Run{Key: cfg.Project, Intent: "onboard", Tier: "T0-onboarding", Status: "needs-onboarding", Detail: []string{o.Gate}})
				os.Exit(3)
			}
		}
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
