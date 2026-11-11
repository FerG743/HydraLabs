package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// hydra map: the site map read from the terminal. The app map (the crawl) and the gap report poll writes are read
// as they are; nothing is run or written.

type mapEl struct {
	Role    string `json:"role"`
	Name    string `json:"name"`
	Tag     string `json:"tag"`
	Locator string `json:"locator"`
	Href    string `json:"href"`
	Stable  bool   `json:"stable"`
}

type mapPage struct {
	URL      string  `json:"url"`
	Title    string  `json:"title"`
	Shot     string  `json:"shot"`
	Elements []mapEl `json:"elements"`
}

type crawlFile struct {
	BaseURL   string    `json:"baseUrl"`
	CrawledAt string    `json:"crawledAt"`
	Pages     []mapPage `json:"pages"`
}

type covFile struct {
	GeneratedAt string `json:"generatedAt"`
	Pages       []struct {
		URL     string   `json:"url"`
		Covered bool     `json:"covered"`
		Gaps    []string `json:"gaps"`
	} `json:"pages"`
}

// the view: the pages with their links both ways, and what the last report says about each page and element
type mapElView struct {
	Label   string `json:"label"`
	Role    string `json:"role"`
	Locator string `json:"locator"`
	Covered bool   `json:"covered"`
}

type mapPageView struct {
	Path     string      `json:"path"`
	URL      string      `json:"url"`
	Title    string      `json:"title"`
	Covered  bool        `json:"covered"`
	Shot     string      `json:"shot"`
	Links    []string    `json:"linksTo"`
	LinkedBy []string    `json:"linkedFrom"`
	Elements []mapElView `json:"elements"`
}

type mapView struct {
	App        string        `json:"app"`
	BaseURL    string        `json:"baseUrl"`
	CrawledAt  string        `json:"crawledAt"`
	Reported   bool          `json:"reported"`
	ReportedAt string        `json:"reportedAt"`
	Links      int           `json:"links"`
	Pages      []mapPageView `json:"pages"`
}

func mapNorm(u string) string {
	p, err := url.Parse(u)
	if err != nil {
		return u
	}
	p.Fragment = ""
	return strings.TrimSuffix(p.String(), "/")
}

func pagePath(u string) string {
	p, err := url.Parse(u)
	if err != nil || p.Path == "" {
		return "/"
	}
	return p.Path + func() string {
		if p.RawQuery != "" {
			return "?" + p.RawQuery
		}
		return ""
	}()
}

func elLabel(e mapEl) string {
	for _, s := range []string{e.Name, e.Tag} {
		if s != "" {
			return s
		}
	}
	return "element"
}

// buildMapView joins the crawl with the gap report. Without a report (no poll yet) every page reads as not reached.
func buildMapView(app string, crawl crawlFile, cov *covFile) mapView {
	v := mapView{App: app, BaseURL: crawl.BaseURL, CrawledAt: crawl.CrawledAt, Pages: []mapPageView{}}
	index := map[string]int{}
	for i, p := range crawl.Pages {
		index[mapNorm(p.URL)] = i
	}
	gaps := map[string]map[string]bool{} // page url -> uncovered locators (nil when no report)
	covered := map[string]bool{}
	if cov != nil {
		v.Reported, v.ReportedAt = true, cov.GeneratedAt
		for _, r := range cov.Pages {
			covered[mapNorm(r.URL)] = r.Covered
			gaps[mapNorm(r.URL)] = map[string]bool{}
			for _, g := range r.Gaps {
				gaps[mapNorm(r.URL)][g] = true
			}
		}
	}
	linked := make([]map[int]bool, len(crawl.Pages))
	linkedBy := make([]map[int]bool, len(crawl.Pages))
	for i := range crawl.Pages {
		linked[i], linkedBy[i] = map[int]bool{}, map[int]bool{}
	}
	for i, p := range crawl.Pages {
		base, err := url.Parse(p.URL)
		if err != nil {
			continue
		}
		for _, e := range p.Elements {
			if e.Href == "" {
				continue
			}
			ref, err := url.Parse(e.Href)
			if err != nil {
				continue
			}
			j, ok := index[mapNorm(base.ResolveReference(ref).String())]
			if ok && j != i && !linked[i][j] {
				linked[i][j], linkedBy[j][i] = true, true
				v.Links++
			}
		}
	}
	for i, p := range crawl.Pages {
		key := mapNorm(p.URL)
		vp := mapPageView{Path: pagePath(p.URL), URL: p.URL, Title: p.Title, Shot: p.Shot, Covered: covered[key]}
		if !v.Reported {
			vp.Covered = false
		}
		for _, j := range sortedKeys(linked[i]) {
			vp.Links = append(vp.Links, pagePath(crawl.Pages[j].URL))
		}
		for _, j := range sortedKeys(linkedBy[i]) {
			vp.LinkedBy = append(vp.LinkedBy, pagePath(crawl.Pages[j].URL))
		}
		for _, e := range p.Elements {
			ok := v.Reported && !gaps[key][e.Locator]
			vp.Elements = append(vp.Elements, mapElView{Label: elLabel(e), Role: e.Role, Locator: e.Locator, Covered: ok})
		}
		v.Pages = append(v.Pages, vp)
	}
	return v
}

