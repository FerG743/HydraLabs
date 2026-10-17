package main

import (
	"fmt"
	"regexp"
	"strings"
)

// "Same as TC-1" preconditions, resolved across the cases already known. Port of tools/jira/resolve_refs.py:
// the referenced text is appended to the original, chains resolve, cycles/ambiguity stay unresolved (never guessed).

const refWarning = "precondition refers to"

func findTarget(ref string, cases []*Case, self string) (*Case, string) {
	var hits []*Case
	for _, c := range cases {
		if c.ID == ref {
			hits = append(hits, c)
		}
	}
	if len(hits) == 0 { // "TC-1" matches "TC-1 – ..." but never "TC-10"
		re := regexp.MustCompile(`^` + regexp.QuoteMeta(ref) + `($|[^\w-])`)
		for _, c := range cases {
			if re.MatchString(strings.TrimSpace(c.Name)) {
				hits = append(hits, c)
			}
		}
	}
	var ok []*Case
	for _, c := range hits {
		if c.ID != self {
			ok = append(ok, c)
		}
	}
	switch len(ok) {
	case 1:
		return ok[0], ""
	case 0:
		return nil, "not found"
	}
	ids := []string{}
	for _, c := range ok {
		ids = append(ids, c.ID)
	}
	return nil, "ambiguous: " + strings.Join(ids, ", ")
}

func original(c *Case) string {
	if c.PreconditionOrig != "" {
		return c.PreconditionOrig
	}
	return c.Precondition
}

func preconditionOf(c *Case, cases []*Case, seen []string) (text, source, err string) {
	if c.PreconditionRef == "" {
		return c.Precondition, "", ""
	}
	for _, s := range seen {
		if s == c.ID {
			return "", "", "cycle: " + strings.Join(append(seen, c.ID), " -> ")
		}
	}
	t, e := findTarget(c.PreconditionRef, cases, c.ID)
	if e != "" {
		return "", "", e
	}
	txt, _, e := preconditionOf(t, cases, append(append([]string{}, seen...), c.ID))
	if e != "" {
		return "", "", e
	}
	return fmt.Sprintf("%s: %s", strings.TrimRight(original(c), ". "), txt), t.ID, ""
}

// ResolveRefs updates cases in place.
func ResolveRefs(cases []*Case) {
	for _, c := range cases {
		if c.PreconditionRef == "" {
			continue
		}
		orig := original(c)
		c.PreconditionOrig = orig
		keep := []string{}
		for _, w := range c.Warnings {
			if !strings.HasPrefix(w, refWarning) {
				keep = append(keep, w)
			}
		}
		probe := *c
		probe.Precondition, probe.PreconditionOrig = orig, orig
		text, src, err := preconditionOf(&probe, cases, nil)
		if err != "" {
			c.Precondition, c.PreconditionSource = orig, ""
			c.Warnings = append(keep, fmt.Sprintf("%s %s: %s", refWarning, c.PreconditionRef, err))
		} else {
			c.Precondition, c.PreconditionSource, c.Warnings = text, src, keep
		}
	}
}

// storeReferenced fetches the siblings a stored case points at ("Same as TC-1") that are not stored yet: a fresh app
// repo has none, and the sibling may carry no automate label of its own. They are only stored, never run. A few
// rounds cover chains (TC-3 -> TC-2 -> TC-1).
func (c Config) storeReferenced(j *Jira) error {
	for round := 0; round < 3; round++ {
		cases, err := c.loadCases()
		if err != nil {
			return err
		}
		added := false
		for _, cs := range cases {
			parent, _ := cs.Meta["parent"].(string)
			if cs.PreconditionRef == "" || parent == "" {
				continue
			}
			if _, e := findTarget(cs.PreconditionRef, cases, cs.ID); e != "not found" {
				continue
			}
			name := regexp.MustCompile(`^` + regexp.QuoteMeta(cs.PreconditionRef) + `($|[^\w-])`)
			issues, err := j.Search(fmt.Sprintf(`parent = %s AND summary ~ "\"%s\""`, parent, cs.PreconditionRef))
			if err != nil {
				return fmt.Errorf("fetching %s for %s: %w", cs.PreconditionRef, cs.ID, err)
			}
			for _, is := range issues {
				if is.Key != cs.ID && name.MatchString(strings.TrimSpace(is.Summary)) {
					if _, err := c.Ingest(is); err != nil {
						return err
					}
					added = true
				}
			}
		}
		if !added {
			return nil
		}
	}
	return nil
}
