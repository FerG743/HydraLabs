package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Stage 0, design: a requirement in, test cases drafted, a person approves. The model only chooses steps and expected
// results from closed lists (code writes the sentences the renderer maps), from the requirement and the page's real elements (the crawl).
// Code judges every draft the way the pipeline later will: the draft becomes the same Jira description a person
// writes, is read back by intake, and must pass the maturity gate, use only values the requirement or the app gives,
// and render at T1 with every step mapped and every expected result asserted. Drafts are saved for reading; with
// --create the good ones go to Jira labeled hydra-draft, never automate: adding automate is the person's approval.
// shortcut: data comes from the requirement; code-enumerated boundary/invalid data is the next step for this stage.

type Draft struct {
	Name         string   `json:"name"`
	Precondition string   `json:"precondition"`
	Steps        []string `json:"steps"`
	Expected     []string `json:"expected"`
}

type DraftResult struct {
	ID       string   `json:"id"`
	Draft    Draft    `json:"draft"`
	OK       bool     `json:"ok"`
	Problems []string `json:"problems,omitempty"`
	Issue    string   `json:"issue,omitempty"` // the Jira key once created
	Case     string   `json:"case,omitempty"`  // the cases/ key once saved
}

type Design struct {
	Source string        `json:"source"`
	Model  string        `json:"model"`
	Tokens int           `json:"tokens"`
	Dir    string        `json:"dir"`
	Needs  []Need        `json:"needs,omitempty"` // what the tester must supply, so nobody has to guess it
	Drafts []DraftResult `json:"drafts"`
}

// Need is one input the requirement depends on: a value from .env, a dropdown's options, or a table's exact title.
type Need struct {
	Kind   string   `json:"kind"`   // "value" | "options" | "table"
	Name   string   `json:"name"`   // the .env name, the dropdown's name, or what the requirement calls the table
	Status string   `json:"status"` // "defined in .env" | "missing from .env" | "" (options and table)
	Hint   string   `json:"hint,omitempty"`
	Values []string `json:"values,omitempty"` // a dropdown's options on the page
}

var tableWord = regexp.MustCompile(`(?i)\b(tabla|table)\b\s*(?:de\s+\w+\s*)?(?:"([^"]+)")?`)

// computeNeeds: the inputs a requirement names, checked against .env and the crawl. Nothing is guessed.
func computeNeeds(requirement string, envKeys map[string]bool, cr *Crawl) []Need {
	var out []Need
	seen := map[string]bool{}
	for _, m := range bracketName.FindAllStringSubmatch(requirement, -1) {
		if seen[m[1]] {
			continue
		}
		seen[m[1]] = true
		st := "missing from .env"
		if envKeys[m[1]] {
			st = "defined in .env"
		}
		out = append(out, Need{Kind: "value", Name: m[1], Status: st, Hint: "set " + m[1] + " in the project's .env"})
	}
	low := strings.ToLower(requirement)
	for _, p := range pagesNamed(cr, low) { // the page's own controls the test must fill: dropdown options, text boxes
		for _, e := range p.Elements {
			switch {
			case e.Role == "combobox" && e.Name != "" && len(e.Options) > 0:
				out = append(out, Need{Kind: "options", Name: e.Name, Values: e.Options, Hint: "pick one of the page's options"})
			case e.Role == "textbox" && e.Name != "" && !strings.Contains(low, strings.ToLower(e.Name)):
				out = append(out, Need{Kind: "value", Name: e.Name, Status: "not given", Hint: "the requirement gives no value for \"" + e.Name + "\": name it as [NAME] and set NAME in .env"})
			}
		}
	}
	for _, e := range crawlOptions(cr) {
		if strings.Contains(low, strings.ToLower(e.Name)) && !seenOptions(out, e.Name) {
			out = append(out, Need{Kind: "options", Name: e.Name, Values: e.Options, Hint: "pick one of the page's options"})
		}
	}
	if m := tableWord.FindStringSubmatch(requirement); m != nil {
		out = append(out, Need{Kind: "table", Name: strings.TrimSpace(m[2]), Hint: "give the table's exact title as the page shows it"})
	}
	return out
}

type crawlSelect struct {
	Name    string
	Options []string
}

// crawlOptions: the dropdowns on the crawled pages that have options.
func crawlOptions(cr *Crawl) []crawlSelect {
	var out []crawlSelect
	if cr == nil {
		return nil
	}
	for _, p := range cr.Pages {
		for _, e := range p.Elements {
			if e.Role == "combobox" && e.Name != "" && len(e.Options) > 0 {
				out = append(out, crawlSelect{Name: e.Name, Options: e.Options})
			}
		}
	}
	return out
}

