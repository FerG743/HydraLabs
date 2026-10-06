package main

import (
	"fmt"
	"regexp"
	"strings"
)

// The maturity gate. Cases are written by a human first; the pipeline only starts on cases that are complete
// enough to automate without guessing. Anything else is sent back as specific questions, before any tokens are spent.

var placeholder = regexp.MustCompile(`(?i)\b(tbd|todo|pendiente|por definir|to be defined|\?\?\?|xxx)\b`)

// Maturity returns what blocks automation (each item phrased as a question for the author) and non-blocking advice.
func Maturity(c *Case) (blocking, advice []string) {
	if len(c.Steps) == 0 {
		blocking = append(blocking, "Which are the steps? The description has no Steps section (or it is empty).")
	}
	if len(c.Expected) == 0 {
		blocking = append(blocking, "What is the expected result? Without it the test would have nothing to assert.")
	}
	for _, s := range c.Steps {
		if len(strings.TrimSpace(s.Action)) < 4 {
			blocking = append(blocking, fmt.Sprintf("Step %s is empty or too short to act on.", s.N))
		} else if placeholder.MatchString(s.Action) {
			blocking = append(blocking, fmt.Sprintf("Step %s still has a placeholder: %q", s.N, s.Action))
		}
	}
	for _, e := range c.Expected {
		if placeholder.MatchString(e.Text) {
			blocking = append(blocking, fmt.Sprintf("An expected result still has a placeholder: %q", e.Text))
		}
	}
	for _, w := range c.Warnings {
		switch {
		case strings.HasPrefix(w, refWarning):
			blocking = append(blocking, "Precondition cannot be resolved: "+strings.TrimPrefix(w, refWarning+" "))
		case strings.HasPrefix(w, "unknown section"), strings.HasPrefix(w, "unlabeled item"):
			advice = append(advice, w+" (ignored)")
		}
	}
	if strings.TrimSpace(c.Precondition) == "" {
		advice = append(advice, "No precondition: state the starting state (logged in? data present? table empty?).")
	}
	return
}
