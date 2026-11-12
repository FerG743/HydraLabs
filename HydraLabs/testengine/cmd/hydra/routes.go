package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Routes: the fewest clicks from the start page to every crawled page, written to out/routes.json by poll, so a route
// can be replayed or read without running anything. The builder is tools/routes.py.

// buildRoutes refreshes out/routes.json from the app map. No crawl yet means nothing is written.
func (c Config) buildRoutes() error {
	return c.buildRoutesWith(filepath.Join(filepath.Dir(c.ToolsDir), "routes.py"))
}

func (c Config) buildRoutesWith(tool string) error {
	crawl := c.crawlPath()
	if !fileExists(crawl) {
		return nil
	}
	out := filepath.Join(c.Out, "routes.json")
	os.MkdirAll(c.Out, 0o755)
	var errb bytes.Buffer
	cmd := exec.Command(c.Python, tool, "--crawl", crawl, "--app", c.App, "--out", out)
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("routes: %w: %s", err, bytes.TrimSpace(errb.Bytes()))
	}
	return nil
}
