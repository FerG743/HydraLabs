package main

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTheBoardSaysWhichResultsMayNoLongerHold(t *testing.T) {
	home := t.TempDir()
	c := Config{Home: home, Out: filepath.Join(home, "hydra", "out"), AppMap: filepath.Join(home, "appmap"), App: "Shop", Project: "SHOP", CaseSource: "local", Lang: "en"}
	os.MkdirAll(c.localCaseDir(), 0o755)
	write := func(k, text string) { os.WriteFile(filepath.Join(c.localCaseDir(), k+".md"), []byte(text), 0o644) }
	write("SHOP-1", signIn)
	write("SHOP-2", strings.Replace(signIn, "valid user", "second user", 1))
	write("SHOP-3", strings.Replace(signIn, "valid user", "third user", 1))

	// SHOP-1 and SHOP-2 ran; SHOP-2's file was edited afterwards; SHOP-3 never ran
	jobs, seen, _ := collect(&c, nil, PollOpts{})
	for _, j := range jobs {
		if j.key != "SHOP-3" {
			seen[j.key] = j.stamp
			writeJSON(filepath.Join(c.Out, "runs", j.key+".json"), Run{Key: j.key, Status: "passed", Tier: "T1-render"})
		}
	}
	writeJSON(statePath(&c), seen)
	write("SHOP-2", strings.Replace(signIn, "valid user", "second user, edited", 1))

	old := time.Now().Add(-time.Hour)
	os.Chtimes(filepath.Join(c.Out, "runs", "SHOP-1.json"), old, old)
	os.MkdirAll(c.AppMap, 0o755)
	os.WriteFile(c.crawlPath(), []byte(`{"pages":[{},{}]}`), 0o644) // crawled after SHOP-1 ran, before SHOP-2 did
	crawled := time.Now().Add(-30 * time.Minute)
	os.Chtimes(c.crawlPath(), crawled, crawled)

	b := c.board()
	got := map[string]BoardCase{}
	for _, bc := range b.Board {
		got[bc.Key] = bc
	}
	if s := got["SHOP-1"].Stale; len(s) != 1 || !strings.Contains(s[0], "crawled again") || got["SHOP-1"].Status != "passed" {
		t.Errorf("SHOP-1 ran before the last crawl: %+v", got["SHOP-1"])
	}
	if s := got["SHOP-2"].Stale; len(s) != 1 || s[0] != "edited after its last run" || got["SHOP-2"].Name != "Sign in with a second user, edited" {
		t.Errorf("SHOP-2 was edited after it ran: %+v", got["SHOP-2"])
	}
	if s := got["SHOP-3"].Stale; len(s) != 1 || s[0] != "not run yet" || got["SHOP-3"].Status != "" {
		t.Errorf("SHOP-3 never ran: %+v", got["SHOP-3"])
	}
	if b.CrawlPages != 2 || b.Source != "local" || b.Passing != 2 {
		t.Errorf("header: %+v", b)
	}
}

func TestEveryViewIsMountedUnderTheBoard(t *testing.T) {
	home := t.TempDir()
	c := Config{Home: home, Out: filepath.Join(home, "out"), App: "Shop", Project: "SHOP"}
	os.MkdirAll(filepath.Join(c.Out, "crawl"), 0o755)
	os.WriteFile(filepath.Join(c.Out, "crawl", "events.jsonl"), []byte(`{"kind":"page"}`+"\n"), 0o644)
	for _, tc := range []struct{ h, path, want string }{
		{"live", "/", "Hydra live"}, {"crawl", "/events?from=0", `"kind":"page"`}, {"cases", "/", "Test cases: Shop"},
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", tc.path, nil)
		switch tc.h {
		case "live":
			c.liveView().ServeHTTP(rec, req)
		case "crawl":
			crawlView(filepath.Join(c.Out, "crawl", "events.jsonl")).ServeHTTP(rec, req)
		case "cases":
			c.casesView().ServeHTTP(rec, req)
		}
		if !strings.Contains(rec.Body.String(), tc.want) {
			t.Errorf("%s %s: %.200s", tc.h, tc.path, rec.Body.String())
		}
	}
	for _, page := range []string{livePage, viewPage, casesPage, boardPage} { // relative URLs, or the board's prefixes break them
		for _, abs := range []string{"fetch('/", "src='/", `href="/`} {
			if strings.Contains(page, abs) {
				t.Errorf("an absolute URL (%s) in a page the board mounts under a prefix", abs)
			}
		}
	}
}
