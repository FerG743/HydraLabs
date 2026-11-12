package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testengine/cmd/hydra/hydramaps"
)

// hydra map: the site map read from the terminal. The app map (the crawl) and the gap report poll writes are read
// as they are; nothing is run or written.

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

// loadMapView reads the app map and the gap report (if any) of this project.
func (c Config) loadMapView() (hydramaps.View, error) {
	b, err := os.ReadFile(c.crawlPath())
	if err != nil {
		return hydramaps.View{}, fmt.Errorf("no app map for %s yet: run hydra crawl first", c.App)
	}
	var crawl hydramaps.Crawl
	if err := json.Unmarshal(b, &crawl); err != nil {
		return hydramaps.View{}, fmt.Errorf("%s: %w", c.crawlPath(), err)
	}
	var cov *hydramaps.Report
	if cb, err := os.ReadFile(filepath.Join(c.Out, "coverage.json")); err == nil {
		var f hydramaps.Report
		if json.Unmarshal(cb, &f) == nil {
			cov = &f
		}
	}
	return hydramaps.BuildView(c.App, crawl, cov), nil
}

func (c Config) printMap(gaps bool, page string, asJSON bool) error {
	if c.App == "" {
		return fmt.Errorf("say which project: hydra map --project KEY (for example DWQ)")
	}
	v, err := c.loadMapView()
	if err != nil {
		return err
	}
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

func printMapSummary(v hydramaps.View) {
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
	if k := v.KPIs; k != nil {
		fmt.Printf("unique locators reached   %d of %d\n", k.CoveredLocators, k.UniqueLocators)
		fmt.Printf("interactive reached       %d of %d (buttons, inputs, tabs)\n", k.InteractiveCovered, k.Interactive)
		var parts []string
		for _, g := range k.BiggestGaps {
			parts = append(parts, fmt.Sprintf("%s %d", g.Path, g.Uncovered))
		}
		if len(parts) > 0 {
			fmt.Printf("biggest gaps (uncovered elements)  %s\n", strings.Join(parts, " · "))
		}
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

func printMapGaps(v hydramaps.View) {
	if !v.Reported {
		fmt.Println("no coverage report yet: a poll writes it")
		return
	}
	any := false
	for _, p := range v.Pages {
		var missing []hydramaps.ElementView
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

func printMapPage(c Config, v hydramaps.View, want string) error {
	for _, p := range v.Pages {
		if p.Path != want && hydramaps.Norm(p.URL) != hydramaps.Norm(want) {
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

// projectPagesArgs rewrites "DWQ pages [flags]" to "map --project DWQ [flags]", so the project comes first.
func projectPagesArgs(reg *Registry, args []string) ([]string, bool) {
	if reg == nil || len(args) < 2 || args[1] != "pages" {
		return nil, false
	}
	if _, ok := reg.Projects[args[0]]; !ok {
		return nil, false
	}
	return append([]string{"map", "--project", args[0]}, args[2:]...), true
}
