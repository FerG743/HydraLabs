package planner

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"uiagent/agent"
)

// Ollama is the local vision planner. It sends the current screenshot and the
// goal to a vision model running in Ollama (no network, no API key) and parses
// the returned action. Use a grounding-capable model such as "qwen3-vl:8b" or
// "qwen2.5vl:7b" — generic vision models are poor at pixel coordinates.
type Ollama struct {
	Model    string
	Endpoint string
	Client   *http.Client
}

func NewOllama(model string) *Ollama {
	return &Ollama{
		Model:    model,
		Endpoint: "http://localhost:11434/api/chat",
		Client:   http.DefaultClient,
	}
}

type chatReq struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
	Format   string        `json:"format,omitempty"`
}

type chatMessage struct {
	Role    string   `json:"role"`
	Content string   `json:"content"`
	Images  []string `json:"images,omitempty"` // base64, no data: prefix
}

type chatResp struct {
	Message chatMessage `json:"message"`
}

const systemPrompt = `You are a UI automation agent. You receive a screenshot of the current screen and a goal. Decide the single next action and respond with ONLY a JSON object, no prose:
{"kind":"click|type|key|scroll|done","x":<int>,"y":<int>,"text":"...","keys":"...","reason":"..."}
For "click", give the pixel x,y of the CENTER of the target element (top-left origin). For "type", set "text". For "key", set "keys" to something like "enter" or "cmd+s". Use "done" when the goal is achieved.`

func (o *Ollama) Next(ctx context.Context, goal agent.Goal, obs agent.Observation, history []agent.Step) (agent.Action, error) {
	user := fmt.Sprintf("Goal: %s\nScreen size: %dx%d pixels\nSteps taken so far: %d\nWhat is the next action?",
		goal.Task, obs.Width, obs.Height, len(history))

	msgs := []chatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: user, Images: []string{base64.StdEncoding.EncodeToString(obs.Screenshot)}},
	}
	body, err := json.Marshal(chatReq{Model: o.Model, Messages: msgs, Stream: false, Format: "json"})
	if err != nil {
		return agent.Action{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.Endpoint, bytes.NewReader(body))
	if err != nil {
		return agent.Action{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.Client.Do(req)
	if err != nil {
		return agent.Action{}, fmt.Errorf("ollama request (is `ollama serve` running?): %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return agent.Action{}, fmt.Errorf("ollama returned %s", resp.Status)
	}

	var cr chatResp
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return agent.Action{}, fmt.Errorf("decode ollama response: %w", err)
	}
	return ParseAction(cr.Message.Content)
}

// ParseAction turns the model's reply into an Action. Models stay chatty even
// when told not to, so we extract the first {...} block before parsing.
func ParseAction(s string) (agent.Action, error) {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '{'); i >= 0 {
		if j := strings.LastIndexByte(s, '}'); j > i {
			s = s[i : j+1]
		}
	}
	var raw struct {
		Kind   string `json:"kind"`
		Ref    string `json:"ref"`
		X      int    `json:"x"`
		Y      int    `json:"y"`
		Text   string `json:"text"`
		Keys   string `json:"keys"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(s), &raw); err != nil {
		return agent.Action{}, fmt.Errorf("model did not return valid action JSON: %w", err)
	}
	kind := agent.ActionKind(raw.Kind)
	switch kind {
	case agent.ActClick, agent.ActType, agent.ActKey, agent.ActScroll, agent.ActDone:
	default:
		return agent.Action{}, fmt.Errorf("unknown action kind %q", raw.Kind)
	}
	return agent.Action{
		Kind: kind, Ref: raw.Ref, X: raw.X, Y: raw.Y,
		Text: raw.Text, Keys: raw.Keys, Reason: raw.Reason,
	}, nil
}
