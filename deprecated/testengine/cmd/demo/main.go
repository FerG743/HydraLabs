package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"

	"testengine/engine"
)

// The example graph is compiled into the binary, so the demo runs from any
// directory. In the real app, graphs come from disk or the canvas via
// engine.LoadGraph.
//
//go:embed login_then_buy.json
var loginThenBuyJSON []byte

// ---- Example blocks. The app registers these; the engine stays generic. ----

type loginBlock struct {
	Username      string `json:"username"`
	ShouldSucceed bool   `json:"shouldSucceed"`
}

func (b loginBlock) Execute(_ context.Context, run *engine.RunContext) (engine.Status, error) {
	if !b.ShouldSucceed {
		run.Logger.Info("login failed", "user", b.Username)
		return engine.StatusFailed, nil
	}
	run.Set("auth_token", "tok_"+b.Username)
	run.Logger.Info("login ok", "user", b.Username)
	return engine.StatusPassed, nil
}

type buyBlock struct {
	Item string `json:"item"`
}

func (b buyBlock) Execute(_ context.Context, run *engine.RunContext) (engine.Status, error) {
	tok, _ := run.Get("auth_token") // guaranteed present: a rule gated this block
	run.Logger.Info("bought item", "item", b.Item, "auth", tok)
	return engine.StatusPassed, nil
}

type logBlock struct {
	Message string `json:"message"`
}

func (b logBlock) Execute(_ context.Context, run *engine.RunContext) (engine.Status, error) {
	run.Logger.Info("report", "message", b.Message)
	return engine.StatusPassed, nil
}

// unmarshalInto builds a block by unmarshaling config into a fresh value.
func unmarshalInto[T engine.Block](defaults T) func(json.RawMessage) (engine.Block, error) {
	return func(c json.RawMessage) (engine.Block, error) {
		b := defaults
		if len(c) > 0 {
			if err := json.Unmarshal(c, &b); err != nil {
				return nil, err
			}
		}
		return b, nil
	}
}

