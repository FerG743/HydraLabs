package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Deliver (stage 6): passing cases -> a review branch in the framework repo, verified by the REAL framework.
//
//	1. take the cases whose last run passed (minus --exclude)
//	2. render them together (shared locators/module are the union of all of them)
//	3. new branch from the base branch, write the files, register the suite in runner_central.py
//	4. run the real pytest with the write-guard plugin
//	5. all green -> commit on the branch.  ANY failure -> nothing is committed, the repo is restored, and the
//	   failures are reported: that is the human gate. A person decides (fix, --exclude, or accept) and re-runs.
// Never merges, never pushes.

type CaseOutcome struct {
	Key    string `json:"key"`
	Status string `json:"status"` // passed | failed | flaky | skipped | missing
	Detail string `json:"detail,omitempty"`
	Passes int    `json:"passes"` // runs it passed, out of Of runs requested
	Of     int    `json:"of"`
}

type Delivery struct {
	Branch    string        `json:"branch,omitempty"`
	Commit    string        `json:"commit,omitempty"`
	Runs      int           `json:"runs"` // consecutive real-framework runs actually executed
	Delivered []string      `json:"delivered,omitempty"`
	Gate      string        `json:"gate,omitempty"` // why nothing was committed
	Note      string        `json:"note,omitempty"` // an outcome that needs no person (e.g. already delivered)
	Outcomes  []CaseOutcome `json:"outcomes"`
}

func expand(p string) string {
	if strings.HasPrefix(p, "~/") {
		h, _ := os.UserHomeDir()
		return filepath.Join(h, p[2:])
	}
	return p
}

func git(repo string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(errb.String()))
	}
	return strings.TrimSpace(out.String()), nil
}

// passedKeys: the keys whose latest run record says passed, filtered by only/exclude.
func (c Config) passedKeys(only []string, exclude map[string]bool) ([]string, error) {
	files, _ := filepath.Glob(filepath.Join(c.Out, "runs", "*.json"))
	want := map[string]bool{}
	for _, k := range only {
		want[k] = true
	}
	var keys []string
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		var r Run
		if json.Unmarshal(b, &r) != nil || r.Status != "passed" || exclude[r.Key] || (len(want) > 0 && !want[r.Key]) {
			continue
		}
		keys = append(keys, r.Key)
	}
	sort.Strings(keys)
	return keys, nil
}

var choicesRe = regexp.MustCompile(`(choices=\[[^\]]*)'all'`)

// registerSuite adds the app to runner_central.py the way the README prescribes. If the file does not look like the
// template it is left alone and the caller says so: a wrong edit to the runner is worse than a missing registration.
func registerSuite(path, app string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	s := string(b)
	if strings.Contains(s, "'"+app+"'") {
		return "", nil
	}
	if !choicesRe.MatchString(s) || !strings.Contains(s, "    elif args.suite == 'all':") {
		return "runner_central.py does not match the template: register the '" + app + "' suite by hand (README, Multi-app)", nil
	}
	s = choicesRe.ReplaceAllString(s, "${1}'"+app+"', 'all'")
	block := fmt.Sprintf("    elif args.suite == '%s':\n        test_targets.append(os.path.join(tests_dir, \"%s\"))\n        report_name = \"report_%s.html\"\n", app, app, app)
	s = strings.Replace(s, "    elif args.suite == 'all':", block+"    elif args.suite == 'all':", 1)
	return "", os.WriteFile(path, []byte(s), 0o644)
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || strings.Contains(p, "__pycache__") || filepath.Base(p) == "locators.confirmed.json" {
			return err // the confirmed-locators file is pipeline metadata, not part of the framework
		}
		rel, _ := filepath.Rel(src, p)
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		out := filepath.Join(dst, rel)
		os.MkdirAll(filepath.Dir(out), 0o755)
		return os.WriteFile(out, b, 0o644)
	})
}

type junitCase struct {
	Class   string `xml:"classname,attr"`
	Name    string `xml:"name,attr"`
	Failure *struct {
		Message string `xml:"message,attr"`
	} `xml:"failure"`
	Error *struct {
		Message string `xml:"message,attr"`
	} `xml:"error"`
	Skipped *struct{} `xml:"skipped"`
}
type junitSuite struct {
	Cases []junitCase `xml:"testcase"`
}

var tokRe = regexp.MustCompile(`test_([a-z]+_\d+)`)