const designRules = `You write manual test cases for a web app from a requirement.
Write ONLY what the requirement states. If it does not say what happens in a situation (empty fields, errors, limits),
write no case for that situation. Never contradict it.
Step 1 is always navigate, to the page the requirement is about: the precondition describes the state, the steps
create it. Each step is one action, picked from a fixed list, with its target and value:
- navigate: value = a path of the app (from the page list)
- enter: target = only the field's name (e.g. "cuenta" for the field "Ingresa la cuenta", never the instruction), value = the text to type; if the requirement names a value as ENV:NAME, the value is that ENV:NAME token
- select: target = the dropdown in plain English words (e.g. "store"), value = one of its options
- click / verify_enabled / verify_disabled: target = the button's exact visible name
Each expected result is one kind, picked from a fixed list:
- disabled_then_enabled: target = a button, between = what the requirement says happens in between (e.g. "the order is added"); never empty
- appears_in: target = the section heading where the processed order shows
- control_visible: target = a button, tab or text box as its exact visible name (the page shows it)
- empty_state_disappears / no_message: target = the exact visible text
- error_shown: an error toast or alert is shown (no target)
Every value must come from the requirement or from a dropdown's options; never invent data.
A value the test types or picks is written as [name] (e.g. [QA_CARD]), never the value itself: the name is a variable in the app's .env.
Never write a file path or a placeholder value. If a case would need a file the requirement does not name, write no case for it.
A precondition is optional: leave it empty unless the requirement asks for a starting state other than the app's default (the app's access note below). Never write the default yourself.
Each case has a short name, a precondition (the starting state), 2 to 8 steps and 1 to 3 expected results.`

// vocabulary: what the model may refer to, from the crawl (paths, fields, buttons, dropdown options).
func vocabulary(cr *Crawl) (string, map[string]bool) {
	var b strings.Builder
	known := map[string]bool{}
	for _, p := range cr.Pages {
		u, _ := url.Parse(p.URL)
		path := "/"
		if u != nil && u.Path != "" {
			path = u.Path
		}
		known[strings.ToLower(path)] = true
		fmt.Fprintf(&b, "Page %s:\n", path)
		for _, e := range p.Elements {
			if e.Name == "" || e.Role == "link" || e.Role == "menuitem" {
				continue
			}
			fmt.Fprintf(&b, "  %s %q", e.Role, e.Name)
			if len(e.Options) > 0 {
				fmt.Fprintf(&b, " options: %s", strings.Join(e.Options, ", "))
				for _, o := range e.Options {
					known[strings.ToLower(o)] = true
				}
			}
			b.WriteString("\n")
		}
	}
	return b.String(), known
}

const designExample = `{"cases":[{"name":"Tarjetas shows Agregar","precondition":"","steps":[{"action":"navigate","target":"","value":"/Tarjetas"},{"action":"verify_enabled","target":"Agregar","value":""}],"expected":[{"kind":"control_visible","target":"Agregar","between":""}]}]}`

// closeJSON: from the first "{" to the end, with the braces the model left open closed.
// shortcut: counts braces inside strings too, so a case text with a brace would be closed wrongly; the case checks still judge every draft.
func closeJSON(content string) string {
	if i := strings.Index(content, "{"); i >= 0 {
		content = content[i:]
	}
	content = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(content), "```"))
	if open := strings.Count(content, "{") - strings.Count(content, "}"); open > 0 {
		content += strings.Repeat("}", open)
	}
	return content
}

// accessNote: from the profile's login (set by init). The model decides per case whether it needs a signed-in user.
func (c Config) accessNote() string {
	if c.LoginFile == "" {
		return "no sign-in: the pages open without a user."
	}
	return "the app has a sign-in page at " + c.LoginPath + ". A page behind it needs a signed-in user, so a case about it starts signed in; a case about the sign-in page itself starts signed out."
}

// defaultPrecondition: the app's starting state, read from its profile (the same login that the board shows).
func (c Config) defaultPrecondition() string {
	if c.LoginFile == "" {
		return "No sign-in is needed."
	}
	return "The user is signed in."
}

// askDrafts is the model's one bounded job. Structured output keeps the answer to the shape; code checks the content.
func (c Config) askDrafts(requirement, access, vocab, existing, plan string, max int) ([]Draft, int, error) {
	rules := designRules
	if c.PlainDrafts {
		rules = plainRules
	}
	body, _ := json.Marshal(map[string]any{"model": c.Model, "temperature": 0, "max_tokens": 3000, "reasoning_effort": "none",
		"messages": []map[string]string{
			{"role": "system", "content": rules},
			{"role": "user", "content": fmt.Sprintf("Requirement:\n%s\n\nThe app's access: %s\n\nThe app (from its crawl):\n%s\n%s\n%sWrite up to %d cases. Answer ONLY with a JSON object shaped like this example, no prose: %s", requirement, access, vocab, existing, planText(plan), max, designExample)}}})
	resp, err := (&http.Client{Timeout: 5 * time.Minute}).Post(c.LMURL+"/chat/completions", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, 0, fmt.Errorf("the model is not reachable at %s: %w", c.LMURL, err)
	}
	defer resp.Body.Close()
	var r struct {
		Choices []struct{ Message struct{ Content string } }
		Usage   struct {
			Total int `json:"total_tokens"`
		}
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil || len(r.Choices) == 0 {
		return nil, 0, fmt.Errorf("the model gave no answer (status %d)", resp.StatusCode)
	}
	var out struct{ Cases []modelCase }
	content := closeJSON(r.Choices[0].Message.Content)
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		return nil, r.Usage.Total, fmt.Errorf("the model's answer is not the agreed JSON (%v); its answer: %s", err, r.Choices[0].Message.Content)
	}
	if len(out.Cases) == 0 { // say what the model said, so an empty answer is not silent
		return nil, r.Usage.Total, fmt.Errorf("the model returned no cases; its answer: %.300s", r.Choices[0].Message.Content)
	}
	if len(out.Cases) > max {
		out.Cases = out.Cases[:max]
	}
	var drafts []Draft
	for _, mc := range out.Cases {
		drafts = append(drafts, mc.draft())
	}
	return drafts, r.Usage.Total, nil
}