func sortedKeys(m map[int]bool) []int {
	var ks []int
	for k := range m {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	return ks
}

// mapAllProjects prints the map of every project in the registry, one block each; the page and JSON views need one project.
func mapAllProjects(reg *Registry, gaps bool, page string, asJSON bool) error {
	if page != "" || asJSON {
		return fmt.Errorf("--page and --json need one project: hydra map --project KEY")
	}
	regPath := filepath.Join(reg.dir, "projects.json")
	for _, k := range reg.Keys() {
		fmt.Printf("== %s\n", k)
		cfg, err := cfgFlags(flag.NewFlagSet("map-"+k, flag.ContinueOnError), []string{"--project", k, "--registry", regPath})
		if err != nil {
			fmt.Fprintf(os.Stderr, "hydra: %s: %v\n", k, err)
			continue
		}
		if err := cfg.printMap(gaps, "", false); err != nil {
			fmt.Fprintf(os.Stderr, "hydra: %s: %v\n", k, err)
		}
		fmt.Println()
	}
	return nil
}

func (c Config) printMap(gaps bool, page string, asJSON bool) error {
	if c.App == "" {
		return fmt.Errorf("say which project: hydra map --project KEY (for example DWQ)")
	}
	b, err := os.ReadFile(c.crawlPath())
	if err != nil {
		return fmt.Errorf("no app map for %s yet: run hydra crawl first", c.App)
	}
	var crawl crawlFile
	if err := json.Unmarshal(b, &crawl); err != nil {
		return fmt.Errorf("%s: %w", c.crawlPath(), err)
	}
	var cov *covFile
	if cb, err := os.ReadFile(filepath.Join(c.Out, "coverage.json")); err == nil {
		var f covFile
		if json.Unmarshal(cb, &f) == nil {
			cov = &f
		}
	}
	v := buildMapView(c.App, crawl, cov)
	switch {
	case asJSON:
		out, _ := json.MarshalIndent(v, "", "  ")
		fmt.Println(string(out))
	case page != "":
		return printMapPage(c, v, page)
	case gaps:
		printMapGaps(v)
	default:
		printMapSummary(v)
	}
	return nil
}

func printMapSummary(v mapView) {
	reached, pages, els, covEls := 0, len(v.Pages), 0, 0
	for _, p := range v.Pages {
		if p.Covered {
			reached++
		}
		for _, e := range p.Elements {
			els++
			if e.Covered {
				covEls++
			}
		}
	}
	fmt.Printf("%s · %d pages · %d links · crawled %s\n", v.App, pages, v.Links, v.CrawledAt)
	if v.Reported {
		fmt.Printf("built tests reach %d of %d pages and %d of %d elements (report %s)\n", reached, pages, covEls, els, v.ReportedAt)
	} else {
		fmt.Println("no coverage report yet: a poll writes it")
	}
	for _, p := range v.Pages {
		mark := " "
		if v.Reported {
			mark = map[bool]string{true: "+", false: "-"}[p.Covered]
		}
		n := 0
		for _, e := range p.Elements {
			if e.Covered {
				n++
			}
		}
		fmt.Printf("%s %-40s %3d/%-3d elements  %s\n", mark, p.Path, n, len(p.Elements), p.Title)
	}
}

func printMapGaps(v mapView) {
	if !v.Reported {
		fmt.Println("no coverage report yet: a poll writes it")
		return
	}
	any := false
	for _, p := range v.Pages {
		var missing []mapElView
		for _, e := range p.Elements {
			if !e.Covered {
				missing = append(missing, e)
			}
		}
		if p.Covered && len(missing) == 0 {
			continue
		}
		any = true
		if !p.Covered {
			fmt.Printf("%s  not reached by any built test\n", p.Path)
		} else {
			fmt.Printf("%s\n", p.Path)
		}
		for _, e := range missing {
			fmt.Printf("    %-12s %-30s %s\n", e.Role, e.Label, e.Locator)
		}
	}
	if !any {
		fmt.Println("no gaps: every page and element is reached by a built test")
	}
}

func printMapPage(c Config, v mapView, want string) error {
	for _, p := range v.Pages {
		if p.Path != want && mapNorm(p.URL) != mapNorm(want) {
			continue
		}
		fmt.Printf("%s\n%s\n", p.Path, p.URL)
		if p.Title != "" {
			fmt.Println(p.Title)
		}
		if v.Reported {
			fmt.Printf("reached by a built test: %v\n", p.Covered)
		}
		fmt.Printf("links to:    %s\n", strings.Join(orNone(p.Links), "  "))
		fmt.Printf("linked from: %s\n", strings.Join(orNone(p.LinkedBy), "  "))
		if p.Shot != "" {
			fmt.Printf("screenshot:  %s\n", filepath.Join(c.Out, "crawl", "shots", p.Shot))
		}
		fmt.Printf("elements (%d):\n", len(p.Elements))
		for _, e := range p.Elements {
			status := ""
			if v.Reported {
				status = map[bool]string{true: "covered", false: "gap"}[e.Covered]
			}
			fmt.Printf("  %-12s %-30s %-8s %s\n", e.Role, e.Label, status, e.Locator)
		}
		return nil
	}
	return fmt.Errorf("no page %s in the map (hydra map lists them)", want)
}

func orNone(s []string) []string {
	if len(s) == 0 {
		return []string{"none"}
	}
	return s
}
