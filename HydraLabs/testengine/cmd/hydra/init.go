package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// hydra init: onboard a team by asking. It collects and checks the answers, shows a summary, and only after a yes
// writes the profile and the registry entry and runs the existing Onboard (repo from the template, first crawl).
// Every answer can come from a file (--answers), so the same thing runs unattended from CI or n8n.
//
// Questions come in sets: "core" for every team, then one set per add-on the team enables. A new add-on is one more
// entry in addons below, with its own questions and the profile fields it writes; nothing else here changes.

type question struct {
	key, prompt, def string
	when             func(a answers) bool                              // nil = always asked
	check            func(v string, a answers) (string, string, error) // -> normalized value, warning, error
}

type addon struct {
	name, title, needs string // needs: what it relies on, said up front instead of failing at the first poll
	questions          []question
}

type answers map[string]string // "section.key" -> value

func (a answers) yes(k string) bool { return strings.HasPrefix(strings.ToLower(a[k]), "y") }

var initAddons = []addon{
	{name: "ploy", title: "HydraPloy: cases in, gates, verify, stability runs, delivery to review branches",
		needs: "automating new cases (intent automate) needs labs too; without it only stored tests run (execute)",
		questions: []question{
			{key: "source", prompt: "Where do the test cases live? (jira | none; CSV and in-repo files are coming)", def: "jira",
				check: oneOf("jira", "none")},
			{key: "jiraKey", prompt: "Jira project key", when: is("ploy.source", "jira"),
				check: upperID}, // default: the project id (filled in when asked)
			{key: "jql", prompt: "Which issues to pick up (JQL)", when: is("ploy.source", "jira"), check: checkJQL},
			{key: "comments", prompt: "Post results back to the Jira issues? (y/N)", def: "n", when: is("ploy.source", "jira"), check: yesNo},
			{key: "stabilityRuns", prompt: "Passes in a row a test needs before it is delivered", def: "3", check: intIn(1, 10)},
		}},
	{name: "labs", title: "HydraLabs: crawl the app, map its elements, build tests from the cases",
		questions: []question{
			{key: "paths", prompt: "Pages to start the crawl from (comma separated)", def: "/", check: checkPaths},
			{key: "model", prompt: "Use the local model when the rules cannot decide (tie-breaks, exploring a page)? (y/N)", def: "n", check: yesNo},
		}},
}

func coreQuestions(reg *Registry) []question {
	return []question{
		{key: "project", prompt: "Project id (short, upper case; the Jira key if the team uses Jira)", check: func(v string, a answers) (string, string, error) {
			v, w, err := upperID(v, a)
			if err == nil && reg != nil {
				if _, taken := reg.Projects[v]; taken {
					err = fmt.Errorf("%s is already registered", v)
				}
			}
			return v, w, err
		}},
		{key: "app", prompt: "App name (one word; the test repo will be Hydra-<App>)", check: func(v string, a answers) (string, string, error) {
			if !regexp.MustCompile(`^[A-Z][A-Za-z0-9]+$`).MatchString(v) {
				return v, "", fmt.Errorf("one word starting with a capital, letters and digits only (e.g. CobroOrdenes)")
			}
			return v, "", nil
		}},
		{key: "addons", prompt: "Add-ons to enable (" + addonNames() + ")", def: addonNames(), check: checkAddons},
		{key: "baseUrl", prompt: "URL of the QA environment of the app", check: checkURL},
		{key: "prod", prompt: "Is that URL production? (y/N)", def: "n", check: yesNo},
		{key: "allowWrites", prompt: "May tests change data there (create, upload, pay...)? (y/N)", def: "n",
			when: func(a answers) bool { return !a.yes("core.prod") }, check: yesNo},
		{key: "baseBranch", prompt: "Base branch of the test repo", def: "main", check: required},
		{key: "remote", prompt: "Remote URL of the test repo (empty = local only for now)", def: "", check: checkRemote},
	}
}

// ---- checks (package vars so tests can replace the ones that touch the network) ----

var (
	reachable = func(url string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			resp.Body.Close()
		}
		return err
	}
	jiraCount = func(jql string) (int, error) {
		j, err := JiraFromEnv()
		if err != nil {
			return 0, err
		}
		is, err := j.Search(jql)
		return len(is), err
	}
	remoteOK = func(url string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		out, err := exec.CommandContext(ctx, "git", "ls-remote", "--heads", url).CombinedOutput()
		if err != nil {
			return fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
		}
		return nil
	}
)