// modelCase is what the model answers: closed choices. Code turns it into the sentences the renderer maps, so the
// model's wording can never drift from the grammar; what it chose is still judged like any other case.
type modelCase struct {
	Name, Precondition string
	Steps              []struct{ Action, Target, Value string }
	Expected           []struct{ Kind, Target, Between string }
}

func (m modelCase) draft() Draft {
	d := Draft{Name: caseKeyPrefix.ReplaceAllString(m.Name, ""), Precondition: m.Precondition}
	q := func(s string) string { return strings.Trim(strings.TrimSpace(s), "\"'`") }
	for _, s := range m.Steps {
		t, v := q(s.Target), q(s.Value)
		if s.Action == "navigate" && v == "" { // the path given as the target instead of the value
			v = t
		}
		line := map[string]string{
			"navigate":        "Navigate to `" + v + "`.",
			"enter":           "Enter " + fieldName(t) + " `" + v + "`.",
			"select":          "Select " + t + " `" + v + "`.",
			"click":           `Click "` + t + `".`,
			"verify_enabled":  `Verify "` + t + `" is enabled.`,
			"verify_disabled": `Verify "` + t + `" is disabled.`,
		}[s.Action]
		if line == "" { // outside the closed list (a model ignoring the schema): kept visible, and the gate rejects it
			line = strings.TrimSpace(s.Action + " " + t + " " + v)
		}
		d.Steps = append(d.Steps, line)
	}
	for _, e := range m.Expected {
		t := q(e.Target)
		line := map[string]string{
			"disabled_then_enabled":  `"` + t + `" is disabled before ` + strings.TrimSpace(e.Between) + ` and enabled after.`,
			"appears_in":             `After processing, the order appears in "` + t + `".`,
			"control_visible":        `The page shows "` + t + `": it appears in "` + t + `".`,
			"empty_state_disappears": `The "` + t + `" empty state disappears.`,
			"no_message":             `No "` + t + `" message appears.`,
			"error_shown":            "An error toast or alert is shown to the user.",
		}[e.Kind]
		if line == "" {
			line = strings.TrimSpace(e.Kind + " " + t)
		}
		d.Expected = append(d.Expected, line)
	}
	return d
}

// draftHTML is the Jira description in the shape people write (and intake reads); values go in <code>.
func draftHTML(d Draft) string {
	inline := func(s string) string {
		parts := strings.Split(s, "`")
		for i := range parts {
			parts[i] = html.EscapeString(parts[i])
			if i%2 == 1 {
				parts[i] = "<code>" + parts[i] + "</code>"
			}
		}
		return strings.Join(parts, "")
	}
	var b strings.Builder
	b.WriteString("<ul><li><p><strong>Preconditions:</strong> " + inline(d.Precondition) + "</p></li><li><p><strong>Steps:</strong></p><ol>")
	for _, s := range d.Steps {
		b.WriteString("<li><p>" + inline(s) + "</p></li>")
	}
	b.WriteString("</ol></li><li><p><strong>Expected Result:</strong></p><ul>")
	for _, e := range d.Expected {
		b.WriteString("<li><p>" + inline(e) + "</p></li>")
	}
	b.WriteString("</ul></li></ul>")
	return b.String()
}

var backticked = regexp.MustCompile("`([^`]+)`")

