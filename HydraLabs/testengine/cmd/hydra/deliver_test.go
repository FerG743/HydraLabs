package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestJUnitIsMappedBackToJiraKeys(t *testing.T) {
	p := filepath.Join(t.TempDir(), "j.xml")
	os.WriteFile(p, []byte(`<?xml version="1.0"?><testsuites><testsuite name="pytest">
<testcase classname="tests.CobroOrdenes.test_dwq_130" name="test_dwq_130_manual"/>
<testcase classname="tests.CobroOrdenes.test_dwq_132" name="test_dwq_132_bad"><failure message="AssertionError: button"/></testcase>
<testcase classname="tests.CobroOrdenes.test_dwq_131" name="test_dwq_131_csv"><skipped/></testcase></testsuite></testsuites>`), 0o644)
	got, err := parseJUnit(p)
	if err != nil {
		t.Fatal(err)
	}
	if got["DWQ-130"].Status != "passed" || got["DWQ-132"].Status != "failed" || got["DWQ-131"].Status != "skipped" {
		t.Errorf("%+v", got)
	}
	if !strings.Contains(got["DWQ-132"].Detail, "AssertionError") {
		t.Errorf("failure reason lost: %+v", got["DWQ-132"])
	}
}

func TestOnlyPassingCasesAreDeliverable(t *testing.T) {
	c := Config{Out: t.TempDir()}
	for k, st := range map[string]string{"DWQ-130": "passed", "DWQ-131": "passed", "DWQ-132": "failed", "DWQ-133": "needs-review"} {
		writeJSON(filepath.Join(c.Out, "runs", k+".json"), Run{Key: k, Status: st})
	}
	got, _ := c.passedKeys(nil, map[string]bool{})
	if strings.Join(got, ",") != "DWQ-130,DWQ-131" {
		t.Errorf("%v", got)
	}
	got, _ = c.passedKeys(nil, map[string]bool{"DWQ-131": true})
	if strings.Join(got, ",") != "DWQ-130" {
		t.Errorf("exclude ignored: %v", got)
	}
}

func oc(status string) CaseOutcome { return CaseOutcome{Status: status, Detail: status + " detail"} }

func TestStabilityVerdicts(t *testing.T) {
	run := func(a, b string) map[string]CaseOutcome { return map[string]CaseOutcome{"A": oc(a), "B": oc(b)} }
	keys := []string{"A", "B"}
	by := func(os []CaseOutcome) map[string]CaseOutcome {
		m := map[string]CaseOutcome{}
		for _, o := range os {
			m[o.Key] = o
		}
		return m
	}

	got := by(judgeRuns(keys, []map[string]CaseOutcome{run("passed", "passed"), run("passed", "passed"), run("passed", "passed")}, 3))
	if got["A"].Status != "passed" || got["A"].Passes != 3 || got["B"].Passes != 3 {
		t.Errorf("three clean runs: %+v", got)
	}

	got = by(judgeRuns(keys, []map[string]CaseOutcome{run("passed", "passed"), run("passed", "failed")}, 3)) // B passed once, then failed
	if got["B"].Status != "flaky" || !strings.Contains(got["B"].Detail, "1 of 2") {
		t.Errorf("a case that passes and then fails is flaky, not failed: %+v", got["B"])
	}
	if got["A"].Status != "passed" || got["A"].Passes != 2 || !strings.Contains(got["A"].Detail, "not completed") {
		t.Errorf("a passing case whose repeats were cut short must say so: %+v", got["A"])
	}

	got = by(judgeRuns(keys, []map[string]CaseOutcome{run("passed", "failed")}, 3)) // stopped at the first failure
	if got["B"].Status != "failed" || got["B"].Passes != 0 {
		t.Errorf("a case that never passed is failed: %+v", got["B"])
	}

	got = by(judgeRuns(keys, nil, 3))
	if got["A"].Status != "missing" {
		t.Errorf("no results at all: %+v", got["A"])
	}
}

func TestAllPassedNeedsEveryKey(t *testing.T) {
	if allPassed([]string{"A", "B"}, map[string]CaseOutcome{"A": oc("passed")}) {
		t.Error("a case with no result is not passed")
	}
}
