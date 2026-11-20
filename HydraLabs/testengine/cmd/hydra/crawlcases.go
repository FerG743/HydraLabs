package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// CrawlCases writes one smoke case per crawled page: open it, see its buttons and tabs. The crawl knows what is on a
// page, not what a button should do, so these only prove the page loads; behaviour cases still need a person.
// Written to a folder for review, not to cases/, so nothing runs until someone moves a file there.
func (c Config) CrawlCases(dir string) (int, error) {
	raw, err := os.ReadFile(c.crawlPath())
	if err != nil {
		return 0, fmt.Errorf("no crawl of %s yet: run hydra crawl --project %s first", c.App, c.Project)
	}
	var cr Crawl
	if err := json.Unmarshal(raw, &cr); err != nil {
		return 0, fmt.Errorf("%s: %w", c.crawlPath(), err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return 0, err
	}
	type page struct {
		path string
		els  []CrawlElement
	}
	var pages []page
	seen, onPages := map[string]bool{}, map[string]int{} // onPages: how many pages each name is on
	for _, p := range cr.Pages {
		u, err := url.Parse(p.URL)
		if err != nil {
			continue
		}
		path := u.Path
		if path == "" {
			path = "/"
		}
		if seen[path] {
			continue
		}
		seen[path] = true
		pages = append(pages, page{path, p.Elements})
		names := map[string]bool{}
		for _, e := range p.Elements {
			names[e.Name] = true
		}
		for nm := range names {
			onPages[nm]++
		}
	}
	n := 0
	for _, p := range pages {
		path := p.path
		var shown []string
		for _, e := range p.els {
			// a name on most pages is the app's chrome (header, theme switch): it proves nothing about this page
			if (e.Role == "button" || e.Role == "tab") && e.Name != "" && onPages[e.Name]*2 <= len(pages) && len(shown) < 3 {
				shown = append(shown, e.Name)
			}
		}
		if len(shown) == 0 { // a page of links only: nothing here to check
			continue
		}
		n++
		var md strings.Builder
		fmt.Fprintf(&md, "# Open the page %s\nlabels: automate\n\n- Preconditions: none.\n- Steps:\n  1. Navigate to %s.\n- Expected result:\n", path, path)
		for _, s := range shown {
			fmt.Fprintf(&md, "  - The page shows \"%s\".\n", s)
		}
		name := filepath.Join(dir, fmt.Sprintf("%s-SMOKE-%d.md", c.Project, n))
		if err := os.WriteFile(name, []byte(md.String()), 0o644); err != nil {
			return n, err
		}
	}
	return n, nil
}
