package mcpserver

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"gitlab-mcp/internal/config"
	"gitlab-mcp/internal/gitlab"
)

var readToolNames = []string{
	"gitlab_get_metadata", "gitlab_get_current_user", "gitlab_list_projects", "gitlab_get_project",
	"gitlab_list_issues", "gitlab_get_issue", "gitlab_list_issue_notes",
	"gitlab_list_merge_requests", "gitlab_get_merge_request", "gitlab_list_merge_request_notes",
	"gitlab_list_repository_tree", "gitlab_get_file", "gitlab_list_branches", "gitlab_list_commits", "gitlab_get_commit",
	"gitlab_list_pipelines", "gitlab_get_pipeline", "gitlab_list_pipeline_jobs", "gitlab_search",
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

func listTools(t *testing.T, server *mcp.Server) map[string]bool {
	t.Helper()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(t.Context(), serverTransport, nil)
	if err != nil {
		t.Fatalf("server.Connect() error = %v", err)
	}
	defer func() { _ = serverSession.Close() }()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	clientSession, err := client.Connect(t.Context(), clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect() error = %v", err)
	}
	defer func() { _ = clientSession.Close() }()
	result, err := clientSession.ListTools(t.Context(), &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	names := make(map[string]bool, len(result.Tools))
	for _, tool := range result.Tools {
		names[tool.Name] = true
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
				if !names[name] {
					t.Errorf("missing read tool %q", name)
				}
			}
			for _, name := range writeToolNames {
				if names[name] != tt.wantWrite {
					t.Errorf("tool %q present=%v, want %v", name, names[name], tt.wantWrite)
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
