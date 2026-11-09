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
		got, ok := b.Resolve(c.role, c.want, "")
		if !ok || got.Locator != c.loc {
			t.Errorf("%q -> %q ok=%v, want %s", c.want, got.Locator, ok, c.loc)
		}
	}
}

func TestUnsureStaysUnresolvedWithoutAModel(t *testing.T) {
	b := &Bridge{Crawl: crawl(el("textbox", "Orden A", "#a"), el("textbox", "Orden B", "#b"))}
	if _, ok := b.Resolve("textbox", "order", ""); ok {
		t.Error("two equally good candidates must not be guessed")
	}
	if _, ok := (&Bridge{Crawl: crawl(el("textbox", "Teléfono", "#t"))}).Resolve("textbox", "order number", ""); ok {
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
		got, ok := b.Resolve("textbox", "order", "")
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

func TestAClickFindsATabOrALinkNotOnlyAButton(t *testing.T) {
	cr := &Crawl{}
	cr.Pages = append(cr.Pages, struct {
		URL      string
		Elements []CrawlElement
	}{URL: "/SPDH", Elements: []CrawlElement{
		{Role: "tab", Name: "Adquirente", Locator: `role=tab[name="Adquirente"]`, Stable: true},
		{Role: "textbox", Name: "Adquirente id", Locator: "#adq", Stable: true}}})
	b := &Bridge{Crawl: cr}
	if el, ok := b.Resolve("button", "Adquirente", ""); !ok || el.Locator != `role=tab[name="Adquirente"]` {
		t.Errorf(`Click "Adquirente" must find the tab: %+v %v`, el, ok)
	}
}

func TestAClickIsNotResolvedToAButtonThatOnlyContainsItsWords(t *testing.T) {
	page := func(els ...CrawlElement) *Bridge {
		cr := &Crawl{}
		cr.Pages = append(cr.Pages, struct {
			URL      string
			Elements []CrawlElement
		}{Elements: els})
		return &Bridge{Crawl: cr}
	}
	other := CrawlElement{Role: "button", Name: "Enviar 1 Orden", Locator: `role=button[name="Enviar 1 Orden"]`, Stable: true}
	mine := CrawlElement{Role: "button", Name: "Enviar", Locator: `role=button[name="Enviar"]`, Stable: true}
	if el, ok := page(other).Resolve("button", "Enviar", ""); ok {
		t.Errorf(`"Enviar" alone must not become another button: %+v`, el)
	}
	if el, ok := page(other, mine).Resolve("button", "Enviar", ""); !ok || el.Locator != mine.Locator {
		t.Errorf("the exact name wins over a longer one: %+v %v", el, ok)
	}
	if el, ok := page(CrawlElement{Role: "button", Name: "Procesar", Locator: "#p", Stable: true}).Resolve("button", "Process", ""); !ok || el.Locator != "#p" {
		t.Errorf("one word for one word through the synonyms still resolves: %+v %v", el, ok)
	}
	field := CrawlElement{Role: "textbox", Name: "Ingresa el número de orden", Locator: "#orderNumber", Stable: true}
	if el, ok := page(field).Resolve("textbox", "order number", ""); !ok || el.Locator != "#orderNumber" {
		t.Errorf("a field's wordy label still matches: %+v %v", el, ok)
	}
}

func TestALocatorIsLookedForOnlyOnItsOwnPage(t *testing.T) {
	cr := &Crawl{}
	add := func(url string, els ...CrawlElement) {
		cr.Pages = append(cr.Pages, struct {
			URL      string
			Elements []CrawlElement
		}{url, els})
	}
	add("http://qa/Generacion_Ordenes_BT", CrawlElement{Role: "button", Name: "Enviar", Locator: "#orden-enviar", Stable: true})
	add("http://qa/SPDH", CrawlElement{Role: "button", Name: "Enviar", Locator: "#spdh-enviar", Stable: true})
	b := &Bridge{Crawl: cr}
	if el, ok := b.Resolve("button", "Enviar", "SPDH"); !ok || el.Locator != "#spdh-enviar" {
		t.Errorf("SPDH's Enviar: %+v %v", el, ok)
	}
	if _, ok := b.Resolve("button", "Enviar", ""); ok {
		t.Error("unscoped, two pages' Enviar are a tie, not a guess")
	}
	if _, ok := b.Resolve("button", "Enviar", "SOPORTE"); ok {
		t.Error("a page that does not have it does not borrow another page's")
	}
	// the renderer's page_token: slug(path).upper(), "INICIO" for the root
	for url, want := range map[string]string{"http://qa/SPDH?x=1": "SPDH", "http://qa/": "INICIO", "http://qa/cobro_ordenes": "COBRO_ORDENES",
		"http://qa/Información": "INFORMACION", "http://qa/Generacion_Ordenes_BT": "GENERACION_ORDENES_BT"} {
		if got := pageOf(url); got != want {
			t.Errorf("pageOf(%s) = %s, want %s", url, got, want)
		}
	}
}

func TestTheBridgeReadsThePageFromTheLocatorName(t *testing.T) {
	cr := crawl()
	cr.Pages[0].URL = "http://qa/SPDH"
	cr.Pages[0].Elements = []CrawlElement{{Role: "button", Name: "Enviar", Locator: `role=button[name="Enviar"]`, Stable: true},
		{Tag: "div", Role: "combobox", Name: "Mensaje", Locator: "#msg", Stable: true}}
	cr.Pages = append(cr.Pages, struct {
		URL      string
		Elements []CrawlElement
	}{"http://qa/otra", []CrawlElement{{Role: "button", Name: "Enviar", Locator: "#other", Stable: true}}})
	py := filepath.Join(t.TempDir(), "app_locators.py")
	os.WriteFile(py, []byte("class AppLocators:\n    SPDH__BOTON_ENVIAR = 'role=button[name=\"Enviar\"]'\n    SPDH__SELECT_MENSAJE = 'role=combobox[name=\"Mensaje\" i]'\n"), 0o644)
	found, left, err := (&Bridge{Crawl: cr}).Run(py, Confirmed{})
	if err != nil || len(left) != 0 || found.Locators["SPDH__BOTON_ENVIAR"] != `role=button[name="Enviar"]` || found.Locators["SPDH__SELECT_MENSAJE"] != "#msg" {
		t.Errorf("%+v left=%v %v", found, left, err)
	}
	if found.Widgets["mensaje"] != "combobox" {
		t.Errorf("a custom combobox is still flagged by its field name, page or not: %v", found.Widgets)
	}
}
