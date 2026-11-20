package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Saving a reviewed run from the board writes its passing drafts once; a run outside the design folder is not read.
func TestDesignSaveWritesThePassingDraftsOfARun(t *testing.T) {
	dir := t.TempDir()
	c := Config{Home: dir, Out: dir, Project: "PQA", App: "PortalQAdev"}
	os.MkdirAll(c.localCaseDir(), 0o755)
	run := filepath.Join(dir, "design", "20990101-000000")
	os.MkdirAll(run, 0o755)
	writeJSON(filepath.Join(run, "design.json"), Design{Drafts: []DraftResult{
		{OK: true, Draft: Draft{Name: "Tarjetas shows Agregar", Steps: []string{"Navigate to /Tarjetas."}, Expected: []string{"The page shows \"Agregar\": it appears in \"Agregar\"."}}},
		{OK: false, Draft: Draft{Name: "rejected"}},
	}})
	h := c.designView()
	post := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/save", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	rec := post(`{"run":"20990101-000000"}`)
	var out struct {
		Keys []string `json:"keys"`
	}
	json.Unmarshal(rec.Body.Bytes(), &out)
	if rec.Code != 200 || len(out.Keys) != 1 || out.Keys[0] != "PQA-1" {
		t.Fatalf("save: %d %s", rec.Code, rec.Body.String())
	}
	if again := post(`{"run":"20990101-000000"}`); strings.Contains(again.Body.String(), "PQA-") {
		t.Fatalf("saved twice: %s", again.Body.String())
	}
	if bad := post(`{"run":"../../etc"}`); bad.Code == 200 {
		t.Fatalf("a run outside the folder was accepted: %s", bad.Body.String())
	}
}
