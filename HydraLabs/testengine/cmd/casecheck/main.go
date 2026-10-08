// casecheck enforces schema/case.schema.json (the normalized-case contract) on stdin.
// Valid input is echoed to stdout unchanged, so it can sit in a pipe; problems go
// to stderr with exit 1. Hand-rolled instead of a schema library: the contract is small.
// Extra fields (warnings, depends_on, ...) are allowed; a misspelled REQUIRED field shows up
// as empty/missing, so typos still fail.
//
//	./join ... | ./casecheck | next-stage
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type Step struct {
	Action string `json:"action"`
}

type Case struct {
	ID     string            `json:"id"`
	Name   string            `json:"name"`
	Status string            `json:"status"`
	Meta   map[string]string `json:"meta"`
	Steps  []Step            `json:"steps"`
	Data   *struct {
		Rows []map[string]string `json:"rows"`
	} `json:"data"`
}

func check(in []byte) []string {
	var doc struct{ Cases []Case }
	dec := json.NewDecoder(bytes.NewReader(in))
	if err := dec.Decode(&struct {
		Cases   *[]Case  `json:"cases"`
		Orphans []string `json:"orphans"`
	}{Cases: &doc.Cases}); err != nil {
		return []string{"not a valid normalized document: " + err.Error()}
	}

	var errs []string
	if doc.Cases == nil {
		errs = append(errs, `missing "cases"`)
	}
	seen := map[string]bool{}
	valid := map[string]bool{"ready": true, "blocked": true, "no-data": true, "depends": true}
	for i, c := range doc.Cases {
		at := fmt.Sprintf("cases[%d] (%q)", i, c.ID)
		switch {
		case c.ID == "":
			errs = append(errs, at+": empty id")
		case c.Status == "ready" && seen[c.ID]: // only cases that write files can collide
			errs = append(errs, at+": duplicate id")
		}
		if c.Status == "ready" {
			seen[c.ID] = true
		}
		if c.Name == "" {
			errs = append(errs, at+": empty name")
		}
		if !valid[c.Status] {
			errs = append(errs, fmt.Sprintf("%s: status %q is not ready|blocked|no-data|depends", at, c.Status))
		}
		for j, s := range c.Steps {
			if s.Action == "" {
				errs = append(errs, fmt.Sprintf("%s: step %d has no action", at, j+1))
			}
		}
		if c.Status == "ready" {
			if len(c.Steps) == 0 {
				errs = append(errs, at+": ready case has no steps")
			}
			if c.Data == nil || len(c.Data.Rows) == 0 {
				errs = append(errs, at+": ready case has no data rows")
			}
		}
	}
	return errs
}

func main() {
	in, _ := io.ReadAll(os.Stdin)
	if errs := check(in); len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintln(os.Stderr, "casecheck:", e)
		}
		os.Exit(1)
	}
	os.Stdout.Write(in)
}