func parseJUnit(path string) (map[string]CaseOutcome, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var many struct {
		Suites []junitSuite `xml:"testsuite"`
	}
	var one junitSuite
	var cases []junitCase
	if xml.Unmarshal(b, &many) == nil && len(many.Suites) > 0 {
		for _, s := range many.Suites {
			cases = append(cases, s.Cases...)
		}
	} else if err := xml.Unmarshal(b, &one); err == nil {
		cases = one.Cases
	}
	out := map[string]CaseOutcome{}
	for _, tc := range cases {
		m := tokRe.FindStringSubmatch(tc.Class + "." + tc.Name)
		if m == nil {
			continue
		}
		key := strings.Replace(strings.ToUpper(m[1]), "_", "-", 1)
		o := CaseOutcome{Key: key, Status: "passed"}
		switch {
		case tc.Failure != nil:
			o.Status, o.Detail = "failed", tc.Failure.Message
		case tc.Error != nil:
			o.Status, o.Detail = "failed", tc.Error.Message
		case tc.Skipped != nil:
			o.Status = "skipped"
		}
		if len(o.Detail) > 400 {
			o.Detail = o.Detail[:400]
		}
		out[key] = o
	}
	return out, nil
}

func (c Config) Deliver(only []string, exclude map[string]bool) (d Delivery, err error) {
	repo, base := expand(c.DeliverRepo), c.BaseBranch
	if repo == "" || c.PortalURL == "" {
		return d, fmt.Errorf("profile needs deliverRepo and baseUrl")
	}
	if !fileExists(filepath.Join(repo, ".git")) {
		return d, fmt.Errorf("%s is not a repo yet: run hydra onboard --project <KEY> first", repo)
	}
	if st, e := git(repo, "status", "--porcelain"); e != nil || st != "" {
		return d, fmt.Errorf("the delivery repo must be clean before delivering (git status --porcelain: %q %v)", st, e)
	}
	if _, e := git(repo, "rev-parse", "--verify", base); e != nil {
		return d, fmt.Errorf("base branch %q has no commits yet: make a baseline commit first", base)
	}
	keys, err := c.passedKeys(only, exclude)
	if err != nil || len(keys) == 0 {
		return d, fmt.Errorf("no passing cases to deliver (%v)", err)
	}

	stamp := time.Now().Format("20060102-150405")
	work := filepath.Join(c.Out, "deliver", stamp)
	stage, render := filepath.Join(work, "cases"), filepath.Join(work, "render")
	for _, k := range keys {
		b, e := os.ReadFile(filepath.Join(c.caseDir(), k+".case.json"))
		if e != nil {
			return d, e
		}
		os.MkdirAll(stage, 0o755)
		os.WriteFile(filepath.Join(stage, k+".case.json"), b, 0o644)
	}
	if _, err = c.renderOnce(stage, render, ""); err != nil { // all together: shared files are the union
		return d, err
	}

	d.Branch = fmt.Sprintf("hydra/%s-%s", appSlug(c.App), stamp)
	if _, err = git(repo, "checkout", "-q", "-b", d.Branch, base); err != nil {
		return d, err
	}
	restore := func() { // back to a clean base; used whenever nothing is committed
		git(repo, "reset", "--hard", "-q")
		git(repo, "clean", "-fdq")
		git(repo, "checkout", "-q", base)
		git(repo, "branch", "-D", "-q", d.Branch)
	}
	for _, sub := range []string{"apps", "tests", "utils"} {
		if err = copyTree(filepath.Join(render, sub), filepath.Join(repo, sub)); err != nil {
			restore()
			return d, err
		}
	}
	note, err := registerSuite(filepath.Join(repo, "runner_central.py"), c.App)
	if err != nil {
		restore()
		return d, err
	}

	// Stability gate: a case counts as automated only if it passes `runs` consecutive runs in the real framework.
	// Repeating stops at the first run where anything fails: a person has to look, more runs would only waste time.
	runs := c.StabilityRuns
	if runs < 1 {
		runs = 1
	}
	var results []map[string]CaseOutcome
	var lastLog string
	for i := 1; i <= runs; i++ {
		got, logPath, perr := c.runPytest(repo, work, i)
		lastLog = logPath
		if perr != nil {
			d.Gate = "pytest did not run: " + perr.Error()
			break
		}
		results = append(results, got)
		if !allPassed(keys, got) {
			break
		}
	}
	d.Runs = len(results)
	d.Outcomes = judgeRuns(keys, results, runs)
	allGreen := d.Gate == "" && len(results) == runs
	for _, o := range d.Outcomes {
		if o.Status != "passed" || o.Passes != runs {
			allGreen = false
		}
	}
	if !allGreen {
		if d.Gate == "" {
			d.Gate = "the real framework did not pass every case on every run: nothing was committed. A person decides: fix, --exclude the case, or accept. Log: " + lastLog
		}
		restore()
		d.Branch = ""
		return d, c.recordDelivery(d, stamp)
	}

	if st, _ := git(repo, "status", "--porcelain"); st == "" { // same files as the base branch: nothing to review
		d.Note = fmt.Sprintf("already delivered: the generated files match %s and the cases are still stable over %d runs", base, runs)
		restore()
		d.Branch = ""
		return d, c.recordDelivery(d, stamp)
	}
	if _, err = git(repo, "add", "apps", "tests", "utils", "runner_central.py"); err != nil {
		restore()
		return d, err
	}
	msg := fmt.Sprintf("hydra: %s (%s)\n\nGenerated from Jira; passed %d consecutive runs in the framework against %s.\n%s", strings.Join(keys, ", "), c.App, runs, c.PortalURL, note)
	if _, err = git(repo, "commit", "-q", "-m", msg); err != nil {
		restore()
		return d, err
	}
	d.Commit, _ = git(repo, "rev-parse", "--short", "HEAD")
	d.Delivered = keys
	git(repo, "checkout", "-q", base) // leave the user on the base branch; the work waits on its review branch
	return d, c.recordDelivery(d, stamp)
}