func addonNames() string {
	var ns []string
	for _, a := range initAddons {
		ns = append(ns, a.name)
	}
	return strings.Join(ns, ",")
}

func is(k, v string) func(a answers) bool { return func(a answers) bool { return a[k] == v } }

func required(v string, _ answers) (string, string, error) {
	if v == "" {
		return v, "", fmt.Errorf("required")
	}
	return v, "", nil
}

func upperID(v string, _ answers) (string, string, error) {
	v = strings.ToUpper(v)
	if !regexp.MustCompile(`^[A-Z][A-Z0-9]{1,9}$`).MatchString(v) {
		return v, "", fmt.Errorf("2-10 letters/digits, starting with a letter (e.g. DWQ)")
	}
	return v, "", nil
}

func yesNo(v string, _ answers) (string, string, error) {
	switch strings.ToLower(v) {
	case "y", "yes", "true":
		return "y", "", nil
	case "n", "no", "false", "":
		return "n", "", nil
	}
	return v, "", fmt.Errorf("y or n")
}

func oneOf(opts ...string) func(string, answers) (string, string, error) {
	return func(v string, _ answers) (string, string, error) {
		for _, o := range opts {
			if strings.EqualFold(v, o) {
				return o, "", nil
			}
		}
		return v, "", fmt.Errorf("one of: %s", strings.Join(opts, ", "))
	}
}

func intIn(lo, hi int) func(string, answers) (string, string, error) {
	return func(v string, _ answers) (string, string, error) {
		n, err := strconv.Atoi(v)
		if err != nil || n < lo || n > hi {
			return v, "", fmt.Errorf("a number from %d to %d", lo, hi)
		}
		return v, "", nil
	}
}

func checkAddons(v string, _ answers) (string, string, error) {
	var got []string
	for _, n := range strings.Split(strings.ReplaceAll(v, " ", ""), ",") {
		if n == "" {
			continue
		}
		if _, ok := findAddon(n); !ok {
			return v, "", fmt.Errorf("unknown add-on %q (known: %s)", n, addonNames())
		}
		got = append(got, n)
	}
	if len(got) == 0 {
		return v, "", fmt.Errorf("at least one of: %s", addonNames())
	}
	warn := ""
	if ploy, _ := findAddon("ploy"); contains(got, "ploy") && !contains(got, "labs") {
		warn = "ploy without labs: " + ploy.needs
	}
	return strings.Join(got, ","), warn, nil
}

