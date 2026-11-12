package main

import (
	"fmt"
	"os"
)

// HydraMaps: everything derived from the app map (the crawl). refreshMaps is the one place that writes the derived files
// (out/coverage.json and out/routes.json); poll calls it after each run, and the board and the CLI only read them.
// A failure is said, never a failed poll: the maps are a by-product of the run.
func (c Config) refreshMaps() {
	if e := c.coverageGaps(); e != nil {
		fmt.Fprintln(os.Stderr, "coverage gaps not updated:", e)
	}
	if e := c.buildRoutes(); e != nil {
		fmt.Fprintln(os.Stderr, "routes not updated:", e)
	}
}