// Design drafts cases and judges each one; nothing leaves the machine.
func (c Config) Design(requirement, source string, max int) (Design, error) {
	d := Design{Source: source, Model: c.Model}
	if c.Model == "" || c.LMURL == "" {
		return d, fmt.Errorf("design needs a model: set model in the profile or pass --model (e.g. prism-ml/bonsai-27b)")
	}
	raw, err := os.ReadFile(c.crawlPath())
	if err != nil {
		return d, fmt.Errorf("no crawl of %s yet: run hydra crawl --project %s first (the model may only use real elements)", c.App, c.Project)
	}
	cr := &Crawl{}
	if err := json.Unmarshal(raw, cr); err != nil {
		return d, fmt.Errorf("%s: %w", c.crawlPath(), err)
	}
	vocab, known := vocabulary(cr)
	envKeys := readEnvKeys(filepath.Join(c.Home, ".env"))
	names, roles := elementNames(cr), elementRoles(cr)
	start := pageNamed(requirement, known)
	existing := ""
	if ix, err := c.BuildIndex(); err == nil { // cases already written for this page or these names: do not repeat them
		if rel := ix.Related(requirement, 5); len(rel) > 0 {
			var b strings.Builder
			b.WriteString("Existing cases about this (do not repeat them):\n")
			for _, r := range rel {
				fmt.Fprintf(&b, "- %s: %s\n", r.Key, r.Title)
			}
			existing = b.String()
		}
	}
	observed := observedStates(c.Out, start)
	d.Needs = computeNeeds(requirement, readEnvKeys(filepath.Join(c.Home, ".env")), cr)
	if c.NoStates {
		observed = nil
	}
	existing += observed.prompt()
	plan, ptokens := "", 0
	if !c.PlainDrafts {
		if plan, ptokens, err = c.askPlan(requirement, c.accessNote(), vocab, existing, max); err != nil {
			d.Tokens = ptokens
			return d, err
		}
	}
	d.Tokens = ptokens
	drafts, tokens, err := c.askDrafts(requirement, c.accessNote(), vocab, existing, plan, max)
	d.Tokens += tokens
	if err != nil {
		return d, err
	}
	for i := range drafts { // the states the exploration observed, for the actions this draft takes
		drafts[i].Expected = append(drafts[i].Expected, observed.expectationsFor(drafts[i].Steps, roles)...)
	}
	for i := range drafts { // the app's default starting state, unless the case asked for another
		if drafts[i].Precondition == "" {
			drafts[i].Precondition = c.defaultPrecondition()
		}
	}
	stored, _ := c.loadCases() // siblings give the renderer its default values, as in the router
	tmp, err := os.MkdirTemp("", "hydra-design-")
	if err != nil {
		return d, err
	}
	defer os.RemoveAll(tmp)
	low := strings.ToLower(requirement)
	seen := map[string]string{} // steps + expected -> the first draft that had them
	for i, dr := range drafts {
		for j, st := range dr.Steps { // "Navigate to Tarjetas" is the page /Tarjetas when the app has that page
			if m := navigateName.FindStringSubmatch(st); m != nil && !strings.HasPrefix(m[1], "/") && known["/"+strings.ToLower(m[1])] {
				dr.Steps[j] = "Navigate to `/" + m[1] + "`."
			}
		}
		if start != "" && (len(dr.Steps) == 0 || !strings.HasPrefix(dr.Steps[0], "Navigate to ")) {
			dr.Steps = append([]string{"Navigate to `" + start + "`."}, dr.Steps...) // where the requirement says it happens
		}
		res := DraftResult{ID: fmt.Sprintf("DRAFT-%d", i+1), Draft: dr}
		cs, err := ToCase(draftHTML(dr), res.ID, dr.Name, "en", "", "")
		if err != nil {
			res.Problems = append(res.Problems, "unreadable as a case: "+err.Error())
			d.Drafts = append(d.Drafts, res)
			continue
		}
		bad, _ := Maturity(cs)
		if len(cs.Steps) > 0 && !strings.HasPrefix(cs.Steps[0].Action, "Navigate to ") {
			bad = append(bad, "does not start by opening a page (step 1 must be navigate): the test would have nowhere to begin")
		}
		key := strings.Join(dr.Steps, "\n") + "\n=>" + strings.Join(dr.Expected, "\n")
		if first, dup := seen[key]; dup {
			bad = append(bad, "same steps and expected results as "+first)
		} else {
			seen[key] = res.ID
		}
		res.Problems = append(res.Problems, bad...)
		for _, st := range dr.Steps {
			res.Problems = append(res.Problems, unsourced(st, envKeys)...)
			res.Problems = append(res.Problems, unmatched(st, names)...)
			res.Problems = append(res.Problems, fieldClicks(st, roles)...)
		}
		for _, ex := range dr.Expected {
			res.Problems = append(res.Problems, unrequested(ex, low)...)
			res.Problems = append(res.Problems, observed.unobserved(ex)...)
			if emptyBefore.MatchString(ex) {
				res.Problems = append(res.Problems, fmt.Sprintf("%q: nothing says what happens in between: write the step that changes the state", ex))
			}
		}
		for _, s := range cs.Steps {
			for _, m := range backticked.FindAllStringSubmatch(s.Action, -1) {
				v := strings.ToLower(m[1])
				// the requirement is untrusted text too: these two never pass because it mentions them
				if strings.HasPrefix(v, "env:") && !isEnvPointer(m[1]) {
					res.Problems = append(res.Problems, fmt.Sprintf("step %s uses %q: a draft may only point at a variable (ENV:NAME), never write the secret", s.N, m[1]))
					continue
				}
				if strings.HasPrefix(s.Action, "Navigate to ") && !known[v] {
					res.Problems = append(res.Problems, fmt.Sprintf("step %s opens %q, which is not a page of the app (crawl)", s.N, m[1]))
					continue
				}
				if !strings.Contains(low, v) && !known[v] {
					res.Problems = append(res.Problems, fmt.Sprintf("step %s uses %q, which is neither in the requirement nor in the app", s.N, m[1]))
				}
			}
		}
		if len(bad) == 0 { // dry render in a throwaway folder: what poll would build from this text
			r := c
			r.Out = filepath.Join(tmp, res.ID)
			pending, spent, _, err := r.render(cs, append(append([]*Case{}, stored...), cs))
			d.Tokens += spent
			switch {
			case err != nil:
				res.Problems = append(res.Problems, "does not render: "+err.Error())
			case len(pending) > 0:
				res.Problems = append(res.Problems, "steps the rules cannot map: "+strings.Join(pending, "; "))
			default:
				test, _ := os.ReadFile(filepath.Join(r.bundleDir(res.ID), "tests", c.App, "test_"+token(res.ID)+".py"))
				if n := strings.Count(string(test), "PENDIENTE: resultado esperado"); n > 0 {
					res.Problems = append(res.Problems, fmt.Sprintf("%d expected result(s) would not be asserted", n))
				}
			}
		}
		res.OK = len(res.Problems) == 0
		d.Drafts = append(d.Drafts, res)
	}
	d.Dir = filepath.Join(c.Out, "design", time.Now().Format("20060102-150405"))
	for _, r := range d.Drafts {
		os.MkdirAll(d.Dir, 0o755)
		os.WriteFile(filepath.Join(d.Dir, r.ID+".html"), []byte(draftHTML(r.Draft)), 0o644)
	}
	return d, writeJSON(filepath.Join(d.Dir, "design.json"), d)
}

