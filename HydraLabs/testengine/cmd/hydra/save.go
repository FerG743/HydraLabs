package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// saveDrafts writes the drafts that passed, once each, as cases/<PROJECT>-<n>.md (the format cases/README.md describes).
// Only a saved run is written, so a person reviewed it first; the run's record keeps each case's key.
func (c Config) saveDrafts(d *Design) ([]string, error) {
	dir := c.localCaseDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	next := c.nextCaseNumber(dir)
	var keys []string
	for i := range d.Drafts {
		r := &d.Drafts[i]
		if !r.OK || r.Case != "" {
			continue
		}
		key := fmt.Sprintf("%s-%d", c.Project, next)
		next++
		if err := os.WriteFile(filepath.Join(dir, key+".md"), []byte(caseMarkdown(r.Draft)), 0o644); err != nil {
			return keys, err
		}
		r.Case = key
		keys = append(keys, key)
	}
	return keys, nil
}

var caseFile = regexp.MustCompile(`^([A-Z][A-Z0-9]*)-(\d+)\.md$`)

// nextCaseNumber: one past the highest <PROJECT>-n in dir, so a new key never overwrites a case.
func (c Config) nextCaseNumber(dir string) int {
	next := 1
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		m := caseFile.FindStringSubmatch(e.Name())
		if m == nil || m[1] != c.Project {
			continue
		}
		if n, err := strconv.Atoi(m[2]); err == nil && n >= next {
			next = n + 1
		}
	}
	return next
}

// caseMarkdown: the draft in the layout cases/README.md shows (labels: automate, the sections, numbered steps).
func caseMarkdown(d Draft) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\nlabels: automate\n\n- Preconditions: %s\n- Steps:\n", d.Name, d.Precondition)
	for i, s := range d.Steps {
		fmt.Fprintf(&b, "  %d. %s\n", i+1, s)
	}
	b.WriteString("- Expected result:\n")
	for _, e := range d.Expected {
		fmt.Fprintf(&b, "  - %s\n", e)
	}
	return b.String()
}
