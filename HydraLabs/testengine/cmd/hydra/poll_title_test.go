package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

// The poll log names each case, not only its ID: a person reading the terminal should know which case a line is about.
func TestReportShowsTheCaseTitleUnderTheID(t *testing.T) {
	r, w, _ := os.Pipe()
	stdout := os.Stdout
	os.Stdout = w
	report(Run{Key: "DWQ-132", Title: "Subir un archivo con el número de orden", Intent: "execute", Tier: "T0", Status: "passed"})
	os.Stdout = stdout
	w.Close()
	out, _ := io.ReadAll(r)
	if !strings.Contains(string(out), "DWQ-132") || !strings.Contains(string(out), "           Subir un archivo con el número de orden") {
		t.Fatalf("title missing from the log line:\n%s", out)
	}
}
