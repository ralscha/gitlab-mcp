package main

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestStdioTransportListTools(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "run", ".", "--gitlab-token=test-token", "--mode=readonly", "--transport=stdio")
	cmd.Env = os.Environ()
	client := mcp.NewClient(&mcp.Implementation{Name: "smoke-test", Version: "0"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer func() { _ = session.Close() }()
	result, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	foundRead := false
	for _, tool := range result.Tools {
		if tool.Name == "gitlab_get_project" {
			foundRead = true
		}
		if tool.Name == "gitlab_create_issue" {
			t.Errorf("write tool present in readonly mode")
		}
	}
	if !foundRead {
		t.Error("gitlab_get_project not registered")
	}
}
