package gitlab

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func testClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := NewClient(server.URL, "test-token", server.Client())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	return client
}

func TestGetIssueEncodesProjectPathAndAuthenticates(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s", r.Method)
		}
		if r.RequestURI != "/api/v4/projects/group%2Fproject/issues/7" {
			t.Errorf("RequestURI = %q", r.RequestURI)
		}
		if r.Header.Get("PRIVATE-TOKEN") != "test-token" {
			t.Errorf("PRIVATE-TOKEN = %q", r.Header.Get("PRIVATE-TOKEN"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":10,"iid":7,"project_id":3,"title":"bug","state":"opened"}`))
	})
	issue, err := client.GetIssue(t.Context(), "group/project", 7)
	if err != nil {
		t.Fatalf("GetIssue() error = %v", err)
	}
	if issue.IID != 7 || issue.Title != "bug" {
		t.Errorf("issue = %#v", issue)
	}
}

func TestListIssuesPaginationAndFilters(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("labels") != "bug,backend" {
			t.Errorf("labels = %q", r.URL.Query().Get("labels"))
		}
		if r.URL.Query().Get("page") != "2" || r.URL.Query().Get("per_page") != "100" {
			t.Errorf("query = %v", r.URL.Query())
		}
		w.Header().Set("X-Total", "201")
		w.Header().Set("X-Total-Pages", "3")
		w.Header().Set("X-Next-Page", "3")
		_, _ = w.Write([]byte(`[{"iid":4,"title":"one","state":"opened"}]`))
	})
	issues, page, err := client.ListIssues(t.Context(), ListIssuesOptions{Project: "1", Labels: []string{"bug", "backend"}, Page: 2, PerPage: 999})
	if err != nil {
		t.Fatalf("ListIssues() error = %v", err)
	}
	if len(issues) != 1 || page.Total != 201 || page.NextPage != 3 || page.LastPage {
		t.Errorf("issues=%v page=%#v", issues, page)
	}
}

func TestGetFileDecodesContent(t *testing.T) {
	content := "hello, GitLab\n"
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI != "/api/v4/projects/team%2Frepo/repository/files/docs%2Freadme.txt?ref=main" {
			t.Errorf("RequestURI = %q", r.RequestURI)
		}
		_ = json.NewEncoder(w).Encode(RepositoryFile{FileName: "readme.txt", FilePath: "docs/readme.txt", Size: len(content), Encoding: "base64", Content: base64.StdEncoding.EncodeToString([]byte(content)), Ref: "main"})
	})
	file, err := client.GetFile(t.Context(), "team/repo", "docs/readme.txt", "main")
	if err != nil {
		t.Fatalf("GetFile() error = %v", err)
	}
	if file.Content != content {
		t.Errorf("Content = %q", file.Content)
	}
}

func TestCreateCommitPayload(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["branch"] != "feature" || body["commit_message"] != "change files" {
			t.Errorf("body = %#v", body)
		}
		actions, ok := body["actions"].([]any)
		if !ok || len(actions) != 2 {
			t.Errorf("actions = %#v", body["actions"])
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"abc","short_id":"abc","title":"change files"}`))
	})
	commit, err := client.CreateCommit(t.Context(), "1", CreateCommitInput{Branch: "feature", CommitMessage: "change files", Actions: []CommitAction{{Action: "update", FilePath: "a.txt", Content: "a"}, {Action: "delete", FilePath: "old.txt"}}})
	if err != nil {
		t.Fatalf("CreateCommit() error = %v", err)
	}
	if commit.ID != "abc" {
		t.Errorf("commit = %#v", commit)
	}
}

func TestCreateCommitRejectsInvalidAction(t *testing.T) {
	client, err := NewClient("https://gitlab.example.com", "token", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.CreateCommit(t.Context(), "1", CreateCommitInput{Branch: "main", CommitMessage: "x", Actions: []CommitAction{{Action: "explode", FilePath: "a"}}})
	if err == nil || !strings.Contains(err.Error(), "invalid action") {
		t.Fatalf("error = %v", err)
	}
}

func TestAPIErrorParsesStructuredMessage(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":{"title":["has already been taken"]}}`))
	})
	_, err := client.GetProject(t.Context(), "1")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 400 || !strings.Contains(apiErr.Message, "title") {
		t.Fatalf("error = %#v", err)
	}
}

func TestRetriesRateLimitResponse(t *testing.T) {
	var calls atomic.Int32
	client := testClient(t, func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{"id":1,"name":"ok"}`))
	})
	project, err := client.GetProject(t.Context(), "1")
	if err != nil || project.Name != "ok" || calls.Load() != 2 {
		t.Fatalf("project=%#v calls=%d err=%v", project, calls.Load(), err)
	}
}

func TestRefusesCrossOriginRedirect(t *testing.T) {
	var redirected atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { redirected.Add(1) }))
	defer target.Close()
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, http.StatusFound) })
	_, err := client.GetProject(t.Context(), "1")
	if err == nil || !strings.Contains(err.Error(), "cross-origin redirect") {
		t.Fatalf("error = %v", err)
	}
	if redirected.Load() != 0 {
		t.Fatal("redirect target was contacted")
	}
}
