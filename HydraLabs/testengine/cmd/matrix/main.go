// matrix turns any "one case, many step rows" test matrix CSV into JSON, ready
// for the LM to rewrite as Gherkin.
//
// Contract (project-agnostic; column names come from flags, nothing is hardcoded):
//
//   - row 1 is the header. A new case starts on every row whose -name cell is
//     non-blank; following rows with a blank -name are its steps (the layout
//     you get when merged cells are exported to CSV).
//
//   - -step / -expected are the action and expected-result columns; -num is the
//     optional step-number column.
//
//   - every other column is case-level metadata (first non-blank value wins),
//     keyed by slugified header. Columns listed in -stepcols are per-step instead.
//
//   - header matching is by slug prefix, so "PASOS* [10]" matches -step PASOS.
//
//   - -idre extracts a numeric id from the case name (default: digits with optional .N) so
//     the case can be joined to a data sheet (see cmd/suite).
//
//     go run ./cmd/matrix -name "NOMBRE DE CASO" -step PASOS -expected "RESULTADOS ESPERADOS" matrix.csv
package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type Step struct {
	N        string            `json:"n,omitempty"`
	Action   string            `json:"action"`
	Expected string            `json:"expected,omitempty"`
	Extra    map[string]string `json:"extra,omitempty"`
}

type Case struct {
	Name     string            `json:"name"`
	ID       string            `json:"id,omitempty"` // from -idre, leading zeros stripped, for joining
	Meta     map[string]string `json:"meta,omitempty"`
	Steps    []Step            `json:"steps"`
	Warnings []string          `json:"warnings,omitempty"`
}

type Config struct {
	Name, Step, Expected, Num string
	StepCols                  []string
	IDRe                      *regexp.Regexp
}

var accents = strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n")

func slug(s string) string {
	s = accents.Replace(strings.ToLower(strings.TrimSpace(s)))
	var b strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		} else if b.Len() > 0 && !strings.HasSuffix(b.String(), "_") {
			b.WriteByte('_')
		}
	}
	return strings.Trim(b.String(), "_")
}

// find returns the index of the first header whose slug starts with want's slug.
func find(header []string, want string) int {
	if want == "" {
		return -1
	}
	w := slug(want)
	for i, h := range header {
		if strings.HasPrefix(h, w) {
			return i
		}
	}
	return -2
}

func parse(recs [][]string, cfg Config) ([]Case, error) {
	header := make([]string, len(recs[0]))
	for i, h := range recs[0] {
		header[i] = slug(h)
	}
	name, step, exp, num := find(header, cfg.Name), find(header, cfg.Step), find(header, cfg.Expected), find(header, cfg.Num)
	for flagName, idx := range map[string]int{"name": name, "step": step, "expected": exp} {
		if idx < 0 {
			return nil, fmt.Errorf("-%s column not found in header %v", flagName, header)
		}
	}
	if num == -2 {
		return nil, fmt.Errorf("-num column not found in header %v", header)
	}
	stepCol := map[int]bool{}
	for _, s := range cfg.StepCols {
		if i := find(header, s); i >= 0 {
			stepCol[i] = true
		} else {
			return nil, fmt.Errorf("-stepcols %q not found in header %v", s, header)
		}
	}

	var cases []Case
	for _, rec := range recs[1:] {
		for len(rec) < len(header) {
			rec = append(rec, "")
		}
		cell := func(i int) string { return strings.TrimSpace(rec[i]) }

		if n := cell(name); n != "" {
			c := Case{Name: n, Meta: map[string]string{}, Steps: []Step{}}
			if m := cfg.IDRe.FindStringSubmatch(n); m != nil {
				c.ID = strings.TrimLeft(m[len(m)-1], "0")
			}
			cases = append(cases, c)
		}
		if len(cases) == 0 {
			continue // junk above the first case
		}
		c := &cases[len(cases)-1]

		for i, h := range header {
			if i == name || i == step || i == exp || i == num || h == "" || stepCol[i] {
				continue
			}
			if _, seen := c.Meta[h]; !seen && cell(i) != "" {
				c.Meta[h] = cell(i)
			}
		}
		if cell(step) == "" && cell(exp) == "" {
			continue
		}
		s := Step{Action: cell(step), Expected: cell(exp)}
		if num >= 0 {
			s.N = cell(num)
		}
		for i := range stepCol {
			if v := cell(i); v != "" {
				if s.Extra == nil {
					s.Extra = map[string]string{}
				}
				s.Extra[header[i]] = v
			}
		}
		if s.Action == "" {
			c.Warnings = append(c.Warnings, fmt.Sprintf("step %q has an expected result but no action", s.N))
		}
		c.Steps = append(c.Steps, s)
	}
	return cases, nil
}

func main() {
	var cfg Config
	var stepCols, idre string
	flag.StringVar(&cfg.Name, "name", "", "case-name column (required)")
	flag.StringVar(&cfg.Step, "step", "", "step/action column (required)")
	flag.StringVar(&cfg.Expected, "expected", "", "expected-result column (required)")
	flag.StringVar(&cfg.Num, "num", "", "step-number column (optional)")
	flag.StringVar(&stepCols, "stepcols", "", "comma-separated per-step extra columns (optional)")
	flag.StringVar(&idre, "idre", `(\d+(?:\.\d+)?)`, "regexp whose last group is the case id (e.g. 0037.1 -> 37.1)")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: matrix -name COL -step COL -expected COL [-num COL] [-stepcols A,B] file.csv")
		os.Exit(2)
	}
	cfg.IDRe = regexp.MustCompile(idre)
	if stepCols != "" {
		cfg.StepCols = strings.Split(stepCols, ",")
	}

	f, err := os.Open(flag.Arg(0))
	if err != nil {
		panic(err)
	}
	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	recs, err := r.ReadAll()
	if err != nil || len(recs) < 2 {
		panic(fmt.Sprint("bad csv: ", err))
	}
	cases, err := parse(recs, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(cases)
}
