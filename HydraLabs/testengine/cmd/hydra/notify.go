package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Outcomes that need a person get one line in Jira (the system of record) and, when a webhook is configured,
// in the automators' Google Chat space. The webhook URL is a secret: it lives in an env var, never in a profile.

func needsPerson(r Run) bool {
	return r.Status == "needs-clarification" || r.Status == "needs-review" || r.Status == "failed" || r.Status == "needs-writes" || r.Status == "env-down" || r.Status == "needs-onboarding"
}

func Message(r Run) string {
	return fmt.Sprintf("HydraPloy [%s / %s]: %s.\n%s", r.Intent, r.Tier, r.Status, strings.Join(r.Detail, "\n"))
}

func ChatNotify(envName string, r Run) error {
	if envName == "" {
		envName = "HYDRA_CHAT_WEBHOOK"
	}
	hook := os.Getenv(envName)
	if hook == "" || !needsPerson(r) {
		return nil
	}
	b, _ := json.Marshal(map[string]string{"text": fmt.Sprintf("*%s* %s", r.Key, Message(r))})
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Post(hook, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("chat webhook: %d", resp.StatusCode)
	}
	return nil
}

// DeliveredMessage is what the issue's owner reads when the case is finished: the result, the cost, and where the test is.
func DeliveredMessage(r Run, d Delivery, repoName, testFile string, runs int) string {
	msg := fmt.Sprintf("HydraPloy: automated and delivered.\n"+
		"Result: passed %d of %d consecutive runs in the framework.\n"+
		"Test: %s\n"+
		"Repo: %s, branch %s, commit %s.\n"+
		"Cost: %d tokens (%s).",
		runs, runs, testFile, repoName, d.Branch, d.Commit, r.Tokens, r.Tier)
	if strings.HasPrefix(d.Branch, "hydra/") { // a review branch: not merged yet
		msg += "\nA person reviews the branch before it is merged."
	}
	return msg
}

// BlockedMessage: a delivery was stopped on this case (flaky or failing in the real framework).
func BlockedMessage(o CaseOutcome, runs int) string {
	return fmt.Sprintf("HydraPloy: delivery blocked, needs a person.\nStatus: %s (passed %d of %d runs).\n%s", o.Status, o.Passes, runs, o.Detail)
}

// PostOnce posts msg to the issue unless this exact news (id) was already posted: re-running a case must not
// spam the board. The record lives in out/hydra/posted.json, so deleting it resets the memory.
func (c Config) PostOnce(j *Jira, key, id, msg string) (bool, error) {
	path := filepath.Join(c.Out, "posted.json")
	posted := map[string]string{}
	if b, err := os.ReadFile(path); err == nil {
		json.Unmarshal(b, &posted)
	}
	if posted[key] == id {
		return false, nil
	}
	if err := j.Comment(key, msg); err != nil {
		return false, err
	}
	posted[key] = id
	return true, writeJSON(path, posted)
}

func shortID(parts ...string) string {
	h := sha1.Sum([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(h[:6])
}
