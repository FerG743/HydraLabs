package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCommentBecomesOneParagraphPerLine(t *testing.T) {
	body := CommentADF("first line\n\n  second line  \nthird")["body"].(map[string]any)
	paras := body["content"].([]any)
	if len(paras) != 3 {
		t.Fatalf("blank lines are dropped and each line is a paragraph: got %d", len(paras))
	}
	b, _ := json.Marshal(paras[1])
	if !strings.Contains(string(b), `"text":"second line"`) {
		t.Errorf("lines are trimmed: %s", b)
	}
}

func TestDeliveredMessageSaysWhatHappenedAndWhereTheTestIs(t *testing.T) {
	m := DeliveredMessage(Run{Tier: "T1-render", Tokens: 0}, Delivery{Branch: "hydra/app-1", Commit: "abc1234"}, "Hydra-App", "tests/App/test_abc_1.py", 3)
	for _, want := range []string{"passed 3 of 3", "tests/App/test_abc_1.py", "Hydra-App", "abc1234", "0 tokens (T1-render)", "reviews the branch"} {
		if !strings.Contains(m, want) {
			t.Errorf("missing %q in:\n%s", want, m)
		}
	}
	if strings.Contains(DeliveredMessage(Run{}, Delivery{Branch: "main", Commit: "x"}, "R", "t.py", 3), "reviews the branch") {
		t.Error("a backfill for a case already on main must not say it awaits review")
	}
}

func TestSameNewsIsNeverPostedTwice(t *testing.T) {
	var posts []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		posts = append(posts, r.URL.Path+" "+string(b))
		w.WriteHeader(201)
	}))
	defer srv.Close()
	c, j := Config{Out: t.TempDir()}, &Jira{Base: srv.URL, Email: "a@b", Token: "t"}

	if ok, err := c.PostOnce(j, "ABC-1", "commit-1", "hello"); !ok || err != nil {
		t.Fatalf("first post: %v %v", ok, err)
	}
	if ok, _ := c.PostOnce(j, "ABC-1", "commit-1", "hello"); ok {
		t.Error("same id must not post again")
	}
	if ok, _ := c.PostOnce(j, "ABC-1", "commit-2", "new commit"); !ok {
		t.Error("new news must post")
	}
	if ok, _ := c.PostOnce(j, "ABC-2", "commit-1", "other issue"); !ok {
		t.Error("another issue has its own memory")
	}
	if len(posts) != 3 || !strings.HasPrefix(posts[0], "/rest/api/3/issue/ABC-1/comment ") {
		t.Errorf("%v", posts)
	}
}

func TestAFailedPostIsNotRemembered(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	defer srv.Close()
	c, j := Config{Out: t.TempDir()}, &Jira{Base: srv.URL, Email: "a@b", Token: "t"}
	if ok, err := c.PostOnce(j, "ABC-1", "x", "m"); ok || err == nil {
		t.Fatalf("a 500 must be an error: %v %v", ok, err)
	}
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(201) }))
	defer srv2.Close()
	j.Base = srv2.URL
	if ok, _ := c.PostOnce(j, "ABC-1", "x", "m"); !ok {
		t.Error("after a failure the same news must be retried, not skipped")
	}
}

func TestWriteBackIsOffUnlessAsked(t *testing.T) {
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hit = true }))
	defer srv.Close()
	t.Setenv("JIRA_BASE_URL", srv.URL)
	t.Setenv("JIRA_EMAIL", "a@b")
	t.Setenv("JIRA_TOKEN", "t")
	Config{Out: t.TempDir(), JiraComments: false}.writeBack(Delivery{Delivered: []string{"ABC-1"}, Commit: "x"})
	if hit {
		t.Error("jiraComments is false: nothing may be sent")
	}
}

func TestOnlyIssuesOfTheJiraProjectGetComments(t *testing.T) {
	c := Config{Project: "PQA", JiraProject: "DWQ"}
	if !c.inJira("DWQ-139") || c.inJira("PQA-3") || c.inJira("DWQX-1") {
		t.Error("cases from files (PQA-3) are not Jira issues; only DWQ-… get comments")
	}
	if !(Config{Project: "DWQ"}).inJira("DWQ-130") {
		t.Error("without jiraProject the registry key is the Jira key, as before")
	}
}
