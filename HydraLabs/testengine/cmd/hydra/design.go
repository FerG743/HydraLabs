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
}

type Design struct {
	Source string        `json:"source"`
	Model  string        `json:"model"`
	Tokens int           `json:"tokens"`
	Dir    string        `json:"dir"`
	Drafts []DraftResult `json:"drafts"`
}

const designRules = `You write manual test cases for a web app from a requirement.
Write ONLY what the requirement states. If it does not say what happens in a situation (empty fields, errors, limits),
write no case for that situation. Never contradict it.
Step 1 is always navigate, to the page the requirement is about: the precondition describes the state, the steps
create it. Each step is one action, picked from a fixed list, with its target and value:
- navigate: value = a path of the app (from the page list)
- enter: target = the text field in plain English words (e.g. "order number"), value = the text to type
- select: target = the dropdown in plain English words (e.g. "store"), value = one of its options
- click / verify_enabled / verify_disabled: target = the button's exact visible name
Each expected result is one kind, picked from a fixed list:
- disabled_then_enabled: target = a button, between = what happens in between (e.g. "the order is added")
- appears_in: target = the section heading where the processed order shows
- empty_state_disappears / no_message: target = the exact visible text
- error_shown: an error toast or alert is shown (no target)
Every value must come from the requirement or from a dropdown's options; never invent data.
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

// askDrafts is the model's one bounded job. Structured output keeps the answer to the shape; code checks the content.
func (c Config) askDrafts(requirement, vocab string, max int) ([]Draft, int, error) {
	str := map[string]any{"type": "string"}
	step := map[string]any{"type": "object", "required": []string{"action", "target", "value"}, "properties": map[string]any{
		"action": map[string]any{"type": "string", "enum": []string{"navigate", "enter", "select", "click", "verify_enabled", "verify_disabled"}},
		"target": str, "value": str}}
	exp := map[string]any{"type": "object", "required": []string{"kind", "target", "between"}, "properties": map[string]any{
		"kind":   map[string]any{"type": "string", "enum": []string{"disabled_then_enabled", "appears_in", "empty_state_disappears", "no_message", "error_shown"}},
		"target": str, "between": str}}
	schema := map[string]any{"type": "object", "required": []string{"cases"}, "properties": map[string]any{
		"cases": map[string]any{"type": "array", "maxItems": max, "items": map[string]any{"type": "object",
			"required": []string{"name", "precondition", "steps", "expected"}, "properties": map[string]any{
				"name": str, "precondition": str,
				"steps":    map[string]any{"type": "array", "items": step},
				"expected": map[string]any{"type": "array", "items": exp}}}}}}
	body, _ := json.Marshal(map[string]any{"model": c.Model, "temperature": 0, "max_tokens": 3000, "reasoning_effort": "none",
		"response_format": map[string]any{"type": "json_schema", "json_schema": map[string]any{"name": "cases", "schema": schema}},
		"messages": []map[string]string{
			{"role": "system", "content": designRules},
			{"role": "user", "content": fmt.Sprintf("Requirement:\n%s\n\nThe app (from its crawl):\n%s\nWrite up to %d cases.", requirement, vocab, max)}}})
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
	content := r.Choices[0].Message.Content
	if i, j := strings.Index(content, "{"), strings.LastIndex(content, "}"); i >= 0 && j > i {
		content = content[i : j+1]
	}
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		return nil, r.Usage.Total, fmt.Errorf("the model's answer is not the agreed JSON: %v", err)
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
	d := Draft{Name: m.Name, Precondition: m.Precondition}
	q := func(s string) string { return strings.Trim(strings.TrimSpace(s), "\"'`") }
	for _, s := range m.Steps {
		t, v := q(s.Target), q(s.Value)
		if s.Action == "navigate" && v == "" { // the path given as the target instead of the value
			v = t
		}
		line := map[string]string{
			"navigate":        "Navigate to `" + v + "`.",
			"enter":           "Enter " + t + " `" + v + "`.",
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
	start := pageNamed(requirement, known)
	drafts, tokens, err := c.askDrafts(requirement, vocab, max)
	d.Tokens = tokens
	if err != nil {
		return d, err
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
		for _, s := range cs.Steps {
			for _, m := range backticked.FindAllStringSubmatch(s.Action, -1) {
				v := strings.ToLower(m[1])
				// the requirement is untrusted text too: these two never pass because it mentions them
				if strings.HasPrefix(v, "env:") {
					res.Problems = append(res.Problems, fmt.Sprintf("step %s uses %q: secrets are never test data in a draft", s.N, m[1]))
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
