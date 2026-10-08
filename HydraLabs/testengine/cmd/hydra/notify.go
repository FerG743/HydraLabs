package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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
