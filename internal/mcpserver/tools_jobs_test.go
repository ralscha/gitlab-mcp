package mcpserver

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"gitlab-mcp/internal/config"
	"gitlab-mcp/internal/gitlab"
)

func TestJobTools(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.EscapedPath() {
		case "/api/v4/projects/group%2Fproject/jobs/42":
			_, _ = io.WriteString(w, `{"id":42,"name":"test","stage":"test","status":"failed"}`)
		case "/api/v4/projects/group%2Fproject/jobs/42/trace":
			_, _ = io.WriteString(w, "FAIL: example\n")
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"message":"404 Job Not Found"}`)
		}
	}))
	t.Cleanup(server.Close)
	client, err := gitlab.NewClient(server.URL, "token", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	session := connectTestClient(t, NewServer(&config.Config{Mode: config.ModeReadOnly}, client))
	for _, tt := range []struct {
		name    string
		jobID   int
		wantErr bool
	}{
		{name: "gitlab_get_job", jobID: 42},
		{name: "gitlab_get_job_trace", jobID: 42},
		{name: "gitlab_get_job_trace", jobID: 999, wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result, err := session.CallTool(t.Context(), &mcp.CallToolParams{
				Name: tt.name, Arguments: map[string]any{"project": "group/project", "job_id": tt.jobID},
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.IsError != tt.wantErr {
				t.Fatalf("IsError=%v, content=%v", result.IsError, result.Content)
			}
			if tt.wantErr {
				return
			}
			encoded, err := json.Marshal(result.StructuredContent)
			if err != nil {
				t.Fatal(err)
			}
			if tt.name == "gitlab_get_job" {
				var job gitlab.Job
				if err := json.Unmarshal(encoded, &job); err != nil || job.ID != 42 || job.Status != "failed" {
					t.Fatalf("job=%s, err=%v", encoded, err)
				}
			} else {
				var trace JobTrace
				if err := json.Unmarshal(encoded, &trace); err != nil || trace.JobID != 42 || trace.Trace != "FAIL: example\n" {
					t.Fatalf("trace=%s, err=%v", encoded, err)
				}
			}
		})
	}
}
