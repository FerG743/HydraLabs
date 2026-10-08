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
