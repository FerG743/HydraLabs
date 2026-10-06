package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// The crawl bridge (the Plan stage): the renderer guesses locators from the case's wording
// (role=textbox[name="order number" i]); the crawler recorded what the page really has. This matches one to the other.
//
//	1. rules   accent/case-insensitive token match through a bilingual synonym table (order~orden, store~tienda ...)
//	2. model   only for what rules cannot settle: pick one of the candidate elements from a closed list (T2, ~100 tokens)
//
// Every answer is cached in <appmap>/<App>.learned.json, so an element is resolved once per app, ever.
// A human-written <App>.json always wins over the learned one.

type CrawlElement struct {
	Tag, Role, Name, ID, Locator, Strategy string
	Placeholder                            string
	Stable                                 bool
	Options                                []string
}
type Crawl struct {
	Pages []struct {
		URL      string
		Elements []CrawlElement
	}
}
type Confirmed struct {
	Locators map[string]string `json:"locators"`
	Widgets  map[string]string `json:"widgets"`
}

var syn = map[string][]string{
	"order": {"orden", "pedido"}, "number": {"numero", "num", "folio", "no"}, "store": {"tienda", "sucursal"}, "name": {"nombre"},
	"user": {"usuario"}, "password": {"contrasena", "clave"}, "email": {"correo", "mail"}, "search": {"buscar", "busqueda"},
	"date": {"fecha"}, "amount": {"monto", "importe"}, "customer": {"cliente"}, "product": {"producto", "articulo"},
	"quantity": {"cantidad"}, "address": {"direccion"}, "phone": {"telefono", "celular"}, "submit": {"enviar"},
	"save": {"guardar"}, "cancel": {"cancelar"}, "add": {"agregar", "anadir"}, "delete": {"eliminar", "borrar"},
	"process": {"procesar"}, "upload": {"subir", "cargar"}, "download": {"descargar"}, "file": {"archivo"},
	"code": {"codigo"}, "card": {"tarjeta"}, "payment": {"pago", "cobro"}, "total": {"total", "importe"},
	"close": {"cerrar"}, "next": {"siguiente"}, "previous": {"anterior"}, "login": {"entrar", "acceder", "iniciar"},
}

// words that carry no identity: verbs in field prompts ("Ingresa la tienda") and articles
var stop = map[string]bool{"the": true, "a": true, "an": true, "el": true, "la": true, "los": true, "las": true, "de": true, "del": true,
	"un": true, "una": true, "enter": true, "select": true, "choose": true, "field": true, "ingresa": true, "ingrese": true,
	"selecciona": true, "seleccione": true, "captura": true, "escribe": true, "i": true}

