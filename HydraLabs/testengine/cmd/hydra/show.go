package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// splitCaseKey: the first argument that looks like a case key, and the other arguments (the flags).
func splitCaseKey(args []string) (string, []string) {
	key, rest := "", []string{}
	for _, a := range args {
		if k := strings.ToUpper(a); key == "" && !strings.HasPrefix(a, "-") && caseKey.MatchString(k) {
			key = k
		} else {
			rest = append(rest, a)
		}
	}
	return key, rest
}

// caseText: the case's Markdown from cases/, then its latest recorded run (if any).
func (c Config) caseText(key string) (string, error) {
	b, err := os.ReadFile(filepath.Join(c.localCaseDir(), key+".md"))
	if err != nil {
		return "", fmt.Errorf("no case %s in %s", key, c.localCaseDir())
	}
	out := string(b)
	if es := c.executions(key); len(es) > 0 {
		last := es[len(es)-1]
		out += fmt.Sprintf("\nLast run: %s, %s (%s)\n", last.At, last.Status, last.Trigger)
	}
	return out, nil
}