// pageNamed: the one app path the requirement mentions ("" when it names none or several: then the model must say).
func pageNamed(requirement string, known map[string]bool) string {
	low, found := strings.ToLower(requirement), ""
	for k := range known {
		if strings.HasPrefix(k, "/") && len(k) > 1 && strings.Contains(low, k) {
			if found != "" {
				return ""
			}
			found = k
		}
	}
	return found
}

// LoadDesign reads a saved run ("last" = the newest), so what is created in Jira is exactly what was reviewed.
func (c Config) LoadDesign(dir string) (Design, error) {
	var d Design
	if dir == "last" {
		runs, _ := filepath.Glob(filepath.Join(c.Out, "design", "*", "design.json"))
		if len(runs) == 0 {
			return d, fmt.Errorf("no saved design run yet: run hydra design --from ... first")
		}
		sort.Strings(runs) // folder names are timestamps
		dir = filepath.Dir(runs[len(runs)-1])
	}
	b, err := os.ReadFile(filepath.Join(expand(dir), "design.json"))
	if err != nil {
		return d, err
	}
	if err := json.Unmarshal(b, &d); err != nil {
		return d, err
	}
	return d, nil
}

// requirementText: a file's text, or a Jira story's description without its markup.
func requirementText(from, story string) (string, string, error) {
	switch {
	case from != "" && story != "":
		return "", "", fmt.Errorf("give --from or --story, not both")
	case from != "":
		b, err := os.ReadFile(expand(from))
		return strings.TrimSpace(string(b)), from, err
	case story != "":
		j, err := JiraFromEnv()
		if err != nil {
			return "", "", err
		}
		is, err := j.Get(story)
		if err != nil {
			return "", "", err
		}
		text := regexp.MustCompile(`<[^>]+>`).ReplaceAllString(strings.NewReplacer("</p>", "\n", "<br>", "\n", "</li>", "\n").Replace(is.HTML), "")
		return strings.TrimSpace(is.Summary + "\n" + html.UnescapeString(text)), story, nil
	}
	return "", "", fmt.Errorf("give the requirement: --from file.txt or --story KEY")
}

func printDesign(d Design) {
	for _, n := range d.Needs {
		switch n.Kind {
		case "value":
			fmt.Printf("NEEDS  [%s] %s: %s\n", n.Name, n.Status, n.Hint)
		case "options":
			fmt.Printf("NEEDS  %s options: %s\n", n.Name, strings.Join(n.Values, ", "))
		default:
			fmt.Printf("NEEDS  table title: %q (%s)\n", n.Name, n.Hint)
		}
	}
	for _, r := range d.Drafts {
		mark := "ok      "
		if !r.OK {
			mark = "rejected"
		}
		fmt.Printf("%-8s %s  %s  (%d steps, %d expected)\n", r.ID, mark, r.Draft.Name, len(r.Draft.Steps), len(r.Draft.Expected))
		for _, p := range r.Problems {
			fmt.Println("           - " + p)
		}
		if r.Issue != "" {
			fmt.Println("           → created " + r.Issue + " (label hydra-draft; add automate to approve)")
		}
	}
	fmt.Printf("\n%d tokens · drafts saved in %s\n", d.Tokens, d.Dir)
}

// ---- Jira: create the approved-for-review drafts ----

// adfInline: text with `values` as code marks, the same thing draftHTML writes as <code>.
func adfInline(s string) []any {
	var out []any
	for i, part := range strings.Split(s, "`") {
		if part == "" {
			continue
		}
		n := map[string]any{"type": "text", "text": part}
		if i%2 == 1 {
			n["marks"] = []any{map[string]any{"type": "code"}}
		}
		out = append(out, n)
	}
	return out
}