func findAddon(n string) (addon, bool) {
	for _, a := range initAddons {
		if a.name == n {
			return a, true
		}
	}
	return addon{}, false
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

func checkURL(v string, _ answers) (string, string, error) {
	v = strings.TrimRight(v, "/")
	if !regexp.MustCompile(`^https?://[^\s/]+`).MatchString(v) {
		return v, "", fmt.Errorf("a full URL, e.g. http://127.0.0.1:3000")
	}
	if err := reachable(v); err != nil { // a warning, not an error: VPN off or app stopped right now is normal
		return v, "not reachable right now (" + err.Error() + "): the first crawl and runs will wait until it is", nil
	}
	return v, "", nil
}

func checkRemote(v string, _ answers) (string, string, error) {
	if v == "" {
		return v, "", nil
	}
	if err := remoteOK(v); err != nil {
		return v, "cannot read it right now (" + err.Error() + "); kept, nothing is pushed until delivery opens PRs", nil
	}
	return v, "", nil
}

func checkJQL(v string, a answers) (string, string, error) {
	if v == "" {
		v = fmt.Sprintf(`project = %s AND labels in ("automate", "execute") ORDER BY key ASC`, a["ploy.jiraKey"])
	}
	n, err := jiraCount(v)
	if err != nil {
		return v, "could not check it against Jira (" + err.Error() + "); kept as written", nil
	}
	return v, fmt.Sprintf("%d issue(s) match today", n), nil
}

func checkPaths(v string, _ answers) (string, string, error) {
	for _, p := range strings.Split(v, ",") {
		if !strings.HasPrefix(strings.TrimSpace(p), "/") {
			return v, "", fmt.Errorf("paths start with /, e.g. /,/login")
		}
	}
	return strings.ReplaceAll(v, " ", ""), "", nil
}

// ---- the conversation ----

type initRun struct {
	in    *bufio.Reader
	out   io.Writer
	given answers // from --answers
	yes   bool    // --yes: no prompts; missing answers take their default
	a     answers
	warns []string
}

func (r *initRun) ask(section string, q question) error {
	k := section + "." + q.key
	if q.when != nil && !q.when(r.a) {
		return nil
	}
	for {
		v, from := q.def, "default"
		if g, ok := r.given[k]; ok {
			v, from = g, "file"
		} else if !r.yes {
			fmt.Fprintf(r.out, "%s", q.prompt)
			if q.def != "" {
				fmt.Fprintf(r.out, " [%s]", q.def)
			}
			fmt.Fprint(r.out, ": ")
			line, err := r.in.ReadString('\n')
			if line = strings.TrimSpace(line); line != "" {
				v, from = line, "typed"
			}
			if err != nil && line == "" && q.def == "" {
				return fmt.Errorf("%s: no answer (input ended)", k)
			}
		}
		val, warn, err := v, "", error(nil)
		if q.check != nil {
			val, warn, err = q.check(v, r.a)
		}
		if err != nil {
			if from != "typed" { // a wrong answer from a file or a default cannot be asked again
				return fmt.Errorf("%s = %q: %v", k, v, err)
			}
			fmt.Fprintf(r.out, "  ✗ %v\n", err)
			continue
		}
		r.a[k] = val
		if warn != "" {
			fmt.Fprintf(r.out, "  ! %s\n", warn)
			r.warns = append(r.warns, q.key+": "+warn)
		}
		return nil
	}
}

func (r *initRun) enabled(name string) bool {
	return contains(strings.Split(r.a["core.addons"], ","), name)
}

// profile turns the answers into the profile file the rest of hydra reads (same fields a person would write).
func (a answers) profile() map[string]any {
	p := defaultProfile(a["core.project"], a["core.app"])
	addons := strings.Split(a["core.addons"], ",")
	p["addons"], p["baseUrl"], p["baseBranch"] = addons, a["core.baseUrl"], a["core.baseBranch"]
	p["allowWrites"] = a.yes("core.allowWrites") && !a.yes("core.prod")
	if a["core.remote"] != "" {
		p["remote"] = a["core.remote"]
	}
	p["_doc"] = "Created by hydra init. Edit by hand or run init again for a new project; a person owns this file."
	delete(p, "jql")
	if contains(addons, "ploy") {
		n, _ := strconv.Atoi(a["ploy.stabilityRuns"])
		p["stabilityRuns"] = n
		if a["ploy.source"] == "jira" {
			p["jql"], p["jiraComments"] = a["ploy.jql"], a.yes("ploy.comments")
		}
	}
	if contains(addons, "labs") {
		p["paths"], p["storeCrawl"] = strings.Split(a["labs.paths"], ","), true
		if !a.yes("labs.model") { // no means none: neither the T2 tie-break nor the T3 agent; a person decides instead
			delete(p, "agent")
			delete(p, "model")
		}
	} else {
		delete(p, "agent")
		delete(p, "model")
	}
	if a.yes("core.prod") {
		p["_environment"] = "PRODUCTION: tests never write here (allowWrites is forced off)."
	}
	return p
}

func (a answers) summary(w io.Writer, profilePath, repo string) {
	fmt.Fprintf(w, "\n  project   %s  →  app %s\n", a["core.project"], a["core.app"])
	fmt.Fprintf(w, "  add-ons   %s\n", a["core.addons"])
	writes := "no (read-only: every non-GET request is blocked)"
	if a.yes("core.prod") {
		writes = "no: production"
	} else if a.yes("core.allowWrites") {
		writes = "YES: tests may change data in this environment"
	}
	fmt.Fprintf(w, "  QA URL    %s   writes: %s\n", a["core.baseUrl"], writes)
	remote := a["core.remote"]
	if remote == "" {
		remote = "local only"
	}
	fmt.Fprintf(w, "  test repo %s (branch %s, remote: %s)\n", repo, a["core.baseBranch"], remote)
	if src := a["ploy.source"]; src != "" {
		fmt.Fprintf(w, "  cases     %s", src)
		if src == "jira" {
			fmt.Fprintf(w, ": %s   write-back: %s", a["ploy.jql"], a["ploy.comments"])
		}
		fmt.Fprintf(w, "\n  delivery  after %s passes in a row\n", a["ploy.stabilityRuns"])
	}
	if p := a["labs.paths"]; p != "" {
		fmt.Fprintf(w, "  crawl     %s   model tier: %s\n", p, a["labs.model"])
	}
	fmt.Fprintf(w, "  writes    %s and an entry in projects.json\n\n", profilePath)
}

// Init runs the whole conversation. preview, when not nil, crawls one page of the app and returns a one-line summary.
func Init(reg *Registry, regPath string, in io.Reader, out io.Writer, given answers, yes bool, preview func(answers) string) (answers, error) {
	r := &initRun{in: bufio.NewReader(in), out: out, given: given, yes: yes, a: answers{}}
	fmt.Fprintln(out, "HydraPloy / HydraLabs onboarding. Nothing is written until you confirm at the end.")
	for _, q := range coreQuestions(reg) {
		if err := r.ask("core", q); err != nil {
			return r.a, err
		}
	}
	for _, ad := range initAddons {
		if !r.enabled(ad.name) {
			continue
		}
		fmt.Fprintf(out, "\n— %s\n", ad.title)
		for _, q := range ad.questions {
			if q.key == "jiraKey" && q.def == "" {
				q.def = r.a["core.project"]
			}
			if err := r.ask(ad.name, q); err != nil {
				return r.a, err
			}
		}
	}

	if preview != nil && r.enabled("labs") {
		fmt.Fprintln(out, "\nPreview: crawling one page (read-only)...")
		fmt.Fprintln(out, "  "+preview(r.a))
	}
	profilePath := filepath.Join(reg.dir, "profiles", strings.ReplaceAll(appSlug(r.a["core.app"]), "_", "-")+".json")
	r.a.summary(out, profilePath, reg.RepoPath(r.a["core.app"]))
	if fileExists(profilePath) {
		return r.a, fmt.Errorf("%s already exists: pick another app name or remove it first", profilePath)
	}
	if !yes {
		fmt.Fprint(out, "Write this and create the repo? (y/N): ")
		line, _ := r.in.ReadString('\n')
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(line)), "y") {
			return r.a, fmt.Errorf("cancelled: nothing was written")
		}
	}
	if err := writeJSON(profilePath, r.a.profile()); err != nil {
		return r.a, err
	}
	rel, _ := filepath.Rel(reg.dir, profilePath)
	if err := addToRegistry(regPath, r.a["core.project"], r.a["core.app"], rel); err != nil {
		os.Remove(profilePath) // no half-registered project
		return r.a, err
	}
	reg.Projects[r.a["core.project"]] = ProjectEntry{App: r.a["core.app"], Profile: rel}
	return r.a, nil
}

