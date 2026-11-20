package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Index is a small lookup over the app's crawl and the local cases: which page a case is about, and the names it uses.
// Design reads it to tell the model which cases already exist for the requirement's page (no graph, no extra tool).
type Index struct {
	Pages []IndexPage `json:"pages"`
	Cases []IndexCase `json:"cases"`
}

type IndexPage struct {
	Path  string   `json:"path"`
	Names []string `json:"names"` // buttons, tabs, fields and headings on the page (not links)
}

type IndexCase struct {
	Key   string   `json:"key"`
	Title string   `json:"title"`
	Page  string   `json:"page"`  // the first path the case navigates to
	Names []string `json:"names"` // the quoted names the case uses
}

var quoted = regexp.MustCompile(`"([^"]+)"`)
var pathWord = regexp.MustCompile(`(?:^|\s)(/[A-Za-z0-9_\-/]*)`)

// BuildIndex reads the crawl map and cases/*.md. A missing crawl or case folder just gives fewer entries.
func (c Config) BuildIndex() (Index, error) {
	var ix Index
	if raw, err := os.ReadFile(c.crawlPath()); err == nil {
		cr := &Crawl{}
		if err := json.Unmarshal(raw, cr); err != nil {
			return ix, fmt.Errorf("%s: %w", c.crawlPath(), err)
		}
		for _, p := range cr.Pages {
			u, _ := url.Parse(p.URL)
			path := "/"
			if u != nil && u.Path != "" {
				path = u.Path
			}
			pg := IndexPage{Path: path}
			for _, e := range p.Elements {
				if e.Name != "" && e.Role != "link" && e.Role != "menuitem" {
					pg.Names = append(pg.Names, e.Name)
				}
			}
			ix.Pages = append(ix.Pages, pg)
		}
	}
	cases, err := localIssues(c.localCaseDir(), "")
	if err != nil {
		return ix, err
	}
	for _, is := range cases {
		b, err := os.ReadFile(filepath.Join(c.localCaseDir(), is.Key+".md"))
		if err != nil {
			return ix, err
		}
		ic := IndexCase{Key: is.Key, Title: is.Summary}
		if m := pathWord.FindStringSubmatch(string(b)); m != nil {
			ic.Page = m[1]
		}
		for _, m := range quoted.FindAllStringSubmatch(string(b), -1) {
			ic.Names = append(ic.Names, m[1])
		}
		ix.Cases = append(ix.Cases, ic)
	}
	sort.Slice(ix.Cases, func(a, b int) bool { return ix.Cases[a].Key < ix.Cases[b].Key })
	return ix, nil
}

// Related: the cases about the requirement's page, then those sharing a name with it. At most max, best first.
func (ix Index) Related(requirement string, max int) []IndexCase {
	low := strings.ToLower(requirement)
	page := ""
	for _, p := range ix.Pages { // "/Tarjetas" or "the Tarjetas page": both name the page
		word := strings.Trim(strings.ToLower(p.Path), "/")
		if word != "" && regexp.MustCompile(`(^|[^a-z0-9_])`+regexp.QuoteMeta(word)+`($|[^a-z0-9_])`).MatchString(low) {
			page = p.Path
		}
	}
	type scored struct {
		c IndexCase
		s int
	}
	var hits []scored
	for _, c := range ix.Cases {
		s := 0
		if page != "" && c.Page == page {
			s += 2
		}
		for _, n := range c.Names {
			if strings.Contains(low, strings.ToLower(n)) {
				s++
			}
		}
		if s > 0 {
			hits = append(hits, scored{c, s})
		}
	}
	sort.SliceStable(hits, func(a, b int) bool { return hits[a].s > hits[b].s })
	var out []IndexCase
	for i := 0; i < len(hits) && i < max; i++ {
		out = append(out, hits[i].c)
	}
	return out
}