func draftADF(d Draft) map[string]any {
	para := func(content ...any) map[string]any { return map[string]any{"type": "paragraph", "content": content} }
	strong := func(s string) map[string]any {
		return map[string]any{"type": "text", "text": s, "marks": []any{map[string]any{"type": "strong"}}}
	}
	items := func(list string, lines []string) map[string]any {
		var li []any
		for _, l := range lines {
			li = append(li, map[string]any{"type": "listItem", "content": []any{para(adfInline(l)...)}})
		}
		return map[string]any{"type": list, "content": li}
	}
	pre := append([]any{strong("Preconditions:"), map[string]any{"type": "text", "text": " "}}, adfInline(d.Precondition)...)
	return map[string]any{"type": "doc", "version": 1, "content": []any{map[string]any{"type": "bulletList", "content": []any{
		map[string]any{"type": "listItem", "content": []any{para(pre...)}},
		map[string]any{"type": "listItem", "content": []any{para(strong("Steps:")), items("orderedList", d.Steps)}},
		map[string]any{"type": "listItem", "content": []any{para(strong("Expected Result:")), items("bulletList", d.Expected)}},
	}}}}
}

// CreateDraft creates one draft under parent: label hydra-draft only, so no poll picks it up before a person does.
func (j *Jira) CreateDraft(project, parent, issueType string, d Draft) (string, error) {
	fields := map[string]any{"project": map[string]string{"key": project}, "issuetype": map[string]string{"name": issueType},
		"summary": d.Name, "labels": []string{"hydra-draft"}, "description": draftADF(d)}
	if parent != "" {
		fields["parent"] = map[string]string{"key": parent}
	}
	var r struct{ Key string }
	err := j.do("POST", "/rest/api/3/issue", url.Values{}, map[string]any{"fields": fields}, &r)
	return r.Key, err
}

// createDrafts sends the drafts that passed, in order; the report records each key.
func (c Config) createDrafts(d *Design, parent, issueType string) error {
	j, err := JiraFromEnv()
	if err != nil {
		return err
	}
	project := c.JiraProject
	if project == "" {
		project = c.Project
	}
	for i := range d.Drafts {
		if !d.Drafts[i].OK || d.Drafts[i].Issue != "" { // already created from this run: never twice
			continue
		}
		key, err := j.CreateDraft(project, parent, issueType, d.Drafts[i].Draft)
		if err != nil {
			return fmt.Errorf("%s: %w", d.Drafts[i].ID, err)
		}
		d.Drafts[i].Issue = key
	}
	return writeJSON(filepath.Join(d.Dir, "design.json"), d)
}

// fieldName: the field's name without an instruction the model copied in ("Ingresa la cuenta" -> "cuenta").
var instruction = regexp.MustCompile(`(?i)^(ingresa|escribe|escriba|ingrese|enter|type|fill|write)\s+(en\s+|la\s+|el\s+|the\s+|a\s+|the\s+)?`)

func fieldName(t string) string {
	if n := strings.TrimSpace(instruction.ReplaceAllString(t, "")); n != "" {
		return n
	}
	return t
}

// isEnvPointer: ENV:NAME, the name of a variable in the project's .env. The value itself never goes into a draft.
var envPointer = regexp.MustCompile(`^ENV:[A-Z][A-Z0-9_]*$`)

func isEnvPointer(v string) bool { return envPointer.MatchString(v) }

// [name] in a step is a value the test reads: it must be a variable in the app's .env (the value never goes into a draft).
var bracketName = regexp.MustCompile(`\[([A-Za-z][A-Za-z0-9_]*)\]`)

// unsourced: one problem for each [name] in the step that .env does not define.
func unsourced(step string, envKeys map[string]bool) []string {
	var out []string
	for _, m := range bracketName.FindAllStringSubmatch(step, -1) {
		if !envKeys[m[1]] {
			out = append(out, fmt.Sprintf("%q: [%s] has no source: add %s to the project's .env", step, m[1], m[1]))
		}
	}
	return out
}

// readEnvKeys: the variable names in a .env file (values are never read here).
func readEnvKeys(path string) map[string]bool {
	keys := map[string]bool{}
	b, err := os.ReadFile(path)
	if err != nil {
		return keys
	}
	for _, line := range strings.Split(string(b), "\n") {
		if k, _, ok := strings.Cut(strings.TrimSpace(line), "="); ok && !strings.HasPrefix(k, "#") {
			keys[strings.TrimSpace(k)] = true
		}
	}
	return keys
}

// elementNames: the names of the app's fields, buttons and dropdowns in the crawl (links are navigation, not targets).
func elementNames(cr *Crawl) []string {
	var out []string
	for _, p := range cr.Pages {
		for _, e := range p.Elements {
			if e.Name != "" && e.Role != "link" && e.Role != "menuitem" {
				out = append(out, e.Name)
			}
		}
	}
	return out
}

var quotedName = regexp.MustCompile(`"([^"]+)"`)
var fieldTarget = regexp.MustCompile("^(?:Enter|Select) (.+?) `")
var accents = strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ñ", "n", "Á", "a", "É", "e", "Í", "i", "Ó", "o", "Ú", "u", "Ñ", "n")

