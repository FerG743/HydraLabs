package main

import (
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// Jira issue description (rendered HTML) -> normalized case. Deterministic, no model.
// Port of tools/jira/jira2case.py; testdata/*.case.json are that script's output.

var sections = map[string]string{ // label (lowercase) -> field
	"preconditions": "precondition", "precondition": "precondition", "precondiciones": "precondition",
	"precondición": "precondition", "precondicion": "precondition",
	"steps": "steps", "pasos": "steps",
	"expected result": "expected", "expected results": "expected",
	"resultado esperado": "expected", "resultados esperados": "expected",
	"pipeline gate": "gate", "gate": "gate",
	"note": "notes", "notes": "notes", "nota": "notes", "notas": "notes",
	"type": "_meta", "tipo": "_meta",
}

type Step struct {
	N      string `json:"n"`
	Action string `json:"action"`
}
type Expected struct {
	Text string `json:"text"`
}
type Case struct {
	ID                 string            `json:"id"`
	Name               string            `json:"name"`
	Language           string            `json:"language"`
	Meta               map[string]any    `json:"meta"`
	Precondition       string            `json:"precondition"`
	PreconditionRef    string            `json:"precondition_ref,omitempty"`
	PreconditionOrig   string            `json:"precondition_original,omitempty"`
	PreconditionSource string            `json:"precondition_resolved_from,omitempty"`
	Steps              []Step            `json:"steps"`
	Expected           []Expected        `json:"expected"`
	Notes              string            `json:"notes,omitempty"`
	Extra              map[string]string `json:"extra,omitempty"`
	Warnings           []string          `json:"warnings"`
}

var (
	ws      = regexp.MustCompile(`\s+`)
	spacePt = regexp.MustCompile(`\s([.,;:!?])(\s|$)`)
	label   = regexp.MustCompile(`^([^:|]{1,40}):\s*(.*)$`)
	sameAs  = regexp.MustCompile(`(?i)^(same as|igual que|mismas? de)\s+([\w-]+)`)
)

func clean(s string) string {
	s = ws.ReplaceAllString(s, " ")
	for { // Jira's link markup leaves "TC-1 ."; a loop because a match eats the trailing space
		n := spacePt.ReplaceAllString(s, "$1$2")
		if n == s {
			break
		}
		s = n
	}
	return strings.TrimSpace(s)
}

func isEl(n *html.Node, tags ...string) bool {
	if n.Type != html.ElementNode {
		return false
	}
	for _, t := range tags {
		if n.Data == t {
			return true
		}
	}
	return false
}

func attr(n *html.Node, k string) string {
	for _, a := range n.Attr {
		if a.Key == k {
			return a.Val
		}
	}
	return ""
}

// text is the inline text of a node, skipping nested lists; <code> keeps its backticks (value anchors).
func text(n *html.Node) string {
	switch {
	case n.Type == html.TextNode:
		return n.Data
	case isEl(n, "br"):
		return " "
	case isEl(n, "ul", "ol"):
		return ""
	case n.Type == html.ElementNode && strings.Contains(attr(n, "class"), "lozenge"): // status label Jira appends to issue keys
		return ""
	}
	var b strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		b.WriteString(text(c))
	}
	if isEl(n, "code", "tt") {
		return "`" + b.String() + "`"
	}
	return b.String()
}

func subItems(li *html.Node) []*html.Node {
	var out []*html.Node
	for c := li.FirstChild; c != nil; c = c.NextSibling {
		if isEl(c, "ul", "ol") {
			for k := c.FirstChild; k != nil; k = k.NextSibling {
				if isEl(k, "li") {
					out = append(out, k)
				}
			}
		}
	}
	return out
}

func item(li *html.Node) string {
	base := clean(text(li))
	var subs []string
	for _, k := range subItems(li) {
		subs = append(subs, item(k))
	}
	if len(subs) == 0 {
		return base
	}
	return strings.TrimSpace(base + " " + strings.Join(subs, "; "))
}

// topItems yields every top-level <li> of every outermost <ul>.
func topItems(n *html.Node, f func(*html.Node)) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if isEl(c, "ul") {
			for k := c.FirstChild; k != nil; k = k.NextSibling {
				if isEl(k, "li") {
					f(k)
				}
			}
		} else {
			topItems(c, f)
		}
	}
}

func ToCase(htmlSrc, key, summary, lang, parent, jiraPriority string) (*Case, error) {
	root, err := html.Parse(strings.NewReader(htmlSrc))
	if err != nil {
		return nil, err
	}
	c := &Case{ID: key, Name: summary, Language: lang, Meta: map[string]any{}, Steps: []Step{}, Expected: []Expected{}, Warnings: []string{}}
	if parent != "" {
		c.Meta["parent"] = parent
	}
	warn := func(f string, a ...any) { c.Warnings = append(c.Warnings, fmt.Sprintf(f, a...)) }

	topItems(root, func(li *html.Node) {
		raw := clean(text(li))
		m := label.FindStringSubmatch(raw)
		if m == nil {
			warn("unlabeled item ignored: %.60s", raw)
			return
		}
		lb, value := strings.TrimSpace(m[1]), strings.TrimSpace(m[2])
		field := sections[strings.ToLower(lb)]
		var items []string
		for _, k := range subItems(li) {
			items = append(items, item(k))
		}
		if len(items) == 0 && value != "" {
			items = []string{value}
		}
		switch field {
		case "_meta": // "Type: Test Case | Priority: High | Labels: `e2e`, `smoke`"
			for _, part := range strings.Split(raw, "|") {
				k, v, _ := strings.Cut(part, ":")
				k, v = strings.ToLower(strings.TrimSpace(k)), strings.ReplaceAll(strings.TrimSpace(v), "`", "")
				switch k {
				case "type", "tipo":
					c.Meta["type"] = v
				case "priority", "prioridad":
					c.Meta["priority"] = v
				case "labels", "etiquetas":
					labels := []string{}
					for _, x := range strings.Split(v, ",") {
						if x = strings.TrimSpace(x); x != "" {
							labels = append(labels, x)
						}
					}
					c.Meta["labels"] = labels
				}
			}
		case "steps":
			for i, t := range items {
				c.Steps = append(c.Steps, Step{N: fmt.Sprint(i + 1), Action: t})
			}
		case "expected":
			for _, t := range items {
				c.Expected = append(c.Expected, Expected{Text: t})
			}
		case "precondition":
			var rest []string
			for _, k := range subItems(li) {
				rest = append(rest, item(k))
			}
			c.Precondition = clean(value + " " + strings.Join(rest, " "))
			if r := sameAs.FindStringSubmatch(c.Precondition); r != nil {
				c.PreconditionRef = strings.TrimRight(r[2], ".")
				warn("precondition refers to %s: resolve it from that case", c.PreconditionRef)
			}
		case "gate":
			c.Meta["gate"] = value
		case "notes":
			c.Notes = value
		default:
			if c.Extra == nil {
				c.Extra = map[string]string{}
			}
			c.Extra[lb] = value
			warn("unknown section kept in 'extra': %s", lb)
		}
	})
	if len(c.Steps) == 0 {
		warn("no steps found")
	}
	if len(c.Expected) == 0 {
		warn("no expected results found")
	}
	if p, _ := c.Meta["priority"].(string); jiraPriority != "" && p != "" && p != jiraPriority {
		warn("priority conflict: description says %s, Jira field says %s", p, jiraPriority)
	}
	return c, nil
}
