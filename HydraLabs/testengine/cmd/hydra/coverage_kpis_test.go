package main

import "testing"

func TestKPIsCountUniqueLocatorsNotRepeatedNavLinks(t *testing.T) {
	nav := mapElView{Locator: "role=link[name=\"Home\"]", Role: "link"}
	btn := mapElView{Locator: "role=button[name=\"Go\"]", Role: "button"}
	v := mapView{Pages: []mapPageView{
		{Path: "/a", Elements: []mapElView{nav, {Locator: btn.Locator, Role: "button", Covered: true}}},
		{Path: "/b", Elements: []mapElView{nav, {Locator: "role=textbox[name=\"Q\"]", Role: "textbox"}}},
	}}
	k := computeKPIs(v)
	if k.UniqueLocators != 3 || k.CoveredLocators != 1 {
		t.Fatalf("unique: %+v", k)
	}
	if k.Interactive != 2 || k.InteractiveCovered != 1 {
		t.Fatalf("interactive: %+v", k)
	}
	if len(k.BiggestGaps) != 2 || k.BiggestGaps[0].Path != "/b" || k.BiggestGaps[0].Uncovered != 2 {
		t.Fatalf("biggest gaps: %+v", k.BiggestGaps)
	}
}
