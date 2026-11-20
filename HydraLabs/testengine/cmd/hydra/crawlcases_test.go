package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestCrawlCasesWritesOneSmokeCasePerPageWithControls(t *testing.T) {
	dir := t.TempDir()
	c := Config{App: "PortalQA", Project: "PQA", AppMap: dir, Out: dir}
	crawl := `{"pages":[
	  {"url":"http://127.0.0.1:3000/cobro_ordenes","elements":[{"role":"button","name":"Procesar"},{"role":"tab","name":"Adquirente"},{"role":"link","name":"Home"}]},
	  {"url":"http://127.0.0.1:3000/","elements":[{"role":"link","name":"Home"}]},
	  {"url":"http://127.0.0.1:3000/cobro_ordenes?x=1","elements":[{"role":"button","name":"Procesar"}]}]}`
	os.WriteFile(c.crawlPath(), []byte(crawl), 0o644)
	out := filepath.Join(dir, "crawl-cases")
	n, err := c.CrawlCases(out)
	if err != nil || n != 1 {
		t.Fatalf("want 1 case (links-only and duplicate pages skipped), got %d %v", n, err)
	}
	b, _ := os.ReadFile(filepath.Join(out, "PQA-SMOKE-1.md"))
	s := string(b)
	for _, want := range []string{"Navigate to /cobro_ordenes.", `The page shows "Procesar".`, `The page shows "Adquirente".`} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
	}
	if !regexp.MustCompile(`^[A-Z][A-Z0-9]*-[A-Z0-9-]+$`).MatchString("PQA-SMOKE-1") {
		t.Error("case key does not match the case key rule")
	}
}

func TestCrawlCasesSkipsTheChromeOnEveryPage(t *testing.T) {
	dir := t.TempDir()
	c := Config{App: "PortalQA", Project: "PQA", AppMap: dir, Out: dir}
	crawl := `{"pages":[
	  {"url":"http://h/a","elements":[{"role":"button","name":"Modo oscuro"},{"role":"button","name":"Guardar"}]},
	  {"url":"http://h/b","elements":[{"role":"button","name":"Modo oscuro"}]}]}`
	os.WriteFile(c.crawlPath(), []byte(crawl), 0o644)
	out := filepath.Join(dir, "crawl-cases")
	c.CrawlCases(out)
	b, _ := os.ReadFile(filepath.Join(out, "PQA-SMOKE-1.md"))
	if strings.Contains(string(b), "Modo oscuro") || !strings.Contains(string(b), "Guardar") {
		t.Errorf("chrome was not skipped:\n%s", b)
	}
}
