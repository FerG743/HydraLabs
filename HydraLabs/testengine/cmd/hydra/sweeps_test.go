package main

import (
	"os"
	"path/filepath"
	"testing"
)

// A swept dropdown's options become messages, each with the fields that message shows.
func TestSweepsBecomeMessagesWithTheirFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "crawl.json")
	os.WriteFile(path, []byte(`{"pages":[{"url":"http://x/SPDH","sweep":{"control":"Mensaje","baseline":["textbox:Monto"],"options":{
		"Greet":{"adds":["textbox:Otro"],"removes":["textbox:Monto"]},
		"Plan 08":{"adds":["textbox:Referencia"]},
		"Enviar":{"skipped":"looks like a send"}}}}]}`), 0o644)
	views := sweepsFromFile(path)
	if len(views) != 1 || views[0].Page != "/SPDH" || views[0].Control != "Mensaje" {
		t.Fatalf("views: %+v", views)
	}
	msgs := map[string][]string{}
	for _, m := range views[0].Messages {
		msgs[m.Name] = m.Fields
	}
	if len(msgs) != 2 || len(msgs["Greet"]) != 1 || msgs["Greet"][0] != "textbox:Otro" {
		t.Fatalf("Greet shows only its own fields: %+v", msgs)
	}
	if len(msgs["Plan 08"]) != 2 {
		t.Fatalf("Plan 08 keeps the baseline and adds its field: %+v", msgs["Plan 08"])
	}
}
