package hydramaps

import (
	"net/url"
	"sort"
	"strings"
)

// The model of the app map: the crawl joined with the gap report, read the same by the CLI, the board and the KPIs.

type Element struct {
	Role    string `json:"role"`
	Name    string `json:"name"`
	Tag     string `json:"tag"`
	Locator string `json:"locator"`
	Href    string `json:"href"`
	Stable  bool   `json:"stable"`
}

type Page struct {
	URL      string    `json:"url"`
	Title    string    `json:"title"`
	Shot     string    `json:"shot"`
	Elements []Element `json:"elements"`
}

type Crawl struct {
	BaseURL   string `json:"baseUrl"`
	CrawledAt string `json:"crawledAt"`
	Pages     []Page `json:"pages"`
}

type Report struct {
	GeneratedAt string `json:"generatedAt"`
	Pages       []struct {
		URL     string   `json:"url"`
		Covered bool     `json:"covered"`
		Gaps    []string `json:"gaps"`
	} `json:"pages"`
}

// the view: the pages with their links both ways, and what the last report says about each page and element
type ElementView struct {
	Label   string `json:"label"`
	Role    string `json:"role"`
	Locator string `json:"locator"`
	Covered bool   `json:"covered"`
}

type PageView struct {
	Path     string        `json:"path"`
	URL      string        `json:"url"`
	Title    string        `json:"title"`
	Covered  bool          `json:"covered"`
	Shot     string        `json:"shot"`
	Links    []string      `json:"linksTo"`
	LinkedBy []string      `json:"linkedFrom"`
	Elements []ElementView `json:"elements"`
}

type View struct {
	App        string     `json:"app"`
	BaseURL    string     `json:"baseUrl"`
	CrawledAt  string     `json:"crawledAt"`
	Reported   bool       `json:"reported"`
	ReportedAt string     `json:"reportedAt"`
	Links      int        `json:"links"`
	Pages      []PageView `json:"pages"`
	KPIs       *KPIs      `json:"kpis,omitempty"`
}

func Norm(u string) string {
	p, err := url.Parse(u)
	if err != nil {
		return u
	}
	p.Fragment = ""
	return strings.TrimSuffix(p.String(), "/")
}

func PagePath(u string) string {
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

func ElLabel(e Element) string {
	for _, s := range []string{e.Name, e.Tag} {
		if s != "" {
			return s
		}
	}
	return "element"
}

// BuildView joins the crawl with the gap report. Without a report (no poll yet) every page reads as not reached.
func BuildView(app string, crawl Crawl, cov *Report) View {
	v := View{App: app, BaseURL: crawl.BaseURL, CrawledAt: crawl.CrawledAt, Pages: []PageView{}}
	index := map[string]int{}
	for i, p := range crawl.Pages {
		index[Norm(p.URL)] = i
	}
	gaps := map[string]map[string]bool{} // page url -> uncovered locators (nil when no report)
	covered := map[string]bool{}
	if cov != nil {
		v.Reported, v.ReportedAt = true, cov.GeneratedAt
		for _, r := range cov.Pages {
			covered[Norm(r.URL)] = r.Covered
			gaps[Norm(r.URL)] = map[string]bool{}
			for _, g := range r.Gaps {
				gaps[Norm(r.URL)][g] = true
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
			j, ok := index[Norm(base.ResolveReference(ref).String())]
			if ok && j != i && !linked[i][j] {
				linked[i][j], linkedBy[j][i] = true, true
				v.Links++
			}
		}
	}
	for i, p := range crawl.Pages {
		key := Norm(p.URL)
		vp := PageView{Path: PagePath(p.URL), URL: p.URL, Title: p.Title, Shot: p.Shot, Covered: covered[key]}
		if !v.Reported {
			vp.Covered = false
		}
		for _, j := range SortedKeys(linked[i]) {
			vp.Links = append(vp.Links, PagePath(crawl.Pages[j].URL))
		}
		for _, j := range SortedKeys(linkedBy[i]) {
			vp.LinkedBy = append(vp.LinkedBy, PagePath(crawl.Pages[j].URL))
		}
		for _, e := range p.Elements {
			ok := v.Reported && !gaps[key][e.Locator]
			vp.Elements = append(vp.Elements, ElementView{Label: ElLabel(e), Role: e.Role, Locator: e.Locator, Covered: ok})
		}
		v.Pages = append(v.Pages, vp)
	}
	if v.Reported {
		k := ComputeKPIs(v)
		v.KPIs = &k
	}
	return v
}

// SortedKeys lists the keys of a map in order.
func SortedKeys(m map[int]bool) []int {
	var ks []int
	for k := range m {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	return ks
}
