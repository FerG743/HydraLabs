package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// The router: cheapest tier that can finish the job, escalating only when the previous one cannot.
//
//	execute  T0 existing  a bundle for this key exists            -> run it                    (0 tokens)
//	automate T1 render    deterministic parse + render + verify   -> passed | failed           (0 tokens)
//	         T4 review    steps the renderer cannot map           -> a human, with the bundle  (0 tokens)
//
// T2 (closed-list matcher) and T3 (page-exploring agent) are the only tiers that spend tokens; they plug in
// between T1 and T4 and are not wired yet, so a case they would handle stops at T4 instead of guessing.
// A case counts as automated only if its generated test passes (Verify), never because it rendered.

// Config is one project's profile (profiles/<name>.json): everything that differs between web apps lives here, never in code.
type Config struct {
	Out         string `json:"out"`
	ToolsDir    string `json:"toolsDir"`
	AppMap      string `json:"appMap"`
	Python      string `json:"python"`
	App         string `json:"app"`
	Lang        string `json:"lang"`
	PortalURL   string `json:"baseUrl"`
	JQL         string `json:"jql"`
	Agent       string `json:"agent"` // path to agent.py; empty disables T3
	Model       string `json:"model"` // LM Studio model for T3
	LMURL       string `json:"lmUrl"`
	MaxTurns    int    `json:"maxTurns"`    // hard budget for T3; past it the case goes to review
	MaxTokens   int    `json:"maxTokens"`   // hard token cap for T3 per case (a stuck run once cost 314k)
	AllowWrites bool   `json:"allowWrites"` // let tests send real writes (default: read-only safety net)

	Crawler      string              `json:"crawler"` // path to tools/crawl/crawl.py
	Paths        []string            `json:"paths"`   // entry paths for the crawl
	MaxPages     int                 `json:"maxPages"`
	Project      string              `json:"-"` // the registered Jira project this run is for
	Registry     *Registry           `json:"-"`
	DeliverRepo  string              `json:"deliverRepo"` // the framework repo that receives verified tests on review branches
	BaseBranch   string              `json:"baseBranch"`
	SlowMS       int                 `json:"slowMs"`     // crawl: reveal elements one at a time, this many ms apart (0 = as fast as possible)
	SafeClicks   bool                `json:"safeClicks"` // open comboboxes/tabs to record their options (never selects or submits)
	LoginFile    string              `json:"loginFile"`  // steps whose values are ENV: tokens, same convention as the framework CSVs
	LoginPath    string              `json:"loginPath"`
	EnvFile      string              `json:"envFile"`        // the framework's .env; the pipeline reads it only to resolve ENV: tokens
	Synonyms     map[string][]string `json:"synonyms"`       // extra word pairs for this app, e.g. {"order": ["remesa"]}
	Knowledge    string              `json:"knowledge"`      // dir with <App>.md: what is peculiar about each application
	JiraComments bool                `json:"jiraComments"`   // post every outcome that needs a person (off until results are trusted)
	ChatWebhook  string              `json:"chatWebhookEnv"` // NAME of the env var holding the Google Chat webhook URL
}

type Run struct {
	Key     string   `json:"key"`
	Intent  string   `json:"intent"`
	Tier    string   `json:"tier"`
	Status  string   `json:"status"` // passed | failed | rendered | needs-review
	Detail  []string `json:"detail,omitempty"`
	Tokens  int      `json:"tokens"`
	Millis  int64    `json:"ms"`
	Updated string   `json:"updated,omitempty"`
}

var envDown = regexp.MustCompile(`ERR_CONNECTION_REFUSED|ERR_CONNECTION_TIMED_OUT|ERR_NAME_NOT_RESOLVED|ERR_ADDRESS_UNREACHABLE|ERR_NETWORK_CHANGED`)

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func token(key string) string {
	return strings.Trim(slugRe.ReplaceAllString(strings.ToLower(key), "_"), "_")
}

