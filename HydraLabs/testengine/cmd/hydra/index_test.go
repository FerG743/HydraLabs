package main

import (
	"os"
	"path/filepath"
	"testing"
)

// The index finds the cases about the requirement's page and the names it shares with them.
func TestIndexFindsRelatedCases(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "cases"), 0o755)
	os.WriteFile(filepath.Join(dir, "cases", "PQA-1.md"), []byte("# Tarjetas shows Agregar\nlabels: automate\n\n- Steps:\n  1. Navigate to /Tarjetas.\n  2. Verify \"Agregar\" is enabled.\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "cases", "PQA-2.md"), []byte("# Cobro page\nlabels: automate\n\n- Steps:\n  1. Navigate to /cobro_ordenes.\n"), 0o644)
	os.MkdirAll(filepath.Join(dir, "map"), 0o755)
	os.WriteFile(filepath.Join(dir, "map", "PQA.crawl.json"), []byte(`{"pages":[{"url":"http://x/Tarjetas","elements":[{"role":"button","name":"Agregar"}]},{"url":"http://x/cobro_ordenes","elements":[]}]}`), 0o644)
	c := Config{Home: dir, App: "PQA", AppMap: filepath.Join(dir, "map")}
	ix, err := c.BuildIndex()
	if err != nil {
		t.Fatal(err)
	}
	got := ix.Related("The Tarjetas page shows the buttons \"Agregar\" and \"Procesar\".", 5)
	if len(got) != 1 || got[0].Key != "PQA-1" {
		t.Fatalf("related: %+v", got)
	}
	if byPage := ix.Related("The Tarjetas page is open.", 5); len(byPage) != 1 || byPage[0].Key != "PQA-1" {
		t.Fatalf("page named in words: %+v", byPage)
	}
	if none := ix.Related("Nothing here about anything.", 5); len(none) != 0 {
		t.Fatalf("unrelated requirement matched: %+v", none)
	}
}
