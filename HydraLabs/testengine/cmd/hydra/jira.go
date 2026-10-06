package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

// Minimal Jira Cloud client: poll, never webhooks (Jira Cloud cannot call localhost).
// Auth: JIRA_BASE_URL, JIRA_EMAIL, JIRA_TOKEN (an API token).

type Issue struct {
	Key, Summary, HTML, Parent, Priority, Updated string
	Labels                                        []string
}

type Jira struct{ Base, Email, Token string }

func JiraFromEnv() (*Jira, error) {
	j := &Jira{os.Getenv("JIRA_BASE_URL"), os.Getenv("JIRA_EMAIL"), os.Getenv("JIRA_TOKEN")}
	if j.Base == "" || j.Email == "" || j.Token == "" {
		return nil, fmt.Errorf("set JIRA_BASE_URL, JIRA_EMAIL and JIRA_TOKEN")
	}
	return j, nil
}

func (j *Jira) do(method, path string, q url.Values, body any, out any) error {
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, j.Base+path+"?"+q.Encode(), r)
	req.SetBasicAuth(j.Email, j.Token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("jira %s %s: %d %.200s", method, path, resp.StatusCode, data)
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(data, out)
}

// Search returns the matching issues in full (one search with the fields we need; descriptions are fetched per issue).
func (j *Jira) Search(jql string) ([]Issue, error) {
	var res struct {
		Issues []struct{ Key string } `json:"issues"`
	}
	if err := j.do("GET", "/rest/api/3/search/jql", url.Values{"jql": {jql}, "fields": {"updated"}, "maxResults": {"100"}}, nil, &res); err != nil {
		return nil, err
	}
	var out []Issue
	for _, it := range res.Issues {
		full, err := j.Get(it.Key)
		if err != nil {
			return nil, err
		}
		out = append(out, *full)
	}
	return out, nil
}

func (j *Jira) Get(key string) (*Issue, error) {
	var r struct {
		Key    string
		Fields struct {
			Summary  string
			Updated  string
			Labels   []string
			Priority *struct{ Name string }
			Parent   *struct{ Key string }
		}
		RenderedFields struct{ Description string }
	}
	q := url.Values{"expand": {"renderedFields"}, "fields": {"summary,description,priority,parent,updated,labels"}}
	if err := j.do("GET", "/rest/api/3/issue/"+key, q, nil, &r); err != nil {
		return nil, err
	}
	is := &Issue{Key: r.Key, Summary: r.Fields.Summary, HTML: r.RenderedFields.Description, Updated: r.Fields.Updated, Labels: r.Fields.Labels}
	if r.Fields.Priority != nil {
		is.Priority = r.Fields.Priority.Name
	}
	if r.Fields.Parent != nil {
		is.Parent = r.Fields.Parent.Key
	}
	return is, nil
}

// Comment posts the run outcome. Off unless asked: write-back waits until the results are trusted.
func (j *Jira) Comment(key, msg string) error {
	adf := map[string]any{"body": map[string]any{"type": "doc", "version": 1, "content": []any{
		map[string]any{"type": "paragraph", "content": []any{map[string]any{"type": "text", "text": msg}}}}}}
	return j.do("POST", "/rest/api/3/issue/"+key+"/comment", url.Values{}, adf, nil)
}