func plainKey(s string) string { return strings.ToLower(accents.Replace(strings.TrimSpace(s))) }

// unmatched: each field or button a step names must be on the app's page (a name in the crawl, ignoring case and accents).
func unmatched(step string, names []string) []string {
	var targets []string
	if m := fieldTarget.FindStringSubmatch(step); m != nil {
		targets = append(targets, m[1])
	}
	for _, m := range quotedName.FindAllStringSubmatch(step, -1) {
		targets = append(targets, m[1])
	}
	var out []string
	for _, t := range targets {
		found := false
		for _, n := range names {
			if strings.Contains(plainKey(n), plainKey(t)) {
				found = true
				break
			}
		}
		if !found {
			out = append(out, fmt.Sprintf("%q: %q is not on the page (crawl)", step, t))
		}
	}
	return out
}

// planRules: the first call reasons in plain words; the second call turns the plan into the agreed JSON, and code checks it.
const planRules = `You plan manual test cases for a web app from a requirement. Reason it through in plain words, not JSON.
For each case, in a numbered list: its name; its starting state (only what the requirement or the app's access note says);
each action in order, named with the exact names from the app's page list; each typed value written as [NAME]
(a variable in the app's .env), never the value itself; each dropdown option is written as the option's own name from the page list; and the expected result. Do not invent names, values or pages.
Every case comes from one sentence of the requirement: quote that sentence first, then the case. Write no case for behavior the
requirement does not state (no reset, no second item, no error, unless the requirement says it). When the requirement says a
Never write an error, message or result the requirement does not state. Never click a text box or dropdown: enter or select it. When the requirement says a
control is enabled or disabled, the expected result states that change (disabled before, enabled after), never only that the
control exists. If the requirement needs something the page list does not show, write one line starting with "Missing:".`

// askPlan: free text, no JSON. The model may reason here; its plan is a hint for askDrafts, which still must pass the checks.
func (c Config) askPlan(requirement, access, vocab, existing string, max int) (string, int, error) {
	body, _ := json.Marshal(map[string]any{"model": c.Model, "temperature": 0, "max_tokens": 6000, "reasoning_effort": "medium",
		"messages": []map[string]string{
			{"role": "system", "content": planRules},
			{"role": "user", "content": fmt.Sprintf("Requirement:\n%s\n\nThe app's access: %s\n\nThe app (from its crawl):\n%s\n%s\nPlan up to %d cases.", requirement, access, vocab, existing, max)}}})
	resp, err := (&http.Client{Timeout: 5 * time.Minute}).Post(c.LMURL+"/chat/completions", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", 0, fmt.Errorf("the model is not reachable at %s: %w", c.LMURL, err)
	}
	defer resp.Body.Close()
	var r struct {
		Choices []struct{ Message struct{ Content string } }
		Usage   struct {
			Total int `json:"total_tokens"`
		}
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil || len(r.Choices) == 0 {
		return "", 0, fmt.Errorf("the model gave no plan (status %d)", resp.StatusCode)
	}
	plan := strings.TrimSpace(r.Choices[0].Message.Content)
	if plan == "" {
		return "", r.Usage.Total, fmt.Errorf("the model gave an empty plan")
	}
	return plan, r.Usage.Total, nil
}

// planText: the plan as input to the drafting call ("" when there is none).
func planText(plan string) string {
	if plan == "" {
		return ""
	}
	return "The plan written from the requirement (use it, and only what the requirement and the page list support):\n" + plan + "\n\n"
}

// elementRoles: each named element's role, by its plain name (a Click on a text box or dropdown is a mistake).
func elementRoles(cr *Crawl) map[string]string {
	out := map[string]string{}
	for _, p := range cr.Pages {
		for _, e := range p.Elements {
			if e.Name != "" {
				out[plainKey(e.Name)] = e.Role
			}
		}
	}
	return out
}

// fieldClicks: "Click <field>" where the field is a text box or dropdown on the page: it must be entered or selected.
func fieldClicks(step string, roles map[string]string) []string {
	if !strings.HasPrefix(step, "Click ") {
		return nil
	}
	var out []string
	for _, m := range quotedName.FindAllStringSubmatch(step, -1) {
		if r := roles[plainKey(m[1])]; r == "textbox" || r == "combobox" {
			out = append(out, fmt.Sprintf("%q: %q is a %s: enter or select it, do not click it", step, m[1], r))
		}
	}
	return out
}

// plainRules: the comparison arm of design --plain. The model gets only the shape and the page list; the checks still run.
const plainRules = `You write manual test cases for a web app from a requirement. Answer with JSON only.`

// caseKeyPrefix: a case key the model copied into a name ("PQA-3 Enter account") is not part of the name; the save step gives keys.
var caseKeyPrefix = regexp.MustCompile(`^[A-Z][A-Z0-9]*-\d+[:\s]+`)

// errorExpected: an expected error the requirement never mentions is invented, so it is refused.
var errorExpected = regexp.MustCompile(`(?i)\berror\b|alert|toast`)
var errorWords = regexp.MustCompile(`(?i)error|alerta|mensaje|toast|falla|rechaz`)

// unrequested: an expected error that the requirement does not describe.
func unrequested(expected, lowRequirement string) []string {
	if errorExpected.MatchString(expected) && !errorWords.MatchString(lowRequirement) {
		return []string{fmt.Sprintf("%q: the requirement does not describe an error, so this expected result is invented", expected)}
	}
	return nil
}

// observedChange is one state change the exploration saw on a page (out/explore/*.json, from explore.py).
type observedChange struct {
	AfterAction string `json:"after_action"`
	Control     string `json:"control"`
	Field       string `json:"field"`
	From        bool   `json:"from"`
	To          bool   `json:"to"`
}

type observedPage []observedChange

// observedStates: every exploration result for this page path, from the project's out/explore folder.
func observedStates(out, path string) observedPage {
	if path == "" {
		return nil
	}
	var all observedPage
	files, _ := filepath.Glob(filepath.Join(out, "explore", "*.json"))
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var r struct {
			Path       string           `json:"path"`
			Candidates []observedChange `json:"candidates"`
		}
		if json.Unmarshal(b, &r) == nil && strings.EqualFold(r.Path, path) { // pageNamed gives the lowercase path
			all = append(all, r.Candidates...)
		}
	}
	return all
}

