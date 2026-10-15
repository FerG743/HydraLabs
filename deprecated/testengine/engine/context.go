package engine

import (
	"log/slog"
	"maps"
	"sync"
)

// RunContext is the per-run state that flows through the graph: a key/value bag
// blocks read and write (the "hybrid" model), plus a logger and run metadata.
// It is safe for concurrent use, but the important isolation comes from Clone:
// the parallel container blocks give each virtual user its own cloned context
// so workers never stomp on each other's state.
type RunContext struct {
	mu     sync.RWMutex
	bag    map[string]any
	Logger *slog.Logger
	RunID  string
}

func NewRunContext(runID string, logger *slog.Logger) *RunContext {
	if logger == nil {
		logger = slog.Default()
	}
	return &RunContext{
		bag:    make(map[string]any),
		Logger: logger,
		RunID:  runID,
	}
}

func (c *RunContext) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.bag[key]
	return v, ok
}

func (c *RunContext) Has(key string) bool {
	_, ok := c.Get(key)
	return ok
}

func (c *RunContext) Set(key string, val any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bag[key] = val
}

// Clone returns an isolated copy for a single virtual user. The map itself is
// copied so workers never share one bag; values are copied by reference, which
// is fine for the scalars, strings, and ids tests usually pass around. If you
// ever store a shared mutable struct in the bag, deep-copy it here.
func (c *RunContext) Clone(runID string) *RunContext {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return &RunContext{
		bag:    maps.Clone(c.bag),
		Logger: c.Logger,
		RunID:  runID,
	}
}
