package hydramaps

import "sort"

// The coverage KPIs of the map: counted per unique locator, so the nav links that repeat on every page count once.

// interactive roles: what a person acts on (a link only navigates, so it is left out of this count)
var interactiveRoles = map[string]bool{"button": true, "textbox": true, "combobox": true, "checkbox": true,
	"radio": true, "switch": true, "tab": true, "spinbutton": true, "slider": true, "searchbox": true}

type Gap struct {
	Path      string `json:"path"`
	Uncovered int    `json:"uncoveredElements"`
}

type KPIs struct {
	UniqueLocators     int   `json:"uniqueLocators"`
	CoveredLocators    int   `json:"coveredLocators"`
	Interactive        int   `json:"interactiveLocators"`
	InteractiveCovered int   `json:"interactiveCovered"`
	BiggestGaps        []Gap `json:"biggestGaps"`
}

// ComputeKPIs needs a report: without one nothing is covered yet, so the KPIs would only say so.
func ComputeKPIs(v View) KPIs {
	type seen struct {
		role    string
		covered bool
	}
	locs := map[string]*seen{}
	var gaps []Gap
	for _, p := range v.Pages {
		uncovered := 0
		for _, e := range p.Elements {
			s := locs[e.Locator]
			if s == nil {
				s = &seen{role: e.Role}
				locs[e.Locator] = s
			}
			if e.Covered {
				s.covered = true
			} else {
				uncovered++
			}
		}
		if uncovered > 0 {
			gaps = append(gaps, Gap{Path: p.Path, Uncovered: uncovered})
		}
	}
	var k KPIs
	for _, s := range locs {
		k.UniqueLocators++
		if s.covered {
			k.CoveredLocators++
		}
		if interactiveRoles[s.role] {
			k.Interactive++
			if s.covered {
				k.InteractiveCovered++
			}
		}
	}
	sort.SliceStable(gaps, func(i, j int) bool { return gaps[i].Uncovered > gaps[j].Uncovered })
	if len(gaps) > 5 {
		gaps = gaps[:5]
	}
	k.BiggestGaps = gaps
	return k
}