// prompt: the observed changes as the only states the drafting may use for expected results.
func (o observedPage) prompt() string {
	if len(o) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("States the app was observed to show on this page (use only these for enabled, disabled or visible expected results):\n")
	for _, c := range o {
		fmt.Fprintf(&b, "- after %s: %q %s %v -> %v\n", c.AfterAction, c.Control, c.Field, c.From, c.To)
	}
	return b.String()
}

var stateBeforeAfter = regexp.MustCompile(`^"(.+?)" is (enabled|disabled) before .+? and (enabled|disabled) after\.?$`)

// unobserved: an expected "X is enabled|disabled before ... and ... after" must match a change the exploration saw.
// When nothing was observed for the page, there is nothing to check against, so nothing is refused.
func (o observedPage) unobserved(expected string) []string {
	m := stateBeforeAfter.FindStringSubmatch(expected)
	if m == nil || len(o) == 0 {
		return nil
	}
	want := m[3] == "enabled"
	for _, c := range o {
		if c.Control == m[1] && c.Field == "enabled" && c.To == want {
			return nil
		}
	}
	return []string{fmt.Sprintf("%q: the exploration never saw %q become %s", expected, m[1], m[3])}
}

// expectationsFor: an observed enable change that this draft's own steps cause becomes a state expectation, written in the
// form the renderer asserts. Code writes these; the model never does.
func (o observedPage) expectationsFor(steps []string, roles map[string]string) []string {
	var out []string
	joined := strings.ToLower(strings.Join(steps, "\n"))
	for _, c := range o {
		if c.Field != "enabled" || c.From == c.To || roles[plainKey(c.Control)] != "button" {
			continue // the renderer checks the state of buttons only
		}
		key := actionKey(c.AfterAction)
		if key == "" || !strings.Contains(joined, strings.ToLower(key)) {
			continue
		}
		out = append(out, fmt.Sprintf(`"%s" is %s before %s and %s after.`, c.Control, stateWord(c.From), c.AfterAction, stateWord(c.To)))
	}
	return out
}

func stateWord(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}

// actionKey: the value a step picks ("DESBLOQUEO" from select "..." = DESBLOQUEO), or the button a step clicks ("Agregar").
var clickedName = regexp.MustCompile(`^click "(.+?)"$`)

func actionKey(action string) string {
	if i := strings.LastIndex(action, "="); i >= 0 {
		return strings.TrimSpace(action[i+1:])
	}
	if m := clickedName.FindStringSubmatch(action); m != nil {
		return m[1]
	}
	return ""
}

var navigateName = regexp.MustCompile("^Navigate to `?([^`.]+?)`?\\.$")

// emptyBefore: a state expectation with nothing between "before" and "and" (the between text is required).
var emptyBefore = regexp.MustCompile(`is (enabled|disabled) before\s+(and|after)\b`)

// pagesNamed: the crawled pages whose path the requirement mentions.
func pagesNamed(cr *Crawl, lowRequirement string) []struct {
	URL      string
	Elements []CrawlElement
} {
	var out []struct {
		URL      string
		Elements []CrawlElement
	}
	if cr == nil {
		return nil
	}
	seen := map[string]bool{} // the crawl can list one page twice
	for _, p := range cr.Pages {
		u, _ := url.Parse(p.URL)
		if u != nil && u.Path != "/" && !seen[u.Path] && strings.Contains(lowRequirement, strings.ToLower(u.Path)) {
			seen[u.Path] = true
			out = append(out, p)
		}
	}
	return out
}

func seenOptions(needs []Need, name string) bool {
	for _, n := range needs {
		if n.Kind == "options" && n.Name == name {
			return true
		}
	}
	return false
}
