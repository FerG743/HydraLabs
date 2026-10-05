// skeleton turns one joined case (cmd/join output, one case on stdin) into a
// Gherkin feature, deterministically. The structure is always valid; the only
// fuzzy part - which words in a step are data - is filled in two layers:
//
//  1. automatic: "Label: value" where the label matches a data field
//     ("Tienda: 0103-..." with field tienda). Row fields become <placeholders>,
//     shared fields are replaced by the data sheet's value.
//  2. optional -map / -map-b64: JSON {"<step n>": [{"field":"rfc","literal":"text in the step"}]}
//     (what the LM returns), for whatever the automatic layer missed.
//
// Without a map, -plan prints what is still unmapped, as compact JSON for the LM.
//
//	skeleton -plan < case.json          # steps + fields still to map
//	skeleton -map map.json < case.json  # {"feature":"...","warnings":[...]}
package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
)

type Step struct{ N, Action, Expected string }

type Case struct {
	ID    string
	Name  string
	Meta  map[string]string
	Steps []Step
	Data  *struct {
		Shared map[string]string
		Rows   []map[string]string
		Lists  map[string][]string
	}
}

type Entry struct{ Field, Literal string }

type Opts struct {
	Map  map[string][]Entry
	Tags []string // meta keys to emit as @tags
}

type Result struct {
	Feature  string   `json:"feature"`
	Warnings []string `json:"warnings"`
}

var (
	accents  = strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n")
	label    = regexp.MustCompile(`^(.*?)([\p{L}][\p{L} ]*?)\s*:\s*(\S.*)$`)
	cred     = regexp.MustCompile(`(?i)^(usuario|contrase[ñn]a|password|user)\s*:\s*.*$`)
	numbered = regexp.MustCompile(`^\s*\d+\s*[.\-)]+\s*`)
	nonWord  = regexp.MustCompile(`[^a-z0-9]+`)
	// "TR (Boleta)": a parenthetical names what a short label means
	alias = regexp.MustCompile(`([\p{L}]{1,6})\s*\(\s*([\p{L} ]+?)\s*\)`)
)

func words(s string) string { // "Correo electrónico " -> "correo electronico"
	return strings.TrimSpace(nonWord.ReplaceAllString(accents.Replace(strings.ToLower(s)), " "))
}

func fieldsOf(c *Case) (rows, shared map[string]bool) {
	rows, shared = map[string]bool{}, map[string]bool{}
	if c.Data == nil {
		return
	}
	for k := range c.Data.Shared {
		shared[k] = true
	}
	for _, r := range c.Data.Rows {
		for k := range r {
			rows[k] = true
		}
	}
	return
}

func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }

// prep splits a step's action into Gherkin lines: credentials replaced, then
// "Label: value" mapped automatically. A numbered list becomes one line per item.
func prep(c *Case, st Step, rows, shared map[string]bool, used map[string]bool) []string {
	var items []string // logical lines (numbered items stay together with their continuations)
	nums := 0
	for _, l := range strings.Split(st.Action, "\n") {
		if numbered.MatchString(l) {
			nums++
		}
	}
	for _, l := range strings.Split(st.Action, "\n") {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		switch {
		case nums >= 2 && numbered.MatchString(l):
			items = append(items, numbered.ReplaceAllString(l, ""))
		case nums >= 2 && len(items) > 0:
			items[len(items)-1] += " " + l
		default:
			items = append(items, l)
		}
	}
	if nums < 2 { // single action: its lines are one sentence
		items = []string{strings.Join(items, "\n")}
	}

	aliases := map[string]string{}
	for _, m := range alias.FindAllStringSubmatch(st.Action, -1) {
		aliases[words(m[1])] = words(m[2])
	}
	var out []string
	for _, it := range items {
		lines := strings.Split(it, "\n")
		for i, l := range lines {
			if m := cred.FindStringSubmatch(strings.TrimSpace(l)); m != nil {
				lines[i] = m[1] + ": <" + strings.ToLower(m[1]) + ">"
				continue
			}
			lines[i] = autoMap(l, c, rows, shared, used, aliases)
		}
		out = append(out, collapse(strings.Join(lines, " ")))
	}
	return out
}

// autoMap rewrites "…Label: value" when Label ends with a known field's words.
func autoMap(l string, c *Case, rows, shared map[string]bool, used map[string]bool, aliases map[string]string) string {
	m := label.FindStringSubmatch(l)
	if m == nil {
		return l
	}
	lab := words(m[2])
	if a, ok := aliases[lab]; ok {
		lab = a
	}
	best := ""
	for _, set := range []map[string]bool{rows, shared} {
		for f := range set {
			fw := strings.ReplaceAll(f, "_", " ")
			if (lab == fw || strings.HasSuffix(lab, " "+fw)) && len(f) > len(best) {
				best = f
			}
		}
	}
	if best == "" {
		return l
	}
	val := strings.TrimSpace(m[3])
	if rows[best] {
		used[best] = true
		return l[:len(l)-len(m[3])] + "<" + best + ">"
	}
	used[best] = true
	return l[:len(l)-len(val)] + c.Data.Shared[best]
}