func (c Config) recordDelivery(d Delivery, stamp string) error {
	return writeJSON(filepath.Join(c.Out, "deliveries", stamp+".json"), d)
}

func tail(s string, n int) string {
	if len(s) > n {
		return s[len(s)-n:]
	}
	return s
}

// runPytest runs the generated suite once in the real framework and returns each case's result.
// pytest runs from inside the repo, so every path handed to it must be absolute.
func (c Config) runPytest(repo, work string, n int) (map[string]CaseOutcome, string, error) {
	agentDir, _ := filepath.Abs(filepath.Dir(expand(c.Agent)))
	junit, _ := filepath.Abs(filepath.Join(work, fmt.Sprintf("junit-%d.xml", n)))
	logPath := filepath.Join(work, fmt.Sprintf("pytest-%d.log", n))
	cmd := exec.Command(c.Python, "-m", "pytest", "tests/"+c.App, "-p", "hydra_guard", "-v", "--tb=short", "--junitxml="+junit)
	cmd.Dir = repo
	allow := "0"
	if c.AllowWrites {
		allow = "1"
	}
	cmd.Env = append(os.Environ(), "PYTHONPATH="+repo+string(os.PathListSeparator)+agentDir,
		"BASE_URL="+c.PortalURL, "HEADLESS=true", "HYDRA_ALLOW_WRITES="+allow, "PYTHONDONTWRITEBYTECODE=1")
	var logb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &logb, &logb
	cmd.Run() // a failing test is a result, not an error; judged from the junit file
	os.WriteFile(logPath, logb.Bytes(), 0o644)
	got, err := parseJUnit(junit)
	if err != nil || len(got) == 0 {
		return nil, logPath, fmt.Errorf("%s", strings.TrimSpace(tail(logb.String(), 400)))
	}
	return got, logPath, nil
}

func allPassed(keys []string, got map[string]CaseOutcome) bool {
	for _, k := range keys {
		if got[k].Status != "passed" {
			return false
		}
	}
	return true
}

// judgeRuns turns the per-run results into one verdict per case:
//
//	passed   passed every run that was executed (and Passes == Of when all runs were done)
//	flaky    passed some runs and failed others: the worst kind, it must never reach the repo
//	failed / skipped / missing   never passed
func judgeRuns(keys []string, results []map[string]CaseOutcome, want int) []CaseOutcome {
	var out []CaseOutcome
	for _, k := range keys {
		o := CaseOutcome{Key: k, Of: want}
		var first *CaseOutcome
		for _, r := range results {
			x, ok := r[k]
			if !ok {
				x = CaseOutcome{Key: k, Status: "missing", Detail: "pytest produced no result for this case"}
			}
			if x.Status == "passed" {
				o.Passes++
			} else if first == nil {
				cp := x
				first = &cp
			}
		}
		switch {
		case len(results) == 0:
			o.Status, o.Detail = "missing", "pytest did not run"
		case first == nil:
			o.Status = "passed"
			if len(results) < want {
				o.Detail = "stability not completed: another case failed first"
			}
		case o.Passes > 0:
			o.Status, o.Detail = "flaky", fmt.Sprintf("passed %d of %d runs; first failure: %s", o.Passes, len(results), first.Detail)
		default:
			o.Status, o.Detail = first.Status, first.Detail
		}
		out = append(out, o)
	}
	return out
}
