package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestASiblingReferencedButNotStoredIsFetchedFromJira(t *testing.T) {
	issues := map[string][2]string{ // key -> summary, description
		"DWQ-130":  {"TC-1 – Manual order entry", "<ul><li><p><strong>Preconditions:</strong> Portal is reachable.</p></li><li><p><strong>Steps:</strong></p><ol><li><p>Navigate to <tt>/cobro_ordenes</tt>.</p></li></ol></li><li><p><strong>Expected Result:</strong></p><ul><li><p>It loads.</p></li></ul></li></ul>"},
		"DWQ-1300": {"TC-10 – Something else", "<p>x</p>"},
	}
	var jql string
	jira := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/search/jql") {
			jql = r.URL.Query().Get("jql")
			w.Write([]byte(`{"issues": [{"key": "DWQ-130"}, {"key": "DWQ-1300"}]}`))
			return
		}
		key := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		is := issues[key]
		json.NewEncoder(w).Encode(map[string]any{"key": key, "fields": map[string]any{"summary": is[0], "parent": map[string]string{"key": "DWQ-129"}},
			"renderedFields": map[string]string{"description": is[1]}})
	}))
	defer jira.Close()

	c := Config{Out: t.TempDir(), Lang: "en"}
	c.Ingest(Issue{Key: "DWQ-131", Summary: "TC-2 – Bulk upload", Parent: "DWQ-129",
		HTML: "<ul><li><p><strong>Preconditions:</strong> Same as TC-1.</p></li><li><p><strong>Steps:</strong></p><ol><li><p>Click &quot;Go&quot;.</p></li></ol></li><li><p><strong>Expected Result:</strong></p><ul><li><p>Done.</p></li></ul></li></ul>"})
	if err := c.storeReferenced(&Jira{Base: jira.URL, Email: "e", Token: "t"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(jql, `parent = DWQ-129`) || !strings.Contains(jql, `summary ~ "\"TC-1\""`) {
		t.Errorf("search must stay under the case's parent: %s", jql)
	}
	cases, _ := c.loadCases()
	stored := map[string]bool{}
	for _, cs := range cases {
		stored[cs.ID] = true
	}
	if !stored["DWQ-130"] || stored["DWQ-1300"] {
		t.Errorf("only TC-1 itself is stored (never TC-10): %v", stored)
	}
	ResolveRefs(cases)
	for _, cs := range cases {
		if cs.ID == "DWQ-131" && (cs.PreconditionSource != "DWQ-130" || !strings.Contains(cs.Precondition, "Portal is reachable")) {
			t.Errorf("the reference must now resolve: %+v", cs)
		}
	}
}
