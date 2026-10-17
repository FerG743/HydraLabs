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
	"strings"
	"syscall"
)

// engineDir: where projects.json, tools/ and templates/ are. The working directory when it has the registry,
// otherwise the folder above the binary (testengine/bin/hydra, also through a symlink), so hydra works from any folder.
func engineDir() string {
	here, _ := os.Getwd()
	if fileExists(filepath.Join(here, "projects.json")) {
		return here
	}
	if exe, err := os.Executable(); err == nil {
		if exe, err = filepath.EvalSymlinks(exe); err == nil {
			if dir := filepath.Dir(filepath.Dir(exe)); fileExists(filepath.Join(dir, "projects.json")) {
				return dir
			}
		}
	}
	return here
}

// cfgFlags: defaults, then the --profile file, then explicit flags. A new web app is a new profile file, not new code.
func cfgFlags(fs *flag.FlagSet, args []string) (*Config, error) {
	here := engineDir()
	defOut, defAppMap := filepath.Join(here, "out", "hydra"), filepath.Join(here, "tools", "jira", "appmap")
	c := &Config{Out: defOut, ToolsDir: filepath.Join(here, "tools", "jira"),
		AppMap: defAppMap, Python: "python3", Lang: "en", MaxTurns: 25,
		LMURL: "http://127.0.0.1:1234/v1", MaxPages: 25, BaseBranch: "main", StabilityRuns: 3, Crawler: filepath.Join(here, "tools", "crawl", "crawl.py"), Knowledge: filepath.Join(here, "knowledge"), JQL: `labels in ("automate", "execute") ORDER BY key ASC`} // "execute" is a reserved JQL word: it must be quoted
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
	if c.Registry != nil && c.App != "" {
		c.Home = expand(c.DeliverRepo) // ...and keeps everything it produces there too
		c.applyHome(defOut, defAppMap)
		c.seedWorking()
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
	fs.StringVar(&c.Trigger, "trigger", "", "why this run happens, kept in the execution log (deploy, nightly...; default: the command)")
	fs.StringVar(&c.AppVersion, "app-version", "", "the build under test (commit, tag, deploy id), kept in the execution log")
	fs.BoolVar(&c.Show, "show", false, "for people watching: a visible, slowed-down browser; deliver also writes and opens the HTML report")
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

// pollMulti: with a registry and no --project/--profile, one pass over every registered project (or the one a
// --only key belongs to) through a single queue, so the cheap work of every project goes first.
func pollMulti(args []string) (handled bool, code int) {
	if argValue(args, "project") != "" || argValue(args, "profile") != "" {
		return false, 0
	}
	here := engineDir()
	regPath := argValue(args, "registry")
	if regPath == "" {
		regPath = filepath.Join(here, "projects.json")
	}
	reg, err := LoadRegistry(regPath)
	if err != nil {
		return false, 0
	}
	fs := flag.NewFlagSet("poll", flag.ExitOnError)
	fs.String("registry", "", "")
	only, intent := fs.String("only", "", ""), fs.String("intent", "", "")
	force, asJSON := fs.Bool("force", false, ""), fs.Bool("json", false, "")
	modelCases := fs.Int("model-cases", reg.ModelCases(), "")
	trigger, appVersion := fs.String("trigger", "", ""), fs.String("app-version", "", "")
	show := fs.Bool("show", false, "")
	fs.Parse(args)
	jsonOut = *asJSON
	keys := reg.Keys()
	if *only != "" {
		k, ok := reg.ProjectFor(*only)
		if !ok {
			fmt.Fprintf(os.Stderr, "hydra: no registered project takes %s (registry key or profile jiraProject; registered: %s)\n", *only, strings.Join(reg.Keys(), ", "))
			return true, 1
		}
		keys = []string{k}
	}
	var cfgs []*Config
	for _, k := range keys {
		cfg, err := cfgFlags(flag.NewFlagSet("poll-"+k, flag.ContinueOnError), []string{"--project", k, "--registry", regPath})
		if err != nil {
			fmt.Fprintf(os.Stderr, "hydra: %s: %v\n", k, err)
			code = 1
			continue
		}
		cfg.Command, cfg.Trigger, cfg.AppVersion, cfg.Show = "poll", *trigger, *appVersion, *show
		if !cfg.hasAddon("ploy") { // a Labs-only team has no case source for the pipeline to poll
			fmt.Fprintf(os.Stderr, "hydra: %s: skipped, the ploy add-on is not enabled\n", k)
			continue
		}
		cfgs = append(cfgs, cfg)
	}
	gated, err := Poll(cfgs, PollOpts{Only: *only, Force: *force, Intent: *intent, ModelCases: *modelCases})
	if err != nil {
		fmt.Fprintln(os.Stderr, "hydra:", err)
		code = 1
	}
	if gated && code == 0 {
		code = 3
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
	// the pipeline's secrets (Jira, chat), as n8n/start.sh loads them, so a plain terminal works the same
	loadHydraEnv("~/.hydra.env", filepath.Join(engineDir(), "..", ".hydra.env"))
	if os.Args[1] == "poll" {
		if handled, code := pollMulti(os.Args[2:]); handled {
			os.Exit(code)
		}
	}
	fs := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
	cfg, err := cfgFlags(fs, os.Args[2:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "hydra:", err)
		os.Exit(1)
	}
	cfg.Command = os.Args[1]
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
		fs.BoolVar(&cfg.StoreCrawl, "store", cfg.StoreCrawl, "after the crawl, store it in the app's repo (apps/<App>/crawl/) on a review branch")
		fs.BoolVar(&cfg.RecordVideo, "video", cfg.RecordVideo, "record the crawl as a .webm video (out/hydra/crawl/videos)")
		fs.IntVar(&cfg.SlowMS, "slow", cfg.SlowMS, "reveal elements one at a time, this many ms apart (for demos)")
		fs.Parse(os.Args[2:])
		if !cfg.hasAddon("labs") {
			err = fmt.Errorf("crawling is part of the labs add-on, which is not enabled for %s (profile: addons)", cfg.App)
			break
		}
		err = cfg.Crawl(*watch, *headed)
		if err == nil && cfg.StoreCrawl {
			var st Stored
			if st, err = cfg.SaveCrawl(); err == nil {
				b, _ := json.MarshalIndent(st, "", "  ")
				fmt.Println(string(b))
			}
		}
		if err == nil && *watch != "" {
			fmt.Println("viewer still up; Ctrl-C to stop")
			sig := make(chan os.Signal, 1)
			signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
			<-sig
		}
	case "watch": // read-only: the live view of every stage this app's runs go through
		addr := fs.String("addr", ":8099", "where to serve the page")
		fs.Parse(os.Args[2:])
		err = cfg.Watch(*addr)
	case "history": // read-only: every recorded execution of a project, or of one case
		key, last, asJSON := fs.String("key", "", "only this case"), fs.Int("last", 30, "newest N (0 = all)"), fs.Bool("json", false, "machine-readable")
		fs.Parse(os.Args[2:])
		printHistory(cfg.executions(*key), *last, *asJSON)
	case "init": // onboard a team by asking; nothing is written before the final yes
		answersPath, yes := fs.String("answers", "", "answers file {core, ploy, labs}: no questions for what it covers"), fs.Bool("yes", false, "no prompts: missing answers take their defaults, no confirmation")
		noPreview := fs.Bool("no-preview", false, "skip the one-page preview crawl")
		fs.Parse(os.Args[2:])
		err = runInit(cfg, *answersPath, *yes, *noPreview)
	case "status": // read-only, no Jira: what every tracked project looks like right now
		asJSON := fs.Bool("json", false, "machine-readable")
		fs.Parse(os.Args[2:])
		if cfg.Registry == nil {
			err = fmt.Errorf("status needs the registry (projects.json)")
			break
		}
		var all []ProjectStatus
		for _, k := range cfg.Registry.Keys() {
			pc, e := cfgFlags(flag.NewFlagSet("status-"+k, flag.ContinueOnError), []string{"--project", k, "--registry", filepath.Join(cfg.Registry.dir, "projects.json")})
			if e != nil {
				err = e
				break
			}
			all = append(all, pc.Status())
		}
		if err == nil {
			printStatus(all, *asJSON)
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
	case "announce": // backfill: comment on cases that were delivered before write-back existed
		post, keys := fs.Bool("post", false, "actually post (default: only print what would be posted)"), fs.String("keys", "", "comma-separated keys (default: every passing case)")
		fs.Parse(os.Args[2:])
		var on []string
		if *keys != "" {
			on = strings.Split(*keys, ",")
		}
		err = cfg.Announce(on, *post)
	case "deliver":
		only, exclude := fs.String("only", "", "comma-separated keys to deliver (default: every passing case)"), fs.String("exclude", "", "comma-separated keys to leave out")
		fs.StringVar(&cfg.DeliverRepo, "repo", cfg.DeliverRepo, "framework repo that receives the tests")
		fs.IntVar(&cfg.StabilityRuns, "runs", cfg.StabilityRuns, "consecutive passes in the real framework a case needs (stability gate)")
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
			if d.Report != "" {
				exec.Command("open", d.Report).Start() // macOS; elsewhere the path above is enough
			}
			cfg.writeBack(d)
			if d.Gate != "" {
				os.Exit(3) // distinct from 1 (error): the pipeline worked and is asking a person
			}
		}
	case "poll":
		jql := fs.String("jql", cfg.JQL, "which issues to look at (profile: jql)")
		comment := fs.Bool("comment", cfg.JiraComments, "post outcomes that need a person to the issue (off until results are trusted)")
		intent, only, force, asJSON := fs.String("intent", "", "with --only: treat the issue as automate|execute even without the label"), fs.String("only", "", "process just this issue key (for a manual trigger)"), fs.Bool("force", false, "ignore the seen-state: run it again"), fs.Bool("json", false, "one JSON object per run (for n8n)")
		modelCases := fs.Int("model-cases", 1, "cases that need the model allowed per poll; the rest wait for the next one")
		if cfg.Registry != nil {
			*modelCases = cfg.Registry.ModelCases()
		}
		fs.Parse(os.Args[2:])
		jsonOut = *asJSON
		gated, perr := Poll([]*Config{cfg}, PollOpts{JQL: *jql, Comment: *comment, Only: *only, Force: *force, Intent: *intent, ModelCases: *modelCases})
		err = perr
		if gated && err == nil {
			os.Exit(3)
		}
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "hydra:", err)
		os.Exit(1)
	}
}