func fold(s string) string { // lowercase, accents removed
	var b strings.Builder
	for _, r := range norm.NFD.String(strings.ToLower(s)) {
		if !unicode.Is(unicode.Mn, r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

var wordRe = regexp.MustCompile(`[a-z0-9]+`)

func tokens(s string) []string {
	var out []string
	for _, w := range wordRe.FindAllString(fold(s), -1) {
		if !stop[w] {
			out = append(out, strings.TrimSuffix(w, "s"))
		}
	}
	return out
}

func (b *Bridge) variants(w string) map[string]bool {
	v := map[string]bool{w: true}
	for _, x := range append(syn[w], b.Extra[w]...) {
		v[strings.TrimSuffix(fold(x), "s")] = true
	}
	return v
}

type Bridge struct {
	Crawl  *Crawl
	Extra  map[string][]string                                // per-app synonyms from the profile
	Pick   func(want string, cands []CrawlElement) (int, int) // optional model: (index or -1, tokens spent)
	Tokens int
}

// score = how many of the wanted words the element's name contains (synonyms allowed); full match = all of them.
func (b *Bridge) score(want string, el CrawlElement) (int, int) {
	w := tokens(want)
	have := map[string]bool{}
	for _, t := range tokens(el.Name + " " + el.Placeholder) {
		have[t] = true
	}
	got := 0
	for _, x := range w {
		for v := range b.variants(x) {
			if have[v] {
				got++
				break
			}
		}
	}
	return got, len(w)
}

var guess = regexp.MustCompile(`^role=(\w+)\[name="(.*?)"( i)?\]$`)

// non-greedy, then an optional trailing comment: the comment may itself contain quotes ("# TODO: ... 'order number'")
var locLine = regexp.MustCompile(`^\s{4}([A-Z][A-Z0-9_]*) = (?:'(.*?)'|"(.*?)")(?:\s+#.*)?$`)
var roleOK = map[string][]string{"textbox": {"textbox", "spinbutton"}, "combobox": {"combobox"}, "button": {"button"}}

func inRoles(role string, el CrawlElement) bool {
	for _, r := range roleOK[role] {
		if el.Role == r {
			return true
		}
	}
	return false
}

// Resolve settles one guessed locator. ok=false means "leave the guess".
func (b *Bridge) Resolve(role, want string) (el CrawlElement, ok bool) {
	var full []CrawlElement
	var near []CrawlElement
	seen := map[string]bool{}
	for _, pg := range b.Crawl.Pages {
		for _, e := range pg.Elements {
			if !inRoles(role, e) || !e.Stable || seen[e.Locator] {
				continue
			}
			g, n := b.score(want, e)
			if n > 0 && g == n {
				full = append(full, e)
				seen[e.Locator] = true
			} else if g > 0 {
				near = append(near, e)
			}
		}
	}
	if len(full) == 1 {
		return full[0], true
	}
	cands := full // several equally good, or none: let the model choose among them (or among the near ones)
	if len(cands) == 0 {
		cands = near
	}
	if b.Pick == nil || len(cands) == 0 {
		return el, false
	}
	if len(cands) > 12 {
		cands = cands[:12]
	}
	i, spent := b.Pick(want, cands)
	b.Tokens += spent
	if i < 0 || i >= len(cands) {
		return el, false
	}
	return cands[i], true
}

// Bridge reads the generated locators file, resolves every guess it can, and returns the new confirmations.
func (b *Bridge) Run(locatorsPy string, already Confirmed) (Confirmed, []string, error) {
	src, err := os.ReadFile(locatorsPy)
	if err != nil {
		return Confirmed{}, nil, err
	}
	out := Confirmed{Locators: map[string]string{}, Widgets: map[string]string{}}
	var left []string
	for _, line := range strings.Split(string(src), "\n") {
		m := locLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name, val := m[1], m[2]+m[3]
		if _, done := already.Locators[name]; done {
			continue
		}
		g := guess.FindStringSubmatch(strings.ReplaceAll(val, `\"`, `"`))
		if g == nil || !(strings.HasPrefix(name, "INPUT_") || strings.HasPrefix(name, "SELECT_") || strings.HasPrefix(name, "BOTON_")) {
			continue
		}
		el, ok := b.Resolve(g[1], g[2])
		if !ok {
			left = append(left, fmt.Sprintf("%s (%s %q)", name, g[1], g[2]))
			continue
		}
		out.Locators[name] = el.Locator
		if strings.HasPrefix(name, "SELECT_") && el.Tag != "select" {
			out.Widgets[strings.ToLower(strings.TrimPrefix(name, "SELECT_"))] = "combobox" // custom widget: open, then pick an option
		}
	}
	sort.Strings(left)
	return out, left, nil
}

func loadConfirmed(path string) Confirmed {
	c := Confirmed{Locators: map[string]string{}, Widgets: map[string]string{}}
	if b, err := os.ReadFile(path); err == nil {
		json.Unmarshal(b, &c)
		if c.Locators == nil {
			c.Locators = map[string]string{}
		}
		if c.Widgets == nil {
			c.Widgets = map[string]string{}
		}
	}
	return c
}

func (a Confirmed) merge(b Confirmed) Confirmed { // b wins
	for k, v := range b.Locators {
		a.Locators[k] = v
	}
	for k, v := range b.Widgets {
		a.Widgets[k] = v
	}
	return a
}

func (c Config) crawlPath() string   { return filepath.Join(c.AppMap, c.App+".crawl.json") }
func (c Config) learnedPath() string { return filepath.Join(c.AppMap, c.App+".learned.json") }
func (c Config) handPath() string    { return filepath.Join(c.AppMap, c.App+".json") }

// confirmed = learned, overridden by what a human wrote.
func (c Config) confirmed() Confirmed {
	return loadConfirmed(c.learnedPath()).merge(loadConfirmed(c.handPath()))
}

// bridge learns from the crawl; returns whether anything new was confirmed, the tokens the model spent, and what is left.
func (c Config) bridge(locatorsPy string) (changed bool, tokens int, left []string, err error) {
	raw, e := os.ReadFile(c.crawlPath())
	if e != nil {
		return false, 0, nil, nil // no crawl yet: the guesses stand, T3 can still explore
	}
	cr := &Crawl{}
	if err := json.Unmarshal(raw, cr); err != nil {
		return false, 0, nil, fmt.Errorf("%s: %w", c.crawlPath(), err)
	}
	b := &Bridge{Crawl: cr, Extra: c.Synonyms}
	if c.Model != "" && c.LMURL != "" {
		b.Pick = c.modelPick
	}
	learned := loadConfirmed(c.learnedPath())
	found, left, err := b.Run(locatorsPy, c.confirmed())
	if err != nil || len(found.Locators) == 0 {
		return false, b.Tokens, left, err
	}
	return true, b.Tokens, left, writeJSON(c.learnedPath(), learned.merge(found))
}

// modelPick is T2: one closed-choice question. The model can only answer with an index we offered, or -1.
func (c Config) modelPick(want string, cands []CrawlElement) (int, int) {
	type opt struct {
		I    int    `json:"i"`
		Role string `json:"role"`
		Name string `json:"name"`
	}
	var opts []opt
	for i, e := range cands {
		opts = append(opts, opt{i, e.Role, e.Name})
	}
	q, _ := json.Marshal(map[string]any{"element_in_test_case": want, "candidates": opts})
	body, _ := json.Marshal(map[string]any{"model": c.Model, "temperature": 0, "max_tokens": 20, "reasoning_effort": "none",
		"messages": []map[string]string{
			{"role": "system", "content": `A web test step names a UI element (possibly in another language than the page). Pick the candidate that is the same element. Answer only JSON: {"i": <index>} or {"i": -1} if none is clearly the same.`},
			{"role": "user", "content": string(q)}}})
	resp, err := (&http.Client{Timeout: 120 * time.Second}).Post(c.LMURL+"/chat/completions", "application/json", bytes.NewReader(body))
	if err != nil {
		return -1, 0
	}
	defer resp.Body.Close()
	var r struct {
		Choices []struct{ Message struct{ Content string } }
		Usage   struct {
			Total int `json:"total_tokens"`
		}
	}
	if json.NewDecoder(resp.Body).Decode(&r) != nil || len(r.Choices) == 0 {
		return -1, 0
	}
	var a struct{ I *int }
	m := regexp.MustCompile(`\{[^}]*\}`).FindString(r.Choices[0].Message.Content)
	if json.Unmarshal([]byte(m), &a) != nil || a.I == nil {
		return -1, r.Usage.Total // unparseable answer = no answer; never guess
	}
	return *a.I, r.Usage.Total
}
