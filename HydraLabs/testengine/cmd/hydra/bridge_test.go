package main

import (
	"os"
	"path/filepath"
	"testing"
)

func el(role, name, loc string) CrawlElement {
	return CrawlElement{Tag: "input", Role: role, Name: name, Locator: loc, Stable: true}
}

func crawl(els ...CrawlElement) *Crawl {
	c := &Crawl{}
	c.Pages = append(c.Pages, struct {
		URL      string
		Elements []CrawlElement
	}{"/p", els})
	return c
}

func TestEnglishCaseWordsMatchSpanishPage(t *testing.T) {
	b := &Bridge{Crawl: crawl(
		el("textbox", "Ingresa el número de orden", "#orderNumber"),
		el("textbox", "Ingresa el nombre", "#nombre"),
		el("combobox", "Ingresa la tienda", "#name"))}
	for _, c := range []struct{ role, want, loc string }{
		{"textbox", "order number", "#orderNumber"},
		{"combobox", "store", "#name"},
	} {
		got, ok := b.Resolve(c.role, c.want)
		if !ok || got.Locator != c.loc {
			t.Errorf("%q -> %q ok=%v, want %s", c.want, got.Locator, ok, c.loc)
		}
	}
}

func TestUnsureStaysUnresolvedWithoutAModel(t *testing.T) {
	b := &Bridge{Crawl: crawl(el("textbox", "Orden A", "#a"), el("textbox", "Orden B", "#b"))}
	if _, ok := b.Resolve("textbox", "order"); ok {
		t.Error("two equally good candidates must not be guessed")
	}
	if _, ok := (&Bridge{Crawl: crawl(el("textbox", "Teléfono", "#t"))}).Resolve("textbox", "order number"); ok {
		t.Error("no shared word must not match")
	}
}

func TestModelOnlyChoosesFromTheOfferedList(t *testing.T) {
	cr := crawl(el("textbox", "Orden A", "#a"), el("textbox", "Orden B", "#b"))
	for _, c := range []struct {
		pick int
		want string
		ok   bool
	}{{1, "#b", true}, {-1, "", false}, {99, "", false}} {
		b := &Bridge{Crawl: cr, Pick: func(string, []CrawlElement) (int, int) { return c.pick, 77 }}
		got, ok := b.Resolve("textbox", "order")
		if ok != c.ok || got.Locator != c.want {
			t.Errorf("pick %d -> %q %v", c.pick, got.Locator, ok)
		}
		if b.Tokens != 77 {
			t.Errorf("tokens not counted: %d", b.Tokens)
		}
	}
}

func TestRunConfirmsGuessesAndMarksCustomWidgets(t *testing.T) {
	py := filepath.Join(t.TempDir(), "l.py")
	os.WriteFile(py, []byte("class L:\n    BOTON_AGREGAR = 'role=button[name=\"Agregar\"]'\n"+
		"    INPUT_ORDER_NUMBER = 'role=textbox[name=\"order number\" i]'  # TODO: confirmar el selector del campo 'order number'\n"+
		"    SELECT_STORE = 'role=combobox[name=\"store\" i]'  # TODO: confirmar 'store' y si va por value o por label\n    SPINNER = 'role=progressbar'\n"), 0o644)
	store := el("combobox", "Ingresa la tienda", "#name")
	store.Tag = "button" // a Radix combobox is a <button>, not a <select>
	b := &Bridge{Crawl: crawl(el("textbox", "Ingresa el número de orden", "#orderNumber"), store, el("button", "Agregar", `role=button[name="Agregar"]`))}
	got, left, err := b.Run(py, Confirmed{Locators: map[string]string{}})
	if err != nil || len(left) != 0 {
		t.Fatalf("%v %v", err, left)
	}
	if got.Locators["INPUT_ORDER_NUMBER"] != "#orderNumber" || got.Locators["SELECT_STORE"] != "#name" || got.Widgets["store"] != "combobox" {
		t.Errorf("%+v", got)
	}
	if _, touched := got.Locators["SPINNER"]; touched {
		t.Error("only INPUT_/SELECT_/BOTON_ guesses are the bridge's business")
	}
}

func TestHumanFileWinsOverLearned(t *testing.T) {
	dir := t.TempDir()
	c := Config{AppMap: dir, App: "App"}
	os.WriteFile(c.learnedPath(), []byte(`{"locators":{"X":"#learned","Y":"#y"}}`), 0o644)
	os.WriteFile(c.handPath(), []byte(`{"locators":{"X":"#human"}}`), 0o644)
	if got := c.confirmed().Locators; got["X"] != "#human" || got["Y"] != "#y" {
		t.Errorf("%v", got)
	}
}