func Intent(labels []string) string {
	got := ""
	for _, l := range labels {
		switch strings.ToLower(l) {
		case "execute":
			return "execute" // asking to run beats asking to build
		case "automate":
			got = "automate"
		}
	}
	return got
}

func (c Config) caseDir() string           { return filepath.Join(c.Out, "cases") }
func (c Config) bundleDir(k string) string { return filepath.Join(c.Out, "bundles", k) }

func writeJSON(path string, v any) error {
	os.MkdirAll(filepath.Dir(path), 0o755)
	b, _ := json.MarshalIndent(v, "", "  ")
	return os.WriteFile(path, b, 0o644)
}

// Ingest stores the case (phase 1). Routing happens after every new case is stored, so references resolve.
func (c Config) Ingest(is Issue) (*Case, error) {
	cs, err := ToCase(is.HTML, is.Key, is.Summary, c.Lang, is.Parent, is.Priority)
	if err != nil {
		return nil, err
	}
	return cs, writeJSON(filepath.Join(c.caseDir(), is.Key+".case.json"), cs)
}

func (c Config) loadCases() ([]*Case, error) {
	files, _ := filepath.Glob(filepath.Join(c.caseDir(), "*.case.json"))
	var out []*Case
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		cs := &Case{}
		if err := json.Unmarshal(b, cs); err != nil {
			return nil, fmt.Errorf("%s: %w", f, err)
		}
		out = append(out, cs)
	}
	return out, nil
}

// Route resolves references across everything stored, then runs the cheapest tier for this issue.
func (c Config) Route(key, intent string) (r Run) {
	t0 := time.Now()
	r = Run{Key: key, Intent: intent}
	defer func() {
		r.Millis = time.Since(t0).Milliseconds()
		writeJSON(filepath.Join(c.Out, "runs", key+".json"), r)
	}()

	if intent == "execute" {
		if _, err := os.Stat(filepath.Join(c.bundleDir(key), "tests")); err == nil {
			r.Tier = "T0-existing"
			v := c.verify(key)
			r.Status, r.Detail = v.Status, v.Detail
			return r
		}
		r.Detail = []string{"execute: no automated test for " + key + " yet, automating it first"}
	}

	cases, err := c.loadCases()
	if err != nil {
		r.Tier, r.Status, r.Detail = "T4-review", "needs-review", []string{err.Error()}
		return r
	}
	ResolveRefs(cases)
	var cs *Case
	for _, x := range cases {
		writeJSON(filepath.Join(c.caseDir(), x.ID+".case.json"), x) // keep resolved text for later runs
		if x.ID == key {
			cs = x
		}
	}
	if cs == nil {
		r.Tier, r.Status, r.Detail = "T4-review", "needs-review", []string{"case not stored"}
		return r
	}
	bad, advice := Maturity(cs)
	if len(bad) > 0 { // before any tokens: send the author specific questions
		r.Tier, r.Status, r.Detail = "T0-gate", "needs-clarification", append(r.Detail, bad...)
		return r
	}
	r.Detail = append(r.Detail, advice...)

	r.Tier = "T1-render"
	pending, spent, err := c.render(cs, cases)
	r.Tokens += spent // T2 (the bridge's closed-list pick), if it was needed
	if err != nil {
		r.Tier, r.Status, r.Detail = "T4-review", "needs-review", append(r.Detail, err.Error())
		return r
	}
	if len(pending) == 0 {
		var v Verdict
		v = c.verify(key)
		r.Status, r.Detail = v.Status, append(r.Detail, v.Detail...)
		if v.Status != "failed" || v.Assertion { // passed, unverified, write-blocked, or a possible app bug: no model can fix those
			return r
		}
	} else {
		r.Detail = append(r.Detail, "T1 could not map: "+strings.Join(pending, "; "))
	}
	if !c.agentOn() {
		if len(pending) > 0 {
			r.Tier, r.Status = "T4-review", "needs-review"
		}
		return r
	}
	return c.escalate(cs, r)
}

