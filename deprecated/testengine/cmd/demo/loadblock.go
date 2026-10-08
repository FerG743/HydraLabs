package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"sync"
	"time"

	"testengine/engine"
)

// loadBlock is a container: it holds a subgraph and runs it Workers×Iterations
// times. Each run gets its own cloned RunContext so the virtual users never
// share state. It reuses the ordinary engine.Interpreter as a sub-interpreter —
// the container adds concurrency, not a second execution model.
type loadBlock struct {
	Workers    int          `json:"workers"`    // concurrent virtual users
	Iterations int          `json:"iterations"` // runs per worker
	Subgraph   engine.Graph `json:"subgraph"`   // what each VU runs
	SaveAs     string       `json:"saveAs"`     // key prefix for stats (default "load")

	reg *engine.Registry // injected by the factory, not from JSON
}

func (b loadBlock) Execute(ctx context.Context, run *engine.RunContext) (engine.Status, error) {
	workers := max(b.Workers, 1)
	iters := max(b.Iterations, 1)

	// Validate the subgraph once, up front, before spawning any workers.
	sub := b.Subgraph
	if errs := sub.Validate(b.reg); len(errs) > 0 {
		return engine.StatusFailed, fmt.Errorf("invalid subgraph: %v", errs[0])
	}

	subInt := engine.NewInterpreter(b.reg)
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil)) // sub-runs don't spam the log

	type agg struct {
		passed, failed int
		lat            []time.Duration
	}
	perWorker := make([]agg, workers)

	var wg sync.WaitGroup
	start := time.Now()
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			a := agg{lat: make([]time.Duration, 0, iters)}
			for i := 0; i < iters; i++ {
				if ctx.Err() != nil {
					break // run cancelled / timed out
				}
				child := run.Clone(fmt.Sprintf("%s-w%d-i%d", run.RunID, w, i))
				child.Logger = quiet

				t0 := time.Now()
				res, err := subInt.Run(ctx, &sub, child)
				a.lat = append(a.lat, time.Since(t0))

				if err == nil && !runFailed(res) {
					a.passed++
				} else {
					a.failed++
				}
			}
			perWorker[w] = a
		}(w)
	}
	wg.Wait()
	elapsed := time.Since(start)

	var passed, failed int
	var all []time.Duration
	for _, a := range perWorker {
		passed += a.passed
		failed += a.failed
		all = append(all, a.lat...)
	}
	total := passed + failed
	p50 := percentile(all, 50)
	p95 := percentile(all, 95)

	prefix := b.SaveAs
	if prefix == "" {
		prefix = "load"
	}
	run.Set(prefix+"_total", total)
	run.Set(prefix+"_passed", passed)
	run.Set(prefix+"_failed", failed)
	run.Set(prefix+"_p50_ms", p50.Milliseconds())
	run.Set(prefix+"_p95_ms", p95.Milliseconds())

	run.Logger.Info("load complete",
		"workers", workers, "iterations", iters,
		"total", total, "passed", passed, "failed", failed,
		"p50_ms", p50.Milliseconds(), "p95_ms", p95.Milliseconds(),
		"wall_ms", elapsed.Milliseconds())

	// It ran. Whether the failure count or latency is acceptable is an assert
	// block's decision — the stats are now in the parent context for one to read.
	return engine.StatusPassed, nil
}

// runFailed treats an iteration as failed if any step errored, failed, or was
// blocked — so a request that branched to a recovery path still counts against
// the load test. Make this configurable later if some subgraphs want laxer rules.
func runFailed(res *engine.RunResult) bool {
	for _, s := range res.Steps {
		if s.Err != nil || s.Status == engine.StatusFailed || s.Status == engine.StatusBlocked {
			return true
		}
	}
	return len(res.Steps) == 0
}

func percentile(d []time.Duration, p int) time.Duration {
	if len(d) == 0 {
		return 0
	}
	s := make([]time.Duration, len(d))
	copy(s, d)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	return s[(p*(len(s)-1))/100]
}

func loadFactory(reg *engine.Registry) engine.Factory {
	return engine.Factory{
		Meta: engine.BlockMeta{
			Type:   "load",
			Writes: []string{"load_total", "load_passed", "load_failed", "load_p50_ms", "load_p95_ms"},
		},
		New: func(c json.RawMessage) (engine.Block, error) {
			var b loadBlock
			if len(c) > 0 {
				if err := json.Unmarshal(c, &b); err != nil {
					return nil, err
				}
			}
			b.reg = reg
			return b, nil
		},
	}
}
