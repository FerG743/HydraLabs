package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func crawlJSON(t *testing.T, at string, pages map[string][]CrawlElement) []byte {
	type pg struct {
		URL      string         `json:"url"`
		Title    string         `json:"title"`
		Elements []CrawlElement `json:"elements"`
	}
	doc := map[string]any{"baseUrl": "http://127.0.0.1:3000", "crawledAt": at}
	var ps []pg
	for u, els := range pages {
		ps = append(ps, pg{"http://127.0.0.1:3000" + u, "T" + u, els})
	}
	doc["pages"] = ps
	b, _ := json.Marshal(doc)
	return b
}

func stable(loc string) CrawlElement {
	return CrawlElement{Role: "button", Name: loc, Locator: loc, Stable: true}
}
func weak(loc string) CrawlElement {
	return CrawlElement{Role: "button", Name: loc, Locator: loc, Stable: false}
}

func TestOnlyTheTimestampIsNotNews(t *testing.T) {
	a := crawlJSON(t, "2026-10-08T10:00:00", map[string][]CrawlElement{"/": {stable("#a")}})
	b := crawlJSON(t, "2026-10-09T09:00:00", map[string][]CrawlElement{"/": {stable("#a")}})
	c := crawlJSON(t, "2026-10-09T09:00:00", map[string][]CrawlElement{"/": {stable("#a"), stable("#b")}})
	if !sameCrawl(a, b) {
		t.Error("a different crawledAt alone must not count as a change")
	}
	if sameCrawl(a, c) {
		t.Error("a new element is a change")
	}
}

func TestChangeLineSaysWhatMoved(t *testing.T) {
	old, _ := loadCrawlDoc(crawlJSON(t, "x", map[string][]CrawlElement{"/": {stable("#a"), stable("#gone")}, "/old": {stable("#o")}}))
	cur, _ := loadCrawlDoc(crawlJSON(t, "y", map[string][]CrawlElement{"/": {stable("#a"), stable("#new1"), stable("#new2")}, "/fresh": {stable("#f")}}))
	got := crawlChange(old, cur)
	for _, want := range []string{"2 pages, 4 elements", "locators +3 -2", "new pages: /fresh", "pages gone: /old"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
	if first := crawlChange(nil, cur); !strings.HasPrefix(first, "first crawl:") {
		t.Errorf("%q", first)
	}
}

func TestSummaryListsWeakLocators(t *testing.T) {
	d, _ := loadCrawlDoc(crawlJSON(t, "2026-10-08", map[string][]CrawlElement{"/": {stable("#a"), weak("[data-testid=dup]")}}))
	s := CrawlSummary("App", d)
	for _, want := range []string{"# Crawl of App", "| `/` | T/ | 2 | 1 | 1 |", "## Weak locators", "[data-testid=dup]"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
	}
}

func TestSaveCrawlWorksOnABranchAndSkipsRepeats(t *testing.T) {
	t.Setenv("GIT_AUTHOR_NAME", "t")
	t.Setenv("GIT_AUTHOR_EMAIL", "t@t")
	t.Setenv("GIT_COMMITTER_NAME", "t")
	t.Setenv("GIT_COMMITTER_EMAIL", "t@t")
	repo := t.TempDir()
	for _, args := range [][]string{{"init", "-q", "-b", "main"}, {"commit", "-q", "--allow-empty", "-m", "base"}} {
		if out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("%v %s", err, out)
		}
	}
	am := t.TempDir()
	c := Config{DeliverRepo: repo, BaseBranch: "main", App: "MiApp", AppMap: am}
	write := func(at string, els ...CrawlElement) {
		os.WriteFile(c.crawlPath(), crawlJSON(t, at, map[string][]CrawlElement{"/": els}), 0o644)
	}

	write("t1", stable("#a"))
	s, err := c.SaveCrawl()
	if err != nil || s.Branch == "" || !strings.HasPrefix(s.Change, "first crawl") {
		t.Fatalf("%+v %v", s, err)
	}
	if cur, _ := git(repo, "branch", "--show-current"); cur != "main" {
		t.Errorf("must leave the repo on main, on %s", cur)
	}
	if _, err := os.Stat(filepath.Join(repo, "hydra", "crawl", "crawl.json")); err == nil {
		t.Error("main must not contain the crawl until the branch is merged")
	}
	for _, f := range []string{"crawl.json", "CRAWL.md"} { // the branch carries both, in the app's hydra/ folder
		if _, err := git(repo, "cat-file", "-e", s.Branch+":hydra/crawl/"+f); err != nil {
			t.Errorf("branch lacks hydra/crawl/%s: %v", f, err)
		}
	}
	// not merged yet: the crawl is awaiting review, so the same crawl must not be stored a second time
	write("t1b-only-the-time-changed", stable("#a"))
	if again, _ := c.SaveCrawl(); again.Branch != "" || !strings.Contains(again.Note, s.Branch) {
		t.Errorf("a crawl already waiting on a branch is not stored twice: %+v", again)
	}
	git(repo, "merge", "-q", "--ff-only", s.Branch)

	write("t2-only-the-time-changed", stable("#a"))
	if s, _ = c.SaveCrawl(); s.Branch != "" || !strings.Contains(s.Note, "unchanged") {
		t.Errorf("an identical crawl must not create a branch: %+v", s)
	}

	write("t3", stable("#a"), stable("#b"))
	s, err = c.SaveCrawl()
	if err != nil || !strings.Contains(s.Change, "locators +1 -0") {
		t.Fatalf("%+v %v", s, err)
	}
	if msg, _ := git(repo, "log", "-1", "--format=%s", s.Branch); !strings.Contains(msg, "locators +1 -0") {
		t.Errorf("the commit message must carry the change: %q", msg)
	}
}

func TestSaveCrawlRefusesADirtyRepo(t *testing.T) {
	repo := t.TempDir()
	exec.Command("git", "-C", repo, "init", "-q", "-b", "main").Run()
	os.WriteFile(filepath.Join(repo, "mine.txt"), []byte("work in progress"), 0o644)
	if _, err := (Config{DeliverRepo: repo, BaseBranch: "main", App: "A", AppMap: t.TempDir()}).SaveCrawl(); err == nil {
		t.Error("must not touch a repo with uncommitted work")
	}
}

func TestBranchNamesNeverCollide(t *testing.T) {
	repo := t.TempDir()
	exec.Command("git", "-C", repo, "init", "-q", "-b", "main").Run()
	exec.Command("git", "-C", repo, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", "b").Run()
	if got := uniqueBranch(repo, "hydra/x"); got != "hydra/x" {
		t.Fatalf("free name is kept: %s", got)
	}
	exec.Command("git", "-C", repo, "branch", "hydra/x").Run()
	exec.Command("git", "-C", repo, "branch", "hydra/x-2").Run()
	if got := uniqueBranch(repo, "hydra/x"); got != "hydra/x-3" {
		t.Errorf("taken names get the next free suffix: %s", got)
	}
}
