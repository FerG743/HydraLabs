package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Saving a chosen value: it appends a new name, never overwrites one already set, and refuses a bad name or a broken line.
func TestSaveEnvValueAppendsOnceAndRefusesBadInput(t *testing.T) {
	home := t.TempDir()
	c := Config{Home: home}
	os.WriteFile(filepath.Join(home, ".env"), []byte("QA_CARD=test-card\n"), 0o600)
	if err := c.saveEnvValue("NUMERO_DE_ORDEN", "1234"); err != nil {
		t.Fatal(err)
	}
	if err := c.saveEnvValue("QA_CARD", "other"); err == nil {
		t.Fatal("an existing name was overwritten")
	}
	if err := c.saveEnvValue("bad name", "x"); err == nil {
		t.Fatal("a bad name was accepted")
	}
	if err := c.saveEnvValue("TIENDA", "LIVR\nINJECTED=1"); err == nil {
		t.Fatal("a value with a line break was accepted")
	}
	b, _ := os.ReadFile(filepath.Join(home, ".env"))
	if !strings.Contains(string(b), "NUMERO_DE_ORDEN=1234\n") || strings.Contains(string(b), "INJECTED") || !strings.Contains(string(b), "QA_CARD=test-card") {
		t.Fatalf(".env: %q", b)
	}
}

// The inputs list gives each field and dropdown its .env name, its options, and a suggestion.
func TestInputsListGivesNamesOptionsAndSuggestions(t *testing.T) {
	cr := &Crawl{Pages: []struct {
		URL      string
		Elements []CrawlElement
	}{{URL: "http://x/cobro_ordenes", Elements: []CrawlElement{
		{Role: "textbox", Name: "Ingresa el número de orden", Placeholder: "123456"},
		{Role: "combobox", Name: "Ingresa la tienda", Options: []string{"LIVR", "SBB"}},
	}}}}
	got := inputsOf(cr, map[string]bool{"TIENDA": true})
	if len(got) != 2 {
		t.Fatalf("inputs: %+v", got)
	}
	for _, in := range got {
		if in.Field == "combobox" && (in.EnvName != "TIENDA" || !in.InEnv || in.Suggestion != "LIVR") {
			t.Fatalf("dropdown: %+v", in)
		}
		if in.Field == "textbox" && (in.EnvName != "NUMERO_DE_ORDEN" || in.Suggestion != "123456") {
			t.Fatalf("text box: %+v", in)
		}
	}
}

// Values the crawl took for names have no word in them; labels do. Values go to their own group.
func TestValuesAreNotLabels(t *testing.T) {
	for _, v := range []string{"01", "1", "1033030823", "303030303035353231303731343334", "0002"} {
		if labelLike(v) {
			t.Fatalf("%q read as a label", v)
		}
	}
	for _, l := range []string{"Cantidad 1", "Ingresa la cuenta", "Consecutivo_Boleta (HEX, 4 hex)"} {
		if !labelLike(l) {
			t.Fatalf("%q not read as a label", l)
		}
	}
}

// Inputs keep their page and tab, so the draft page can group them.
func TestInputsKeepPageAndTab(t *testing.T) {
	cr := &Crawl{Pages: []struct {
		URL      string
		Elements []CrawlElement
	}{{URL: "http://x/SPDH", Elements: []CrawlElement{
		{Role: "combobox", Name: "Mensaje", Options: []string{"Greet"}, Under: "Adquirente"},
		{Role: "textbox", Name: "01"},
	}}}}
	got := inputsOf(cr, map[string]bool{})
	if len(got) != 2 {
		t.Fatalf("inputs: %+v", got)
	}
	for _, in := range got {
		if in.Page != "/SPDH" {
			t.Fatalf("page: %+v", in)
		}
		if in.Name == "Mensaje" && (in.Group != "Adquirente" || !in.Labelled) {
			t.Fatalf("group: %+v", in)
		}
		if in.Name == "01" && in.Labelled {
			t.Fatalf("value read as a label: %+v", in)
		}
	}
}

// A dropdown the crawl could not open gets the options a person typed in, marked as such; the crawl's own options win.
func TestHandOptionsFillOnlyWhatTheCrawlMissed(t *testing.T) {
	cr := &Crawl{Pages: []struct {
		URL      string
		Elements []CrawlElement
	}{{URL: "http://x/SPDH", Elements: []CrawlElement{
		{Role: "combobox", Name: "Formato_Negocio (EBCDIC, 4 hex)"},
		{Role: "combobox", Name: "Mensaje", Options: []string{"Greet"}},
	}}}}
	hand := map[string][]string{"Formato_Negocio (EBCDIC, 4 hex)": {"LV — Liverpool"}, "Mensaje": {"Otra"}}
	for _, in := range inputsOfHand(cr, map[string]bool{}, hand) {
		if in.Name == "Formato_Negocio (EBCDIC, 4 hex)" && (!in.HandEntered || len(in.Options) != 1) {
			t.Fatalf("hand options not used: %+v", in)
		}
		if in.Name == "Mensaje" && (in.HandEntered || in.Options[0] != "Greet") {
			t.Fatalf("the crawl's options were replaced: %+v", in)
		}
	}
}

// Saving a dropdown's options keeps one per line, drops blanks, and survives a reload.
func TestSaveHandOptionsRoundTrips(t *testing.T) {
	c := Config{Out: t.TempDir()}
	if err := c.saveHandOption("Tipo_Mensaje (EBCDIC, 2 hex)", []string{"0 — Compra espera respuesta", "", "4 — Mensaje de Reversa"}); err != nil {
		t.Fatal(err)
	}
	got := loadHandOptions(c.handOptionsPath())["Tipo_Mensaje (EBCDIC, 2 hex)"]
	if len(got) != 2 || got[1] != "4 — Mensaje de Reversa" {
		t.Fatalf("saved: %v", got)
	}
	if err := c.saveHandOption("X", []string{" "}); err == nil {
		t.Fatal("empty options accepted")
	}
}

// An input without what it needs is flagged with the reason; a complete one is not.
func TestIncompleteInputsAreFlagged(t *testing.T) {
	if incompleteReason(Input{Field: "combobox"}) == "" {
		t.Fatal("a dropdown with no options is not flagged")
	}
	if incompleteReason(Input{Field: "textbox", Labelled: false}) == "" {
		t.Fatal("a value read as a name is not flagged")
	}
	if incompleteReason(Input{Field: "combobox", Options: []string{"LIVR"}}) != "" || incompleteReason(Input{Field: "textbox", Labelled: true}) != "" {
		t.Fatal("a complete input is flagged")
	}
}