func (c Config) agentOn() bool { return c.Agent != "" && c.PortalURL != "" && c.Model != "" }

// escalate is T3: the agent explores the real page (read-only) and fills a validated spec; code renders the files.
// Its budget is a hard cap. The agent never grades itself: the same verify() decides.
func (c Config) escalate(cs *Case, r Run) Run {
	r.Tier = "T3-agent"
	stage, bundle := filepath.Join(c.Out, "stage", cs.ID), c.bundleDir(cs.ID)
	os.RemoveAll(bundle)
	args := []string{c.Agent, "--case", filepath.Join(stage, cs.ID+".case.json"), "--repo", bundle, "--app", c.App,
		"--base-url", c.PortalURL, "--model", c.Model, "--max-turns", fmt.Sprint(c.MaxTurns)}
	if fileExists(c.crawlPath()) {
		args = append(args, "--map", c.crawlPath()) // what was crawled is given, not rediscovered
	}
	if c.MaxTokens > 0 {
		args = append(args, "--max-tokens", fmt.Sprint(c.MaxTokens))
	}
	if notes := filepath.Join(c.Knowledge, c.App+".md"); fileExists(notes) {
		args = append(args, "--notes", notes) // the app's known quirks go to the model; they are never rediscovered
	}
	if c.LMURL != "" {
		args = append(args, "--lm-url", c.LMURL)
	}
	var out, errb bytes.Buffer
	cmd := exec.Command(c.Python, args...)
	cmd.Stdout, cmd.Stderr = &out, &errb
	cmd.Run() // a non-zero exit still prints a report; judge by the report
	var res struct {
		Finished bool
		Problems []string
		Summary  string
		Tokens   int
	}
	if err := json.Unmarshal(out.Bytes(), &res); err != nil {
		r.Tier, r.Status = "T4-review", "needs-review"
		r.Detail = append(r.Detail, fmt.Sprintf("agent output unreadable: %.300s", errb.String()))
		return r
	}
	r.Tokens += res.Tokens
	if !res.Finished || len(res.Problems) > 0 {
		r.Tier, r.Status = "T4-review", "needs-review"
		r.Detail = append(r.Detail, "agent did not finish clean within its budget: "+res.Summary)
		r.Detail = append(r.Detail, res.Problems...)
		return r
	}
	v := c.verify(cs.ID)
	r.Status, r.Detail = v.Status, append(r.Detail, v.Detail...)
	if v.Status == "failed" {
		r.Tier, r.Status = "T4-review", "needs-review" // the agent's output did not pass an independent run
	}
	return r
}

// render runs the renderer on this one case; returns the steps it could not map and the tokens the bridge's model spent.
// With a crawl available it renders, lets the bridge confirm the guessed locators against the real page, and renders again.
func (c Config) render(cs *Case, all []*Case) (pending []string, tokens int, err error) {
	stage := filepath.Join(c.Out, "stage", cs.ID)
	os.RemoveAll(stage)
	for _, x := range all { // every stored case: the renderer learns default values (a valid order, a valid store) from the others
		if err := writeJSON(filepath.Join(stage, x.ID+".case.json"), x); err != nil {
			return nil, 0, err
		}
	}
	bundle := c.bundleDir(cs.ID)
	pending, err = c.renderOnce(stage, bundle, cs.ID)
	if err != nil {
		return nil, 0, err
	}
	changed, spent, left, err := c.bridge(filepath.Join(bundle, "apps", c.App, "locators", appSlug(c.App)+"_locators.py"))
	if err != nil {
		return nil, spent, err
	}
	if changed {
		if pending, err = c.renderOnce(stage, bundle, cs.ID); err != nil {
			return nil, spent, err
		}
	}
	if len(left) > 0 {
		pending = append(pending, "locators not found on the crawled app: "+strings.Join(left, "; "))
	}
	return pending, spent, nil
}

var camel = regexp.MustCompile(`([a-z0-9])([A-Z])`)

