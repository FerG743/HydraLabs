package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"testengine/engine"
)

// httpRequestBlock makes a real HTTP call and writes the response into the
// context. SaveAs is an optional key prefix so two requests in one test don't
// clobber each other's results ("login" -> login_status/login_body).
type httpRequestBlock struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
	SaveAs  string            `json:"saveAs"`
}

func (b httpRequestBlock) statusKey() string {
	if b.SaveAs == "" {
		return "resp_status"
	}
	return b.SaveAs + "_status"
}

func (b httpRequestBlock) bodyKey() string {
	if b.SaveAs == "" {
		return "resp_body"
	}
	return b.SaveAs + "_body"
}

func (b httpRequestBlock) Execute(ctx context.Context, run *engine.RunContext) (engine.Status, error) {
	method := b.Method
	if method == "" {
		method = http.MethodGet
	}
	var body io.Reader
	if b.Body != "" {
		body = strings.NewReader(b.Body)
	}

	req, err := http.NewRequestWithContext(ctx, method, b.URL, body)
	if err != nil {
		// Malformed config is an ENGINE error: abort the run.
		return engine.StatusFailed, fmt.Errorf("build request: %w", err)
	}
	for k, v := range b.Headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// Transport failure (refused/timeout/DNS) is a TEST result, not a
		// crash: record it and let the graph branch on the failed edge.
		run.Logger.Warn("http request failed", "url", b.URL, "err", err)
		run.Set(b.statusKey(), 0)
		run.Set(b.bodyKey(), "")
		return engine.StatusFailed, nil
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	run.Set(b.statusKey(), resp.StatusCode)
	run.Set(b.bodyKey(), string(data))
	run.Logger.Info("http request", "method", method, "url", b.URL, "status", resp.StatusCode)

	// Got a response — that's a pass for the request itself. Whether the
	// status code is acceptable is the assert block's job.
	return engine.StatusPassed, nil
}

// assertBlock checks one thing about the context and passes or fails. Same
// philosophy as rules: a small typed set of kinds, not an expression language.
type assertBlock struct {
	Kind     string `json:"kind"`     // status_equals | body_contains | key_equals
	Key      string `json:"key"`      // context key (sensible default per kind)
	Expected any    `json:"expected"` // status_equals, key_equals
	Substr   string `json:"substr"`   // body_contains
}

func (b assertBlock) Execute(_ context.Context, run *engine.RunContext) (engine.Status, error) {
	switch b.Kind {
	case "status_equals":
		return b.equals(run, orDefault(b.Key, "resp_status"))
	case "key_equals":
		return b.equals(run, b.Key)
	case "body_contains":
		got, _ := run.Get(orDefault(b.Key, "resp_body"))
		s, _ := got.(string)
		if strings.Contains(s, b.Substr) {
			return engine.StatusPassed, nil
		}
		run.Logger.Warn("assert failed", "kind", b.Kind, "want_substr", b.Substr)
		return engine.StatusFailed, nil
	default:
		return engine.StatusFailed, fmt.Errorf("unknown assert kind %q", b.Kind)
	}
}

func (b assertBlock) equals(run *engine.RunContext, key string) (engine.Status, error) {
	got, _ := run.Get(key)
	// Same string-compare shortcut as rules — extract a shared typed compare
	// once this shows up a third time.
	if fmt.Sprint(got) == fmt.Sprint(b.Expected) {
		return engine.StatusPassed, nil
	}
	run.Logger.Warn("assert failed", "kind", b.Kind, "key", key, "got", got, "expected", b.Expected)
	return engine.StatusFailed, nil
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
