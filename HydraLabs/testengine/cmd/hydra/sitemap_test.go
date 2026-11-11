package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSiteMapServesTheCrawlTheAppMapWasBuiltFrom(t *testing.T) {
	c := Config{App: "Shop", AppMap: t.TempDir()}
	if got := string(c.sitemapData()); got != `{"pages":[]}` {
		t.Fatalf("no crawl yet should give an empty map, got %s", got)
	}
	crawl := `{"baseUrl":"http://x/","pages":[{"url":"http://x/","title":"Home"}]}`
	os.WriteFile(filepath.Join(c.AppMap, "Shop.crawl.json"), []byte(crawl), 0o644)
	if got := string(c.sitemapData()); got != crawl {
		t.Fatalf("the map must be the crawl as written, got %s", got)
	}
}