func appSlug(app string) string { return strings.ToLower(camel.ReplaceAllString(app, "${1}_${2}")) } // CobroOrdenes -> cobro_ordenes

func (c Config) renderOnce(stage, bundle, only string) ([]string, error) {
	os.RemoveAll(bundle)
	// the app map: locators confirmed on the real page (learned from the crawl, or written by a human) replace the guesses
	if conf := c.confirmed(); len(conf.Locators) > 0 || len(conf.Widgets) > 0 {
		dst := filepath.Join(bundle, "apps", c.App)
		os.MkdirAll(dst, 0o755)
		if err := writeJSON(filepath.Join(dst, "locators.confirmed.json"), conf); err != nil {
			return nil, err
		}
	}
	// ponytail: renderer is still Python (golden-tested); port when it changes next.
	cmd := exec.Command(c.Python, filepath.Join(c.ToolsDir, "render_pytest.py"), "--cases", stage, "--only", only, "--app", c.App, "--out", bundle)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("render failed: %v: %.300s", err, errb.String())
	}
	var rep []struct{ Pending []string }
	if err := json.Unmarshal(out.Bytes(), &rep); err != nil {
		return nil, fmt.Errorf("render report unreadable: %v", err)
	}
	var pending []string
	for _, x := range rep {
		pending = append(pending, x.Pending...)
	}
	return pending, nil
}

type Verdict struct {
	Status    string // passed | failed | rendered (not verified) | needs-writes (only the read-only guard stopped it)
	Detail    []string
	Assertion bool // failed on an assertion about the app's behavior: a possible app bug, not a test bug
}

// verify runs the generated test (read-only unless AllowWrites). Without a base URL it stays "rendered".
func (c Config) verify(key string) Verdict {
	if c.PortalURL == "" {
		return Verdict{Status: "rendered", Detail: []string{"no baseUrl: bundle written, not verified (not counted as automated)"}}
	}
	args := []string{filepath.Join(c.ToolsDir, "run_generated.py"), "--out", c.bundleDir(key), "--test", token(key), "--base-url", c.PortalURL}
	if c.AllowWrites {
		args = append(args, "--allow-writes")
	}
	var out, errb bytes.Buffer
	cmd := exec.Command(c.Python, args...)
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	var res struct {
		Status        string
		Error         string
		AbortedWrites []string `json:"aborted_writes"`
	}
	if json.Unmarshal(out.Bytes(), &res) != nil {
		return Verdict{Status: "failed", Detail: []string{fmt.Sprintf("runner output unreadable: %.300s", errb.String())}}
	}
	if err == nil && res.Status == "passed" {
		return Verdict{Status: "passed"}
	}
	msg := strings.TrimSpace(res.Error)
	if len(res.AbortedWrites) > 0 { // the test needed a real write the safety net refused; not a locator problem
		return Verdict{Status: "needs-writes", Detail: []string{"blocked write(s): " + strings.Join(res.AbortedWrites, ", ") + "; set allowWrites in the profile to run for real", msg[:min(300, len(msg))]}}
	}
	if envDown.MatchString(msg + " " + string(out.Bytes())) {
		// Not a test problem and not an app bug: the app (or the VPN, or the backend) is not there. A model cannot fix it.
		return Verdict{Status: "env-down", Detail: []string{"the app under test is unreachable (" + c.PortalURL + "): is it running, is the VPN on? " + msg[:min(200, len(msg))]}}
	}
	if strings.HasPrefix(msg, "AssertionError") {
		// The test found the elements and the app did something else than the case expects. That is what a test is for:
		// a person decides whether it is a bug, and a model must not "fix" it by loosening the check.
		return Verdict{Status: "failed", Detail: []string{"assertion failed (possible app bug, not a test bug): " + msg[:min(500, len(msg))]}, Assertion: true}
	}
	return Verdict{Status: "failed", Detail: []string{msg[:min(600, len(msg))]}}
}

func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }
