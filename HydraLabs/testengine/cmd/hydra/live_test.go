package main

import (
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestLiveEventsAreAppendedInOrder(t *testing.T) {
	c := Config{Out: t.TempDir(), App: "App"}
	c.live("case", "A-1", map[string]any{"intent": "automate"})
	c.live("map", "A-1", map[string]any{"steps": []StepMap{{N: "1", Text: "Click", Locator: "#go", Confirmed: true, Color: "#16a34a"}}})
	w := httptest.NewRecorder()
	tailFrom(w, filepath.Join(c.liveDir(), "events.jsonl"), "1")
	got := w.Body.String()
	if strings.Count(got, "\n") != 1 || !strings.Contains(got, `"kind":"map"`) || !strings.Contains(got, `"locator":"#go"`) {
		t.Errorf("from=1 must return only the second event: %q", got)
	}
}

func TestColorsFollowTheCrawlLegend(t *testing.T) {
	for loc, want := range map[string]string{`[data-testid="go"]`: "#16a34a", "#orderNumber": "#16a34a", `role=button[name="Agregar"]`: "#2563eb", "button": "#ea580c", "": ""} {
		if got := colorOf(loc, true); got != want {
			t.Errorf("%q: %s, want %s", loc, got, want)
		}
	}
	if colorOf("#orderNumber", false) != "#ea580c" {
		t.Error("a selector not confirmed on the real app is a guess, whatever its shape")
	}
}

func TestTheLivePageOnlyOpensDeliveryReports(t *testing.T) {
	c := Config{Out: "/repo/hydra/out"}
	for p, ok := range map[string]bool{
		"/repo/hydra/out/deliver/20261017-110201/report_x.html":          true,
		"/repo/hydra/out/deliver/../../../.env":                          false,
		"/repo/hydra/out/deliver/20261017-110201/../../../conftest.html": false,
		"/repo/hydra/out/deliver/20261017-110201/pytest-1.log":           false,
		"relative/deliver/report.html":                                   false,
		"/etc/passwd.html":                                               false,
	} {
		if _, got := c.reportPath(p); got != ok {
			t.Errorf("%s: allowed=%v, want %v", p, got, ok)
		}
	}
}

func TestTheEventCountLetsThePageNoticeAClearedLog(t *testing.T) {
	c := Config{Out: t.TempDir(), App: "App"}
	c.live("case", "A-1", nil)
	c.live("case", "A-2", nil)
	w := httptest.NewRecorder()
	tailFrom(w, filepath.Join(c.liveDir(), "events.jsonl"), "5")
	if w.Header().Get("X-Total") != "2" || w.Body.Len() != 0 {
		t.Errorf("a page that read 5 lines of a 2-line log must learn it was cleared: %v %q", w.Header(), w.Body.String())
	}
}