func build(c *Case, o Opts) (Result, []Plan, map[string]bool) {
	rows, shared := fieldsOf(c)
	used := map[string]bool{}
	var warn []string
	var plan []Plan

	type gstep struct{ action, expected string }
	var steps []gstep
	for _, st := range c.Steps {
		lines := prep(c, st, rows, shared, used)
		for i, l := range lines {
			for _, e := range o.Map[st.N] {
				if !strings.Contains(l, e.Literal) {
					continue
				}
				switch {
				case rows[e.Field]:
					l, used[e.Field] = strings.Replace(l, e.Literal, "<"+e.Field+">", 1), true
				case shared[e.Field]:
					l, used[e.Field] = strings.Replace(l, e.Literal, c.Data.Shared[e.Field], 1), true
				default:
					warn = append(warn, fmt.Sprintf("step %s: unknown field %q ignored", st.N, e.Field))
				}
				o.Map[st.N] = removeEntry(o.Map[st.N], e)
				break
			}
			lines[i] = l
		}
		for _, e := range o.Map[st.N] {
			warn = append(warn, fmt.Sprintf("step %s: literal %q not found, ignored", st.N, e.Literal))
		}
		if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
			warn = append(warn, fmt.Sprintf("step %s has no action: skipped", st.N))
			continue
		}
		for i, l := range lines {
			g := gstep{action: l}
			if i == len(lines)-1 {
				g.expected = collapse(st.Expected)
			}
			steps = append(steps, g)
		}
		plan = append(plan, Plan{N: st.N, Text: strings.Join(lines, " / ")})
	}

	var cols []string
	for f := range used {
		if rows[f] { // shared fields are written literally, never as columns
			cols = append(cols, f)
		}
	}
	sort.Strings(cols)
	for f := range rows {
		if !used[f] {
			warn = append(warn, fmt.Sprintf("data column %q is not used in any step: dropped", f))
		}
	}
	for f := range c.Data.Lists {
		warn = append(warn, fmt.Sprintf("data %q has values that don't align with rows: not used", f))
	}

	var b strings.Builder
	b.WriteString("# language: es\n")
	if tags := tagsOf(c, o.Tags); tags != "" {
		b.WriteString(tags + "\n")
	}
	title := c.Meta["descripcion"]
	if title == "" {
		title = c.Name
	}
	fmt.Fprintf(&b, "Característica: %s\n\n", collapse(title))
	if p := collapse(c.Meta["precondicion"]); p != "" {
		fmt.Fprintf(&b, "  Antecedentes:\n    Dado %s\n\n", p)
	}
	kind := "Escenario"
	if len(cols) > 0 {
		kind = "Esquema del escenario"
	}
	fmt.Fprintf(&b, "  %s: %s\n", kind, collapse(c.Name))
	for _, s := range steps {
		fmt.Fprintf(&b, "    Cuando %s\n", s.action)
		if s.expected != "" {
			fmt.Fprintf(&b, "    Entonces %s\n", s.expected)
		}
	}
	if len(cols) > 0 {
		b.WriteString("\n    Ejemplos:\n      | " + strings.Join(cols, " | ") + " |\n")
		for _, r := range c.Data.Rows {
			cells := make([]string, len(cols))
			for i, f := range cols {
				cells[i] = strings.ReplaceAll(r[f], "|", `\|`)
				if cells[i] == "" {
					warn = append(warn, fmt.Sprintf("row has an empty %q cell", f))
				}
			}
			b.WriteString("      | " + strings.Join(cells, " | ") + " |\n")
		}
	}
	if warn == nil {
		warn = []string{}
	}
	return Result{Feature: b.String(), Warnings: warn}, plan, used
}

func removeEntry(es []Entry, e Entry) []Entry {
	for i := range es {
		if es[i] == e {
			return append(es[:i:i], es[i+1:]...)
		}
	}
	return es
}

func tagsOf(c *Case, keys []string) string {
	var t []string
	if i := strings.IndexByte(strings.TrimPrefix(c.Name, "_"), '_'); i > 0 {
		if j := strings.IndexByte(c.Name[i+1:], '_'); j > 0 {
			t = append(t, "@"+c.Name[:i+1+j])
		}
	}
	for _, k := range keys {
		if v := collapse(c.Meta[k]); v != "" {
			t = append(t, "@"+strings.ReplaceAll(v, " ", "_"))
		}
	}
	return strings.Join(t, " ")
}

type Plan struct {
	N    string `json:"n"`
	Text string `json:"text"` // step as it stands after the automatic pass
}

type PlanOut struct {
	Steps  []Plan   `json:"steps"`
	Fields []string `json:"fields"` // data fields not yet used by any step
}

func main() {
	plan := flag.Bool("plan", false, "print the unmapped steps and fields as JSON for the LM")
	mapFile := flag.String("map", "", "JSON file with the LM's mapping")
	mapB64 := flag.String("map-b64", "", "same, base64-encoded (for shells that mangle quotes)")
	tags := flag.String("tags", "prioridad,ciclo,nivel_de_prueba,parent", "meta keys to emit as @tags")
	flag.Parse()

	var c Case
	in, _ := io.ReadAll(os.Stdin)
	if err := json.Unmarshal(in, &c); err != nil {
		fmt.Fprintln(os.Stderr, "stdin is not a joined case:", err)
		os.Exit(1)
	}
	if c.Data == nil {
		fmt.Fprintln(os.Stderr, "case has no data (status must be ready)")
		os.Exit(1)
	}

	o := Opts{Map: map[string][]Entry{}, Tags: strings.Split(*tags, ",")}
	var raw []byte
	switch {
	case *mapB64 != "":
		raw, _ = base64.StdEncoding.DecodeString(*mapB64)
	case *mapFile != "":
		raw, _ = os.ReadFile(*mapFile)
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &o.Map); err != nil {
			fmt.Fprintln(os.Stderr, "bad map:", err)
			os.Exit(1)
		}
	}

	res, steps, used := build(&c, o)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if *plan {
		rows, shared := fieldsOf(&c)
		var left []string
		for _, set := range []map[string]bool{rows, shared} {
			for f := range set {
				if !used[f] {
					left = append(left, f)
				}
			}
		}
		sort.Strings(left)
		enc.Encode(PlanOut{Steps: steps, Fields: left})
		return
	}
	enc.Encode(res)
}
