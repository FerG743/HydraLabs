package hydramaps

import "testing"

func TestKPIsCountUniqueLocatorsNotRepeatedNavLinks(t *testing.T) {
	nav := ElementView{Locator: "role=link[name=\"Home\"]", Role: "link"}
	btn := ElementView{Locator: "role=button[name=\"Go\"]", Role: "button"}
	v := View{Pages: []PageView{
		{Path: "/a", Elements: []ElementView{nav, {Locator: btn.Locator, Role: "button", Covered: true}}},
		{Path: "/b", Elements: []ElementView{nav, {Locator: "role=textbox[name=\"Q\"]", Role: "textbox"}}},
	}}
	k := ComputeKPIs(v)
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
