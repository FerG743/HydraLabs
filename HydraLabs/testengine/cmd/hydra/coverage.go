package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Coverage gaps: which pages and elements of the app map the built tests never reach. Poll writes the answer to
// out/coverage.json after every run, so the board and anyone else can read the gaps without running anything.

// coverageGaps refreshes out/coverage.json from the app map and every built bundle. Nothing to compare yet (no crawl)
// means nothing is written.
func (c Config) coverageGaps() error {
	return c.coverageGapsWith(filepath.Join(filepath.Dir(c.ToolsDir), "coverage.py"))
}

func (c Config) coverageGapsWith(tool string) error {
	crawl := c.crawlPath()
	if !fileExists(crawl) {
		return nil
	}
	bundles, _ := filepath.Glob(filepath.Join(c.Out, "bundles", "*"))
	argv := []string{tool}
	for _, b := range bundles {
		argv = append(argv, "--suite", b)
	}
	out := filepath.Join(c.Out, "coverage.json")
	os.MkdirAll(c.Out, 0o755)
	argv = append(argv, "--app", c.App, "--crawl", crawl, "--out", out)
	var errb bytes.Buffer
	cmd := exec.Command(c.Python, argv...)
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("coverage: %w: %s", err, bytes.TrimSpace(errb.Bytes()))
	}
	return nil
}

// coverageData is the last gap report, or an empty one before the first poll has written it.
func (c Config) coverageData() []byte {
	b, err := os.ReadFile(filepath.Join(c.Out, "coverage.json"))
	if err != nil {
		return []byte(`{"pages":[]}`)
	}
	return b
}
