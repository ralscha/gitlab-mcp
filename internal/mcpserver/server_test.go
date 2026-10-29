package mcpserver

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"gitlab-mcp/internal/config"
	"gitlab-mcp/internal/gitlab"
)

var readToolNames = []string{
	"gitlab_get_metadata", "gitlab_get_current_user", "gitlab_list_projects", "gitlab_get_project",
	"gitlab_list_issues", "gitlab_get_issue", "gitlab_list_issue_notes",
	"gitlab_list_merge_requests", "gitlab_get_merge_request", "gitlab_list_merge_request_diffs", "gitlab_list_merge_request_notes",
	"gitlab_list_repository_tree", "gitlab_get_file", "gitlab_list_branches", "gitlab_list_commits", "gitlab_get_commit",
	"gitlab_list_pipelines", "gitlab_get_pipeline", "gitlab_list_pipeline_jobs", "gitlab_search",
	"gitlab_get_job", "gitlab_get_job_trace",
}

var writeToolNames = []string{
	"gitlab_create_issue", "gitlab_update_issue", "gitlab_add_issue_note",
	"gitlab_create_merge_request", "gitlab_update_merge_request", "gitlab_add_merge_request_note", "gitlab_accept_merge_request",
	"gitlab_create_branch", "gitlab_create_commit", "gitlab_create_pipeline", "gitlab_retry_pipeline",
}

func testClient(t *testing.T) *gitlab.Client {
	t.Helper()
	client, err := gitlab.NewClient("https://gitlab.example.com", "token", nil)
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func connectTestClient(t *testing.T, server *mcp.Server) *mcp.ClientSession {
	t.Helper()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(t.Context(), serverTransport, nil)
	if err != nil {
		t.Fatalf("server.Connect() error = %v", err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	clientSession, err := client.Connect(t.Context(), clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect() error = %v", err)
	}
	t.Cleanup(func() { _ = clientSession.Close() })
	return clientSession
}

func listTools(t *testing.T, server *mcp.Server) map[string]*mcp.Tool {
	t.Helper()
	clientSession := connectTestClient(t, server)
	result, err := clientSession.ListTools(t.Context(), &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	names := make(map[string]*mcp.Tool, len(result.Tools))
	for _, tool := range result.Tools {
		names[tool.Name] = tool
	}
	return names
}

func TestServerToolModes(t *testing.T) {
	tests := []struct {
		name      string
		mode      config.Mode
		wantWrite bool
	}{
		{"readonly", config.ModeReadOnly, false},
		{"readwrite", config.ModeReadWrite, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			names := listTools(t, NewServer(&config.Config{Mode: tt.mode}, testClient(t)))
			for _, name := range readToolNames {
				if names[name] == nil {
					t.Errorf("missing read tool %q", name)
				}
			}
			for _, name := range writeToolNames {
				if present := names[name] != nil; present != tt.wantWrite {
					t.Errorf("tool %q present=%v, want %v", name, present, tt.wantWrite)
				}
			}
			wantCount := len(readToolNames)
			if tt.wantWrite {
				wantCount += len(writeToolNames)
			}
			if len(names) != wantCount {
				t.Errorf("tool count = %d, want %d", len(names), wantCount)
			}
		})
	}
}

func TestToolsAdvertiseDestructiveChanges(t *testing.T) {
	tools := listTools(t, NewServer(&config.Config{Mode: config.ModeReadWrite}, testClient(t)))
	for _, name := range []string{"gitlab_update_issue", "gitlab_update_merge_request", "gitlab_add_issue_note", "gitlab_add_merge_request_note"} {
		tool := tools[name]
		if tool == nil || tool.Annotations == nil {
			t.Fatalf("tool %s is missing annotations", name)
		}
		if tool.Annotations.ReadOnlyHint || tool.Annotations.DestructiveHint == nil || !*tool.Annotations.DestructiveHint {
			t.Errorf("tool %s must advertise that it can replace existing data: %+v", name, tool.Annotations)
		}
		if tool.Annotations.IdempotentHint {
			t.Errorf("tool %s can execute quick actions and must not promise idempotence", name)
		}
	}
}

func TestGetFileDefaultRef(t *testing.T) {
	for _, ref := range []string{"", "release/v1"} {
		name := ref
		if name == "" {
			name = "default branch"
		}
		t.Run(name, func(t *testing.T) {
			wantRef := ref
			if wantRef == "" {
				wantRef = "HEAD"
			}
			api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.URL.Query().Get("ref"); got != wantRef {
					t.Errorf("ref = %q, want %q", got, wantRef)
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(gitlab.RepositoryFile{
					FileName: "README.md", FilePath: "README.md", Size: 5,
					Content: base64.StdEncoding.EncodeToString([]byte("hello")), Encoding: "base64", Ref: wantRef,
				})
			}))
			t.Cleanup(api.Close)
			client, err := gitlab.NewClient(api.URL, "token", api.Client())
			if err != nil {
				t.Fatal(err)
			}
			session := connectTestClient(t, NewServer(&config.Config{Mode: config.ModeReadOnly}, client))
			args := map[string]any{"project": "group/project", "file_path": "README.md"}
			if ref != "" {
				args["ref"] = ref
			}
			result, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "gitlab_get_file", Arguments: args})
			if err != nil {
				t.Fatalf("CallTool() error = %v", err)
			}
			if result.IsError {
				t.Fatalf("CallTool() result = %+v", result)
			}
			output, ok := result.StructuredContent.(map[string]any)
			if !ok || output["content"] != "hello" || output["ref"] != wantRef {
				t.Errorf("unexpected file output: %#v", result.StructuredContent)
			}
		})
	}
}

func TestCreateCommitEmptyContent(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Actions []map[string]any `json:"actions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode commit: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if len(payload.Actions) != 1 {
			t.Errorf("actions = %#v, want one action", payload.Actions)
		} else if content, present := payload.Actions[0]["content"]; !present || content != "" {
			t.Errorf("empty content was lost: %#v", payload.Actions[0])
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(gitlab.Commit{ID: "abc123"})
	}))
	t.Cleanup(api.Close)
	client, err := gitlab.NewClient(api.URL, "token", api.Client())
	if err != nil {
		t.Fatal(err)
	}
	session := connectTestClient(t, NewServer(&config.Config{Mode: config.ModeReadWrite}, client))
	for _, includeContent := range []bool{true, false} {
		action := map[string]any{"action": "update", "file_path": "empty.txt"}
		if includeContent {
			action["content"] = ""
		}
		result, err := session.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "gitlab_create_commit", Arguments: map[string]any{
				"project": "group/project", "branch": "main", "commit_message": "Empty file",
				"actions": []any{action},
			},
		})
		if err != nil {
			t.Fatalf("CallTool() error = %v", err)
		}
		if result.IsError != !includeContent {
			t.Errorf("includeContent=%v: IsError=%v, content=%v", includeContent, result.IsError, result.Content)
		}
	}
}
