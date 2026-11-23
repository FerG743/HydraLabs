package main

import (
	"strings"
	"testing"
)

// Each sweep in the profile reaches the crawler as its own --sweep argument.
func TestCrawlArgsPassEverySweep(t *testing.T) {
	c := Config{Crawler: "crawl.py", PortalURL: "http://x", Sweeps: []string{"/SPDH=Mensaje", "/Otra=Control"}}
	args := strings.Join(c.crawlArgs("ev.jsonl", "out.json", false), " ")
	if !strings.Contains(args, "--sweep /SPDH=Mensaje") || !strings.Contains(args, "--sweep /Otra=Control") {
		t.Fatalf("sweeps not passed: %s", args)
	}
}