func registry() *engine.Registry {
	r := engine.NewRegistry()

	r.Register(engine.Factory{
		Meta: engine.BlockMeta{Type: "login", Writes: []string{"auth_token"}},
		New:  unmarshalInto[loginBlock](loginBlock{ShouldSucceed: true}),
	})
	r.Register(engine.Factory{
		Meta: engine.BlockMeta{
			Type:  "buy",
			Reads: []string{"auth_token"},
			DefaultRules: []engine.Rule{
				{Kind: engine.RuleRequiresKey, Key: "auth_token"},
				{Kind: engine.RuleRequiresBlockUpstream, Block: "login"},
			},
		},
		New: unmarshalInto[buyBlock](buyBlock{}),
	})
	r.Register(engine.Factory{
		Meta: engine.BlockMeta{Type: "log"},
		New:  unmarshalInto[logBlock](logBlock{}),
	})
	r.Register(engine.Factory{
		Meta: engine.BlockMeta{
			Type:   "http_request",
			Writes: []string{"resp_status", "resp_body"},
		},
		New: unmarshalInto[httpRequestBlock](httpRequestBlock{}),
	})
	r.Register(engine.Factory{
		Meta: engine.BlockMeta{
			Type:  "assert",
			Reads: []string{"resp_status", "resp_body"},
		},
		New: unmarshalInto[assertBlock](assertBlock{}),
	})
	r.Register(loadFactory(r))
	return r
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	reg := registry()
	in := engine.NewInterpreter(reg)

	// --- Scenario 1: valid graph, login runs before buy ---
	fmt.Println("=== Scenario 1: login -> buy (valid) ===")
	g1, err := engine.LoadGraph(loginThenBuyJSON)
	if err != nil {
		panic(err)
	}
	reportValidation(g1, reg)
	res1, err := in.Run(context.Background(), g1, engine.NewRunContext("run-1", logger))
	if err != nil {
		fmt.Println("  run error:", err)
	}
	printSteps(res1)

	// --- Scenario 2: buy with NO login upstream ---
	// Structural rule catches it at design time; if you run it anyway the
	// runtime requires_key rule blocks the buy.
	fmt.Println("\n=== Scenario 2: buy with no login (invalid) ===")
	g2, err := engine.LoadGraph([]byte(`{
		"start": "n_buy",
		"nodes": [
			{"id": "n_buy", "type": "buy", "config": {"item": "widget"},
			 "rules": [
			   {"kind": "requires_key", "key": "auth_token"},
			   {"kind": "requires_block_upstream", "block": "login"}
			 ]},
			{"id": "n_report", "type": "log", "config": {"message": "done"}}
		],
		"edges": [{"from": "n_buy", "to": "n_report"}]
	}`))
	if err != nil {
		panic(err)
	}
	reportValidation(g2, reg)
	fmt.Println("  running anyway to show the runtime guard:")
	res2, _ := in.Run(context.Background(), g2, engine.NewRunContext("run-2", logger))
	printSteps(res2)

	// --- Scenario 3: a real HTTP request + assert against a local server ---
	fmt.Println("\n=== Scenario 3: http request -> assert (live round-trip) ===")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"ok":true}`)
	}))
	defer srv.Close()

	g3, err := engine.LoadGraph([]byte(fmt.Sprintf(`{
		"start": "n_req",
		"nodes": [
			{"id": "n_req", "type": "http_request", "config": {"method": "GET", "url": %q}},
			{"id": "n_assert", "type": "assert",
			 "config": {"kind": "status_equals", "expected": 200},
			 "rules": [{"kind": "requires_key", "key": "resp_status"}]},
			{"id": "n_ok",  "type": "log", "config": {"message": "endpoint healthy"}},
			{"id": "n_bad", "type": "log", "config": {"message": "endpoint NOT healthy"}}
		],
		"edges": [
			{"from": "n_req",    "to": "n_assert", "on": "passed"},
			{"from": "n_req",    "to": "n_bad",    "on": "failed"},
			{"from": "n_assert", "to": "n_ok",     "on": "passed"},
			{"from": "n_assert", "to": "n_bad",    "on": "failed"}
		]
	}`, srv.URL)))
	if err != nil {
		panic(err)
	}
	reportValidation(g3, reg)
	res3, _ := in.Run(context.Background(), g3, engine.NewRunContext("run-3", logger))
	printSteps(res3)

	// --- Scenario 4: a load block — 20 workers × 50 iterations = 1000 runs ---
	fmt.Println("\n=== Scenario 4: load (20 x 50) -> assert zero failures ===")
	loadSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"ok":true}`)
	}))
	defer loadSrv.Close()

	g4, err := engine.LoadGraph([]byte(fmt.Sprintf(`{
		"start": "n_load",
		"nodes": [
			{"id": "n_load", "type": "load", "config": {
				"workers": 20, "iterations": 50,
				"subgraph": {
					"start": "s_req",
					"nodes": [
						{"id": "s_req", "type": "http_request", "config": {"url": %q}},
						{"id": "s_assert", "type": "assert",
						 "config": {"kind": "status_equals", "expected": 200},
						 "rules": [{"kind": "requires_key", "key": "resp_status"}]}
					],
					"edges": [{"from": "s_req", "to": "s_assert", "on": "passed"}]
				}
			}},
			{"id": "n_check", "type": "assert", "config": {"kind": "key_equals", "key": "load_failed", "expected": 0}},
			{"id": "n_ok",  "type": "log", "config": {"message": "load passed: zero failures"}},
			{"id": "n_bad", "type": "log", "config": {"message": "load had failures"}}
		],
		"edges": [
			{"from": "n_load",  "to": "n_check", "on": "passed"},
			{"from": "n_check", "to": "n_ok",    "on": "passed"},
			{"from": "n_check", "to": "n_bad",   "on": "failed"}
		]
	}`, loadSrv.URL)))
	if err != nil {
		panic(err)
	}
	reportValidation(g4, reg)
	res4, _ := in.Run(context.Background(), g4, engine.NewRunContext("run-4", logger))
	printSteps(res4)
}

func reportValidation(g *engine.Graph, reg *engine.Registry) {
	errs := g.Validate(reg)
	if len(errs) == 0 {
		fmt.Println("  validation: ok")
		return
	}
	for _, e := range errs {
		fmt.Println("  validation error:", e)
	}
}

func printSteps(res *engine.RunResult) {
	for _, s := range res.Steps {
		line := fmt.Sprintf("  step %-9s -> %s", s.NodeID, s.Status)
		if s.Reason != "" {
			line += " (" + s.Reason + ")"
		}
		fmt.Println(line)
	}
}