// addToRegistry edits projects.json as a generic document, so fields this code does not know (_doc...) survive.
func addToRegistry(path, project, app, profile string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	ps, _ := doc["projects"].(map[string]any)
	if ps == nil {
		ps = map[string]any{}
	}
	if _, taken := ps[project]; taken {
		return fmt.Errorf("%s is already in %s", project, path)
	}
	ps[project] = map[string]string{"app": app, "profile": profile}
	doc["projects"] = ps
	return writeJSON(path, doc)
}

// loadAnswers reads {"core": {...}, "ploy": {...}, "labs": {...}} into "section.key" answers.
func loadAnswers(path string) (answers, error) {
	a := answers{}
	if path == "" {
		return a, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc map[string]map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	for sec, kv := range doc {
		for k, v := range kv {
			switch x := v.(type) {
			case bool:
				a[sec+"."+k] = map[bool]string{true: "y", false: "n"}[x]
			case []any:
				var parts []string
				for _, p := range x {
					parts = append(parts, fmt.Sprint(p))
				}
				a[sec+"."+k] = strings.Join(parts, ",")
			default:
				a[sec+"."+k] = fmt.Sprint(x)
			}
		}
	}
	return a, nil
}

// loadHydraEnv reads the pipeline's secrets file the way n8n/start.sh does, so init can check Jira from a plain
// terminal. Only names it does not already have are set; values are never printed.
func loadHydraEnv(paths ...string) {
	for _, p := range paths {
		b, err := os.ReadFile(expand(p))
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(b), "\n") {
			k, v, ok := strings.Cut(strings.TrimSpace(strings.TrimPrefix(line, "export ")), "=")
			if !ok || strings.HasPrefix(k, "#") || os.Getenv(k) != "" {
				continue
			}
			os.Setenv(k, strings.Trim(v, `"'`))
		}
	}
}

