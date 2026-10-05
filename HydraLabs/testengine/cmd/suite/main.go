// suite turns any "test data per case" CSV into JSON ready for Gherkin Examples.
//
// Contract (project-agnostic):
//
//   - column 0 is the case id; row 1 is the header; other columns are fields,
//     keyed by their slugified header (usable as <placeholders>).
//
//   - -key names the column that defines one scenario per line (e.g. boleta, sku).
//
//   - multi-line cells are positional: line i of every aligned column belongs to row i.
//
//   - a column with a single value is shared by all rows of the case.
//
//   - a column that is multi-valued but not aligned with -key goes to "lists"
//     (surfaced, never guessed) and adds a warning.
//
//     go run ./cmd/suite -key boleta data.csv > suite.json
package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

type Case struct {
	ID       string              `json:"id"`
	Status   string              `json:"status"` // ready | empty | free-text
	Note     string              `json:"note,omitempty"`
	Shared   map[string]string   `json:"shared,omitempty"`
	Rows     []map[string]string `json:"rows,omitempty"`
	Lists    map[string][]string `json:"lists,omitempty"`
	Warnings []string            `json:"warnings,omitempty"`
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

// lines splits a cell on newlines, trims, strips Excel's leading apostrophe,
// and drops trailing blanks (but keeps inner ones: position matters).
func lines(cell string) []string {
	out := strings.Split(cell, "\n")
	for i, s := range out {
		out[i] = strings.TrimPrefix(strings.TrimSpace(s), "'")
	}
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return out
}

func nonBlank(ss []string) []string {
	var out []string
	for _, s := range ss {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func main() {
	key := flag.String("key", "", "column that defines one scenario per line (required)")
	flag.Parse()
	if *key == "" || flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: suite -key <column> file.csv")
		os.Exit(2)
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

	header := make([]string, len(recs[0]))
	keyIdx := -1
	for i, h := range recs[0] {
		header[i] = slug(h)
		if i > 0 && header[i] == slug(*key) {
			keyIdx = i
		}
	}
	if keyIdx < 0 {
		panic(fmt.Sprintf("key column %q not in header %v", *key, header))
	}

	var cases []Case
	for _, rec := range recs[1:] {
		for len(rec) < len(header) {
			rec = append(rec, "")
		}
		c := Case{ID: strings.TrimSpace(rec[0])}
		if c.ID == "" {
			continue
		}
		keyLines := lines(rec[keyIdx])
		if len(nonBlank(keyLines)) == 0 { // nothing to iterate on: empty or free text
			var text []string
			for _, cell := range rec[1:] {
				text = append(text, nonBlank(lines(cell))...)
			}
			c.Status, c.Note = "empty", ""
			if len(text) > 0 {
				c.Status, c.Note = "free-text", strings.Join(text, " | ")
			}
			cases = append(cases, c)
			continue
		}

		c.Status, c.Shared = "ready", map[string]string{}
		rows := make([]map[string]string, len(keyLines))
		for i := range rows {
			rows[i] = map[string]string{}
		}
		for i := 1; i < len(header); i++ {
			col, vals := header[i], lines(rec[i])
			nb := nonBlank(vals)
			switch {
			case i == keyIdx || len(vals) == len(keyLines):
				for j := range rows {
					rows[j][col] = vals[j]
					if vals[j] == "" {
						c.Warnings = append(c.Warnings, fmt.Sprintf("%s=%s (line %d): %s is blank", header[keyIdx], keyLines[j], j+1, col))
					}
				}
			case len(nb) == 1:
				c.Shared[col] = nb[0]
			case len(nb) > 1:
				if c.Lists == nil {
					c.Lists = map[string][]string{}
				}
				c.Lists[col] = nb
				c.Warnings = append(c.Warnings, fmt.Sprintf("%s: %d values don't align with %d %s rows; not paired", col, len(nb), len(keyLines), header[keyIdx]))
			}
		}
		for _, row := range rows {
			if row[header[keyIdx]] != "" { // skip blank-key lines
				c.Rows = append(c.Rows, row)
			}
		}
		cases = append(cases, c)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(cases)
}
