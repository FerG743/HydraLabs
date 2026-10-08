// join merges cmd/matrix output (steps) with cmd/suite output (data rows) by
// case id, producing one self-contained input per case for the Gherkin step.
//
// Status per case:
//
//	blocked  - no steps, or any step matches -blocked (e.g. "Dependencia de insumos")
//	no-data  - has steps but the data sheet has no usable row for it
//	depends  - the data sheet says "use another case's inputs" (free text)
//	ready    - steps and data rows present
//
// Ids are compared with leading zeros stripped ("01" == "1"). Suite ids with no
// matching matrix case are reported as orphans, never dropped silently.
//
//	go run ./cmd/join [-blocked REGEX] matrix.json suite.json > cases.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type Matrix struct {
	ID       string           `json:"id"`
	Name     string           `json:"name"`
	Meta     map[string]any   `json:"meta,omitempty"`
	Steps    []map[string]any `json:"steps"`
	Warnings []string         `json:"warnings,omitempty"`
}

type Suite struct {
	ID       string              `json:"id"`
	Status   string              `json:"status"`
	Note     string              `json:"note,omitempty"`
	Shared   map[string]string   `json:"shared,omitempty"`
	Rows     []map[string]string `json:"rows,omitempty"`
	Lists    map[string][]string `json:"lists,omitempty"`
	Warnings []string            `json:"warnings,omitempty"`
}

type Data struct {
	Shared map[string]string   `json:"shared,omitempty"`
	Rows   []map[string]string `json:"rows"`
	Lists  map[string][]string `json:"lists,omitempty"`
}

type Case struct {
	ID       string           `json:"id"`
	Name     string           `json:"name"`
	Status   string           `json:"status"`
	Note     string           `json:"note,omitempty"`
	Meta     map[string]any   `json:"meta,omitempty"`
	Steps    []map[string]any `json:"steps"`
	Data     *Data            `json:"data,omitempty"`
	Warnings []string         `json:"warnings,omitempty"`
}

type Out struct {
	Cases   []Case   `json:"cases"`
	Orphans []string `json:"orphans,omitempty"` // suite ids with no matrix case
}

func norm(id string) string { return strings.TrimLeft(strings.TrimSpace(id), "0") }

func load(path string, v any) {
	b, err := os.ReadFile(path)
	if err == nil {
		err = json.Unmarshal(b, v)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, path+":", err)
		os.Exit(1)
	}
}

func join(matrix []Matrix, suite []Suite, blocked *regexp.Regexp) Out {
	byID := map[string]Suite{}
	for _, s := range suite {
		byID[norm(s.ID)] = s
	}

	var out Out
	seen := map[string]bool{}
	for _, m := range matrix {
		c := Case{ID: m.ID, Name: m.Name, Meta: m.Meta, Steps: m.Steps, Warnings: m.Warnings}
		id := norm(m.ID)
		if seen[id] {
			c.Warnings = append(c.Warnings, "duplicate case id "+m.ID)
		}
		seen[id] = true
		s, hasSuite := byID[id]

		switch {
		case len(m.Steps) == 0 || (blocked != nil && matchesAny(m.Steps, blocked)):
			c.Status = "blocked"
		case !hasSuite || s.Status == "empty" || (s.Status == "ready" && len(s.Rows) == 0):
			c.Status = "no-data"
		case s.Status == "free-text":
			c.Status, c.Note = "depends", s.Note
		default:
			c.Status = "ready"
			c.Data = &Data{Shared: s.Shared, Rows: s.Rows, Lists: s.Lists}
			c.Warnings = append(c.Warnings, s.Warnings...)
		}
		out.Cases = append(out.Cases, c)
	}
	for _, s := range suite {
		if !seen[norm(s.ID)] && s.Status != "empty" {
			out.Orphans = append(out.Orphans, s.ID)
		}
	}
	return out
}

func matchesAny(steps []map[string]any, re *regexp.Regexp) bool {
	for _, st := range steps {
		for _, v := range st {
			if s, ok := v.(string); ok && re.MatchString(s) {
				return true
			}
		}
	}
	return false
}

func main() {
	pat := flag.String("blocked", "", "regexp; a case with any step matching it is blocked")
	flag.Parse()
	if flag.NArg() != 2 {
		fmt.Fprintln(os.Stderr, "usage: join [-blocked REGEX] matrix.json suite.json")
		os.Exit(2)
	}
	var re *regexp.Regexp
	if *pat != "" {
		re = regexp.MustCompile(*pat)
	}
	var m []Matrix
	var s []Suite
	load(flag.Arg(0), &m)
	load(flag.Arg(1), &s)

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(join(m, s, re))
}