// hasAddon: a profile written before add-ons existed has none listed and keeps everything it had.
func (c Config) hasAddon(name string) bool { return len(c.Addons) == 0 || contains(c.Addons, name) }

// runInit is the command: the conversation, then the existing Onboard for the new project.
func runInit(cfg *Config, answersPath string, yes, noPreview bool) error {
	if cfg.Registry == nil {
		return fmt.Errorf("init needs the registry (projects.json) in the working directory")
	}
	regPath := filepath.Join(cfg.Registry.dir, "projects.json")
	loadHydraEnv("~/.hydra.env", filepath.Join(cfg.Registry.dir, "..", ".hydra.env"))
	given, err := loadAnswers(answersPath)
	if err != nil {
		return err
	}
	var preview func(answers) string
	if !noPreview {
		preview = func(a answers) string { return previewCrawl(*cfg, a) }
	}
	a, err := Init(cfg.Registry, regPath, os.Stdin, os.Stdout, given, yes, preview)
	if err != nil {
		return err
	}
	key := a["core.project"]
	pc, err := cfgFlags(flag.NewFlagSet("init-"+key, flag.ContinueOnError), []string{"--project", key, "--registry", regPath})
	if err != nil {
		return err
	}
	fmt.Printf("Registered %s. Creating %s and, if the app answers, its first crawl...\n", key, pc.Home)
	o, err := pc.Registry.Onboard(key, pc)
	if err != nil {
		return err
	}
	if pc.Remote != "" {
		if _, e := git(o.Repo, "remote", "get-url", "origin"); e != nil {
			if _, e := git(o.Repo, "remote", "add", "origin", pc.Remote); e != nil {
				fmt.Fprintln(os.Stderr, "hydra: could not set the remote:", e)
			}
		}
	}
	b, _ := json.MarshalIndent(o, "", "  ")
	fmt.Println(string(b))
	if o.Gate != "" {
		os.Exit(3) // registered and created, but a person must act before the pipeline can continue
	}
	fmt.Printf("\nDone. Next: hydra status   ·   hydra crawl --project %s --watch :8099\n", key)
	return nil
}

// previewCrawl maps one page into a throwaway folder (read-only, like every crawl) and says what it found.
func previewCrawl(cfg Config, a answers) string {
	tmp, err := os.MkdirTemp("", "hydra-init-")
	if err != nil {
		return "preview skipped: " + err.Error()
	}
	defer os.RemoveAll(tmp)
	c := cfg
	c.Home, c.Registry, c.Out, c.AppMap, c.Quiet = "", nil, tmp, tmp, true
	c.App, c.PortalURL, c.MaxPages, c.SafeClicks, c.LoginFile, c.StoreCrawl = a["core.app"], a["core.baseUrl"], 1, false, "", false
	c.Paths = []string{strings.SplitN(a["labs.paths"]+",", ",", 2)[0]}
	if err := c.Crawl("", false); err != nil {
		return "preview failed (" + err.Error() + "); the first crawl after setup will try again"
	}
	var m struct {
		Pages []struct {
			URL      string `json:"url"`
			Settled  *bool  `json:"settled"`
			Elements []struct {
				Strategy string `json:"strategy"`
				Stable   bool   `json:"stable"`
			} `json:"elements"`
		} `json:"pages"`
	}
	if b, err := os.ReadFile(c.crawlPath()); err != nil || json.Unmarshal(b, &m) != nil || len(m.Pages) == 0 {
		return "preview found no page (a login, or nothing rendered?)"
	}
	p := m.Pages[0]
	strong, role, weak := 0, 0, 0
	for _, e := range p.Elements {
		switch {
		case !e.Stable:
			weak++
		case e.Strategy == "testid" || e.Strategy == "id":
			strong++
		default:
			role++
		}
	}
	n := len(p.Elements)
	if n == 0 {
		return "the page loaded but has no interactive elements (a login wall or a canvas app?)"
	}
	s := fmt.Sprintf("%s: %d elements · %d%% test id / stable id · %d%% role + name · %d weak", p.URL, n, strong*100/n, role*100/n, weak)
	if p.Settled != nil && !*p.Settled {
		s += " · the page never went quiet (bot protection or a slow API?): parts may be missing"
	}
	return s
}
